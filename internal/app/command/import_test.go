package command

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

type routeImportRepository struct{ nodes []domain.Node }

func (r *routeImportRepository) ImportTopology(_ context.Context, _ domain.Laboratory, nodes []domain.Node, _ []domain.Interface, _ []domain.Link, _ []domain.NetworkObject) error {
	r.nodes = nodes
	return nil
}

func TestImportPreservesAndCanonicalizesDockerStaticRoutes(t *testing.T) {
	repository := &routeImportRepository{}
	bundle := LaboratoryExport{
		SchemaVersion: 1,
		Laboratory:    ExportLaboratory{Name: "import", RecoveryPolicy: domain.RecoveryRemainStopped},
		Nodes: []ExportNode{{ExportID: "docker-1", Name: "docker", Kind: "docker", Config: map[string]any{"network_interfaces": []any{map[string]any{
			"name": "eth0", "modes": []any{"static"}, "addresses": []any{"192.0.2.10/24"},
			"routes": []any{map[string]any{"destination": "198.51.100.99/24", "gateway": "192.0.2.1"}},
		}}}}},
		Redaction: ExportRedaction{ImagesExcluded: true, CredentialsExcluded: true, BootstrapSecretsExcluded: true, CapturesExcluded: true},
	}
	if _, err := NewImportService(repository, nil).ImportAs(context.Background(), "lab", bundle); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(repository.nodes[0].Config)
	if !strings.Contains(string(body), `"destination":"198.51.100.0/24"`) {
		t.Fatalf("config=%s", body)
	}
}
