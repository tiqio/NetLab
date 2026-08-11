package httpapi

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNetworkRecoveryRoutesAreRegistered(t *testing.T) {
	engine := gin.New()
	(&NetworkHandlers{}).Register(engine)
	routes := map[string]bool{}
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	for _, want := range []string{"POST /api/v1/network-objects/:objectId/reconcile", "POST /api/v1/network-object-links/:linkId/reconcile", "GET /api/v1/network-objects/:objectId/diagnostics"} {
		if !routes[want] {
			t.Fatalf("missing route %s", want)
		}
	}
}
