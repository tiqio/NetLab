package command_test

import (
	"context"
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
	if err = templates.UpsertTemplate(ctx, domain.DeviceTemplate{Key: "ubuntu-test", DisplayName: "Ubuntu Test", RuntimeKind: domain.RuntimeQEMU, Versions: []domain.TemplateVersion{{ID: versionID, Version: "24.04", ManifestVersion: 1, ImageVersionID: image.ID, Defaults: domain.TemplateDefaults{CPUCount: 2, MemoryMiB: 2048, Interfaces: 2, InterfaceNameFormat: "ens%d"}, NICDrivers: []string{"virtio-net-pci"}, Enabled: true}}}); err != nil {
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
