package query

import (
	"context"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

type unsortedPlacementReader struct{}

func (unsortedPlacementReader) ListPlacements(context.Context, domain.ID) ([]domain.TopologyPlacement, error) {
	return []domain.TopologyPlacement{{ResourceID: "z"}, {ResourceID: "a"}, {ResourceID: "m"}}, nil
}

func TestTopologyPlacementServiceSortsRepositoryResults(t *testing.T) {
	values, err := NewTopologyPlacementService(unsortedPlacementReader{}).List(context.Background(), "lab")
	if err != nil || len(values) != 3 || values[0].ResourceID != "a" || values[2].ResourceID != "z" {
		t.Fatalf("values=%+v err=%v", values, err)
	}
}

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
