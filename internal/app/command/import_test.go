package command

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

type routeImportRepository struct {
	nodes              []domain.Node
	networkObjects     []domain.NetworkObject
	networkObjectLinks []domain.NetworkObjectLink
	placements         []domain.TopologyPlacement
}

func (r *routeImportRepository) ImportTopology(_ context.Context, _ domain.Laboratory, nodes []domain.Node, _ []domain.Interface, _ []domain.Link, networkObjects []domain.NetworkObject, objectLinks []domain.NetworkObjectLink, placements []domain.TopologyPlacement) error {
	r.nodes = nodes
	r.networkObjects = networkObjects
	r.networkObjectLinks = objectLinks
	r.placements = placements
	return nil
}

func TestImportPreservesLegacySinglePortNetworkObject(t *testing.T) {
	repository := &routeImportRepository{}
	bundle := LaboratoryExport{SchemaVersion: 1, Laboratory: ExportLaboratory{Name: "legacy", RecoveryPolicy: domain.RecoveryRemainStopped}, NetworkObjects: []map[string]any{{"export_id": "legacy-switch", "name": "Legacy", "kind": "switch_l2", "config": map[string]any{"ports": []any{map[string]any{"name": "lan0"}}}}}, Redaction: ExportRedaction{ImagesExcluded: true, CredentialsExcluded: true, BootstrapSecretsExcluded: true, CapturesExcluded: true}}
	if _, err := NewImportService(repository, nil).ImportAs(context.Background(), "legacy", bundle); err != nil {
		t.Fatal(err)
	}
	ports := repository.networkObjects[0].Config["ports"].([]any)
	if len(ports) != 1 || ports[0].(map[string]any)["name"] != "lan0" {
		t.Fatalf("import expanded legacy ports: %+v", repository.networkObjects[0].Config)
	}
}

func TestImportRemapsNetworkObjectLinkEndpoints(t *testing.T) {
	repository := &routeImportRepository{}
	bundle := LaboratoryExport{SchemaVersion: 1, Laboratory: ExportLaboratory{Name: "links", RecoveryPolicy: domain.RecoveryRemainStopped}, NetworkObjects: []map[string]any{{"export_id": "switch-a", "name": "A", "kind": "switch_l2", "config": map[string]any{"ports": []any{map[string]any{"name": "swp1"}}}}, {"export_id": "switch-b", "name": "B", "kind": "switch_l2", "config": map[string]any{"ports": []any{map[string]any{"name": "swp2"}}}}}, NetworkObjectLinks: []ExportNetworkObjectLink{{ObjectAExportID: "switch-a", PortAName: "swp1", ObjectBExportID: "switch-b", PortBName: "swp2", DesiredState: "connected"}}, Redaction: ExportRedaction{ImagesExcluded: true, CredentialsExcluded: true, BootstrapSecretsExcluded: true, CapturesExcluded: true}}
	if _, err := NewImportService(repository, nil).ImportAs(context.Background(), "lab", bundle); err != nil {
		t.Fatal(err)
	}
	if len(repository.networkObjectLinks) != 1 || repository.networkObjectLinks[0].ObjectAID == "switch-a" || repository.networkObjectLinks[0].ObjectBID == "switch-b" || repository.networkObjectLinks[0].PortAName != "swp1" {
		t.Fatalf("links=%+v", repository.networkObjectLinks)
	}
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

func TestImportPreservesExportedPlacementAndAllocatesOnlyMissingResources(t *testing.T) {
	repository := &routeImportRepository{}
	bundle := LaboratoryExport{
		SchemaVersion: 1,
		Laboratory:    ExportLaboratory{Name: "placements", RecoveryPolicy: domain.RecoveryRemainStopped},
		Nodes: []ExportNode{
			{ExportID: "node-a", Name: "A", Kind: "docker", Config: map[string]any{}},
			{ExportID: "node-b", Name: "B", Kind: "docker", Config: map[string]any{}},
		},
		Placements: []ExportPlacement{{ResourceExportID: "node-a", ResourceType: domain.PlacementNode, X: 320, Y: -140, Revision: 4}},
		Redaction:  ExportRedaction{ImagesExcluded: true, CredentialsExcluded: true, BootstrapSecretsExcluded: true, CapturesExcluded: true},
	}
	if _, err := NewImportService(repository, nil).ImportAs(context.Background(), "lab", bundle); err != nil {
		t.Fatal(err)
	}
	if len(repository.placements) != 2 {
		t.Fatalf("placements=%+v", repository.placements)
	}
	byID := map[domain.ID]domain.TopologyPlacement{}
	for _, placement := range repository.placements {
		byID[placement.ResourceID] = placement
	}
	if preserved := byID[repository.nodes[0].ID]; preserved.X != 320 || preserved.Y != -140 || preserved.Revision != 4 {
		t.Fatalf("preserved=%+v", preserved)
	}
	if fallback := byID[repository.nodes[1].ID]; fallback.Revision != 1 || (fallback.X == 320 && fallback.Y == -140) {
		t.Fatalf("fallback=%+v", fallback)
	}
}
