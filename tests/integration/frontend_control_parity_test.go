package integration

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/api/mcp"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestFrontendAPIAndMCPCreateConvergeOnAuthoritativeState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:frontend-control-parity?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := storesqlite.NewTopologyRepository(database)
	labs := command.NewLaboratoryService(topology)
	queries := query.NewLaboratoryService(topology)

	frontendLab, err := labs.Create(ctx, "frontend-created", "parity", domain.RecoveryRemainStopped)
	if err != nil {
		t.Fatal(err)
	}
	tools := mcp.Tools(mcp.Services{Labs: labs, LabQueries: queries})
	var create mcp.Tool
	for _, tool := range tools {
		if tool.Name == "netlab.labs.create" {
			create = tool
			break
		}
	}
	if create.Handler == nil {
		t.Fatal("netlab.labs.create MCP tool missing")
	}
	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	value, err := create.Handler(ginContext, map[string]any{
		"name":            "mcp-created",
		"description":     "parity",
		"recovery_policy": string(domain.RecoveryRemainStopped),
	})
	if err != nil {
		t.Fatal(err)
	}
	mcpLab, ok := value.(domain.Laboratory)
	if !ok {
		t.Fatalf("unexpected MCP result type %T", value)
	}
	for _, lab := range []domain.Laboratory{frontendLab, mcpLab} {
		snapshot, snapshotErr := queries.Snapshot(ctx, lab.ID)
		if snapshotErr != nil {
			t.Fatal(snapshotErr)
		}
		if snapshot.Laboratory.Description != "parity" || snapshot.Laboratory.RecoveryPolicy != domain.RecoveryRemainStopped || snapshot.Laboratory.Revision != 1 {
			t.Fatalf("non-convergent authoritative state: %+v", snapshot.Laboratory)
		}
	}
}
