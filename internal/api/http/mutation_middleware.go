package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/audit"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
)

type MutationTaskStore interface {
}

type captureWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (w *captureWriter) Write(body []byte) (int, error) {
	w.body.Write(body)
	return w.ResponseWriter.Write(body)
}
func (w *captureWriter) WriteString(value string) (int, error) {
	w.body.WriteString(value)
	return w.ResponseWriter.WriteString(value)
}

func MutationAutomation(idempotency *command.IdempotencyService, _ MutationTaskStore, audits *audit.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api/v1/") || !isMutation(c.Request.Method) {
			c.Next()
			return
		}
		if durableTaskMutation(c.Request.Method, c.Request.URL.Path) {
			writer := &captureWriter{ResponseWriter: c.Writer}
			c.Writer = writer
			c.Next()
			if audits != nil {
				resourceType, resourceID := mutationResource(c.Request.URL.Path)
				taskID := domain.ID("")
				var envelope struct {
					Task domain.OperationTask `json:"task"`
				}
				if json.Unmarshal(writer.body.Bytes(), &envelope) == nil && envelope.Task.ID != "" {
					taskID = envelope.Task.ID
					resourceType = envelope.Task.ResourceType
					resourceID = envelope.Task.ResourceID
				}
				outcome := "accepted"
				if c.Writer.Status() >= 400 {
					outcome = "failed"
				}
				_, _ = audits.Record(context.Background(), "api", c.Request.Method+":"+c.FullPath(), resourceType, resourceID, taskID, outcome, string(taskID), map[string]any{"status": c.Writer.Status()})
			}
			return
		}
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			handleError(c, err)
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		run := func(ctx context.Context) (int, []byte, error) {
			resourceType, resourceID := mutationResource(c.Request.URL.Path)
			writer := &captureWriter{ResponseWriter: c.Writer}
			c.Writer = writer
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
			c.Next()
			status := c.Writer.Status()
			outcome := "succeeded"
			if status >= 400 {
				outcome = "failed"
			}
			if audits != nil {
				_, _ = audits.Record(context.Background(), "api", c.Request.Method+":"+c.FullPath(), resourceType, resourceID, "", outcome, "", map[string]any{"status": status})
			}
			return status, writer.body.Bytes(), nil
		}
		key := c.GetHeader("Idempotency-Key")
		if key == "" || idempotency == nil || selfManagedIdempotency(c.Request.Method, c.Request.URL.Path) {
			_, _, _ = run(c)
			return
		}
		fingerprintBody := append([]byte(c.Request.URL.RawQuery+"\n"), body...)
		result, executeErr := idempotency.Execute(c, c.Request.Method+":"+c.Request.URL.Path, key, fingerprintBody, run)
		if executeErr != nil {
			if !WriteIdempotencyError(c, executeErr) {
				handleError(c, executeErr)
			}
			c.Abort()
			return
		}
		if result.Replay {
			c.Header("Idempotency-Replayed", "true")
			c.Data(result.StatusCode, "application/json", result.Body)
			c.Abort()
		}
	}
}

func selfManagedIdempotency(method, path string) bool {
	return method == http.MethodPost && path == "/api/v1/labs"
}

func durableTaskMutation(method, path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 || parts[0] != "api" || parts[1] != "v1" {
		return false
	}
	if method == http.MethodPut && len(parts) == 5 && parts[2] == "nodes" && parts[4] == "state" {
		return true
	}
	if method == http.MethodDelete && len(parts) == 4 && (parts[2] == "nodes" || parts[2] == "links" || parts[2] == "interfaces" || parts[2] == "port-mappings") {
		return true
	}
	if method == http.MethodDelete && len(parts) == 4 && parts[2] == "network-objects" {
		return true
	}
	if method == http.MethodDelete && len(parts) == 4 && parts[2] == "labs" {
		return true
	}
	if method == http.MethodPost && len(parts) == 5 && parts[2] == "labs" && parts[4] == "links" {
		return true
	}
	if method == http.MethodPost && len(parts) == 5 && parts[2] == "labs" && (parts[4] == "exports" || parts[4] == "duplicate") {
		return true
	}
	if method == http.MethodPost && len(parts) == 5 && parts[2] == "labs" && parts[4] == "network-objects" {
		return true
	}
	if method == http.MethodPost && len(parts) == 3 && parts[2] == "lab-imports" {
		return true
	}
	if len(parts) == 3 && method == http.MethodPost && (parts[2] == "captures" || parts[2] == "traffic-filters") {
		return true
	}
	if len(parts) == 4 && method == http.MethodDelete && (parts[2] == "captures" || parts[2] == "traffic-filters") {
		return true
	}
	if len(parts) == 5 && method == http.MethodDelete && parts[2] == "traffic-filters" && parts[4] == "history" {
		return true
	}
	return method == http.MethodPost && len(parts) == 5 && parts[2] == "nodes" && (parts[4] == "interfaces" || parts[4] == "guest-exec" || parts[4] == "port-mappings")
}

func isMutation(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}

func mutationResource(path string) (string, domain.ID) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	collections := map[string]string{"labs": "laboratory", "nodes": "node", "links": "link", "interfaces": "interface", "network-objects": "network_object", "port-mappings": "port_mapping", "captures": "capture", "traffic-filters": "traffic_filter", "tasks": "operation_task"}
	for index := len(parts) - 2; index >= 0; index-- {
		if resourceType := collections[parts[index]]; resourceType != "" && index+1 < len(parts) {
			return resourceType, domain.ID(parts[index+1])
		}
	}
	return "operation", "unknown"
}
