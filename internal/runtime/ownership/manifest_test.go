package ownership

import (
	"github.com/netlab/netlab/internal/domain"
	"testing"
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
