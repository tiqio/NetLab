package httpapi

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/domain"
	captureRuntime "github.com/netlab/netlab/internal/runtime/capture"
)

type CaptureHandlers struct {
	captures   *reconcile.CaptureManager
	filters    *reconcile.TrafficFilterManager
	operations *reconcile.CaptureTaskService
}

func NewCaptureHandlers(captures *reconcile.CaptureManager, filters *reconcile.TrafficFilterManager, operations *reconcile.CaptureTaskService) *CaptureHandlers {
	return &CaptureHandlers{captures: captures, filters: filters, operations: operations}
}

func (h *CaptureHandlers) Register(engine *gin.Engine) {
	api := engine.Group("/api/v1")
	api.GET("/captures", h.listCaptures)
	api.POST("/captures", h.startCapture)
	api.GET("/captures/:captureId", h.getCapture)
	api.DELETE("/captures/:captureId", h.stopCapture)
	api.GET("/captures/:captureId/stream", h.streamCapture)
	api.GET("/traffic-filters", h.listTrafficFilters)
	api.POST("/traffic-filters", h.startTrafficFilter)
	api.GET("/traffic-filters/:filterId", h.getTrafficFilter)
	api.DELETE("/traffic-filters/:filterId", h.stopTrafficFilter)
	api.DELETE("/traffic-filters/:filterId/history", h.deleteTrafficFilterHistory)
}

func (h *CaptureHandlers) listCaptures(c *gin.Context) {
	values := h.captures.ListLaboratory(domain.ID(c.Query("laboratory_id")))
	if c.Query("include_internal") != "true" {
		visible := make([]domain.Capture, 0, len(values))
		for _, value := range values {
			if value.Purpose == "" {
				visible = append(visible, value)
			}
		}
		values = visible
	}
	c.JSON(http.StatusOK, values)
}

func (h *CaptureHandlers) listTrafficFilters(c *gin.Context) {
	values := h.filters.List(domain.ID(c.Query("laboratory_id")))
	result := make([]gin.H, 0, len(values))
	for _, value := range values {
		current, ambiguous, err := h.filters.Get(value.ID)
		if err == nil {
			result = append(result, gin.H{"traffic_filter": current, "ambiguous": ambiguous})
		}
	}
	c.JSON(http.StatusOK, result)
}

func (h *CaptureHandlers) startCapture(c *gin.Context) {
	var body struct {
		LaboratoryID    domain.ID `json:"laboratory_id"`
		SourceType      string    `json:"source_type"`
		SourceID        domain.ID `json:"source_id"`
		Interface       string    `json:"interface"`
		Filter          string    `json:"filter"`
		Format          string    `json:"format"`
		Retain          bool      `json:"retain"`
		MaxBytes        int64     `json:"max_bytes"`
		DurationSeconds int       `json:"duration_seconds"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	if h.operations == nil {
		handleError(c, domain.Problem{Code: "operation_unavailable", Message: "capture automation unavailable"})
		return
	}
	value, taskValue, err := h.operations.StartCapture(c, reconcile.CaptureRequest{LaboratoryID: body.LaboratoryID, SourceType: body.SourceType, SourceID: body.SourceID, Interface: body.Interface, Filter: body.Filter, Format: body.Format, Retain: body.Retain, MaxBytes: body.MaxBytes, Duration: time.Duration(body.DurationSeconds) * time.Second}, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"capture": value, "task": taskValue, "stream_url": "/api/v1/captures/" + string(value.ID) + "/stream", "wireshark": gin.H{"mode": "http_stream", "media_type": "application/vnd.tcpdump.pcap"}})
}

func (h *CaptureHandlers) getCapture(c *gin.Context) {
	value, err := h.captures.Get(domain.ID(c.Param("captureId")))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *CaptureHandlers) stopCapture(c *gin.Context) {
	if h.operations == nil {
		handleError(c, domain.Problem{Code: "operation_unavailable", Message: "capture automation unavailable"})
		return
	}
	value, err := h.operations.StopCapture(c, domain.ID(c.Param("captureId")), c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task": value})
}

func (h *CaptureHandlers) streamCapture(c *gin.Context) {
	metadata, err := h.captures.Get(domain.ID(c.Param("captureId")))
	if err != nil {
		handleError(c, err)
		return
	}
	stream, cancel, err := h.captures.Subscribe(metadata.ID)
	if err != nil {
		handleError(c, err)
		return
	}
	defer cancel()
	mediaType := "application/vnd.tcpdump.pcap"
	if metadata.Format == "pcapng" {
		mediaType = "application/x-pcapng"
	}
	c.Header("Content-Type", mediaType)
	c.Header("X-NetLab-Capture-ID", string(metadata.ID))
	c.Header("X-NetLab-Capture-Filter", metadata.Filter)
	c.Status(http.StatusOK)
	c.Stream(func(writer io.Writer) bool {
		chunk, ok := <-stream
		if !ok {
			return false
		}
		_, _ = writer.Write(chunk)
		return true
	})
}

func (h *CaptureHandlers) startTrafficFilter(c *gin.Context) {
	var body struct {
		LaboratoryID         domain.ID            `json:"laboratory_id"`
		Match                captureRuntime.Match `json:"match"`
		MaxObservations      int                  `json:"max_observations"`
		InterfaceIDs         []domain.ID          `json:"interface_ids"`
		LinkIDs              []domain.ID          `json:"link_ids"`
		NetworkObjectLinkIDs []domain.ID          `json:"network_object_link_ids"`
		Color                string               `json:"color"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		handleError(c, err)
		return
	}
	if h.operations == nil {
		handleError(c, domain.Problem{Code: "operation_unavailable", Message: "traffic filter automation unavailable"})
		return
	}
	value, taskValue, err := h.operations.StartFilterWithObjectLinks(c, body.LaboratoryID, body.Match, body.MaxObservations, body.InterfaceIDs, body.LinkIDs, body.NetworkObjectLinkIDs, body.Color, c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"traffic_filter": value, "task": taskValue})
}

func (h *CaptureHandlers) getTrafficFilter(c *gin.Context) {
	value, ambiguous, err := h.filters.Get(domain.ID(c.Param("filterId")))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"traffic_filter": value, "ambiguous": ambiguous})
}

func (h *CaptureHandlers) stopTrafficFilter(c *gin.Context) {
	if h.operations == nil {
		handleError(c, domain.Problem{Code: "operation_unavailable", Message: "traffic filter automation unavailable"})
		return
	}
	value, err := h.operations.StopFilter(c, domain.ID(c.Param("filterId")), c.GetHeader("Idempotency-Key"))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"task": value})
}

func (h *CaptureHandlers) deleteTrafficFilterHistory(c *gin.Context) {
	if h.operations == nil {
		handleError(c, domain.Problem{Code: "operation_unavailable", Message: "traffic filter automation unavailable"})
		return
	}
	value, err := h.operations.DeleteFilterHistory(domain.ID(c.Param("filterId")))
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"traffic_filter": value})
}
