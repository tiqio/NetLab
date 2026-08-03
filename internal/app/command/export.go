package command

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/app/audit"
	"github.com/netlab/netlab/internal/domain"
)

var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

type ExportRedaction struct {
	ImagesExcluded           bool `json:"images_excluded"`
	CredentialsExcluded      bool `json:"credentials_excluded"`
	BootstrapSecretsExcluded bool `json:"bootstrap_secrets_excluded"`
	CapturesExcluded         bool `json:"captures_excluded"`
}

type ExportLaboratory struct {
	Name           string                `json:"name"`
	Description    string                `json:"description"`
	RecoveryPolicy domain.RecoveryPolicy `json:"recovery_policy"`
}

type ExportTemplateVersion struct {
	TemplateKey     string `json:"template_key"`
	ManifestVersion int    `json:"manifest_version"`
	ImageDigest     string `json:"image_digest"`
}

type ExportNode struct {
	ExportID     string              `json:"export_id"`
	Name         string              `json:"name"`
	Kind         string              `json:"kind"`
	TemplateKey  string              `json:"template_key,omitempty"`
	ImageDigest  string              `json:"image_digest,omitempty"`
	DesiredState domain.DesiredState `json:"desired_state"`
	Config       map[string]any      `json:"config"`
}

type ExportLink struct {
	EndpointA string `json:"endpoint_a"`
	EndpointB string `json:"endpoint_b"`
}

type ExportNetworkObjectLink struct {
	ObjectAExportID string `json:"object_a_export_id"`
	PortAName       string `json:"port_a_name"`
	ObjectBExportID string `json:"object_b_export_id"`
	PortBName       string `json:"port_b_name"`
	DesiredState    string `json:"desired_state"`
}

type LaboratoryExport struct {
	SchemaVersion      int                       `json:"schema_version"`
	ExportedAt         time.Time                 `json:"exported_at"`
	Laboratory         ExportLaboratory          `json:"laboratory"`
	TemplateVersions   []ExportTemplateVersion   `json:"template_versions"`
	Nodes              []ExportNode              `json:"nodes"`
	Links              []ExportLink              `json:"links"`
	NetworkObjects     []map[string]any          `json:"network_objects"`
	NetworkObjectLinks []ExportNetworkObjectLink `json:"network_object_links,omitempty"`
	Redaction          ExportRedaction           `json:"redaction"`
}

type ExportSnapshotReader interface {
	Snapshot(context.Context, domain.ID) (domain.TopologySnapshot, error)
}

type ExportArtifactWriter interface {
	Create(context.Context, string, string, string, domain.ID, []byte, time.Duration) (domain.Artifact, error)
}

type ExportArtifactIDWriter interface {
	CreateWithID(context.Context, domain.ID, string, string, string, domain.ID, []byte, time.Duration) (domain.Artifact, error)
}

type ExportService struct {
	reader    ExportSnapshotReader
	artifacts ExportArtifactWriter
}

func NewExportService(reader ExportSnapshotReader, artifacts ExportArtifactWriter) *ExportService {
	return &ExportService{reader: reader, artifacts: artifacts}
}

func (s *ExportService) Build(ctx context.Context, labID domain.ID) (LaboratoryExport, error) {
	snapshot, err := s.reader.Snapshot(ctx, labID)
	if err != nil {
		return LaboratoryExport{}, err
	}
	value := LaboratoryExport{
		SchemaVersion:  1,
		ExportedAt:     time.Now().UTC(),
		Laboratory:     ExportLaboratory{Name: snapshot.Laboratory.Name, Description: snapshot.Laboratory.Description, RecoveryPolicy: snapshot.Laboratory.RecoveryPolicy},
		NetworkObjects: []map[string]any{},
		Redaction:      ExportRedaction{ImagesExcluded: true, CredentialsExcluded: true, BootstrapSecretsExcluded: true, CapturesExcluded: true},
	}
	interfaces := make(map[domain.ID]domain.Interface, len(snapshot.Interfaces))
	nodes := make(map[domain.ID]domain.Node, len(snapshot.Nodes))
	dependencies := map[string]ExportTemplateVersion{}
	for _, iface := range snapshot.Interfaces {
		interfaces[iface.ID] = iface
	}
	for _, node := range snapshot.Nodes {
		nodes[node.ID] = node
		config := exportConfig(node.Config)
		templateKey, _ := config["template_key"].(string)
		imageDigest, _ := config["image_digest"].(string)
		if imageDigest != "" && !digestPattern.MatchString(imageDigest) {
			return LaboratoryExport{}, fmt.Errorf("node %s has invalid image digest", node.Name)
		}
		value.Nodes = append(value.Nodes, ExportNode{ExportID: string(node.ID), Name: node.Name, Kind: node.Kind, TemplateKey: templateKey, ImageDigest: imageDigest, DesiredState: node.DesiredState, Config: config})
		if templateKey != "" && imageDigest != "" {
			manifestVersion := 1
			if raw, ok := config["manifest_version"].(float64); ok && raw >= 1 {
				manifestVersion = int(raw)
			}
			dependencies[templateKey+"@"+imageDigest] = ExportTemplateVersion{TemplateKey: templateKey, ManifestVersion: manifestVersion, ImageDigest: imageDigest}
		}
	}
	for _, link := range snapshot.Links {
		a, aOK := interfaces[link.EndpointAID]
		b, bOK := interfaces[link.EndpointBID]
		if !aOK || !bOK {
			return LaboratoryExport{}, fmt.Errorf("link %s references missing interface", link.ID)
		}
		value.Links = append(value.Links, ExportLink{EndpointA: fmt.Sprintf("%s:%d", nodes[a.NodeID].ID, a.Slot), EndpointB: fmt.Sprintf("%s:%d", nodes[b.NodeID].ID, b.Slot)})
	}
	for _, networkObject := range snapshot.NetworkObjects {
		value.NetworkObjects = append(value.NetworkObjects, map[string]any{
			"export_id": networkObject.ID,
			"name":      networkObject.Name,
			"kind":      networkObject.Kind,
			"config":    exportConfig(networkObject.Config),
		})
	}
	for _, link := range snapshot.NetworkObjectLinks {
		value.NetworkObjectLinks = append(value.NetworkObjectLinks, ExportNetworkObjectLink{ObjectAExportID: string(link.ObjectAID), PortAName: link.PortAName, ObjectBExportID: string(link.ObjectBID), PortBName: link.PortBName, DesiredState: link.DesiredState})
	}
	for _, dependency := range dependencies {
		value.TemplateVersions = append(value.TemplateVersions, dependency)
	}
	sort.Slice(value.TemplateVersions, func(i, j int) bool {
		return value.TemplateVersions[i].TemplateKey < value.TemplateVersions[j].TemplateKey
	})
	return value, nil
}

func exportConfig(input map[string]any) map[string]any {
	return audit.Redact(stripRuntimeExportValues(input))
}

func stripRuntimeExportValues(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	result := make(map[string]any, len(input))
	for key, value := range input {
		if runtimeExportKey(key) {
			continue
		}
		switch typed := value.(type) {
		case map[string]any:
			result[key] = stripRuntimeExportValues(typed)
		case []any:
			items := make([]any, 0, len(typed))
			for _, item := range typed {
				if nested, ok := item.(map[string]any); ok {
					items = append(items, stripRuntimeExportValues(nested))
				} else {
					items = append(items, item)
				}
			}
			result[key] = items
		default:
			result[key] = value
		}
	}
	return result
}

func runtimeExportKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	for _, marker := range []string{"container_pid", "namespace_pid", "namespace_name", "runtime_interface", "runtime_locator", "host_interface", "packet_payload", "packet_bytes", "capture_payload"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func (s *ExportService) CreateArtifact(ctx context.Context, labID domain.ID, ttl time.Duration) (domain.Artifact, error) {
	return s.CreateArtifactAs(ctx, domain.NewID(), labID, ttl)
}

func (s *ExportService) CreateArtifactAs(ctx context.Context, artifactID, labID domain.ID, ttl time.Duration) (domain.Artifact, error) {
	value, err := s.Build(ctx, labID)
	if err != nil {
		return domain.Artifact{}, err
	}
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return domain.Artifact{}, err
	}
	if writer, ok := s.artifacts.(ExportArtifactIDWriter); ok {
		return writer.CreateWithID(ctx, artifactID, "laboratory_export", "application/vnd.netlab.lab+json", "laboratory", labID, body, ttl)
	}
	return s.artifacts.Create(ctx, "laboratory_export", "application/vnd.netlab.lab+json", "laboratory", labID, body, ttl)
}
