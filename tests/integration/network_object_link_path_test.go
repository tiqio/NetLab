package integration

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

func TestPrivilegedNetworkObjectLinkThreeObjectPath(t *testing.T) {
	if os.Getenv("NETLAB_PRIVILEGED") != "1" {
		t.Skip("set NETLAB_PRIVILEGED=1 on the acceptance host")
	}
	if os.Geteuid() != 0 {
		t.Skip("NETLAB_PRIVILEGED requires root")
	}
	for _, tool := range []string{"ip", "ping", "python3"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("acceptance host missing %s: %v", tool, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	if len(suffix) > 7 {
		suffix = suffix[len(suffix)-7:]
	}
	objectA := networkObjectLinkL3Object("path-a-"+suffix, map[string]string{
		"a0": "10.41.0.1/30",
		"a1": "10.41.1.1/30",
	})
	objectB := networkObjectLinkL3Object("path-b-"+suffix, map[string]string{
		"b0": "10.41.0.2/30",
		"b1": "10.41.1.2/30",
		"b2": "10.42.0.1/30",
	})
	objectC := networkObjectLinkL3Object("path-c-"+suffix, map[string]string{
		"c0": "10.42.0.2/30",
	})
	objects := []domain.NetworkObject{objectA, objectB, objectC}
	for _, object := range objects {
		namespace, err := linuxnet.NetworkObjectNamespaceName(object)
		if err != nil {
			t.Fatal(err)
		}
		runObjectLinkCommand(t, ctx, "ip", "netns", "add", namespace)
		runObjectLinkCommand(t, ctx, "ip", "-n", namespace, "link", "set", "lo", "up")
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cleanupCancel()
		for _, object := range objects {
			namespace, _ := linuxnet.NetworkObjectNamespaceName(object)
			_ = exec.CommandContext(cleanupCtx, "ip", "netns", "delete", namespace).Run()
		}
	}()

	namespaceA, _ := linuxnet.NetworkObjectNamespaceName(objectA)
	namespaceC, _ := linuxnet.NetworkObjectNamespaceName(objectC)
	primary := domain.NetworkObjectLink{ID: domain.ID("path-primary-" + suffix), ObjectAID: objectA.ID, PortAName: "a0", ObjectBID: objectB.ID, PortBName: "b0"}
	parallel := domain.NetworkObjectLink{ID: domain.ID("path-parallel-" + suffix), ObjectAID: objectA.ID, PortAName: "a1", ObjectBID: objectB.ID, PortBName: "b1"}
	downstream := domain.NetworkObjectLink{ID: domain.ID("path-downstream-" + suffix), ObjectAID: objectB.ID, PortAName: "b2", ObjectBID: objectC.ID, PortBName: "c0"}
	dataPlane, err := linuxnet.NewDataPlane(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		link domain.NetworkObjectLink
		a    domain.NetworkObject
		b    domain.NetworkObject
	}{{primary, objectA, objectB}, {parallel, objectA, objectB}, {downstream, objectB, objectC}} {
		if err = dataPlane.EnsureNetworkObjectLink(ctx, item.link, item.a, item.b); err != nil {
			t.Fatalf("ensure %s: %v", item.link.ID, err)
		}
	}
	runObjectLinkCommand(t, ctx, "ip", "-n", namespaceA, "route", "replace", "10.42.0.0/30", "via", "10.41.0.2")
	runObjectLinkCommand(t, ctx, "ip", "-n", namespaceC, "route", "replace", "10.41.0.0/30", "via", "10.42.0.1")

	assertObjectLinkPing(t, ctx, namespaceA, "10.41.0.2")
	assertObjectLinkPing(t, ctx, namespaceA, "10.41.1.2")
	assertObjectLinkPing(t, ctx, namespaceA, "10.42.0.2")
	assertObjectLinkPing(t, ctx, namespaceC, "10.41.0.1")
	assertObjectLinkSocket(t, ctx, "tcp", namespaceA, namespaceC, "10.42.0.2", 18401)
	assertObjectLinkSocket(t, ctx, "tcp", namespaceC, namespaceA, "10.41.0.1", 18402)
	assertObjectLinkSocket(t, ctx, "udp", namespaceA, namespaceC, "10.42.0.2", 18403)
	assertObjectLinkSocket(t, ctx, "udp", namespaceC, namespaceA, "10.41.0.1", 18404)

	if err = dataPlane.DeleteNetworkObjectLink(ctx, primary, objectA, objectB); err != nil {
		t.Fatalf("delete primary parallel link: %v", err)
	}
	assertObjectLinkPing(t, ctx, namespaceA, "10.41.1.2")
	runObjectLinkCommand(t, ctx, "ip", "-n", namespaceA, "route", "replace", "10.42.0.0/30", "via", "10.41.1.2")
	runObjectLinkCommand(t, ctx, "ip", "-n", namespaceC, "route", "replace", "10.41.1.0/30", "via", "10.42.0.1")
	assertObjectLinkPing(t, ctx, namespaceA, "10.42.0.2")

	recovered, err := linuxnet.NewDataPlane(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		link domain.NetworkObjectLink
		a    domain.NetworkObject
		b    domain.NetworkObject
	}{{parallel, objectA, objectB}, {downstream, objectB, objectC}} {
		if err = recovered.EnsureNetworkObjectLink(ctx, item.link, item.a, item.b); err != nil {
			t.Fatalf("recover %s: %v", item.link.ID, err)
		}
	}
	assertObjectLinkPing(t, ctx, namespaceA, "10.42.0.2")
}

func networkObjectLinkL3Object(id string, addresses map[string]string) domain.NetworkObject {
	interfaces := make([]any, 0, len(addresses))
	for name, address := range addresses {
		interfaces = append(interfaces, map[string]any{"name": name, "addresses": []any{address}})
	}
	return domain.NetworkObject{ID: domain.ID(id), Kind: domain.NetworkSwitchL3, Config: map[string]any{"interfaces": interfaces, "forward_ipv4": true}}
}

func runObjectLinkCommand(t *testing.T, ctx context.Context, name string, args ...string) string {
	t.Helper()
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v: %s", name, strings.Join(args, " "), err, output)
	}
	return string(output)
}

func assertObjectLinkPing(t *testing.T, ctx context.Context, namespace, destination string) {
	t.Helper()
	runObjectLinkCommand(t, ctx, "ip", "netns", "exec", namespace, "ping", "-c", "2", "-W", "1", destination)
}

func assertObjectLinkSocket(t *testing.T, ctx context.Context, network, sourceNamespace, destinationNamespace, destination string, port int) {
	t.Helper()
	serverCode := `import socket,sys;s=socket.socket(socket.AF_INET,socket.SOCK_STREAM if sys.argv[1]=='tcp' else socket.SOCK_DGRAM);s.bind(('0.0.0.0',int(sys.argv[2])));s.settimeout(5);` +
		`exec("s.listen(1);c,a=s.accept();d=c.recv(64);c.sendall(d)" if sys.argv[1]=='tcp' else "d,a=s.recvfrom(64);s.sendto(d,a)")`
	server := exec.CommandContext(ctx, "ip", "netns", "exec", destinationNamespace, "python3", "-c", serverCode, network, strconv.Itoa(port))
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Process.Kill() })
	time.Sleep(150 * time.Millisecond)
	clientCode := `import socket,sys;s=socket.socket(socket.AF_INET,socket.SOCK_STREAM if sys.argv[1]=='tcp' else socket.SOCK_DGRAM);s.settimeout(3);s.connect((sys.argv[2],int(sys.argv[3])));s.sendall(b'netlab');d=s.recv(64);assert d==b'netlab',d`
	runObjectLinkCommand(t, ctx, "ip", "netns", "exec", sourceNamespace, "python3", "-c", clientCode, network, destination, strconv.Itoa(port))
	if err := server.Wait(); err != nil {
		t.Fatalf("%s server: %v", network, err)
	}
}
