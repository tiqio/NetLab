package query

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/netlab/netlab/internal/domain"
	"gopkg.in/yaml.v3"
)

type TemplateStore interface {
	UpsertTemplate(context.Context, domain.DeviceTemplate) error
	ListTemplates(context.Context) ([]domain.DeviceTemplate, error)
	ListImages(context.Context) ([]domain.ImageVersion, error)
}
type TemplateService struct {
	store     TemplateStore
	readiness map[string]domain.TemplateReadiness
}

func NewTemplateService(store TemplateStore) *TemplateService { return &TemplateService{store: store} }

func (s *TemplateService) TemplateReadiness(templateKey string) (domain.TemplateReadiness, bool) {
	value, ok := s.readiness[templateKey]
	return value, ok
}

type manifest struct {
	SchemaVersion int                     `yaml:"schema_version"`
	Templates     []domain.DeviceTemplate `yaml:"templates"`
}

func (s *TemplateService) LoadBuiltins(ctx context.Context, root string) error {
	for _, kind := range []string{"qemu", "docker"} {
		body, err := os.ReadFile(filepath.Join(root, kind, "manifest.yaml"))
		if err != nil {
			return err
		}
		var doc manifest
		if err = yaml.Unmarshal(body, &doc); err != nil {
			return err
		}
		if doc.SchemaVersion != 1 {
			return fmt.Errorf("unsupported manifest schema %d", doc.SchemaVersion)
		}
		for _, template := range doc.Templates {
			if err = template.Validate(); err != nil {
				return err
			}
			template.CreatedAt = time.Now().UTC()
			for i := range template.Versions {
				template.Versions[i].CreatedAt = template.CreatedAt
			}
			if err = s.store.UpsertTemplate(ctx, template); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *TemplateService) LoadReadiness(path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var document struct {
		Templates []struct {
			TemplateKey     string         `json:"template_key"`
			Status          string         `json:"status"`
			GenuineWorkload bool           `json:"genuine_workload"`
			Bootstrap       map[string]any `json:"bootstrap"`
			Console         map[string]any `json:"console"`
			Capabilities    map[string]any `json:"capabilities"`
			Lifecycle       map[string]any `json:"lifecycle"`
			Cleanup         map[string]any `json:"cleanup"`
			ExceptionID     *string        `json:"exception_id"`
		} `json:"templates"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		return err
	}
	s.readiness = map[string]domain.TemplateReadiness{}
	for _, item := range document.Templates {
		s.readiness[item.TemplateKey] = domain.TemplateReadiness{
			Status:          item.Status,
			GenuineWorkload: item.GenuineWorkload,
			Checks: map[string]any{
				"bootstrap": item.Bootstrap, "console": item.Console,
				"capabilities": item.Capabilities, "lifecycle": item.Lifecycle,
				"cleanup": item.Cleanup,
			},
			ExceptionID: item.ExceptionID,
		}
	}
	return nil
}
func (s *TemplateService) List(ctx context.Context) ([]domain.DeviceTemplate, error) {
	values, err := s.store.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	images, err := s.store.ListImages(ctx)
	if err != nil {
		return nil, err
	}
	for templateIndex := range values {
		for versionIndex := range values[templateIndex].Versions {
			version := &values[templateIndex].Versions[versionIndex]
			if version.ImageVersionID == "" {
				version.ImageVersionID = recommendedImageID(version.RuntimeOptions, values[templateIndex].RuntimeKind, images)
			}
			if readiness, ok := s.readiness[values[templateIndex].Key]; ok {
				value := readiness
				version.Readiness = &value
			}
		}
	}
	return values, nil
}

func recommendedImageID(options map[string]any, runtimeKind domain.RuntimeKind, images []domain.ImageVersion) domain.ID {
	name, _ := options["recommended_image_name"].(string)
	version, _ := options["recommended_image_version"].(string)
	if name == "" || version == "" {
		return ""
	}
	for _, image := range images {
		if image.RuntimeKind == runtimeKind && image.Name == name && image.Version == version && image.Availability == domain.ImageAvailable && image.LicenseStatus == domain.LicenseReviewed {
			return image.ID
		}
	}
	return ""
}
func (s *TemplateService) Images(ctx context.Context) ([]domain.ImageVersion, error) {
	return s.store.ListImages(ctx)
}
