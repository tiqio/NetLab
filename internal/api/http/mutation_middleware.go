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
		c.Request = c.Request.WithContext(command.WithTopologyConnectionEntryPoint(c.Request.Context(), "compatibility_http"))
		c.Set(command.TopologyConnectionEntryPointContextKey, "compatibility_http")
		if durableTaskMutation(c.Request.Method, c.Request.URL.Path) {
			body, readErr := io.ReadAll(c.Request.Body)
			if readErr != nil {
				handleError(c, readErr)
				c.Abort()
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
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
				var problem domain.Problem
				if c.Writer.Status() >= 400 {
					outcome = "failed"
					if json.Unmarshal(writer.body.Bytes(), &problem) == nil && (problem.Code == "port_in_use" || problem.Code == "revision_conflict" || problem.Code == "idempotency_conflict") {
						outcome = "conflict"
					}
				}
				entryPoint := c.GetString("topology_entry_point")
				if entryPoint == "" {
					entryPoint = command.TopologyConnectionEntryPoint(c.Request.Context(), "compatibility_http")
				}
				details := topologyMutationAuditDetails(c.Request.URL.Path, body)
				details["status"] = c.Writer.Status()
				details["entry_point"] = entryPoint
				if problem.Code != "" {
					details["problem_code"] = problem.Code
					details["cleanup"] = problem.Cleanup
				}
				_, _ = audits.Record(context.Background(), "api", c.Request.Method+":"+c.FullPath(), resourceType, resourceID, taskID, outcome, string(taskID), details)
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
				_, _ = audits.Record(context.Background(), "api", c.Request.Method+":"+c.FullPath(), resourceType, resourceID, "", outcome, "", map[string]any{"status": status, "entry_point": command.TopologyConnectionEntryPoint(c.Request.Context(), "compatibility_http")})
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

func topologyMutationAuditDetails(path string, body []byte) map[string]any {
	details := map[string]any{}
	if len(body) == 0 {
		return details
	}
	var input map[string]any
	if json.Unmarshal(body, &input) != nil {
		return details
	}
	for _, key := range []string{"source", "target", "config", "endpoint_a_id", "endpoint_b_id", "interface_id", "port_name", "object_a_id", "port_a_name", "object_b_id", "port_b_name"} {
		if value, ok := input[key]; ok {
			details[key] = value
		}
	}
	if strings.Contains(path, "/connections") || strings.Contains(path, "/links") || strings.Contains(path, "/attachments") {
		details["connection_summary"] = true
	}
	return details
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
	if method == http.MethodPost && len(parts) == 6 && parts[2] == "nodes" && parts[4] == "credentials" && parts[5] == "console-admin" {
		return false
	}
	if method == http.MethodPost && len(parts) == 7 && parts[2] == "nodes" && parts[4] == "credentials" && parts[5] == "console-admin" && parts[6] == "verify" {
		return true
	}
	if method == http.MethodPost && len(parts) == 6 && parts[2] == "nodes" && parts[4] == "bootstrap" && parts[5] == "fortigate" {
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
	if method == http.MethodPost && len(parts) == 5 && parts[2] == "labs" && parts[4] == "connections" {
		return true
	}
	if method == http.MethodDelete && len(parts) == 4 && parts[2] == "connections" {
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
	collections := map[string]string{"labs": "laboratory", "nodes": "node", "links": "link", "connections": "topology_connection", "interfaces": "interface", "network-objects": "network_object", "port-mappings": "port_mapping", "captures": "capture", "traffic-filters": "traffic_filter", "tasks": "operation_task"}
	for index := len(parts) - 2; index >= 0; index-- {
		if resourceType := collections[parts[index]]; resourceType != "" && index+1 < len(parts) {
			return resourceType, domain.ID(parts[index+1])
		}
	}
	return "operation", "unknown"
}
