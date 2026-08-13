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

func TestNamespaceAttachmentRestoresForwardingAcrossTenRuntimeRestarts(t *testing.T) {
	if os.Getenv("NETLAB_PRIVILEGED_TESTS") != "1" {
		t.Skip("set NETLAB_PRIVILEGED_TESTS=1 to run attachment forwarding recovery")
	}
	if os.Geteuid() != 0 {
		t.Fatal("attachment forwarding recovery requires root")
	}
	for _, tool := range []string{"ip", "bridge", "ping"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("acceptance host missing %s: %v", tool, err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	if len(suffix) > 5 {
		suffix = suffix[len(suffix)-5:]
	}
	object := domain.NetworkObject{ID: domain.ID("af-" + suffix), Kind: domain.NetworkSwitchL2, Config: map[string]any{
		"vlan_filtering": true,
		"ports": []any{
			map[string]any{"name": "lan0", "pvid": 20, "tagged": []any{}},
			map[string]any{"name": "lan1", "pvid": 20, "tagged": []any{}},
		},
	}}
	attachments := []domain.NetworkAttachment{
		{ID: domain.ID("aa-" + suffix), InterfaceID: domain.ID("ia-" + suffix), PortName: "lan0", Config: map[string]any{"pvid": 20}},
		{ID: domain.ID("ab-" + suffix), InterfaceID: domain.ID("ib-" + suffix), PortName: "lan1", Config: map[string]any{"pvid": 20}},
	}
	clientNamespaces := []string{"nca" + suffix, "ncb" + suffix}
	clientPorts := []string{"eca" + suffix, "ecb" + suffix}
	addresses := []string{"192.0.2.11/24", "192.0.2.12/24"}
	for index, attachment := range attachments {
		hostPort := linuxnet.HostInterfaceName(attachment.InterfaceID)
		vlanRecoveryCommand(t, ctx, "ip", "netns", "add", clientNamespaces[index])
		vlanRecoveryCommand(t, ctx, "ip", "link", "add", hostPort, "type", "veth", "peer", "name", clientPorts[index])
		vlanRecoveryCommand(t, ctx, "ip", "link", "set", clientPorts[index], "netns", clientNamespaces[index])
		vlanRecoveryCommand(t, ctx, "ip", "link", "set", hostPort, "up")
		vlanRecoveryCommand(t, ctx, "ip", "-n", clientNamespaces[index], "link", "set", "lo", "up")
		vlanRecoveryCommand(t, ctx, "ip", "-n", clientNamespaces[index], "link", "set", clientPorts[index], "up")
		vlanRecoveryCommand(t, ctx, "ip", "-n", clientNamespaces[index], "address", "add", addresses[index], "dev", clientPorts[index])
	}
	t.Cleanup(func() {
		for _, attachment := range attachments {
			if runtime, err := linuxnet.NewDataPlane(nil); err == nil {
				_ = runtime.DeleteAttachment(context.Background(), attachment)
			}
			_ = exec.Command("ip", "link", "delete", linuxnet.HostInterfaceName(attachment.InterfaceID)).Run()
		}
		for _, namespace := range clientNamespaces {
			_ = exec.Command("ip", "netns", "delete", namespace).Run()
		}
		if runtime, err := linuxnet.NewSwitchL2Runtime(nil); err == nil {
			_ = runtime.Delete(context.Background(), object.ID)
		}
	})

	switchRuntime, err := linuxnet.NewSwitchL2Runtime(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = switchRuntime.Configure(ctx, object); err != nil {
		t.Fatal(err)
	}
	namespace := linuxnet.SwitchL2NamespaceName(object.ID)
	for cycle := 1; cycle <= 10; cycle++ {
		if cycle > 1 {
			for _, port := range []string{"lan0", "lan1"} {
				vlanRecoveryCommand(t, ctx, "bridge", "-n", namespace, "vlan", "del", "dev", port, "vid", "1-4094")
				vlanRecoveryCommand(t, ctx, "bridge", "-n", namespace, "vlan", "add", "dev", port, "vid", "1", "pvid", "untagged")
				vlanRecoveryCommand(t, ctx, "ip", "-n", namespace, "link", "set", port, "nomaster")
				vlanRecoveryCommand(t, ctx, "ip", "-n", namespace, "link", "set", port, "down")
			}
		}
		dataPlane, dataPlaneErr := linuxnet.NewDataPlane(nil)
		if dataPlaneErr != nil {
			t.Fatal(dataPlaneErr)
		}
		for _, attachment := range attachments {
			if err = dataPlane.AttachNamespace(ctx, attachment, domain.Interface{ID: attachment.InterfaceID}, object); err != nil {
				t.Fatalf("restart cycle %d attach %s: %v", cycle, attachment.PortName, err)
			}
		}
		converged, diagnostics, diagnosticsErr := switchRuntime.ConfigurationConverged(ctx, object)
		if diagnosticsErr != nil || !converged {
			t.Fatalf("restart cycle %d membership=%+v err=%v", cycle, diagnostics, diagnosticsErr)
		}
		body, pingErr := exec.CommandContext(ctx, "ip", "netns", "exec", clientNamespaces[0], "ping", "-c", "3", "-W", "1", "192.0.2.12").CombinedOutput()
		if pingErr != nil {
			t.Fatalf("restart cycle %d forwarding failed: %v: %s", cycle, pingErr, body)
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
