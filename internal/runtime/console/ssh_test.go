package console

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
	qemuRuntime "github.com/netlab/netlab/internal/runtime/qemu"
)

type sshAddressSourceStub struct{}

func (sshAddressSourceStub) ListNodeNATLeasePaths(context.Context, domain.ID) ([]string, error) {
	return nil, nil
}

type sshCredentialSourceStub struct{}

func (sshCredentialSourceStub) Credentials(context.Context, string) (qemuRuntime.BootstrapCredentials, error) {
	return qemuRuntime.BootstrapCredentials{Username: "ubuntu", Password: "temporary-password"}, nil
}

func TestAddressesFromLeaseMatchesCurrentNodeMAC(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	body := []byte("1800000300 02:00:00:00:00:01 10.10.0.21 ubuntu *\n1799999999 02:00:00:00:00:01 10.10.0.20 expired *\n1800000300 02:00:00:00:00:02 10.10.0.22 other *\n")
	values := addressesFromLease(body, map[string]bool{"02:00:00:00:00:01": true}, now)
	if len(values) != 1 || values[0] != "10.10.0.21" {
		t.Fatalf("addresses=%v", values)
	}
}

func TestConfiguredAddressesUsesStaticInterfaceCIDRs(t *testing.T) {
	node := domain.Node{Config: map[string]any{"network_interfaces": []any{
		map[string]any{"name": "ens0", "addresses": []any{"192.0.2.10/24", "2001:db8::10/64"}},
	}}}
	values := configuredAddresses(node)
	if len(values) != 2 || values[0] != "192.0.2.10" || values[1] != "2001:db8::10" {
		t.Fatalf("addresses=%v", values)
	}
}

func TestSSHAvailabilityRequiresCredentialsAndReachableEndpoint(t *testing.T) {
	backend := NewSSHBackend(sshAddressSourceStub{}, sshCredentialSourceStub{})
	node := domain.Node{ID: "node-1", ObservedState: domain.ObservedRunning, Config: map[string]any{"network_interfaces": []any{map[string]any{"addresses": []any{"127.0.0.1/32"}}}}}
	if err := backend.Available(context.Background(), node); err == nil {
		t.Fatal("SSH was advertised without bootstrap credentials")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	backend.port = strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	node.Config["seed_iso"] = "/tmp/seed.iso"
	if err = backend.Available(context.Background(), node); err != nil {
		t.Fatalf("reachable SSH endpoint was not advertised: %v", err)
	}
}
