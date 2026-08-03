package reconcile

import (
	"context"
	"errors"
	"testing"
)

func TestLinuxOwnershipScannerRequiresExplicitLinkAlias(t *testing.T) {
	scanner := &LinuxOwnershipScanner{ip: "ip", output: func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "netns" {
			return nil, nil
		}
		return []byte(`[
          {"ifname":"nlt-owned","ifalias":"netlab:iface-1","linkinfo":{"info_kind":"tun"}},
          {"ifname":"nlt-unowned","linkinfo":{"info_kind":"tun"}},
          {"ifname":"eth0","ifalias":"operator:uplink"}
        ]`), nil
	}}

	values, err := scanner.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].ResourceType != "interface" || values[0].ResourceID != "iface-1" || values[0].ObjectName != "nlt-owned" {
		t.Fatalf("values=%+v", values)
	}
}

func TestLinuxOwnershipScannerKeepsSuccessfulPartialInventory(t *testing.T) {
	scanner := &LinuxOwnershipScanner{ip: "ip", output: func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "netns" {
			return nil, errors.New("netns unavailable")
		}
		return []byte(`[ {"ifname":"nli-owned","ifalias":"netlab:iface-1"} ]`), nil
	}}
	values, err := scanner.Discover(context.Background())
	if err != nil || len(values) != 1 {
		t.Fatalf("values=%+v err=%v", values, err)
	}
}

func TestProcessOwnerRequiresExplicitEnvironmentMarker(t *testing.T) {
	resourceType, resourceID := processOwner([]byte("PATH=/bin\x00NETLAB_OWNERSHIP=capture:capture-1\x00"))
	if resourceType != "capture" || resourceID != "capture-1" {
		t.Fatalf("resource_type=%q resource_id=%q", resourceType, resourceID)
	}
	resourceType, resourceID = processOwner([]byte("PATH=/bin\x00"))
	if resourceType != "" || resourceID != "" {
		t.Fatalf("unmarked process claimed: %q %q", resourceType, resourceID)
	}
}
