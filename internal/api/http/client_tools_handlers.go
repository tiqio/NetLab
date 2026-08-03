package httpapi

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/domain"
)

type ClientToolsHandlers struct {
	directory string
}

func NewClientToolsHandlers(stateDirectory string) *ClientToolsHandlers {
	return &ClientToolsHandlers{directory: filepath.Join(stateDirectory, "client-tools")}
}

func (h *ClientToolsHandlers) Register(engine *gin.Engine) {
	engine.GET("/api/v1/client-tools/wireshark-helper/:platform", h.downloadWiresharkHelper)
}

func (h *ClientToolsHandlers) downloadWiresharkHelper(c *gin.Context) {
	filename, ok := wiresharkHelperFilename(c.Param("platform"))
	if !ok {
		writeProblem(c, http.StatusNotFound, domain.Problem{Code: "client_tool_not_found", Message: "unsupported Wireshark helper platform"})
		return
	}
	path := filepath.Join(h.directory, filename)
	file, err := os.Open(path)
	if err != nil {
		writeProblem(c, http.StatusNotFound, domain.Problem{Code: "client_tool_unavailable", Message: "Wireshark helper package is not available on this server", OperatorHint: "build and publish the requested client helper binary"})
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		writeProblem(c, http.StatusNotFound, domain.Problem{Code: "client_tool_unavailable", Message: "Wireshark helper package is not a regular file"})
		return
	}
	digest := sha256.New()
	if _, err = io.Copy(digest, file); err != nil {
		writeProblem(c, http.StatusInternalServerError, domain.Problem{Code: "client_tool_read_failed", Message: "could not read Wireshark helper package"})
		return
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		writeProblem(c, http.StatusInternalServerError, domain.Problem{Code: "client_tool_read_failed", Message: "could not prepare Wireshark helper download"})
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Header("X-Checksum-SHA256", fmt.Sprintf("%x", digest.Sum(nil)))
	c.Header("Cache-Control", "no-store")
	c.DataFromReader(http.StatusOK, info.Size(), "application/octet-stream", file, nil)
}

func wiresharkHelperFilename(platform string) (string, bool) {
	values := map[string]string{
		"linux-amd64":   "netlab-wireshark-helper-linux-amd64",
		"windows-amd64": "netlab-wireshark-helper-windows-amd64.exe",
		"darwin-amd64":  "netlab-wireshark-helper-darwin-amd64",
		"darwin-arm64":  "netlab-wireshark-helper-darwin-arm64",
	}
	value, ok := values[platform]
	return value, ok
}
