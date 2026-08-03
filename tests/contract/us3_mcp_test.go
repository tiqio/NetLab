package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/api/mcp"
)

func TestMCPToolSchemaTransportAndOriginValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	mcp.NewServer(mcp.Tools(mcp.Services{})).Register(engine)

	request := func(origin string, payload map[string]any) *httptest.ResponseRecorder {
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "http://netlab.local/mcp", bytes.NewReader(body))
		req.Host = "netlab.local"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, req)
		return response
	}

	initialize := request("http://netlab.local", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	if initialize.Code != http.StatusOK || initialize.Header().Get("Mcp-Session-Id") == "" {
		t.Fatalf("initialize: %d %s", initialize.Code, initialize.Body.String())
	}
	list := request("", map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": map[string]any{}})
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte("netlab.captures.start")) || !bytes.Contains(list.Body.Bytes(), []byte("inputSchema")) {
		t.Fatalf("tools/list: %d %s", list.Code, list.Body.String())
	}
	rejected := request("https://attacker.invalid", map[string]any{"jsonrpc": "2.0", "id": 3, "method": "tools/list", "params": map[string]any{}})
	if rejected.Code != http.StatusForbidden || !bytes.Contains(rejected.Body.Bytes(), []byte("origin_rejected")) {
		t.Fatalf("origin: %d %s", rejected.Code, rejected.Body.String())
	}
	unsupported := request("", map[string]any{"jsonrpc": "2.0", "id": 4, "method": "tools/call", "params": map[string]any{"name": "netlab.captures.start", "arguments": map[string]any{}}})
	if !bytes.Contains(unsupported.Body.Bytes(), []byte("capability_unsupported")) || !bytes.Contains(unsupported.Body.Bytes(), []byte("isError")) {
		t.Fatal(unsupported.Body.String())
	}
}
