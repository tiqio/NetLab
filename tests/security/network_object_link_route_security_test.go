package security_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
)

type securityExecutor struct {
	programs []string
	args     [][]string
}

func (e *securityExecutor) Run(_ context.Context, name string, args ...string) error {
	e.programs = append(e.programs, name)
	e.args = append(e.args, append([]string(nil), args...))
	return nil
}

func (e *securityExecutor) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, context.Canceled
}

type securityExportReader struct {
	snapshot domain.TopologySnapshot
}

func (r securityExportReader) Snapshot(context.Context, domain.ID) (domain.TopologySnapshot, error) {
	return r.snapshot, nil
}

func TestNetworkValuesCannotBecomeInterpolatedShellCommands(t *testing.T) {
	for _, value := range []string{"swp1;touch /tmp/netlab-owned", "$(id)", "eth0 && reboot", "a|cat /etc/shadow"} {
		if err := domain.ValidateNetworkObjectPortName(value); err == nil {
			t.Fatalf("accepted executable endpoint name %q", value)
		}
		interfaces := []domain.NodeNetworkInterfaceSettings{{Name: "eth0", Addresses: []string{"192.0.2.10/24"}, Routes: []domain.RouteConfig{{Destination: value, Gateway: "192.0.2.1"}}}}
		if err := domain.ValidateNodeNetworkInterfaces(interfaces); err == nil {
			t.Fatalf("accepted executable route destination %q", value)
		}
	}

	executor := &securityExecutor{}
	dataPlane, err := linuxnet.NewDataPlane(executor)
	if err != nil {
		t.Fatal(err)
	}
	link := domain.Link{ID: "safe-link"}
	if err = dataPlane.EnsureLink(context.Background(), link, domain.Interface{ID: "safe-a"}, domain.Interface{ID: "safe-b"}); err != nil {
		t.Fatal(err)
	}
	for _, program := range executor.programs {
		if program == "sh" || program == "bash" || strings.ContainsAny(program, " ;|&$`") {
			t.Fatalf("runtime invoked a shell program: %q", program)
		}
	}
}

func TestExportsOmitRuntimeSecretsAndPacketPayloads(t *testing.T) {
	snapshot := domain.TopologySnapshot{
		Laboratory: domain.Laboratory{Name: "security", RecoveryPolicy: domain.RecoveryRemainStopped},
		Nodes: []domain.Node{{ID: "node", Name: "docker", Kind: "docker", Config: map[string]any{
			"password": "target-secret", "container_pid": 4242, "namespace_name": "netlab-secret", "packet_payload": "secret-packet",
			"network_interfaces": []any{map[string]any{"name": "eth0", "routes": []any{map[string]any{"destination": "198.51.100.0/24", "gateway": "192.0.2.1"}}}},
		}}},
		NetworkObjects:     []domain.NetworkObject{{ID: "a", Name: "A", Kind: domain.NetworkSwitchL2, Config: map[string]any{"credential": "target-secret", "runtime_interface": "nva-secret"}}, {ID: "b", Name: "B", Kind: domain.NetworkSwitchL2}},
		NetworkObjectLinks: []domain.NetworkObjectLink{{ID: "runtime-link-id", ObjectAID: "a", PortAName: "swp1", ObjectBID: "b", PortBName: "swp1", DesiredState: "connected"}},
	}
	bundle, err := command.NewExportService(securityExportReader{snapshot: snapshot}, nil).Build(context.Background(), "lab")
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, prohibited := range []string{"target-secret", "secret-packet", "netlab-secret", "nva-secret", "runtime-link-id", "container_pid", "namespace_name", "packet_payload"} {
		if strings.Contains(text, prohibited) {
			t.Fatalf("export leaked %q: %s", prohibited, text)
		}
	}
	if !strings.Contains(text, "198.51.100.0/24") || !strings.Contains(text, "[REDACTED]") {
		t.Fatalf("export lost declared routes or redaction marker: %s", text)
	}
}
