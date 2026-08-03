package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func TestTemplateUpsertUpdatesImageBinding(t *testing.T) {
	database, err := Open(context.Background(), filepath.Join(t.TempDir(), "netlab.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repository := NewTemplateRepository(database)
	for _, id := range []domain.ID{"image-old", "image-new"} {
		if err = repository.CreateImage(context.Background(), domain.ImageVersion{ID: id, RuntimeKind: domain.RuntimeQEMU, Name: string(id), Version: "1", Digest: "sha256:" + string(id), SourceType: "test", SourceReference: string(id), Format: "qcow2", Availability: domain.ImageAvailable, LicenseStatus: domain.LicenseReviewed}); err != nil {
			t.Fatal(err)
		}
	}
	value := domain.DeviceTemplate{Key: "ubuntu-acceptance", DisplayName: "Ubuntu Acceptance", RuntimeKind: domain.RuntimeQEMU, Versions: []domain.TemplateVersion{{Version: "24.04", ManifestVersion: 1, ImageVersionID: "image-old", Defaults: domain.TemplateDefaults{CPUCount: 1, MemoryMiB: 512, Interfaces: 1}, NICDrivers: []string{"virtio-net-pci"}, Enabled: true}}}
	if err = repository.UpsertTemplate(context.Background(), value); err != nil {
		t.Fatal(err)
	}
	value.Versions[0].ImageVersionID = "image-new"
	if err = repository.UpsertTemplate(context.Background(), value); err != nil {
		t.Fatal(err)
	}
	templates, err := repository.ListTemplates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(templates) != 1 || templates[0].Versions[0].ImageVersionID != "image-new" {
		t.Fatalf("templates=%+v", templates)
	}
}
