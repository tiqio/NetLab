package query

import (
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func TestResolveTopologyPlacementsIsCompleteStableAndPreservesStoredValues(t *testing.T) {
	snapshot := domain.TopologySnapshot{
		Laboratory: domain.Laboratory{ID: "lab-1"},
		Nodes:      []domain.Node{{ID: "node-b"}, {ID: "node-a"}},
		Placements: []domain.TopologyPlacement{{LaboratoryID: "lab-1", ResourceID: "node-a", ResourceType: domain.PlacementNode, X: 4, Y: 5, Revision: 2}},
	}
	first := ResolveTopologyPlacements(snapshot)
	second := ResolveTopologyPlacements(snapshot)
	if len(first) != 2 || first[0].ResourceID != "node-a" || first[0].X != 4 || first[1] != second[1] {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
}
