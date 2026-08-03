package integration

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/api/mcp"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestPlacementBatchIdempotencyConflictEventsAndMCPParity(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:placement-integration?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := storesqlite.NewTopologyRepository(database)
	repositories := storesqlite.NewRepositories(database)
	lab, err := command.NewLaboratoryService(topology).Create(ctx, "placements", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	node, _, err := command.NewNodeService(topology).Create(ctx, lab.ID, "node", "pc", 1)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := query.NewLaboratoryService(topology).Snapshot(ctx, lab.ID)
	if err != nil || len(snapshot.Placements) != 1 {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	service := command.NewTopologyPlacementService(topology)
	idempotency := command.NewIdempotencyService(repositories, time.Hour)
	request, _ := json.Marshal(map[string]any{"x": 10, "y": 20})
	operation := func(context.Context) (int, []byte, error) {
		result, updateErr := service.Update(ctx, lab.ID, snapshot.Laboratory.Revision, []domain.PlacementUpdate{{ResourceID: node.ID, ResourceType: domain.PlacementNode, X: 10, Y: 20}})
		body, _ := json.Marshal(result)
		return 200, body, updateErr
	}
	first, err := idempotency.Execute(ctx, "placement", "drag-1", request, operation)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := idempotency.Execute(ctx, "placement", "drag-1", request, operation)
	if err != nil || !replay.Replay || string(first.Body) != string(replay.Body) {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	var result command.TopologyPlacementResult
	_ = json.Unmarshal(first.Body, &result)
	if _, err = service.Update(ctx, lab.ID, snapshot.Laboratory.Revision, []domain.PlacementUpdate{{ResourceID: node.ID, ResourceType: domain.PlacementNode, X: 99, Y: 99, Revision: 1}}); err == nil {
		t.Fatal("expected stale laboratory revision conflict")
	}
	tool := mcp.TopologyPlacementTools(service)[0]
	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	_, err = tool.Handler(ginContext, map[string]any{"laboratory_id": string(lab.ID), "expected_revision": float64(result.LaboratoryRevision), "idempotency_key": "drag-2", "placements": []any{map[string]any{"resource_id": string(node.ID), "resource_type": "node", "x": float64(30), "y": float64(40), "revision": float64(1)}}})
	if err != nil {
		t.Fatal(err)
	}
	final, err := query.NewLaboratoryService(topology).Snapshot(ctx, lab.ID)
	if err != nil || final.Placements[0].X != 30 || final.Placements[0].Y != 40 {
		t.Fatalf("final=%+v err=%v", final.Placements, err)
	}
	events, err := repositories.OutboxAfter(ctx, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, event := range events {
		if event.Type == "topology.placements_changed" {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("placement event count=%d events=%+v", count, events)
	}
}
