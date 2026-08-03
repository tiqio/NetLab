package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/audit"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
)

type ToolHandler func(*gin.Context, map[string]any) (any, error)

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Handler     ToolHandler    `json:"-"`
}

type Server struct {
	tools       map[string]Tool
	idempotency *command.IdempotencyService
	audit       *audit.Service
}

func NewServer(tools []Tool, options ...any) *Server {
	server := &Server{tools: make(map[string]Tool, len(tools))}
	for _, option := range options {
		if value, ok := option.(*command.IdempotencyService); ok {
			server.idempotency = value
		}
		if value, ok := option.(*audit.Service); ok {
			server.audit = value
		}
	}
	for _, tool := range tools {
		server.tools[tool.Name] = tool
	}
	return server
}

func (s *Server) Register(engine *gin.Engine) {
	engine.POST("/mcp", s.handle)
	engine.GET("/mcp", func(c *gin.Context) {
		c.Header("Allow", http.MethodPost)
		c.Status(http.StatusMethodNotAllowed)
	})
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

func (s *Server) handle(c *gin.Context) {
	if !validOrigin(c.Request) {
		c.JSON(http.StatusForbidden, response{JSONRPC: "2.0", Error: &rpcError{Code: -32000, Message: "origin is not allowed", Data: map[string]any{"code": "origin_rejected", "retryable": false}}})
		return
	}
	if !strings.Contains(c.GetHeader("Accept"), "application/json") && c.GetHeader("Accept") != "" && !strings.Contains(c.GetHeader("Accept"), "text/event-stream") {
		c.JSON(http.StatusNotAcceptable, response{JSONRPC: "2.0", Error: &rpcError{Code: -32600, Message: "Accept must allow application/json or text/event-stream"}})
		return
	}
	var request request
	if err := c.ShouldBindJSON(&request); err != nil || request.JSONRPC != "2.0" {
		c.JSON(http.StatusBadRequest, response{JSONRPC: "2.0", ID: request.ID, Error: &rpcError{Code: -32700, Message: "invalid JSON-RPC request"}})
		return
	}
	switch request.Method {
	case "initialize":
		c.Header("Mcp-Session-Id", string(domain.NewID()))
		c.JSON(http.StatusOK, response{JSONRPC: "2.0", ID: request.ID, Result: map[string]any{"protocolVersion": "2025-06-18", "capabilities": map[string]any{"tools": map[string]any{"listChanged": false}}, "serverInfo": map[string]any{"name": "netlab", "version": "1.0.0"}}})
	case "notifications/initialized":
		c.Status(http.StatusAccepted)
	case "tools/list":
		tools := make([]Tool, 0, len(s.tools))
		for _, tool := range s.tools {
			tools = append(tools, tool)
		}
		c.JSON(http.StatusOK, response{JSONRPC: "2.0", ID: request.ID, Result: map[string]any{"tools": tools}})
	case "tools/call":
		s.callTool(c, request)
	default:
		c.JSON(http.StatusOK, response{JSONRPC: "2.0", ID: request.ID, Error: &rpcError{Code: -32601, Message: "method not found"}})
	}
}

func (s *Server) callTool(c *gin.Context, request request) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(request.Params, &params); err != nil {
		c.JSON(http.StatusOK, response{JSONRPC: "2.0", ID: request.ID, Error: &rpcError{Code: -32602, Message: "invalid tool parameters"}})
		return
	}
	tool, ok := s.tools[params.Name]
	if !ok {
		c.JSON(http.StatusOK, response{JSONRPC: "2.0", ID: request.ID, Error: &rpcError{Code: -32602, Message: "unknown tool"}})
		return
	}
	type invocation struct {
		Value   any             `json:"value,omitempty"`
		Problem *domain.Problem `json:"problem,omitempty"`
	}
	invoke := func(ctx context.Context) (int, []byte, error) {
		value, err := tool.Handler(c, params.Arguments)
		if err != nil {
			problem := problemFromError(err)
			body, marshalErr := json.Marshal(invocation{Problem: &problem})
			return http.StatusOK, body, marshalErr
		}
		body, marshalErr := json.Marshal(invocation{Value: value})
		return http.StatusOK, body, marshalErr
	}
	var resultValue invocation
	var err error
	key, _ := params.Arguments["idempotency_key"].(string)
	if s.idempotency != nil && key != "" && mutationTool(params.Name) {
		requestBody, _ := json.Marshal(params.Arguments)
		result, executeErr := s.idempotency.Execute(c, "MCP:"+params.Name, key, requestBody, invoke)
		err = executeErr
		if err == nil {
			err = json.Unmarshal(result.Body, &resultValue)
		}
	} else {
		_, body, invokeErr := invoke(c)
		err = invokeErr
		if err == nil {
			err = json.Unmarshal(body, &resultValue)
		}
	}
	if err == nil && resultValue.Problem != nil {
		err = *resultValue.Problem
	}
	if mutationTool(params.Name) && s.audit != nil {
		outcome := "succeeded"
		if err != nil {
			outcome = "failed"
		}
		resourceType, resourceID := mcpResource(params.Arguments)
		_, _ = s.audit.Record(context.Background(), "mcp", params.Name, resourceType, resourceID, "", outcome, string(domain.NewID()), map[string]any{"idempotency": key != ""})
	}
	if err != nil {
		problem := problemFromError(err)
		body, _ := json.Marshal(problem)
		c.JSON(http.StatusOK, response{JSONRPC: "2.0", ID: request.ID, Result: map[string]any{"isError": true, "content": []map[string]any{{"type": "text", "text": string(body)}}, "structuredContent": problem}})
		return
	}
	body, _ := json.Marshal(resultValue.Value)
	c.JSON(http.StatusOK, response{JSONRPC: "2.0", ID: request.ID, Result: map[string]any{"content": []map[string]any{{"type": "text", "text": string(body)}}, "structuredContent": resultValue.Value}})
}

func mutationTool(name string) bool {
	for _, suffix := range []string{".list", ".get", ".capabilities"} {
		if strings.HasSuffix(name, suffix) {
			return false
		}
	}
	return name != "netlab.capabilities"
}

func mcpResource(args map[string]any) (string, domain.ID) {
	for _, key := range []string{"node_id", "lab_id", "laboratory_id", "link_id", "interface_id", "capture_id", "filter_id", "mapping_id", "task_id", "object_id"} {
		if value, ok := args[key].(string); ok && value != "" {
			return strings.TrimSuffix(key, "_id"), domain.ID(value)
		}
	}
	return "operation", "unknown"
}

func validOrigin(request *http.Request) bool {
	origin := request.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Host, request.Host)
}

func problemFromError(err error) domain.Problem {
	if errors.Is(err, command.ErrIdempotencyConflict) {
		return domain.Problem{Code: "idempotency_conflict", Message: err.Error(), Retryable: false, Phase: "idempotency", Cleanup: "no duplicate mutation executed", OperatorHint: "reuse the key only with the original payload or choose a new key"}
	}
	if errors.Is(err, command.ErrIdempotencyPending) {
		return domain.Problem{Code: "idempotency_pending", Message: err.Error(), Retryable: true, Phase: "idempotency", Cleanup: "original mutation remains in progress", OperatorHint: "retry the same key after the indicated delay", RetryAfterSeconds: 1}
	}
	return domain.NormalizeProblem(err, domain.Problem{Code: "invalid_request", Retryable: false, Phase: "mcp_tool", Cleanup: "no cleanup required", OperatorHint: "correct the request parameters and retry"})
}

func requiredObject(properties map[string]any, required ...string) map[string]any {
	return map[string]any{"type": "object", "additionalProperties": false, "properties": properties, "required": required}
}

func stringProperty(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func argumentString(arguments map[string]any, name string) (string, error) {
	value, ok := arguments[name].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}
