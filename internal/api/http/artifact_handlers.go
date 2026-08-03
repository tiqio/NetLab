package httpapi

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/artifact"
	"github.com/netlab/netlab/internal/domain"
)

type ArtifactHandlers struct{ service *artifact.Service }

func NewArtifactHandlers(service *artifact.Service) *ArtifactHandlers {
	return &ArtifactHandlers{service: service}
}

func (h *ArtifactHandlers) Register(engine *gin.Engine) {
	engine.GET("/api/v1/artifacts/:artifactId", h.download)
}

func (h *ArtifactHandlers) download(c *gin.Context) {
	metadata, file, err := h.service.Open(c, domain.ID(c.Param("artifactId")))
	if err != nil {
		handleError(c, err)
		return
	}
	defer file.Close()
	c.Header("Content-Disposition", `attachment; filename="`+filepath.Base(string(metadata.ID))+`"`)
	c.Header("Digest", "sha-256="+metadata.SHA256)
	c.DataFromReader(http.StatusOK, metadata.SizeBytes, metadata.MediaType, file, nil)
}
