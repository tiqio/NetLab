package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	httpapi "github.com/netlab/netlab/internal/api/http"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/query"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
	"github.com/netlab/netlab/internal/support/observability"
)

func TestSharedTopologyContract(t *testing.T) {
	database, err := storesqlite.Open(context.Background(), "file:contract-us1?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	server := httpapi.NewServer(":0", slog.New(slog.NewTextHandler(io.Discard, nil)), &observability.Metrics{})
	httpapi.NewTopologyHandlers(command.NewLaboratoryService(topology), query.NewLaboratoryService(topology), command.NewNodeService(topology), command.NewLinkService(topology), repositories).Register(server.Engine())

	created := request(t, server.Engine(), http.MethodPost, "/api/v1/labs", "", map[string]any{"name": "shared"}, http.StatusCreated)
	var lab map[string]any
	if err = json.Unmarshal(created.Body.Bytes(), &lab); err != nil {
		t.Fatal(err)
	}
	id := lab["id"].(string)
	request(t, server.Engine(), http.MethodPost, "/api/v1/labs/"+id+"/nodes", "1", map[string]any{"name": "pc1", "kind": "pc", "interface_count": 2}, http.StatusCreated)
	snapshot := request(t, server.Engine(), http.MethodGet, "/api/v1/labs/"+id, "", nil, http.StatusOK)
	if snapshot.Header().Get("X-Event-Sequence") == "" {
		t.Fatal("missing event sequence")
	}
	stale := request(t, server.Engine(), http.MethodPatch, "/api/v1/labs/"+id, "99", map[string]any{"name": "stale", "recovery_policy": "auto_restore"}, http.StatusPreconditionFailed)
	if !bytes.Contains(stale.Body.Bytes(), []byte("revision_conflict")) {
		t.Fatal(stale.Body.String())
	}
}

func request(t *testing.T, handler http.Handler, method, path, revision string, body any, expected int) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, _ := json.Marshal(body)
		reader = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if revision != "" {
		req.Header.Set("If-Match", revision)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != expected {
		t.Fatalf("%s %s: got %d want %d body=%s", method, path, response.Code, expected, response.Body.String())
	}
	return response
}
