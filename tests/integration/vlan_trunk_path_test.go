package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
)

func TestPrivilegedVLANAccessTrunkAndIsolation(t *testing.T) {
	if os.Getenv("NETLAB_PRIVILEGED_TESTS") != "1" {
		t.Skip("set NETLAB_PRIVILEGED_TESTS=1 to run VLAN path acceptance")
	}
	if os.Geteuid() != 0 {
		t.Fatal("VLAN path acceptance requires root")
	}
	for _, tool := range []string{"ip", "bridge", "ping"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("acceptance host missing %s: %v", tool, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	if len(suffix) > 6 {
		suffix = suffix[len(suffix)-6:]
	}
	switchA := vlanSwitchObject("vsa-"+suffix, []domain.VLANPort{
		{Name: "access10", PVID: 10},
		{Name: "access20", PVID: 20},
		{Name: "trunk0", Tagged: []int{10, 20}},
	})
	switchB := vlanSwitchObject("vsb-"+suffix, []domain.VLANPort{
		{Name: "access10", PVID: 10},
		{Name: "access20", PVID: 20},
		{Name: "trunk0", Tagged: []int{10, 20}},
	})
	pc10A := vlanPCObject("p10a-"+suffix, "10.70.10.1/24")
	pc10B := vlanPCObject("p10b-"+suffix, "10.70.10.2/24")
	pc20A := vlanPCObject("p20a-"+suffix, "10.70.20.1/24")
	pc20B := vlanPCObject("p20b-"+suffix, "10.70.20.2/24")
	objects := []domain.NetworkObject{switchA, switchB, pc10A, pc10B, pc20A, pc20B}
	for _, object := range objects {
		namespace, err := linuxnet.NetworkObjectNamespaceName(object)
		if err != nil {
			t.Fatal(err)
		}
		vlanCommand(t, ctx, "ip", "netns", "add", namespace)
		vlanCommand(t, ctx, "ip", "-n", namespace, "link", "set", "lo", "up")
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cleanupCancel()
		for _, object := range objects {
			namespace, _ := linuxnet.NetworkObjectNamespaceName(object)
			_ = exec.CommandContext(cleanupCtx, "ip", "netns", "delete", namespace).Run()
		}
	})

	dataPlane, err := linuxnet.NewDataPlane(nil)
	if err != nil {
		t.Fatal(err)
	}
	l2Runtime, err := linuxnet.NewSwitchL2Runtime(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range []domain.NetworkObject{switchA, switchB} {
		if err = l2Runtime.Configure(ctx, object); err != nil {
			t.Fatalf("provision %s before late ports: %v", object.ID, err)
		}
	}
	links := []struct {
		link domain.NetworkObjectLink
		a    domain.NetworkObject
		b    domain.NetworkObject
	}{
		{vlanObjectLink("v10a-"+suffix, pc10A, "eth0", switchA, "access10"), pc10A, switchA},
		{vlanObjectLink("v20a-"+suffix, pc20A, "eth0", switchA, "access20"), pc20A, switchA},
		{vlanObjectLink("trunk-"+suffix, switchA, "trunk0", switchB, "trunk0"), switchA, switchB},
		{vlanObjectLink("v10b-"+suffix, pc10B, "eth0", switchB, "access10"), pc10B, switchB},
		{vlanObjectLink("v20b-"+suffix, pc20B, "eth0", switchB, "access20"), pc20B, switchB},
	}
	for _, item := range links {
		if err = dataPlane.EnsureNetworkObjectLink(ctx, item.link, item.a, item.b); err != nil {
			t.Fatalf("ensure %s: %v", item.link.ID, err)
		}
	}
	for _, object := range []domain.NetworkObject{switchA, switchB} {
		if err = l2Runtime.Configure(ctx, object); err != nil {
			t.Fatalf("configure %s: %v", object.ID, err)
		}
		converged, diagnostics, diagnosticsErr := l2Runtime.ConfigurationConverged(ctx, object)
		if diagnosticsErr != nil || !converged {
			t.Fatalf("VLAN membership did not converge for %s: %+v err=%v", object.ID, diagnostics, diagnosticsErr)
		}
	}
	pcRuntime, err := linuxnet.NewPCRuntime(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range []domain.NetworkObject{pc10A, pc10B, pc20A, pc20B} {
		if err = pcRuntime.Configure(ctx, object); err != nil {
			t.Fatalf("configure %s: %v", object.ID, err)
		}
	}

	vlanAssertPingCount(t, ctx, pc10A, "10.70.10.2", 100, 99)
	vlanAssertPingCount(t, ctx, pc20A, "10.70.20.2", 100, 99)
	vlanAssertBlocked(t, ctx, pc10A, "10.70.20.2")
	vlanAssertBlocked(t, ctx, pc20A, "10.70.10.2")
}

func vlanSwitchObject(id string, ports []domain.VLANPort) domain.NetworkObject {
	values := make([]any, 0, len(ports))
	for _, port := range ports {
		tagged := make([]any, 0, len(port.Tagged))
		for _, vlan := range port.Tagged {
			tagged = append(tagged, vlan)
		}
		values = append(values, map[string]any{"name": port.Name, "pvid": port.PVID, "tagged": tagged})
	}
	return domain.NetworkObject{ID: domain.ID(id), Kind: domain.NetworkSwitchL2, Config: map[string]any{"vlan_filtering": true, "ports": values}}
}

func vlanPCObject(id, address string) domain.NetworkObject {
	return domain.NetworkObject{ID: domain.ID(id), Kind: domain.NetworkPC, Config: map[string]any{"interfaces": []any{map[string]any{"name": "eth0", "modes": []any{"static"}, "addresses": []any{address}}}}}
}

func vlanObjectLink(id string, objectA domain.NetworkObject, portA string, objectB domain.NetworkObject, portB string) domain.NetworkObjectLink {
	return domain.NetworkObjectLink{ID: domain.ID(id), ObjectAID: objectA.ID, PortAName: portA, ObjectBID: objectB.ID, PortBName: portB}
}

func vlanCommand(t *testing.T, ctx context.Context, name string, args ...string) []byte {
	t.Helper()
	body, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v: %s", name, strings.Join(args, " "), err, body)
	}
	return body
}

func vlanAssertPingCount(t *testing.T, ctx context.Context, source domain.NetworkObject, destination string, probes, minimum int) {
	t.Helper()
	namespace, _ := linuxnet.NetworkObjectNamespaceName(source)
	body := vlanCommand(t, ctx, "ip", "netns", "exec", namespace, "ping", "-i", "0.01", "-c", fmt.Sprint(probes), "-W", "1", destination)
	marker := fmt.Sprintf("%d received", minimum)
	if minimum == probes {
		marker = fmt.Sprintf("%d received", probes)
	}
	for received := probes; received >= minimum; received-- {
		if strings.Contains(string(body), fmt.Sprintf("%d received", received)) {
			return
		}
	}
	t.Fatalf("expected at least %s from %s: %s", marker, source.ID, body)
}

func vlanAssertBlocked(t *testing.T, ctx context.Context, source domain.NetworkObject, destination string) {
	t.Helper()
	namespace, _ := linuxnet.NetworkObjectNamespaceName(source)
	body, err := exec.CommandContext(ctx, "ip", "netns", "exec", namespace, "ping", "-c", "3", "-W", "1", destination).CombinedOutput()
	blocked := strings.Contains(string(body), "0 received") || strings.Contains(string(body), "Network is unreachable") || strings.Contains(string(body), "Destination Host Unreachable")
	if err == nil || !blocked {
		t.Fatalf("cross-VLAN path unexpectedly succeeded from %s to %s: %s", source.ID, destination, body)
	}
}
