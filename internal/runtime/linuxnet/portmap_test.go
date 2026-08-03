package linuxnet

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

type fakeNft struct {
	commands []string
	listing  string
}

func (f *fakeNft) Run(_ context.Context, name string, args ...string) error {
	f.commands = append(f.commands, name+" "+strings.Join(args, " "))
	return nil
}
func (f *fakeNft) Output(context.Context, string, ...string) ([]byte, error) {
	return []byte(f.listing), nil
}

func TestPortMappingRulesOwnershipAndCleanup(t *testing.T) {
	runner := &fakeNft{}
	mapper, _ := NewPortMapper(runner)
	mapping := domain.PortMapping{ID: "map-1", Protocol: "tcp", HostAddress: "127.0.0.1", HostPort: 0, GuestAddress: "192.0.2.10", GuestPort: 22}
	mapping.HostPort = freeTCPPort(t)
	if err := mapper.Apply(context.Background(), mapping); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(runner.commands, "\n")
	if !strings.Contains(commands, "ip daddr 127.0.0.1 tcp dport") || !strings.Contains(commands, "dnat ip to 192.0.2.10:22") || !strings.Contains(commands, `comment "netlab:map-1"`) {
		t.Fatalf("commands=%s", commands)
	}
	runner.listing = `tcp dport 2222 comment "netlab:map-1" # handle 7`
	if err := mapper.Delete(context.Background(), mapping.ID); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(runner.commands, "\n"), "handle 7") {
		t.Fatal("owned rule not deleted")
	}
}

func TestIPv6PortMappingUsesExplicitAddressFamily(t *testing.T) {
	runner := &fakeNft{}
	mapper, _ := NewPortMapper(runner)
	mapping := domain.PortMapping{ID: "map-v6", Protocol: "tcp", HostAddress: "::1", HostPort: freeTCPPort(t), GuestAddress: "2001:db8::10", GuestPort: 22}
	if err := mapper.Apply(context.Background(), mapping); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(runner.commands, "\n")
	if !strings.Contains(commands, "ip6 daddr ::1 tcp dport") || !strings.Contains(commands, "dnat ip6 to [2001:db8::10]:22") || !strings.Contains(commands, "postrouting ip6 daddr 2001:db8::10") {
		t.Fatalf("commands=%s", commands)
	}
}

func freeTCPPort(t *testing.T) int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	return port
}
