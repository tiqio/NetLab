package domain

import (
	"fmt"
	"strings"
	"time"
)

type RuntimeKind string

const (
	RuntimeQEMU   RuntimeKind = "qemu"
	RuntimeDocker RuntimeKind = "docker"
)

type ImageAvailability string

const (
	ImageStaging   ImageAvailability = "staging"
	ImageAvailable ImageAvailability = "available"
	ImageInvalid   ImageAvailability = "invalid"
	ImageMissing   ImageAvailability = "missing"
	ImageDeleting  ImageAvailability = "deleting"
)

type LicenseStatus string

const (
	LicenseReviewed   LicenseStatus = "reviewed"
	LicenseUnreviewed LicenseStatus = "unreviewed"
	LicenseRejected   LicenseStatus = "rejected"
)

type DeviceTemplate struct {
	ID          ID                `json:"id" yaml:"-"`
	Key         string            `json:"template_key" yaml:"key"`
	DisplayName string            `json:"display_name" yaml:"display_name"`
	RuntimeKind RuntimeKind       `json:"runtime_kind" yaml:"runtime_kind"`
	Versions    []TemplateVersion `json:"versions" yaml:"versions"`
	CreatedAt   time.Time         `json:"created_at" yaml:"-"`
}
type TemplateDefaults struct {
	CPUCount            int    `json:"cpu_count" yaml:"cpu_count"`
	CPUQuotaMicros      int64  `json:"cpu_quota_micros,omitempty" yaml:"cpu_quota_micros,omitempty"`
	MemoryMiB           int    `json:"memory_mib" yaml:"memory_mib"`
	DiskGiB             int    `json:"disk_gib,omitempty" yaml:"disk_gib,omitempty"`
	Interfaces          int    `json:"interfaces" yaml:"interfaces"`
	InterfaceNameFormat string `json:"interface_name_format" yaml:"interface_name_format"`
}
type TemplateVersion struct {
	ID              ID                 `json:"id" yaml:"-"`
	TemplateID      ID                 `json:"template_id" yaml:"-"`
	Version         string             `json:"version" yaml:"version"`
	ManifestVersion int                `json:"manifest_version" yaml:"manifest_version"`
	ImageVersionID  ID                 `json:"image_version_id,omitempty" yaml:"image_version_id,omitempty"`
	Defaults        TemplateDefaults   `json:"defaults" yaml:"defaults"`
	Capabilities    []string           `json:"capabilities" yaml:"capabilities"`
	NICDrivers      []string           `json:"supported_nic_drivers" yaml:"nic_drivers"`
	ConsoleModes    []string           `json:"console_modes" yaml:"console_modes"`
	RuntimeOptions  map[string]any     `json:"runtime_options" yaml:"runtime_options"`
	Enabled         bool               `json:"enabled" yaml:"enabled"`
	Readiness       *TemplateReadiness `json:"readiness,omitempty" yaml:"-"`
	CreatedAt       time.Time          `json:"created_at" yaml:"-"`
}

type TemplateReadiness struct {
	Status          string         `json:"status"`
	GenuineWorkload bool           `json:"genuine_workload"`
	Checks          map[string]any `json:"checks,omitempty"`
	ExceptionID     *string        `json:"exception_id,omitempty"`
}
type ImageVersion struct {
	ID              ID                `json:"id"`
	RuntimeKind     RuntimeKind       `json:"runtime_kind"`
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Digest          string            `json:"digest"`
	SourceType      string            `json:"source_type"`
	SourceReference string            `json:"source_reference"`
	Format          string            `json:"format"`
	SizeBytes       int64             `json:"size_bytes"`
	Availability    ImageAvailability `json:"availability"`
	LicenseStatus   LicenseStatus     `json:"license_status"`
	LicenseNotes    string            `json:"license_notes"`
	Validation      map[string]any    `json:"validation_result"`
	CreatedAt       time.Time         `json:"created_at"`
}

func (t DeviceTemplate) Validate() error {
	if t.Key == "" || strings.ToLower(t.Key) != t.Key {
		return fmt.Errorf("template key must be lowercase")
	}
	if t.DisplayName == "" {
		return fmt.Errorf("display name required")
	}
	if t.RuntimeKind != RuntimeQEMU && t.RuntimeKind != RuntimeDocker {
		return fmt.Errorf("invalid runtime kind")
	}
	if len(t.Versions) == 0 {
		return fmt.Errorf("template versions required")
	}
	for _, v := range t.Versions {
		if err := v.Validate(t.RuntimeKind); err != nil {
			return fmt.Errorf("version %s: %w", v.Version, err)
		}
	}
	return nil
}
func (v TemplateVersion) Validate(kind RuntimeKind) error {
	if v.Version == "" || v.ManifestVersion < 1 {
		return fmt.Errorf("version and manifest version required")
	}
	if v.Defaults.CPUCount < 1 || v.Defaults.MemoryMiB < 1 || v.Defaults.Interfaces < 1 {
		return fmt.Errorf("positive defaults required")
	}
	if kind == RuntimeQEMU && len(v.NICDrivers) == 0 {
		return fmt.Errorf("qemu nic drivers required")
	}
	return nil
}
func (i ImageVersion) CanStart() error {
	if i.Availability != ImageAvailable {
		return fmt.Errorf("image is %s", i.Availability)
	}
	if i.LicenseStatus != LicenseReviewed {
		return fmt.Errorf("image license is %s", i.LicenseStatus)
	}
	if !strings.HasPrefix(i.Digest, "sha256:") {
		return fmt.Errorf("sha256 digest required")
	}
	return nil
}
func (v TemplateVersion) HasCapability(capability string) bool {
	for _, candidate := range v.Capabilities {
		if candidate == capability {
			return true
		}
	}
	return false
}
