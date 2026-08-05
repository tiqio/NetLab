package console

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/netlab/netlab/internal/domain"
	qemuRuntime "github.com/netlab/netlab/internal/runtime/qemu"
	"golang.org/x/crypto/ssh"
)

type SSHAddressSource interface {
	ListNodeNATLeasePaths(context.Context, domain.ID) ([]string, error)
}

type SSHCredentialSource interface {
	Credentials(context.Context, string) (qemuRuntime.BootstrapCredentials, error)
}

type SSHBackend struct {
	addresses    SSHAddressSource
	credentials  SSHCredentialSource
	timeout      time.Duration
	probeTimeout time.Duration
	port         string
}

func NewSSHBackend(addresses SSHAddressSource, credentials SSHCredentialSource) *SSHBackend {
	return &SSHBackend{addresses: addresses, credentials: credentials, timeout: 5 * time.Second, probeTimeout: 300 * time.Millisecond, port: "22"}
}

func (b *SSHBackend) Available(ctx context.Context, node domain.Node) error {
	if node.ObservedState != domain.ObservedRunning {
		return fmt.Errorf("SSH console requires a running node")
	}
	if b.addresses == nil || b.credentials == nil {
		return fmt.Errorf("SSH console resolver is unavailable")
	}
	seedPath, _ := node.Config["seed_iso"].(string)
	if seedPath == "" {
		return fmt.Errorf("SSH credentials are unavailable for this node")
	}
	if _, err := b.credentials.Credentials(ctx, seedPath); err != nil {
		return fmt.Errorf("resolve SSH credentials: %w", err)
	}
	addresses, err := b.ResolveAddresses(ctx, node)
	if err != nil {
		return err
	}
	dialer := net.Dialer{Timeout: b.probeTimeout}
	for _, address := range addresses {
		connection, dialErr := dialer.DialContext(ctx, "tcp", net.JoinHostPort(address, b.port))
		if dialErr == nil {
			_ = connection.Close()
			return nil
		}
	}
	return fmt.Errorf("SSH endpoint is not reachable from the NetLab host")
}

func (b *SSHBackend) OpenConsole(ctx context.Context, node domain.Node) (io.ReadWriteCloser, error) {
	if node.ObservedState != domain.ObservedRunning {
		return nil, fmt.Errorf("SSH console requires a running node")
	}
	if b.addresses == nil || b.credentials == nil {
		return nil, fmt.Errorf("SSH console resolver is unavailable")
	}
	seedPath, _ := node.Config["seed_iso"].(string)
	if seedPath == "" {
		return nil, fmt.Errorf("SSH credentials are unavailable for this node")
	}
	credentials, err := b.credentials.Credentials(ctx, seedPath)
	if err != nil {
		return nil, fmt.Errorf("resolve SSH credentials: %w", err)
	}
	addresses, err := b.ResolveAddresses(ctx, node)
	if err != nil {
		return nil, err
	}
	var failures []string
	for _, address := range addresses {
		connection, openErr := b.open(ctx, net.JoinHostPort(address, b.port), credentials)
		if openErr == nil {
			return connection, nil
		}
		failures = append(failures, address+": "+openErr.Error())
	}
	return nil, fmt.Errorf("SSH is not ready on the node addresses (%s)", strings.Join(failures, "; "))
}

func (b *SSHBackend) ResolveAddresses(ctx context.Context, node domain.Node) ([]string, error) {
	values := configuredAddresses(node)
	leasePaths, err := b.addresses.ListNodeNATLeasePaths(ctx, node.ID)
	if err != nil {
		return nil, fmt.Errorf("resolve NAT lease files: %w", err)
	}
	macs := nodeMACAddresses(node)
	for _, path := range leasePaths {
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		values = append(values, addressesFromLease(body, macs, time.Now())...)
	}
	values = uniqueStrings(values)
	if len(values) == 0 {
		return nil, fmt.Errorf("no SSH address found; attach a DHCP-enabled interface to a running NAT bridge or configure a static address")
	}
	return values, nil
}

func (b *SSHBackend) open(ctx context.Context, address string, credentials qemuRuntime.BootstrapCredentials) (io.ReadWriteCloser, error) {
	dialer := net.Dialer{Timeout: b.timeout}
	networkConnection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	sshConfig := &ssh.ClientConfig{
		User:            credentials.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(credentials.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         b.timeout,
	}
	clientConnection, channels, requests, err := ssh.NewClientConn(networkConnection, address, sshConfig)
	if err != nil {
		_ = networkConnection.Close()
		return nil, err
	}
	client := ssh.NewClient(clientConnection, channels, requests)
	session, err := client.NewSession()
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	terminal, peer := net.Pipe()
	session.Stdin = peer
	session.Stdout = peer
	session.Stderr = peer
	if err = session.RequestPty("xterm-256color", 40, 120, ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 115200, ssh.TTY_OP_OSPEED: 115200}); err != nil {
		_ = peer.Close()
		_ = terminal.Close()
		_ = session.Close()
		_ = client.Close()
		return nil, err
	}
	if err = session.Shell(); err != nil {
		_ = peer.Close()
		_ = terminal.Close()
		_ = session.Close()
		_ = client.Close()
		return nil, err
	}
	return &sshConsole{Conn: terminal, peer: peer, session: session, client: client}, nil
}

type sshConsole struct {
	net.Conn
	peer      net.Conn
	session   *ssh.Session
	client    *ssh.Client
	closeOnce sync.Once
}

func (c *sshConsole) Close() error {
	c.closeOnce.Do(func() {
		_ = c.Conn.Close()
		_ = c.peer.Close()
		_ = c.session.Close()
		_ = c.client.Close()
	})
	return nil
}

func configuredAddresses(node domain.Node) []string {
	raw, _ := node.Config["network_interfaces"].([]any)
	if direct, ok := node.Config["network_interfaces"].([]map[string]any); ok {
		raw = make([]any, len(direct))
		for index := range direct {
			raw[index] = direct[index]
		}
	}
	var values []string
	for _, item := range raw {
		entry, _ := item.(map[string]any)
		addresses, _ := entry["addresses"].([]any)
		if direct, ok := entry["addresses"].([]string); ok {
			for _, address := range direct {
				if prefix, err := netip.ParsePrefix(address); err == nil {
					values = append(values, prefix.Addr().String())
				}
			}
			continue
		}
		for _, rawAddress := range addresses {
			address, _ := rawAddress.(string)
			if prefix, err := netip.ParsePrefix(address); err == nil {
				values = append(values, prefix.Addr().String())
			}
		}
	}
	return values
}

func nodeMACAddresses(node domain.Node) map[string]bool {
	result := map[string]bool{}
	raw, _ := node.Config["interfaces"].([]any)
	if direct, ok := node.Config["interfaces"].([]map[string]any); ok {
		raw = make([]any, len(direct))
		for index := range direct {
			raw[index] = direct[index]
		}
	}
	for _, item := range raw {
		entry, _ := item.(map[string]any)
		mac, _ := entry["mac_address"].(string)
		if mac != "" {
			result[strings.ToLower(mac)] = true
		}
	}
	return result
}

func addressesFromLease(body []byte, macs map[string]bool, now time.Time) []string {
	var result []string
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || !macs[strings.ToLower(fields[1])] {
			continue
		}
		var expires int64
		if _, err := fmt.Sscan(fields[0], &expires); err != nil || (expires != 0 && expires <= now.Unix()) {
			continue
		}
		if address, err := netip.ParseAddr(fields[2]); err == nil {
			result = append(result, address.String())
		}
	}
	return result
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
