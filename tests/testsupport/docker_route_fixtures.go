package testsupport

import "github.com/netlab/netlab/internal/domain"

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
