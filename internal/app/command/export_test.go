package command

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

type routeExportReader struct{ snapshot domain.TopologySnapshot }

func (r routeExportReader) Snapshot(context.Context, domain.ID) (domain.TopologySnapshot, error) {
	return r.snapshot, nil
}

func TestExportPreservesDockerStaticRoutes(t *testing.T) {
	node := domain.Node{ID: "node", Name: "docker", Kind: "docker", Config: map[string]any{"network_interfaces": []any{map[string]any{
		"name": "eth0", "modes": []any{"static"}, "addresses": []any{"192.0.2.10/24"},
		"routes": []any{map[string]any{"destination": "0.0.0.0/0", "gateway": "192.0.2.1", "metric": float64(10)}},
	}}}}
	bundle, err := NewExportService(routeExportReader{snapshot: domain.TopologySnapshot{Laboratory: domain.Laboratory{Name: "lab", RecoveryPolicy: domain.RecoveryRemainStopped}, Nodes: []domain.Node{node}}}, nil).Build(context.Background(), "lab")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(bundle.Nodes[0].Config)
	if !strings.Contains(string(body), `"destination":"0.0.0.0/0"`) || !strings.Contains(string(body), `"metric":10`) {
		t.Fatalf("config=%s", body)
	}
}
