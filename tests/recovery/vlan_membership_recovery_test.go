package recovery_test

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
)

func TestVLANMembershipPersistsAcrossTenRuntimeRestarts(t *testing.T) {
	if os.Getenv("NETLAB_PRIVILEGED_TESTS") != "1" {
		t.Skip("set NETLAB_PRIVILEGED_TESTS=1 to run VLAN recovery acceptance")
	}
	if os.Geteuid() != 0 {
		t.Fatal("VLAN recovery acceptance requires root")
	}
	for _, tool := range []string{"ip", "bridge"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("acceptance host missing %s: %v", tool, err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	if len(suffix) > 6 {
		suffix = suffix[len(suffix)-6:]
	}
	object := domain.NetworkObject{ID: domain.ID("vr-" + suffix), Kind: domain.NetworkSwitchL2, Config: map[string]any{
		"vlan_filtering": true,
		"ports": []any{
			map[string]any{"name": "access10", "pvid": 10, "tagged": []any{}},
			map[string]any{"name": "trunk0", "pvid": 0, "tagged": []any{10, 20}},
			map[string]any{"name": "unused0", "pvid": 30, "tagged": []any{}},
		},
	}}
	namespace := linuxnet.SwitchL2NamespaceName(object.ID)
	vlanRecoveryCommand(t, ctx, "ip", "netns", "add", namespace)
	vlanRecoveryCommand(t, ctx, "ip", "-n", namespace, "link", "set", "lo", "up")
	t.Cleanup(func() { _ = exec.Command("ip", "netns", "delete", namespace).Run() })
	for _, port := range []string{"access10", "trunk0"} {
		host := "h" + port
		if len(host) > 15 {
			host = host[:15]
		}
		vlanRecoveryCommand(t, ctx, "ip", "link", "add", host, "type", "veth", "peer", "name", port)
		t.Cleanup(func() { _ = exec.Command("ip", "link", "delete", host).Run() })
		vlanRecoveryCommand(t, ctx, "ip", "link", "set", port, "netns", namespace)
		vlanRecoveryCommand(t, ctx, "ip", "link", "set", host, "up")
	}

	for cycle := 1; cycle <= 10; cycle++ {
		runtime, err := linuxnet.NewSwitchL2Runtime(nil)
		if err != nil {
			t.Fatal(err)
		}
		if err = runtime.Configure(ctx, object); err != nil {
			t.Fatalf("restart cycle %d configure: %v", cycle, err)
		}
		converged, diagnostics, diagnosticsErr := runtime.ConfigurationConverged(ctx, object)
		if diagnosticsErr != nil || !converged {
			t.Fatalf("restart cycle %d membership=%+v err=%v", cycle, diagnostics, diagnosticsErr)
		}
		ports := diagnostics["observed"].(map[string]any)["ports"].([]linuxnet.SwitchL2PortObservation)
		if len(ports) != 3 || ports[2].Name != "unused0" || ports[2].Attached {
			t.Fatalf("restart cycle %d logical port observation=%+v", cycle, ports)
		}
	}
}

func vlanRecoveryCommand(t *testing.T, ctx context.Context, name string, args ...string) {
	t.Helper()
	body, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v: %s", name, strings.Join(args, " "), err, body)
	}
}
