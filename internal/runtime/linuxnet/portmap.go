package linuxnet

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"

	"github.com/netlab/netlab/internal/domain"
)

type NftRunner interface {
	Run(context.Context, string, ...string) error
	Output(context.Context, string, ...string) ([]byte, error)
}

type ExecNftRunner struct{}

func (ExecNftRunner) Run(ctx context.Context, name string, args ...string) error {
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %s: %w", name, strings.Join(args, " "), strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (ExecNftRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

type PortMapper struct {
	nft    string
	runner NftRunner
}

func NewPortMapper(runner NftRunner) (*PortMapper, error) {
	if runner != nil {
		return &PortMapper{nft: "nft", runner: runner}, nil
	}
	path, err := exec.LookPath("nft")
	if err != nil {
		return nil, err
	}
	return &PortMapper{nft: path, runner: ExecNftRunner{}}, nil
}

func CheckHostPort(address, protocol string, port int) error {
	if address == "" {
		address = "0.0.0.0"
	}
	endpoint := net.JoinHostPort(address, fmt.Sprint(port))
	if protocol == "udp" {
		connection, err := net.ListenPacket("udp", endpoint)
		if err != nil {
			return err
		}
		return connection.Close()
	}
	listener, err := net.Listen("tcp", endpoint)
	if err != nil {
		return err
	}
	return listener.Close()
}

func (m *PortMapper) CheckHostPort(address, protocol string, port int) error {
	return CheckHostPort(address, protocol, port)
}

func (m *PortMapper) EnsureBase(ctx context.Context) error {
	commands := [][]string{
		{"add", "table", "inet", "netlab"},
		{"add", "chain", "inet", "netlab", "prerouting", "{", "type", "nat", "hook", "prerouting", "priority", "dstnat", ";", "}"},
		{"add", "chain", "inet", "netlab", "output", "{", "type", "nat", "hook", "output", "priority", "-100", ";", "}"},
		{"add", "chain", "inet", "netlab", "postrouting", "{", "type", "nat", "hook", "postrouting", "priority", "srcnat", ";", "}"},
	}
	for _, args := range commands {
		_ = m.runner.Run(ctx, m.nft, args...)
	}
	return nil
}

func (m *PortMapper) Apply(ctx context.Context, mapping domain.PortMapping) error {
	if mapping.Protocol != "tcp" && mapping.Protocol != "udp" {
		return fmt.Errorf("protocol must be tcp or udp")
	}
	hostIP := net.ParseIP(mapping.HostAddress)
	guestIP := net.ParseIP(mapping.GuestAddress)
	if mapping.HostPort < 1 || mapping.HostPort > 65535 || mapping.GuestPort < 1 || mapping.GuestPort > 65535 || hostIP == nil || guestIP == nil {
		return fmt.Errorf("invalid port mapping")
	}
	if (hostIP.To4() == nil) != (guestIP.To4() == nil) {
		return fmt.Errorf("host and guest addresses must use the same IP family")
	}
	if err := CheckHostPort(mapping.HostAddress, mapping.Protocol, mapping.HostPort); err != nil {
		return domain.Problem{Code: "port_conflict", Message: err.Error(), Retryable: false}
	}
	_ = m.EnsureBase(ctx)
	comment := `"netlab:` + string(mapping.ID) + `"`
	destination := net.JoinHostPort(mapping.GuestAddress, fmt.Sprint(mapping.GuestPort))
	addressFamily := "ip"
	if net.ParseIP(mapping.GuestAddress).To4() == nil {
		addressFamily = "ip6"
	}
	for _, chain := range []string{"prerouting", "output"} {
		if m.hasOwnedRule(ctx, chain, mapping.ID) {
			continue
		}
		args := []string{"add", "rule", "inet", "netlab", chain}
		if !hostIP.IsUnspecified() {
			args = append(args, addressFamily, "daddr", mapping.HostAddress)
		}
		args = append(args, mapping.Protocol, "dport", fmt.Sprint(mapping.HostPort), "dnat", addressFamily, "to", destination, "comment", comment)
		if err := m.runner.Run(ctx, m.nft, args...); err != nil {
			_ = m.Delete(ctx, mapping.ID)
			return err
		}
	}
	if m.hasOwnedRule(ctx, "postrouting", mapping.ID) {
		return nil
	}
	return m.runner.Run(ctx, m.nft, "add", "rule", "inet", "netlab", "postrouting", addressFamily, "daddr", mapping.GuestAddress, mapping.Protocol, "dport", fmt.Sprint(mapping.GuestPort), "masquerade", "comment", comment)
}

func (m *PortMapper) hasOwnedRule(ctx context.Context, chain string, id domain.ID) bool {
	body, err := m.runner.Output(ctx, m.nft, "list", "chain", "inet", "netlab", chain)
	return err == nil && strings.Contains(string(body), "netlab:"+string(id))
}

func (m *PortMapper) Delete(ctx context.Context, id domain.ID) error {
	comment := "netlab:" + regexp.QuoteMeta(string(id))
	for _, chain := range []string{"prerouting", "output", "postrouting"} {
		body, err := m.runner.Output(ctx, m.nft, "-a", "list", "chain", "inet", "netlab", chain)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(strings.NewReader(string(body)))
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.Contains(line, comment) && !strings.Contains(line, "netlab:"+string(id)) {
				continue
			}
			fields := strings.Fields(line)
			for index := range fields {
				if fields[index] == "handle" && index+1 < len(fields) {
					_ = m.runner.Run(ctx, m.nft, "delete", "rule", "inet", "netlab", chain, "handle", fields[index+1])
				}
			}
		}
	}
	return nil
}
