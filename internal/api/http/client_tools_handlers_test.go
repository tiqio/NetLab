package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWiresharkHelperDownloadUsesWhitelistedPackage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stateDirectory := t.TempDir()
	toolsDirectory := filepath.Join(stateDirectory, "client-tools")
	if err := os.MkdirAll(toolsDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(toolsDirectory, "netlab-wireshark-helper-linux-amd64")
	if err := os.WriteFile(path, []byte("helper"), 0o700); err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	NewClientToolsHandlers(stateDirectory).Register(engine)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/client-tools/wireshark-helper/linux-amd64", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "helper" {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Header().Get("Content-Disposition"), "linux-amd64") || response.Header().Get("X-Checksum-SHA256") == "" {
		t.Fatalf("headers=%v", response.Header())
	}
}

func TestWiresharkHelperDownloadRejectsUnknownPlatform(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	NewClientToolsHandlers(t.TempDir()).Register(engine)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/client-tools/wireshark-helper/../../etc/passwd", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code == http.StatusOK {
		t.Fatal("unexpected client tool download")
	}
}
