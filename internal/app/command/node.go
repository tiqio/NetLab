package command

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/domain"
	qemuRuntime "github.com/netlab/netlab/internal/runtime/qemu"
)

type NodeRepository interface {
	CreateNode(context.Context, domain.Node, []domain.Interface) error
	GetNode(context.Context, domain.ID) (domain.Node, error)
	SetNodeDesiredState(context.Context, domain.ID, domain.Revision, domain.DesiredState) (domain.Node, error)
	DeleteNode(context.Context, domain.ID, domain.Revision) error
}
type NodeTemplateRepository interface {
	GetTemplateVersion(context.Context, domain.ID) (domain.DeviceTemplate, domain.TemplateVersion, error)
	GetImage(context.Context, domain.ID) (domain.ImageVersion, error)
}

type CreateNodeRequest struct {
	Name                string
	Kind                string
	TemplateVersionID   domain.ID
	ImageVersionID      domain.ID
	CPUCount            int
	CPUQuotaMicros      int64
	MemoryMiB           int
	StorageGiB          int
	InterfaceLimit      int
	ProcessLimit        int
	NICDriver           string
	InterfaceCount      int
	Config              map[string]any
	Bootstrap           qemuRuntime.SeedSpec
	ExpectedLabRevision domain.Revision
	PlacementIntent     *domain.PlacementIntent
	Entry               string
	PlacementResult     *CreateNodePlacementResult
}

type CreateNodePlacementResult struct {
	PlacementAssignment *domain.PlacementAssignment `json:"placement_assignment,omitempty"`
	LaboratoryRevision  domain.Revision             `json:"laboratory_revision,omitempty"`
}

type nodePlacementRepository interface {
	CreateNodeWithPlacement(context.Context, domain.Node, []domain.Interface, domain.Revision, *domain.PlacementIntent, string) (domain.PlacementAssignment, domain.Revision, error)
}

type SeedBuilder interface {
	Build(context.Context, domain.ID, domain.ID, qemuRuntime.SeedSpec) (string, error)
}

type NodeService struct {
	repository NodeRepository
	templates  NodeTemplateRepository
	seeds      SeedBuilder
	readiness  interface {
		TemplateReadiness(string) (domain.TemplateReadiness, bool)
	}
}

func (s *NodeService) SetSeedBuilder(builder SeedBuilder) { s.seeds = builder }
func (s *NodeService) SetTemplateReadinessResolver(resolver interface {
	TemplateReadiness(string) (domain.TemplateReadiness, bool)
}) {
	s.readiness = resolver
}

func NewNodeService(repository NodeRepository, templates ...NodeTemplateRepository) *NodeService {
	service := &NodeService{repository: repository}
	if len(templates) > 0 {
		service.templates = templates[0]
	}
	return service
}
func (s *NodeService) Create(ctx context.Context, labID domain.ID, name, kind string, interfaceCount int) (domain.Node, []domain.Interface, error) {
	return s.CreateConfigured(ctx, labID, CreateNodeRequest{Name: name, Kind: kind, InterfaceCount: interfaceCount})
}

func (s *NodeService) CreateConfigured(ctx context.Context, labID domain.ID, request CreateNodeRequest) (domain.Node, []domain.Interface, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return domain.Node{}, nil, fmt.Errorf("node name required")
	}
	kind := request.Kind
	config := cloneMap(request.Config)
	if err := normalizeNodeNetworkConfig(config); err != nil {
		return domain.Node{}, nil, err
	}
	interfaceCount := request.InterfaceCount
	cpuCount, cpuQuota, memoryMiB := request.CPUCount, request.CPUQuotaMicros, request.MemoryMiB
	storageGiB, interfaceLimit, processLimit := request.StorageGiB, request.InterfaceLimit, request.ProcessLimit
	var templateVersion domain.TemplateVersion
	var templateKey string
	if request.TemplateVersionID != "" {
		if s.templates == nil {
			return domain.Node{}, nil, domain.Problem{Code: "capability_unsupported", Message: "template resolver unavailable"}
		}
		template, version, err := s.templates.GetTemplateVersion(ctx, request.TemplateVersionID)
		if err != nil {
			return domain.Node{}, nil, err
		}
		if !version.Enabled {
			return domain.Node{}, nil, domain.Problem{Code: "template_disabled", Message: "template version is disabled"}
		}
		templateVersion = version
		templateKey = template.Key
		kind = string(template.RuntimeKind)
		config["template_key"] = template.Key
		if s.readiness != nil {
			if readiness, ok := s.readiness.TemplateReadiness(template.Key); ok {
				config["template_readiness"] = map[string]any{
					"status":           readiness.Status,
					"genuine_workload": readiness.GenuineWorkload,
					"exception_id":     readiness.ExceptionID,
				}
			}
		}
		if cpuCount == 0 {
			cpuCount = version.Defaults.CPUCount
		}
		if cpuQuota == 0 {
			cpuQuota = version.Defaults.CPUQuotaMicros
		}
		if memoryMiB == 0 {
			memoryMiB = version.Defaults.MemoryMiB
		}
		if interfaceCount == 0 {
			interfaceCount = version.Defaults.Interfaces
		}
		if storageGiB == 0 {
			storageGiB = version.Defaults.DiskGiB
		}
		for key, value := range version.RuntimeOptions {
			if _, exists := config[key]; !exists {
				config[key] = value
			}
		}
		config["console_modes"] = append([]string(nil), version.ConsoleModes...)
		imageID := request.ImageVersionID
		if imageID == "" {
			imageID = version.ImageVersionID
		}
		if imageID == "" {
			return domain.Node{}, nil, domain.Problem{Code: "image_required", Message: "an image version must be selected", ResourceType: "template_version", ResourceID: version.ID}
		}
		if version.ImageVersionID != "" && version.ImageVersionID != imageID {
			return domain.Node{}, nil, domain.Problem{Code: "image_incompatible", Message: "image is not assigned to the selected template version"}
		}
		image, err := s.templates.GetImage(ctx, imageID)
		if err != nil {
			return domain.Node{}, nil, err
		}
		if image.RuntimeKind != template.RuntimeKind {
			return domain.Node{}, nil, domain.Problem{Code: "image_incompatible", Message: "image runtime does not match template"}
		}
		if !domain.ImageCompatibleWithTemplate(image, template, version) {
			return domain.Node{}, nil, domain.Problem{Code: "image_incompatible", Message: "image does not match the selected device template"}
		}
		if err = image.CanStart(); err != nil {
			return domain.Node{}, nil, domain.Problem{Code: "image_unavailable", Message: err.Error(), ResourceType: "image_version", ResourceID: image.ID}
		}
		config["image_version_id"] = string(image.ID)
		config["image_digest"] = image.Digest
		if template.RuntimeKind == domain.RuntimeQEMU {
			path, _ := image.Validation["path"].(string)
			if path == "" {
				return domain.Node{}, nil, domain.Problem{Code: "image_missing", Message: "validated QEMU image path is unavailable", ResourceType: "image_version", ResourceID: image.ID}
			}
			if _, err = os.Stat(path); err != nil {
				return domain.Node{}, nil, domain.Problem{Code: "image_missing", Message: err.Error(), ResourceType: "image_version", ResourceID: image.ID}
			}
			config["image_path"] = path
		} else {
			config["image"] = image.SourceReference
			if image.SourceType != "oci_local" && image.Digest != "" && !strings.Contains(image.SourceReference, "@sha256:") {
				config["image"] = image.SourceReference + "@" + image.Digest
			}
		}
	}
	if kind == "" {
		return domain.Node{}, nil, fmt.Errorf("node kind required")
	}
	if interfaceCount < 1 {
		interfaceCount = 1
	}
	if interfaceLimit == 0 {
		interfaceLimit = max(interfaceCount, 64)
	}
	if processLimit == 0 {
		processLimit = 4096
	}
	now := time.Now().UTC()
	node := domain.Node{ID: domain.NewID(), LaboratoryID: labID, Name: name, Kind: kind, TemplateVersionID: request.TemplateVersionID, Revision: 1, DesiredState: domain.DesiredStopped, ObservedState: domain.ObservedStopped, CPUCount: cpuCount, CPUQuotaMicros: cpuQuota, MemoryMiB: memoryMiB, StorageGiB: storageGiB, InterfaceLimit: interfaceLimit, ProcessLimit: processLimit, Config: config, CreatedAt: now, UpdatedAt: now}
	interfaceNameFormat := templateVersion.Defaults.InterfaceNameFormat
	if configured, ok := config["interface_name_format"].(string); ok && strings.TrimSpace(configured) != "" {
		interfaceNameFormat = configured
	}
	if interfaceNameFormat != "" {
		config["interface_name_format"] = interfaceNameFormat
	}
	interfaces := make([]domain.Interface, interfaceCount)
	sequentialMACPrefix := [3]byte{}
	sequentialMACs, _ := config["mac_address_mode"].(string)
	internalInterfaceCount := configInt(config["internal_interface_count"])
	if sequentialMACs == "sequential" {
		_, _ = rand.Read(sequentialMACPrefix[:])
	}
	for i := range interfaces {
		interfaceName, internalInterface := templateInterfaceName(interfaceNameFormat, i, internalInterfaceCount)
		driver := ""
		if len(templateVersion.NICDrivers) > 0 {
			driver = templateVersion.NICDrivers[0]
		}
		if request.NICDriver != "" {
			if !slices.Contains(templateVersion.NICDrivers, request.NICDriver) {
				return domain.Node{}, nil, domain.Problem{Code: "capability_unsupported", Message: "NIC driver is not supported by the template"}
			}
			driver = request.NICDriver
		}
		macAddress := randomMAC()
		if sequentialMACs == "sequential" {
			macAddress = sequentialMAC(sequentialMACPrefix, i)
		}
		interfaces[i] = domain.Interface{ID: domain.NewID(), NodeID: node.ID, Slot: i, Name: interfaceName, Driver: driver, MACAddress: macAddress, OperationalState: "down", Revision: 1}
		if internalInterface {
			interfaces[i].Driver = driver
		}
	}
	descriptors := make([]map[string]any, 0, len(interfaces))
	for _, iface := range interfaces {
		descriptors = append(descriptors, map[string]any{"id": string(iface.ID), "slot": iface.Slot, "name": iface.Name, "driver": iface.Driver, "mac_address": iface.MACAddress, "internal": iface.Slot < internalInterfaceCount})
	}
	node.Config["interfaces"] = descriptors
	if templateKey == "ubuntu-qemu" && templateVersion.HasCapability("cloud_init") {
		if request.Bootstrap.UserData == "" {
			request.Bootstrap.UserData = "#cloud-config\n{}\n"
		}
		if request.Bootstrap.VendorData == "" {
			request.Bootstrap.VendorData = ubuntuQGAVendorData
		}
	}
	if request.Bootstrap.UserData != "" {
		if !templateVersion.HasCapability("cloud_init") {
			return domain.Node{}, nil, domain.Problem{Code: "capability_unsupported", Message: "template does not support cloud-init"}
		}
		if s.seeds == nil {
			return domain.Node{}, nil, domain.Problem{Code: "capability_unsupported", Message: "cloud-init seed builder unavailable"}
		}
		if request.Bootstrap.NetworkConfig == "" && kind == string(domain.RuntimeQEMU) {
			networkConfig, networkErr := BuildCloudInitNetworkConfig(interfaces, config["network_interfaces"])
			if networkErr != nil {
				return domain.Node{}, nil, networkErr
			}
			request.Bootstrap.NetworkConfig = networkConfig
		}
		seedPath, err := s.seeds.Build(ctx, labID, node.ID, request.Bootstrap)
		if err != nil {
			return domain.Node{}, nil, err
		}
		config["seed_iso"] = seedPath
	}
	var createErr error
	if request.ExpectedLabRevision > 0 {
		placementRepository, ok := s.repository.(nodePlacementRepository)
		if !ok {
			createErr = domain.Problem{Code: "capability_unsupported", Message: "authoritative placement is unavailable", ResourceType: "laboratory", ResourceID: labID}
		} else {
			assignment, laboratoryRevision, err := placementRepository.CreateNodeWithPlacement(ctx, node, interfaces, request.ExpectedLabRevision, request.PlacementIntent, request.Entry)
			createErr = err
			if err == nil && request.PlacementResult != nil {
				request.PlacementResult.PlacementAssignment = &assignment
				request.PlacementResult.LaboratoryRevision = laboratoryRevision
			}
		}
	} else {
		createErr = s.repository.CreateNode(ctx, node, interfaces)
	}
	if createErr != nil {
		if seedPath, ok := config["seed_iso"].(string); ok {
			_ = os.RemoveAll(filepath.Dir(seedPath))
		}
		return node, interfaces, createErr
	}
	return node, interfaces, nil
}

const ubuntuQGAVendorData = `#cloud-config
package_update: true
packages:
  - qemu-guest-agent
runcmd:
  - [systemctl, enable, --now, qemu-guest-agent.service]
`

func normalizeNodeNetworkConfig(config map[string]any) error {
	raw, ok := config["network_interfaces"]
	if !ok {
		return nil
	}
	var interfaces []domain.NodeNetworkInterfaceSettings
	body, err := json.Marshal(raw)
	if err != nil {
		return domain.Problem{Code: "invalid_node_network", Message: "network interface configuration is invalid"}
	}
	if err := json.Unmarshal(body, &interfaces); err != nil {
		return domain.Problem{Code: "invalid_node_network", Message: "network interface configuration is invalid"}
	}
	if err = domain.ValidateNodeNetworkInterfaces(interfaces); err != nil {
		code := "invalid_node_network"
		var configError domain.NetworkConfigError
		if errors.As(err, &configError) {
			code = configError.Code
		}
		return domain.Problem{Code: code, Message: err.Error()}
	}
	body, err = json.Marshal(interfaces)
	if err != nil {
		return domain.Problem{Code: "invalid_node_network", Message: "network interface configuration could not be normalized"}
	}
	var normalized any
	if err := json.Unmarshal(body, &normalized); err != nil {
		return domain.Problem{Code: "invalid_node_network", Message: "network interface configuration could not be normalized"}
	}
	config["network_interfaces"] = normalized
	return nil
}

func BuildCloudInitNetworkConfig(interfaces []domain.Interface, raw any) (string, error) {
	values, _ := raw.([]any)
	if direct, ok := raw.([]map[string]any); ok {
		values = make([]any, len(direct))
		for index := range direct {
			values[index] = direct[index]
		}
	}
	configured := make(map[string]map[string]any, len(values))
	for _, rawValue := range values {
		value, _ := rawValue.(map[string]any)
		name, _ := value["name"].(string)
		if name != "" {
			configured[name] = value
		}
	}
	if len(configured) == 0 {
		return "", nil
	}
	ethernets := make(map[string]any, len(configured))
	for _, iface := range interfaces {
		value, exists := configured[iface.Name]
		if !exists {
			continue
		}
		modes := make(map[string]bool)
		for _, mode := range stringValues(value["modes"]) {
			modes[strings.ToLower(mode)] = true
		}
		entry := map[string]any{
			"match":    map[string]any{"macaddress": iface.MACAddress},
			"set-name": iface.Name,
			"optional": true,
			"dhcp4":    modes["dhcpv4"],
			"dhcp6":    modes["dhcpv6"],
		}
		if modes["slaac"] {
			entry["accept-ra"] = true
		}
		if addresses := stringValues(value["addresses"]); len(addresses) > 0 {
			entry["addresses"] = addresses
		}
		ethernets[iface.Name] = entry
	}
	if len(ethernets) == 0 {
		return "", nil
	}
	body, err := json.Marshal(map[string]any{"version": 2, "ethernets": ethernets})
	if err != nil {
		return "", fmt.Errorf("encode cloud-init network config: %w", err)
	}
	return string(body), nil
}

func stringValues(value any) []string {
	var result []string
	switch values := value.(type) {
	case []string:
		result = append(result, values...)
	case []any:
		for _, raw := range values {
			if text, ok := raw.(string); ok && text != "" {
				result = append(result, text)
			}
		}
	}
	return result
}

func cloneMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input)+4)
	for key, value := range input {
		output[key] = value
	}
	return output
}
func (s *NodeService) SetState(ctx context.Context, id domain.ID, revision domain.Revision, state domain.DesiredState) (domain.Node, error) {
	if state != domain.DesiredRunning && state != domain.DesiredStopped {
		return domain.Node{}, fmt.Errorf("invalid desired state")
	}
	return s.repository.SetNodeDesiredState(ctx, id, revision, state)
}
func (s *NodeService) Delete(ctx context.Context, id domain.ID, revision domain.Revision) error {
	return s.repository.DeleteNode(ctx, id, revision)
}
func randomMAC() string {
	var b [3]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("02:00:00:%02x:%02x:%02x", b[0], b[1], b[2])
}

func sequentialMAC(prefix [3]byte, index int) string {
	return fmt.Sprintf("02:00:%02x:%02x:%02x:%02x", prefix[0], prefix[1], prefix[2], index)
}

func templateInterfaceName(format string, slot, internalCount int) (string, bool) {
	if slot < internalCount {
		return fmt.Sprintf("internal%d", slot), true
	}
	index := slot - internalCount
	if format != "" {
		return fmt.Sprintf(format, index), false
	}
	return fmt.Sprintf("eth%d", index), false
}

func configInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}
