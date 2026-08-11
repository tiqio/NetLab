package testsupport

import (
	"context"
	"fmt"

	"github.com/netlab/netlab/internal/domain"
)

type DualStackProbe struct {
	Name        string
	Source      string
	Destination string
	Family      string
	Attempts    int
	MinimumOK   int
}

type DualStackProber interface {
	Probe(context.Context, DualStackProbe) (int, error)
}

type DualStackPathFixture struct {
	DockerRouter domain.Node
	UbuntuRoutes []domain.RouteConfig
	VyOSRoutes   []domain.RouteConfig
	Probes       []DualStackProbe
}

func ComponentMatrixDualStackFixture() DualStackPathFixture {
	routerInterfaces := []any{
		map[string]any{"name": "service0", "modes": []any{"static"}, "addresses": []any{"10.40.40.1/24", "fd40::1/64"}},
		map[string]any{"name": "uplink0", "modes": []any{"static"}, "addresses": []any{"10.40.41.1/30", "fd40:41::1/64"}, "routes": []any{
			map[string]any{"destination": "172.16.0.0/30", "gateway": "10.40.41.2"},
			map[string]any{"destination": "fd16::/64", "gateway": "fd40:41::2"},
		}},
	}
	ubuntuRoutes := []domain.RouteConfig{
		{Destination: "10.10.10.0/24", Gateway: "172.16.0.1"},
		{Destination: "10.20.20.0/24", Gateway: "172.16.0.1"},
		{Destination: "10.30.30.0/24", Gateway: "172.16.0.1"},
		{Destination: "fd10::/64", Gateway: "fd16::1"},
		{Destination: "fd20::/64", Gateway: "fd16::1"},
		{Destination: "fd30::/64", Gateway: "fd16::1"},
	}
	vyosRoutes := []domain.RouteConfig{
		{Destination: "10.40.40.0/24", Gateway: "172.16.0.2"},
		{Destination: "fd40::/64", Gateway: "fd16::2"},
	}
	probes := []DualStackProbe{
		{Name: "busybox-service-ipv4", Source: "10.40.40.11", Destination: "10.40.40.1", Family: "ipv4", Attempts: 100, MinimumOK: 99},
		{Name: "busybox-service-ipv6", Source: "fd40::11", Destination: "fd40::1", Family: "ipv6", Attempts: 100, MinimumOK: 99},
		{Name: "ubuntu-vyos-ipv4", Source: "172.16.0.2", Destination: "172.16.0.1", Family: "ipv4", Attempts: 100, MinimumOK: 99},
		{Name: "ubuntu-vyos-ipv6", Source: "fd16::2", Destination: "fd16::1", Family: "ipv6", Attempts: 100, MinimumOK: 99},
		{Name: "ubuntu-core-ipv4", Source: "172.16.0.2", Destination: "10.10.10.1", Family: "ipv4", Attempts: 100, MinimumOK: 99},
		{Name: "ubuntu-core-ipv6", Source: "fd16::2", Destination: "fd10::1", Family: "ipv6", Attempts: 100, MinimumOK: 99},
		{Name: "ubuntu-dmz-ipv4", Source: "172.16.0.2", Destination: "10.20.20.10", Family: "ipv4", Attempts: 100, MinimumOK: 99},
		{Name: "ubuntu-dmz-ipv6", Source: "fd16::2", Destination: "fd20::10", Family: "ipv6", Attempts: 100, MinimumOK: 99},
		{Name: "ubuntu-management-ipv4", Source: "172.16.0.2", Destination: "10.30.30.10", Family: "ipv4", Attempts: 100, MinimumOK: 99},
		{Name: "ubuntu-management-ipv6", Source: "fd16::2", Destination: "fd30::10", Family: "ipv6", Attempts: 100, MinimumOK: 99},
	}
	return DualStackPathFixture{
		DockerRouter: domain.Node{ID: "service-router", Kind: string(domain.RuntimeDocker), Config: map[string]any{"forward_ipv4": true, "forward_ipv6": true, "network_interfaces": routerInterfaces}},
		UbuntuRoutes: ubuntuRoutes,
		VyOSRoutes:   vyosRoutes,
		Probes:       probes,
	}
}

func (fixture DualStackPathFixture) Validate() error {
	if fixture.DockerRouter.Config["forward_ipv4"] != true || fixture.DockerRouter.Config["forward_ipv6"] != true {
		return fmt.Errorf("service router dual-stack forwarding is not enabled")
	}
	if len(fixture.UbuntuRoutes) != 6 || len(fixture.VyOSRoutes) != 2 {
		return fmt.Errorf("return route coverage is incomplete")
	}
	families := map[string]int{}
	for _, probe := range fixture.Probes {
		if probe.Attempts != 100 || probe.MinimumOK != 99 {
			return fmt.Errorf("probe %s does not enforce 99/100", probe.Name)
		}
		families[probe.Family]++
	}
	if families["ipv4"] != 5 || families["ipv6"] != 5 {
		return fmt.Errorf("dual-stack probe coverage is incomplete")
	}
	return nil
}

func (fixture DualStackPathFixture) Run(ctx context.Context, prober DualStackProber) error {
	if err := fixture.Validate(); err != nil {
		return err
	}
	for _, probe := range fixture.Probes {
		succeeded, err := prober.Probe(ctx, probe)
		if err != nil {
			return fmt.Errorf("probe %s: %w", probe.Name, err)
		}
		if succeeded < probe.MinimumOK {
			return fmt.Errorf("probe %s succeeded %d/%d", probe.Name, succeeded, probe.Attempts)
		}
	}
	return nil
}
