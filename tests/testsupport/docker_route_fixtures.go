package testsupport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func DockerDualStackInterfaceSettings() domain.NodeNetworkInterfaceSettings {
	return domain.NodeNetworkInterfaceSettings{
		ID:        "docker-eth0",
		Name:      "eth0",
		Driver:    "veth",
		Modes:     []string{"static"},
		Addresses: []string{"10.10.1.2/24", "2001:db8:1::2/64"},
	}
}

func DockerDualStackRouteConfig() []domain.RouteConfig {
	return []domain.RouteConfig{
		{Destination: "10.30.0.0/24", Gateway: "10.10.1.1", Metric: 100},
		{Destination: "2001:db8:3::/64", Gateway: "2001:db8:1::1"},
	}
}

func AssertDockerRouteOutput(t testing.TB, output, interfaceName string, route domain.RouteConfig) {
	t.Helper()
	line := strings.TrimSpace(output)
	for _, fragment := range []string{route.Destination, "dev " + interfaceName} {
		if !strings.Contains(line, fragment) {
			t.Fatalf("route output %q does not contain %q", line, fragment)
		}
	}
	if route.Gateway != "" && !strings.Contains(line, "via "+route.Gateway) {
		t.Fatalf("route output %q does not contain gateway %q", line, route.Gateway)
	}
	if route.Metric > 0 && !strings.Contains(line, fmt.Sprintf("metric %d", route.Metric)) {
		t.Fatalf("route output %q does not contain metric %d", line, route.Metric)
	}
}
