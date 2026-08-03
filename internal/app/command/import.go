package command

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type ImportRepository interface {
	ImportTopology(context.Context, domain.Laboratory, []domain.Node, []domain.Interface, []domain.Link, []domain.NetworkObject, []domain.NetworkObjectLink) error
}

type ImportLaboratoryReader interface {
	GetLaboratory(context.Context, domain.ID) (domain.Laboratory, error)
}

type DependencyResolver interface {
	HasImageDigest(context.Context, string) (bool, error)
}

type ImportService struct {
	repository   ImportRepository
	dependencies DependencyResolver
}

func NewImportService(repository ImportRepository, dependencies DependencyResolver) *ImportService {
	return &ImportService{repository: repository, dependencies: dependencies}
}

func (s *ImportService) Preflight(ctx context.Context, bundle LaboratoryExport) error {
	if bundle.SchemaVersion != 1 {
		return fmt.Errorf("unsupported export schema version %d", bundle.SchemaVersion)
	}
	if strings.TrimSpace(bundle.Laboratory.Name) == "" {
		return fmt.Errorf("laboratory name required")
	}
	if !bundle.Redaction.ImagesExcluded || !bundle.Redaction.CredentialsExcluded || !bundle.Redaction.BootstrapSecretsExcluded || !bundle.Redaction.CapturesExcluded {
		return fmt.Errorf("export redaction report is incomplete")
	}
	for _, dependency := range bundle.TemplateVersions {
		if !digestPattern.MatchString(dependency.ImageDigest) {
			return fmt.Errorf("invalid image digest for %s", dependency.TemplateKey)
		}
		if s.dependencies != nil {
			available, err := s.dependencies.HasImageDigest(ctx, dependency.ImageDigest)
			if err != nil {
				return err
			}
			if !available {
				return fmt.Errorf("missing dependency %s (%s); substitution is not allowed", dependency.TemplateKey, dependency.ImageDigest)
			}
		}
	}
	return nil
}

func (s *ImportService) Import(ctx context.Context, bundle LaboratoryExport) (domain.Laboratory, error) {
	return s.ImportAs(ctx, domain.NewID(), bundle)
}

func (s *ImportService) ImportAs(ctx context.Context, laboratoryID domain.ID, bundle LaboratoryExport) (domain.Laboratory, error) {
	if err := s.Preflight(ctx, bundle); err != nil {
		return domain.Laboratory{}, err
	}
	if reader, ok := s.repository.(ImportLaboratoryReader); ok {
		if existing, err := reader.GetLaboratory(ctx, laboratoryID); err == nil {
			return existing, nil
		}
	}
	now := time.Now().UTC()
	lab := domain.Laboratory{ID: laboratoryID, Name: bundle.Laboratory.Name, Description: bundle.Laboratory.Description, RecoveryPolicy: bundle.Laboratory.RecoveryPolicy, Revision: 1, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if lab.RecoveryPolicy == "" {
		lab.RecoveryPolicy = domain.RecoveryRemainStopped
	}
	nodeIDs := make(map[string]domain.ID, len(bundle.Nodes))
	interfaceIDs := map[string]domain.ID{}
	var nodes []domain.Node
	var interfaces []domain.Interface
	for _, exported := range bundle.Nodes {
		if exported.ExportID == "" || exported.Name == "" {
			return domain.Laboratory{}, fmt.Errorf("node export_id and name required")
		}
		nodeID := domain.NewID()
		nodeIDs[exported.ExportID] = nodeID
		config := exported.Config
		if config == nil {
			config = map[string]any{}
		}
		config = cloneMap(config)
		if err := normalizeNodeNetworkConfig(config); err != nil {
			return domain.Laboratory{}, fmt.Errorf("node %s network configuration: %w", exported.Name, err)
		}
		if exported.TemplateKey != "" {
			config["template_key"] = exported.TemplateKey
		}
		if exported.ImageDigest != "" {
			config["image_digest"] = exported.ImageDigest
		}
		desired := exported.DesiredState
		if desired != domain.DesiredRunning {
			desired = domain.DesiredStopped
		}
		nodes = append(nodes, domain.Node{ID: nodeID, LaboratoryID: lab.ID, Name: exported.Name, Kind: exported.Kind, Revision: 1, DesiredState: desired, ObservedState: domain.ObservedStopped, Config: config, CreatedAt: now, UpdatedAt: now})
	}
	maxSlots := map[string]int{}
	for _, link := range bundle.Links {
		for _, endpoint := range []string{link.EndpointA, link.EndpointB} {
			nodeExportID, slot, err := parseExportEndpoint(endpoint)
			if err != nil {
				return domain.Laboratory{}, err
			}
			if _, exists := nodeIDs[nodeExportID]; !exists {
				return domain.Laboratory{}, fmt.Errorf("link endpoint references missing node %s", nodeExportID)
			}
			if slot+1 > maxSlots[nodeExportID] {
				maxSlots[nodeExportID] = slot + 1
			}
		}
	}
	for exportID, nodeID := range nodeIDs {
		count := maxSlots[exportID]
		if count < 1 {
			count = 1
		}
		for slot := 0; slot < count; slot++ {
			id := domain.NewID()
			interfaceIDs[fmt.Sprintf("%s:%d", exportID, slot)] = id
			interfaces = append(interfaces, domain.Interface{ID: id, NodeID: nodeID, Slot: slot, Name: fmt.Sprintf("eth%d", slot), MACAddress: randomMAC(), OperationalState: "down", Revision: 1})
		}
	}
	var links []domain.Link
	for _, exported := range bundle.Links {
		links = append(links, domain.Link{ID: domain.NewID(), LaboratoryID: lab.ID, EndpointAID: interfaceIDs[exported.EndpointA], EndpointBID: interfaceIDs[exported.EndpointB], Revision: 1, DesiredState: "connected", ObservedState: "pending"})
	}
	var networkObjects []domain.NetworkObject
	networkObjectIDs := make(map[string]domain.ID, len(bundle.NetworkObjects))
	for _, exported := range bundle.NetworkObjects {
		exportID := exportedText(exported["export_id"])
		name, _ := exported["name"].(string)
		kind, _ := exported["kind"].(string)
		config, _ := exported["config"].(map[string]any)
		if exportID == "" || name == "" || domain.ValidateNetworkKind(kind) != nil {
			return domain.Laboratory{}, fmt.Errorf("invalid exported network object")
		}
		if _, exists := networkObjectIDs[exportID]; exists {
			return domain.Laboratory{}, fmt.Errorf("duplicate network object export_id %s", exportID)
		}
		objectID := domain.NewID()
		networkObjectIDs[exportID] = objectID
		networkObjects = append(networkObjects, domain.NetworkObject{ID: objectID, LaboratoryID: lab.ID, Name: name, Kind: kind, Revision: 1, DesiredState: "active", ObservedState: "pending", Config: config, CreatedAt: now, UpdatedAt: now})
	}
	var networkObjectLinks []domain.NetworkObjectLink
	for _, exported := range bundle.NetworkObjectLinks {
		objectAID, objectAExists := networkObjectIDs[exported.ObjectAExportID]
		objectBID, objectBExists := networkObjectIDs[exported.ObjectBExportID]
		if !objectAExists || !objectBExists || objectAID == objectBID {
			return domain.Laboratory{}, fmt.Errorf("network object link references invalid endpoints")
		}
		if err := domain.ValidateNetworkObjectPortName(exported.PortAName); err != nil {
			return domain.Laboratory{}, err
		}
		if err := domain.ValidateNetworkObjectPortName(exported.PortBName); err != nil {
			return domain.Laboratory{}, err
		}
		desiredState := exported.DesiredState
		if desiredState != "deleted" {
			desiredState = "connected"
		}
		networkObjectLinks = append(networkObjectLinks, domain.NetworkObjectLink{ID: domain.NewID(), LaboratoryID: lab.ID, ObjectAID: objectAID, PortAName: exported.PortAName, ObjectBID: objectBID, PortBName: exported.PortBName, Revision: 1, DesiredState: desiredState, ObservedState: "pending"})
	}
	return lab, s.repository.ImportTopology(ctx, lab, nodes, interfaces, links, networkObjects, networkObjectLinks)
}

func exportedText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case domain.ID:
		return string(typed)
	default:
		return ""
	}
}

func parseExportEndpoint(value string) (string, int, error) {
	index := strings.LastIndexByte(value, ':')
	if index < 1 {
		return "", 0, fmt.Errorf("invalid endpoint %q", value)
	}
	slot, err := strconv.Atoi(value[index+1:])
	if err != nil || slot < 0 || slot > 255 {
		return "", 0, fmt.Errorf("invalid endpoint slot %q", value)
	}
	return value[:index], slot, nil
}
