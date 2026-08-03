package console

import (
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func TestAddressesFromLeaseMatchesCurrentNodeMAC(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	body := []byte("1800000300 02:00:00:00:00:01 10.10.0.21 ubuntu *\n1799999999 02:00:00:00:00:01 10.10.0.20 expired *\n1800000300 02:00:00:00:00:02 10.10.0.22 other *\n")
	values := addressesFromLease(body, map[string]bool{"02:00:00:00:00:01": true}, now)
	if len(values) != 1 || values[0] != "10.10.0.21" {
		t.Fatalf("addresses=%v", values)
	}
}

func TestConfiguredAddressesUsesStaticInterfaceCIDRs(t *testing.T) {
	node := domain.Node{Config: map[string]any{"network_interfaces": []any{
		map[string]any{"name": "ens0", "addresses": []any{"192.0.2.10/24", "2001:db8::10/64"}},
	}}}
	values := configuredAddresses(node)
	if len(values) != 2 || values[0] != "192.0.2.10" || values[1] != "2001:db8::10" {
		t.Fatalf("addresses=%v", values)
	}
}
