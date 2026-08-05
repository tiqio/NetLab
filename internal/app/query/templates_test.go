package query

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func TestLoadReadinessRejectsDifferentCandidate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "readiness.json")
	if err := os.WriteFile(path, []byte(`{"candidate_id":"old","templates":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := NewTemplateService(&templateStoreStub{}).LoadReadinessForCandidate(path, "new"); err == nil {
		t.Fatal("expected candidate mismatch")
	}
}

type templateStoreStub struct {
	templates []domain.DeviceTemplate
	images    []domain.ImageVersion
}

func (s *templateStoreStub) UpsertTemplate(context.Context, domain.DeviceTemplate) error {
	return nil
}

func (s *templateStoreStub) ListTemplates(context.Context) ([]domain.DeviceTemplate, error) {
	return s.templates, nil
}

func (s *templateStoreStub) ListImages(context.Context) ([]domain.ImageVersion, error) {
	return s.images, nil
}

func TestListBindsRecommendedAvailableImage(t *testing.T) {
	store := &templateStoreStub{
		templates: []domain.DeviceTemplate{{
			Key: "ruijie-router", RuntimeKind: domain.RuntimeQEMU,
			Versions: []domain.TemplateVersion{{RuntimeOptions: map[string]any{"recommended_image_name": "Ruijie Router", "recommended_image_version": "V1.06"}}},
		}},
		images: []domain.ImageVersion{{ID: "image-1", RuntimeKind: domain.RuntimeQEMU, Name: "Ruijie Router", Version: "V1.06", Availability: domain.ImageAvailable, LicenseStatus: domain.LicenseReviewed}},
	}
	values, err := NewTemplateService(store).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := values[0].Versions[0].ImageVersionID; got != "image-1" {
		t.Fatalf("recommended image=%q", got)
	}
	if got := values[0].Versions[0].CompatibleImageVersionIDs; len(got) != 1 || got[0] != "image-1" {
		t.Fatalf("compatible images=%v", got)
	}
}

func TestListSeparatesUnboundQEMUImageFamilies(t *testing.T) {
	store := &templateStoreStub{
		templates: []domain.DeviceTemplate{{
			Key: "vyos", DisplayName: "VyOS", RuntimeKind: domain.RuntimeQEMU,
			Versions: []domain.TemplateVersion{{Version: "rolling"}},
		}},
		images: []domain.ImageVersion{
			{ID: "vyos-image", RuntimeKind: domain.RuntimeQEMU, Name: "VyOS", SourceReference: "vyos-rolling.qcow2"},
			{ID: "ubuntu-image", RuntimeKind: domain.RuntimeQEMU, Name: "Ubuntu", SourceReference: "ubuntu-24.04.qcow2"},
			{ID: "fortigate-image", RuntimeKind: domain.RuntimeQEMU, Name: "FortiGate", SourceReference: "fortinet-FGT.qcow2"},
		},
	}
	values, err := NewTemplateService(store).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got := values[0].Versions[0].CompatibleImageVersionIDs
	if len(got) != 1 || got[0] != "vyos-image" {
		t.Fatalf("VyOS compatible images=%v", got)
	}
}
