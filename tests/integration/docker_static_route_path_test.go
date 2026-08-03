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
	dockeradapter "github.com/netlab/netlab/internal/runtime/docker"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
	"github.com/netlab/netlab/tests/testsupport"
)

func TestPrivilegedDockerStaticRoutePath(t *testing.T) {
	if os.Getenv("NETLAB_PRIVILEGED") != "1" {
		t.Skip("set NETLAB_PRIVILEGED=1 on the acceptance host")
	}
	if os.Geteuid() != 0 {
		t.Skip("NETLAB_PRIVILEGED requires root")
	}
	image := strings.TrimSpace(os.Getenv("NETLAB_DOCKER_ROUTE_IMAGE"))
	if image == "" {
		t.Skip("set NETLAB_DOCKER_ROUTE_IMAGE to an approved image pinned by digest")
	}
	if !strings.Contains(image, "@sha256:") {
		t.Fatal("NETLAB_DOCKER_ROUTE_IMAGE must be pinned by sha256 digest")
	}
	for _, tool := range []string{"docker", "ip", "nsenter"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("acceptance host missing %s: %v", tool, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	runRouteCommand(t, ctx, "docker", "image", "inspect", image)

	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	if len(suffix) > 7 {
		suffix = suffix[len(suffix)-7:]
	}
	routerNS := "nlr" + suffix
	bridgeA := "nlba" + suffix
	bridgeB := "nlbb" + suffix
	routerHostA := "nlha" + suffix
	routerPeerA := "nlpa" + suffix
	routerHostB := "nlhb" + suffix
	routerPeerB := "nlpb" + suffix

	nodeA := dockerRouteNode("route-a-"+suffix, "if-a-"+suffix, image, "10.10.1.2/24", "2001:db8:1::2/64", []domain.RouteConfig{
		{Destination: "10.30.0.0/24", Gateway: "10.10.1.1", Metric: 100},
		{Destination: "2001:db8:3::/64", Gateway: "2001:db8:1::1"},
	})
	nodeB := dockerRouteNode("route-b-"+suffix, "if-b-"+suffix, image, "10.30.0.2/24", "2001:db8:3::2/64", []domain.RouteConfig{
		{Destination: "10.10.1.0/24", Gateway: "10.30.0.1", Metric: 100},
		{Destination: "2001:db8:1::/64", Gateway: "2001:db8:3::1"},
	})

	adapter, err := dockeradapter.NewAdapter()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if t.Failed() {
			for _, node := range []domain.Node{nodeA, nodeB} {
				body, _ := exec.CommandContext(cleanupCtx, "docker", "inspect", "netlab-"+string(node.ID), "--format", `{{json .State}}`).CombinedOutput()
				t.Logf("container %s state: %s", node.ID, strings.TrimSpace(string(body)))
			}
		}
		_ = adapter.Delete(cleanupCtx, nodeA)
		_ = adapter.Delete(cleanupCtx, nodeB)
		runRouteCleanup(cleanupCtx, "ip", "link", "delete", bridgeA)
		runRouteCleanup(cleanupCtx, "ip", "link", "delete", bridgeB)
		runRouteCleanup(cleanupCtx, "ip", "netns", "delete", routerNS)
	}()

	createRouteBridge(t, ctx, bridgeA)
	createRouteBridge(t, ctx, bridgeB)
	runRouteCommand(t, ctx, "ip", "netns", "add", routerNS)
	runRouteCommand(t, ctx, "ip", "-n", routerNS, "link", "set", "lo", "up")
	attachRouterPort(t, ctx, routerNS, bridgeA, routerHostA, routerPeerA, "lan0", "10.10.1.1/24", "2001:db8:1::1/64")
	attachRouterPort(t, ctx, routerNS, bridgeB, routerHostB, routerPeerB, "lan1", "10.30.0.1/24", "2001:db8:3::1/64")
	runRouteCommand(t, ctx, "ip", "netns", "exec", routerNS, "sysctl", "-qw", "net.ipv4.ip_forward=1")
	runRouteCommand(t, ctx, "ip", "netns", "exec", routerNS, "sysctl", "-qw", "net.ipv6.conf.all.forwarding=1")

	if err := adapter.Start(ctx, nodeA); err != nil {
		t.Fatalf("start node A: %v", err)
	}
	if err := adapter.Start(ctx, nodeB); err != nil {
		t.Fatalf("start node B: %v", err)
	}
	attachDockerEndpoint(t, ctx, nodeA, bridgeA)
	attachDockerEndpoint(t, ctx, nodeB, bridgeB)
	assertDockerRoutes(t, ctx, nodeA, []domain.RouteConfig{
		{Destination: "10.30.0.0/24", Gateway: "10.10.1.1", Metric: 100},
		{Destination: "2001:db8:3::/64", Gateway: "2001:db8:1::1"},
	})
	assertDockerRoutes(t, ctx, nodeB, []domain.RouteConfig{
		{Destination: "10.10.1.0/24", Gateway: "10.30.0.1", Metric: 100},
		{Destination: "2001:db8:1::/64", Gateway: "2001:db8:3::1"},
	})
	assertDockerRouteTraffic(t, ctx, nodeA, nodeB)

	if err := adapter.Stop(ctx, nodeA); err != nil {
		t.Fatalf("stop node A: %v", err)
	}
	if err := adapter.Start(ctx, nodeA); err != nil {
		t.Fatalf("restart node A: %v", err)
	}
	attachDockerEndpoint(t, ctx, nodeA, bridgeA)
	assertDockerRoutes(t, ctx, nodeA, []domain.RouteConfig{
		{Destination: "10.30.0.0/24", Gateway: "10.10.1.1", Metric: 100},
		{Destination: "2001:db8:3::/64", Gateway: "2001:db8:1::1"},
	})

	recoveredAdapter, err := dockeradapter.NewAdapter()
	if err != nil {
		t.Fatal(err)
	}
	if err = recoveredAdapter.Start(ctx, nodeA); err != nil {
		t.Fatalf("service recovery node A: %v", err)
	}
	if err = recoveredAdapter.Start(ctx, nodeB); err != nil {
		t.Fatalf("service recovery node B: %v", err)
	}
	assertDockerRoutes(t, ctx, nodeA, []domain.RouteConfig{
		{Destination: "10.30.0.0/24", Gateway: "10.10.1.1", Metric: 100},
		{Destination: "2001:db8:3::/64", Gateway: "2001:db8:1::1"},
	})
	assertDockerRouteTraffic(t, ctx, nodeA, nodeB)
}

func dockerRouteNode(id, interfaceID, image, ipv4, ipv6 string, routes []domain.RouteConfig) domain.Node {
	return domain.Node{
		ID: domain.ID(id), LaboratoryID: "docker-route-integration", Name: id, Kind: string(domain.RuntimeDocker),
		CPUCount: 1, MemoryMiB: 128, InterfaceLimit: 1, ProcessLimit: 128,
		Config: map[string]any{
			"image":      image,
			"command":    []any{"sleep", "31536000"},
			"interfaces": []map[string]any{{"id": interfaceID, "name": "eth0", "mac_address": ""}},
			"network_interfaces": []map[string]any{{
				"id": interfaceID, "name": "eth0", "driver": "veth", "modes": []any{"static"},
				"addresses": []any{ipv4, ipv6}, "routes": routeMaps(routes),
			}},
		},
	}
}

func routeMaps(routes []domain.RouteConfig) []any {
	result := make([]any, 0, len(routes))
	for _, route := range routes {
		result = append(result, map[string]any{"destination": route.Destination, "gateway": route.Gateway, "metric": route.Metric})
	}
	return result
}

func createRouteBridge(t *testing.T, ctx context.Context, name string) {
	t.Helper()
	runRouteCommand(t, ctx, "ip", "link", "add", name, "type", "bridge")
	runRouteCommand(t, ctx, "ip", "link", "set", name, "up")
}

func attachRouterPort(t *testing.T, ctx context.Context, namespace, bridge, host, peer, target, ipv4, ipv6 string) {
	t.Helper()
	runRouteCommand(t, ctx, "ip", "link", "add", host, "type", "veth", "peer", "name", peer)
	runRouteCommand(t, ctx, "ip", "link", "set", host, "master", bridge)
	runRouteCommand(t, ctx, "ip", "link", "set", host, "up")
	runRouteCommand(t, ctx, "ip", "link", "set", peer, "netns", namespace)
	runRouteCommand(t, ctx, "ip", "-n", namespace, "link", "set", peer, "name", target)
	runRouteCommand(t, ctx, "ip", "-n", namespace, "link", "set", target, "up")
	runRouteCommand(t, ctx, "ip", "-n", namespace, "address", "replace", ipv4, "dev", target)
	runRouteCommand(t, ctx, "ip", "-n", namespace, "address", "replace", ipv6, "dev", target)
}

func attachDockerEndpoint(t *testing.T, ctx context.Context, node domain.Node, bridge string) {
	t.Helper()
	interfaceID := domain.ID(node.Config["interfaces"].([]map[string]any)[0]["id"].(string))
	runRouteCommand(t, ctx, "ip", "link", "set", linuxnet.HostInterfaceName(interfaceID), "master", bridge)
	runRouteCommand(t, ctx, "ip", "link", "set", linuxnet.HostInterfaceName(interfaceID), "up")
}

func assertDockerRoutes(t *testing.T, ctx context.Context, node domain.Node, routes []domain.RouteConfig) {
	t.Helper()
	for _, route := range routes {
		family := "-4"
		if strings.Contains(route.Destination, ":") {
			family = "-6"
		}
		output := runRouteCommand(t, ctx, "docker", "exec", "netlab-"+string(node.ID), "ip", family, "route", "show", route.Destination)
		testsupport.AssertDockerRouteOutput(t, string(output), "eth0", route)
	}
}

func assertDockerRouteTraffic(t *testing.T, ctx context.Context, source, target domain.Node) {
	t.Helper()
	sourceName := "netlab-" + string(source.ID)
	targetName := "netlab-" + string(target.ID)
	runRouteCommand(t, ctx, "docker", "exec", sourceName, "ping", "-6", "-c", "1", "-W", "2", "2001:db8:1::1")
	runRouteCommand(t, ctx, "docker", "exec", targetName, "ping", "-6", "-c", "1", "-W", "2", "2001:db8:3::1")
	runRouteCommand(t, ctx, "docker", "exec", sourceName, "ping", "-c", "1", "-W", "2", "10.30.0.2")
	runRouteCommand(t, ctx, "docker", "exec", sourceName, "ping", "-6", "-c", "1", "-W", "2", "2001:db8:3::2")

	runRouteCommand(t, ctx, "docker", "exec", targetName, "sh", "-c", "rm -f /tmp/netlab-tcp; timeout 8 nc -l -p 19001 > /tmp/netlab-tcp &")
	time.Sleep(200 * time.Millisecond)
	runRouteCommand(t, ctx, "docker", "exec", sourceName, "sh", "-c", "printf route-tcp | nc -w 3 10.30.0.2 19001")
	waitDockerFile(t, ctx, targetName, "/tmp/netlab-tcp", "route-tcp")

	runRouteCommand(t, ctx, "docker", "exec", targetName, "sh", "-c", "rm -f /tmp/netlab-udp; timeout 8 nc -u -l -p 19002 > /tmp/netlab-udp &")
	time.Sleep(200 * time.Millisecond)
	runRouteCommand(t, ctx, "docker", "exec", sourceName, "sh", "-c", "printf route-udp | nc -u -w 1 10.30.0.2 19002")
	waitDockerFile(t, ctx, targetName, "/tmp/netlab-udp", "route-udp")
}

func waitDockerFile(t *testing.T, ctx context.Context, container, path, expected string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		command := exec.CommandContext(ctx, "docker", "exec", container, "cat", path)
		if body, err := command.Output(); err == nil && strings.Contains(string(body), expected) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("container %s did not receive %q in %s", container, expected, path)
}

func runRouteCommand(t *testing.T, ctx context.Context, name string, args ...string) []byte {
	t.Helper()
	command := exec.CommandContext(ctx, name, args...)
	command.Env = append(os.Environ(), "LC_ALL=C")
	body, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, body)
	}
	return body
}

func runRouteCleanup(ctx context.Context, name string, args ...string) {
	command := exec.CommandContext(ctx, name, args...)
	command.Env = append(os.Environ(), "LC_ALL=C")
	_, _ = command.CombinedOutput()
}
