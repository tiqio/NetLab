package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/support/observability"
)

type Server struct {
	engine *gin.Engine
	server *http.Server
	logger *slog.Logger
}

func NewServer(address string, logger *slog.Logger, metrics *observability.Metrics) *Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery(), requestID(), cors(), RequestLimits(4<<20))
	engine.GET("/healthz", gin.WrapF(observability.HealthHandler))
	engine.GET("/readyz", gin.WrapF(observability.HealthHandler))
	engine.GET("/metrics", gin.WrapH(observability.MetricsHandler(metrics)))
	assets, _ := fs.Sub(WebAssets, "webdist")
	staticAssets, _ := fs.Sub(WebAssets, "webdist/assets")
	engine.StaticFS("/assets", http.FS(staticAssets))
	engine.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			writeProblem(c, http.StatusNotFound, domain.Problem{Code: "not_found", Message: "resource not found"})
			return
		}
		body, err := fs.ReadFile(assets, "index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", body)
	})
	return &Server{engine: engine, server: &http.Server{Addr: address, Handler: engine, ReadHeaderTimeout: 10 * time.Second}, logger: logger}
}
func (s *Server) Engine() *gin.Engine { return s.engine }
func (s *Server) Start() error {
	if s.logger != nil {
		s.logger.Info("http server starting", "address", s.server.Addr)
	}
	err := s.server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
func (s *Server) Shutdown(ctx context.Context) error { return s.server.Shutdown(ctx) }
func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = string(domain.NewID())
		}
		c.Header("X-Request-ID", id)
		c.Set("request_id", id)
		c.Next()
	}
}
func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Allow-Headers", "Content-Type,Idempotency-Key,If-Match,X-Request-ID")
			c.Header("Access-Control-Allow-Methods", "GET,POST,PATCH,PUT,DELETE,OPTIONS")
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}
func writeProblem(c *gin.Context, status int, problem domain.Problem) {
	SetRetryHeaders(c, problem)
	c.Header("Content-Type", "application/problem+json")
	c.Status(status)
	_ = json.NewEncoder(c.Writer).Encode(problem)
}
func ParseRevision(value string) (domain.Revision, error) {
	var revision int64
	if _, err := fmt.Sscanf(strings.Trim(value, "\""), "%d", &revision); err != nil {
		return 0, err
	}
	return domain.Revision(revision), nil
}
