package query

import (
	"context"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

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
}
