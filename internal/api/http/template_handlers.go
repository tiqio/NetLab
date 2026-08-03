package httpapi

import (
	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/domain"
	runtimeimage "github.com/netlab/netlab/internal/runtime/image"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
	"net/http"
)

type TemplateHandlers struct {
	queries  *query.TemplateService
	images   *storesqlite.TemplateRepository
	importer *runtimeimage.Importer
}

func NewTemplateHandlers(queries *query.TemplateService, images *storesqlite.TemplateRepository, importer *runtimeimage.Importer) *TemplateHandlers {
	return &TemplateHandlers{queries: queries, images: images, importer: importer}
}
func (h *TemplateHandlers) Register(engine *gin.Engine) {
	api := engine.Group("/api/v1")
	api.GET("/templates", h.listTemplates)
	api.GET("/images", h.listImages)
	api.POST("/images", h.importImage)
}
func (h *TemplateHandlers) listTemplates(c *gin.Context) {
	values, err := h.queries.List(c)
	if err != nil {
		handleError(c, err)
		return
	}
	if values == nil {
		values = []domain.DeviceTemplate{}
	}
	c.JSON(200, values)
}
func (h *TemplateHandlers) listImages(c *gin.Context) {
	values, err := h.queries.Images(c)
	if err != nil {
		handleError(c, err)
		return
	}
	if values == nil {
		values = []domain.ImageVersion{}
	}
	c.JSON(200, values)
}
func (h *TemplateHandlers) importImage(c *gin.Context) {
	var body struct {
		Name           string             `json:"name"`
		Version        string             `json:"version"`
		RuntimeKind    domain.RuntimeKind `json:"runtime_kind"`
		SourcePath     string             `json:"source_reference"`
		ExpectedSHA256 string             `json:"expected_sha256"`
		LicenseNotes   string             `json:"license_notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	image, err := h.importer.Import(c, runtimeimage.ImportRequest{Name: body.Name, Version: body.Version, RuntimeKind: body.RuntimeKind, SourcePath: body.SourcePath, ExpectedSHA256: body.ExpectedSHA256, LicenseNotes: body.LicenseNotes})
	if err != nil {
		handleError(c, err)
		return
	}
	if err = h.images.CreateImage(c, image); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, image)
}
