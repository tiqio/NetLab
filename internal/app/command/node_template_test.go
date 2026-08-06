package command_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
	qemuRuntime "github.com/netlab/netlab/internal/runtime/qemu"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

type staticReadiness struct{ value domain.TemplateReadiness }

type capturingSeedBuilder struct{ spec qemuRuntime.SeedSpec }

func (b *capturingSeedBuilder) Build(_ context.Context, _, _ domain.ID, spec qemuRuntime.SeedSpec) (string, error) {
	b.spec = spec
	return "/tmp/netlab-test-seed.iso", nil
}

func (s staticReadiness) TemplateReadiness(string) (domain.TemplateReadiness, bool) {
	return s.value, true
}

func TestCreateNodePinsValidatedTemplateImage(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:node-template?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := storesqlite.NewTopologyRepository(database)
	templates := storesqlite.NewTemplateRepository(database)
	lab, err := command.NewLaboratoryService(topology).Create(ctx, "template-lab", "", domain.RecoveryRemainStopped)
	if err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(t.TempDir(), "ubuntu.qcow2")
	if err = os.WriteFile(imagePath, []byte("qcow2-test"), 0o600); err != nil {
		t.Fatal(err)
	}
	image := domain.ImageVersion{ID: domain.NewID(), RuntimeKind: domain.RuntimeQEMU, Name: "ubuntu", Version: "24.04", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", SourceType: "local_import", SourceReference: "ubuntu.qcow2", Format: "qcow2", Availability: domain.ImageAvailable, LicenseStatus: domain.LicenseReviewed, LicenseNotes: "operator supplied", Validation: map[string]any{"path": imagePath}, CreatedAt: time.Now().UTC()}
	if err = templates.CreateImage(ctx, image); err != nil {
		t.Fatal(err)
	}
	versionID := domain.NewID()
	if err = templates.UpsertTemplate(ctx, domain.DeviceTemplate{Key: "ubuntu-test", DisplayName: "Ubuntu Test", RuntimeKind: domain.RuntimeQEMU, Versions: []domain.TemplateVersion{{ID: versionID, Version: "24.04", ManifestVersion: 1, ImageVersionID: image.ID, Defaults: domain.TemplateDefaults{CPUCount: 2, MemoryMiB: 2048, Interfaces: 2, InterfaceNameFormat: "ens%d"}, NICDrivers: []string{"virtio-net-pci", "e1000"}, Enabled: true}}}); err != nil {
		t.Fatal(err)
	}
	service := command.NewNodeService(topology, templates)
	service.SetTemplateReadinessResolver(staticReadiness{value: domain.TemplateReadiness{Status: "mechanics_validated", GenuineWorkload: true}})
	node, interfaces, err := service.CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "ubuntu", TemplateVersionID: versionID})
	if err != nil {
		t.Fatal(err)
	}
	if node.TemplateVersionID != versionID || node.Config["image_version_id"] != string(image.ID) || node.Config["image_path"] != imagePath {
		t.Fatalf("node=%+v config=%v", node, node.Config)
	}
	readiness, ok := node.Config["template_readiness"].(map[string]any)
	if !ok || readiness["status"] != "mechanics_validated" || readiness["genuine_workload"] != true {
		t.Fatalf("template readiness was not frozen at launch: %#v", node.Config["template_readiness"])
	}
	if node.CPUCount != 2 || node.MemoryMiB != 2048 || len(interfaces) != 2 || interfaces[0].Name != "ens0" || interfaces[0].Driver != "virtio-net-pci" {
		t.Fatalf("defaults not applied: node=%+v interfaces=%+v", node, interfaces)
	}
	_, selectedInterfaces, err := service.CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "ubuntu-e1000", TemplateVersionID: versionID, NICDriver: "e1000"})
	if err != nil {
		t.Fatal(err)
	}
	if selectedInterfaces[0].Driver != "e1000" {
		t.Fatalf("requested driver not applied: %+v", selectedInterfaces)
	}
	if _, _, err = service.CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "ubuntu-invalid-driver", TemplateVersionID: versionID, NICDriver: "vmxnet3"}); err == nil {
		t.Fatal("unsupported driver should fail")
	}
}

func TestCreateNodeRejectsUnreviewedImage(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:node-template-reject?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := storesqlite.NewTopologyRepository(database)
	templates := storesqlite.NewTemplateRepository(database)
	lab, _ := command.NewLaboratoryService(topology).Create(ctx, "reject-lab", "", domain.RecoveryRemainStopped)
	image := domain.ImageVersion{ID: domain.NewID(), RuntimeKind: domain.RuntimeDocker, Name: "busybox", Version: "1", Digest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", SourceType: "oci_registry", SourceReference: "busybox", Format: "oci", Availability: domain.ImageAvailable, LicenseStatus: domain.LicenseUnreviewed, LicenseNotes: "pending", CreatedAt: time.Now().UTC()}
	if err = templates.CreateImage(ctx, image); err != nil {
		t.Fatal(err)
	}
	versionID := domain.NewID()
	if err = templates.UpsertTemplate(ctx, domain.DeviceTemplate{Key: "busybox-test", DisplayName: "BusyBox Test", RuntimeKind: domain.RuntimeDocker, Versions: []domain.TemplateVersion{{ID: versionID, Version: "1", ManifestVersion: 1, ImageVersionID: image.ID, Defaults: domain.TemplateDefaults{CPUCount: 1, MemoryMiB: 64, Interfaces: 1}, Enabled: true}}}); err != nil {
		t.Fatal(err)
	}
	_, _, err = command.NewNodeService(topology, templates).CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "busybox", TemplateVersionID: versionID})
	if err == nil {
		t.Fatal("unreviewed image accepted")
	}
}

func TestCreateNginxNodeUsesForegroundCommandAndMatchingImage(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:nginx-template?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := storesqlite.NewTopologyRepository(database)
	templates := storesqlite.NewTemplateRepository(database)
	lab, _ := command.NewLaboratoryService(topology).Create(ctx, "nginx-lab", "", domain.RecoveryRemainStopped)
	image := domain.ImageVersion{ID: domain.NewID(), RuntimeKind: domain.RuntimeDocker, Name: "nginx", Version: "1.30-alpine", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", SourceType: "oci_registry", SourceReference: "nginx:1.30-alpine", Format: "oci", Availability: domain.ImageAvailable, LicenseStatus: domain.LicenseReviewed, LicenseNotes: "official image", CreatedAt: time.Now().UTC()}
	if err = templates.CreateImage(ctx, image); err != nil {
		t.Fatal(err)
	}
	versionID := domain.NewID()
	if err = templates.UpsertTemplate(ctx, domain.DeviceTemplate{Key: "nginx-container", DisplayName: "Nginx", RuntimeKind: domain.RuntimeDocker, Versions: []domain.TemplateVersion{{ID: versionID, Version: "1.30-alpine", ManifestVersion: 1, Defaults: domain.TemplateDefaults{CPUCount: 1, MemoryMiB: 128, Interfaces: 1, InterfaceNameFormat: "eth%d"}, NICDrivers: []string{"veth"}, Capabilities: []string{"http_server"}, RuntimeOptions: map[string]any{"command": []any{"nginx", "-g", "daemon off;"}, "recommended_image_name": "nginx", "recommended_image_version": "1.30-alpine"}, Enabled: true}}}); err != nil {
		t.Fatal(err)
	}
	node, interfaces, err := command.NewNodeService(topology, templates).CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "web", TemplateVersionID: versionID, ImageVersionID: image.ID})
	if err != nil {
		t.Fatal(err)
	}
	commandValue, ok := node.Config["command"].([]any)
	if !ok || len(commandValue) != 3 || commandValue[0] != "nginx" || commandValue[2] != "daemon off;" {
		t.Fatalf("nginx command=%#v", node.Config["command"])
	}
	if node.Config["image"] != "nginx:1.30-alpine@"+image.Digest || len(interfaces) != 1 || interfaces[0].Name != "eth0" {
		t.Fatalf("node=%+v interfaces=%+v", node, interfaces)
	}
}

func TestCreateNodeRejectsImageFromDifferentQEMUDeviceFamily(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:node-template-family?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := storesqlite.NewTopologyRepository(database)
	templates := storesqlite.NewTemplateRepository(database)
	lab, _ := command.NewLaboratoryService(topology).Create(ctx, "family-lab", "", domain.RecoveryRemainStopped)
	imagePath := filepath.Join(t.TempDir(), "vyos.qcow2")
	if err = os.WriteFile(imagePath, []byte("qcow2-test"), 0o600); err != nil {
		t.Fatal(err)
	}
	vyosImage := domain.ImageVersion{ID: domain.NewID(), RuntimeKind: domain.RuntimeQEMU, Name: "VyOS", Version: "rolling", Digest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", SourceType: "local_import", SourceReference: "vyos-rolling.qcow2", Format: "qcow2", Availability: domain.ImageAvailable, LicenseStatus: domain.LicenseReviewed, Validation: map[string]any{"path": imagePath}, CreatedAt: time.Now().UTC()}
	ubuntuImage := domain.ImageVersion{ID: domain.NewID(), RuntimeKind: domain.RuntimeQEMU, Name: "Ubuntu", Version: "24.04", Digest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", SourceType: "local_import", SourceReference: "ubuntu-24.04.qcow2", Format: "qcow2", Availability: domain.ImageAvailable, LicenseStatus: domain.LicenseReviewed, Validation: map[string]any{"path": imagePath}, CreatedAt: time.Now().UTC()}
	for _, image := range []domain.ImageVersion{vyosImage, ubuntuImage} {
		if err = templates.CreateImage(ctx, image); err != nil {
			t.Fatal(err)
		}
	}
	versionID := domain.NewID()
	if err = templates.UpsertTemplate(ctx, domain.DeviceTemplate{Key: "vyos", DisplayName: "VyOS", RuntimeKind: domain.RuntimeQEMU, Versions: []domain.TemplateVersion{{ID: versionID, Version: "rolling", ManifestVersion: 1, Defaults: domain.TemplateDefaults{CPUCount: 1, MemoryMiB: 1024, Interfaces: 1, InterfaceNameFormat: "eth%d"}, NICDrivers: []string{"virtio-net-pci"}, Enabled: true}}}); err != nil {
		t.Fatal(err)
	}
	service := command.NewNodeService(topology, templates)
	if _, _, err = service.CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "wrong-image", TemplateVersionID: versionID, ImageVersionID: ubuntuImage.ID}); err == nil {
		t.Fatal("cross-family image was accepted")
	} else if problem, ok := err.(domain.Problem); !ok || problem.Code != "image_incompatible" {
		t.Fatalf("cross-family image error=%#v", err)
	}
	if _, _, err = service.CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "vyos", TemplateVersionID: versionID, ImageVersionID: vyosImage.ID}); err != nil {
		t.Fatalf("compatible VyOS image rejected: %v", err)
	}
}

func TestCreateQEMUNodeBuildsMACMatchedCloudInitNetworkConfig(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:node-network-config?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := storesqlite.NewTopologyRepository(database)
	templates := storesqlite.NewTemplateRepository(database)
	lab, _ := command.NewLaboratoryService(topology).Create(ctx, "network-config-lab", "", domain.RecoveryRemainStopped)
	imagePath := filepath.Join(t.TempDir(), "ubuntu.qcow2")
	if err = os.WriteFile(imagePath, []byte("qcow2-test"), 0o600); err != nil {
		t.Fatal(err)
	}
	image := domain.ImageVersion{ID: domain.NewID(), RuntimeKind: domain.RuntimeQEMU, Name: "ubuntu", Version: "24.04", Digest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", SourceType: "local_import", SourceReference: "ubuntu.qcow2", Format: "qcow2", Availability: domain.ImageAvailable, LicenseStatus: domain.LicenseReviewed, LicenseNotes: "operator supplied", Validation: map[string]any{"path": imagePath}, CreatedAt: time.Now().UTC()}
	if err = templates.CreateImage(ctx, image); err != nil {
		t.Fatal(err)
	}
	versionID := domain.NewID()
	if err = templates.UpsertTemplate(ctx, domain.DeviceTemplate{Key: "ubuntu-network", DisplayName: "Ubuntu", RuntimeKind: domain.RuntimeQEMU, Versions: []domain.TemplateVersion{{ID: versionID, Version: "24.04", ManifestVersion: 1, ImageVersionID: image.ID, Defaults: domain.TemplateDefaults{CPUCount: 1, MemoryMiB: 1024, Interfaces: 1, InterfaceNameFormat: "ens%d"}, Capabilities: []string{"cloud_init"}, NICDrivers: []string{"virtio-net-pci"}, Enabled: true}}}); err != nil {
		t.Fatal(err)
	}
	builder := &capturingSeedBuilder{}
	service := command.NewNodeService(topology, templates)
	service.SetSeedBuilder(builder)
	_, interfaces, err := service.CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "ubuntu", TemplateVersionID: versionID, Bootstrap: qemuRuntime.SeedSpec{UserData: "#cloud-config\n{}"}, Config: map[string]any{"network_interfaces": []any{map[string]any{"name": "ens0", "modes": []any{"static", "slaac"}, "addresses": []any{"192.0.2.10/24", "2001:db8::10/64"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(interfaces) != 1 || builder.spec.NetworkConfig == "" {
		t.Fatalf("interfaces=%+v network_config=%q", interfaces, builder.spec.NetworkConfig)
	}
	for _, expected := range []string{interfaces[0].MACAddress, `"set-name":"ens0"`, `"192.0.2.10/24"`, `"accept-ra":true`} {
		if !strings.Contains(builder.spec.NetworkConfig, expected) {
			t.Fatalf("missing %s in %s", expected, builder.spec.NetworkConfig)
		}
	}
}

func TestCreateDockerNodePreservesCanonicalStaticRoutes(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:docker-route-create?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := storesqlite.NewTopologyRepository(database)
	lab, _ := command.NewLaboratoryService(topology).Create(ctx, "docker-routes", "", domain.RecoveryRemainStopped)
	node, _, err := command.NewNodeService(topology).CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{
		Name: "router", Kind: "docker", InterfaceCount: 1,
		Config: map[string]any{"network_interfaces": []any{map[string]any{
			"name": "eth0", "modes": []any{"static"}, "addresses": []any{"192.0.2.10/24"},
			"routes": []any{map[string]any{"destination": "198.51.100.99/24", "gateway": "192.0.2.1", "metric": 10}},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(node.Config["network_interfaces"])
	if !strings.Contains(string(body), `"destination":"198.51.100.0/24"`) || !strings.Contains(string(body), `"gateway":"192.0.2.1"`) {
		t.Fatalf("network_interfaces=%s", body)
	}
}
