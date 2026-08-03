package integration

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/api/stream"
	"github.com/netlab/netlab/internal/app/audit"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/domain"
	captureRuntime "github.com/netlab/netlab/internal/runtime/capture"
	consoleRuntime "github.com/netlab/netlab/internal/runtime/console"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestFrontendDiagnosticsExposeConsoleTrafficObservationsAndRedaction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:frontend-diagnostics?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := storesqlite.NewTopologyRepository(database)
	lab, _ := command.NewLaboratoryService(topology).Create(ctx, "diagnostics", "", domain.RecoveryRemainStopped)
	node, _, err := command.NewNodeService(topology).CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "console node", Kind: "qemu", InterfaceCount: 1, Config: map[string]any{"console_modes": []any{"telnet", "vnc"}}})
	if err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	stream.NewConsoleHandlers(t.TempDir(), consoleRuntime.Limits{IdleTimeout: time.Minute}, topology).Register(engine)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/nodes/"+string(node.ID)+"/consoles", nil))
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte("telnet")) || !bytes.Contains(response.Body.Bytes(), []byte("vnc")) {
		t.Fatalf("console metadata=%d %s", response.Code, response.Body.String())
	}

	filters := reconcile.NewTrafficFilterManager(nil)
	filter, err := filters.StartScoped(lab.ID, captureRuntime.Match{Protocol: "tcp", DestinationPort: 443}, 10, []domain.ID{"interface-a", "interface-b"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err = filters.Observe(filter.ID, "flow-1", "interface-a", "", "egress", 128, now); err != nil {
		t.Fatal(err)
	}
	if err = filters.Observe(filter.ID, "flow-1", "interface-b", "link-1", "ingress", 128, now.Add(time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	observed, ambiguous, err := filters.Get(filter.ID)
	if err != nil || ambiguous || len(observed.Observations) == 0 {
		t.Fatalf("traffic observations=%+v ambiguous=%v err=%v", observed.Observations, ambiguous, err)
	}
	stopped, err := filters.Stop(filter.ID)
	if err != nil || stopped.State != "stopped" {
		t.Fatalf("traffic filter stop=%+v err=%v", stopped, err)
	}

	redacted := audit.Redact(map[string]any{"capture_id": "capture-1", "password": "secret", "command": []any{"tcpdump", "-i", "eth0"}})
	if redacted["password"] != "[REDACTED]" || redacted["command"] != "[REDACTED]" || redacted["capture_id"] != "capture-1" {
		t.Fatalf("diagnostic metadata redaction failed: %+v", redacted)
	}
}
