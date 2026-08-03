package ownership

import (
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func TestNamesAndOwnership(t *testing.T) {
	id := domain.NewID()
	a := Name("nl-tap", id, 15)
	b := Name("nl-tap", id, 15)
	if a != b || len(a) > 15 {
		t.Fatal("name not deterministic")
	}
	m := Manifest{ResourceType: "node", ResourceID: id}
	if err := m.Add("tap", a, nil); err != nil {
		t.Fatal(err)
	}
	if err := RequireOwned(m, "tap", a); err != nil {
		t.Fatal(err)
	}
	if RequireOwned(m, "tap", "other") == nil {
		t.Fatal("expected guard")
	}
}

func TestDirectVethPairManifestOwnsBothNamespaceEndpoints(t *testing.T) {
	manifest := DirectVethPairManifest("link-1", "ns-a", "swp1", "ns-b", "swp2")
	if manifest.ResourceType != "network_object_link" || manifest.ResourceID != "link-1" || len(manifest.Objects) != 2 {
		t.Fatalf("manifest=%+v", manifest)
	}
	if !manifest.Owns("network_object_link_endpoint", "ns-a:swp1") || !manifest.Owns("network_object_link_endpoint", "ns-b:swp2") {
		t.Fatalf("manifest=%+v", manifest)
	}
}
