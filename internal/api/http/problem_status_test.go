package httpapi

import (
	"net/http"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func TestNodeNameConflictUsesHTTPConflict(t *testing.T) {
	if status := problemHTTPStatus(domain.Problem{Code: "node_name_conflict"}); status != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, status)
	}
}
