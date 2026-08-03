package contract

import (
	"encoding/json"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func TestTopologySnapshotAndPlacementEventJSONContract(t *testing.T) {
	placement := domain.TopologyPlacement{LaboratoryID: "lab-1", ResourceID: "node-1", ResourceType: domain.PlacementNode, X: 10, Y: 20, Revision: 2}
	snapshot := domain.TopologySnapshot{Placements: []domain.TopologyPlacement{placement}}
	body, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err = json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, ok := decoded["placements"]; !ok {
		t.Fatal("snapshot missing placements")
	}
	event := domain.OutboxEvent{Type: "topology.placements_changed", LaboratoryID: "lab-1", ResourceType: "laboratory", ResourceID: "lab-1", Revision: 3, Data: map[string]any{"placements": []domain.TopologyPlacement{placement}}}
	body, err = json.Marshal(event)
	if err != nil || !json.Valid(body) {
		t.Fatalf("event=%s err=%v", body, err)
	}
}
