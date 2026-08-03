package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/netlab/netlab/internal/domain"
	"time"
)

type TemplateRepository struct{ database *Database }

func NewTemplateRepository(database *Database) *TemplateRepository {
	return &TemplateRepository{database: database}
}
func (r *TemplateRepository) UpsertTemplate(ctx context.Context, t domain.DeviceTemplate) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		if t.ID == "" {
			t.ID = domain.NewID()
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO device_templates(id,template_key,display_name,runtime_kind,created_at) VALUES(?,?,?,?,?) ON CONFLICT(template_key) DO UPDATE SET display_name=excluded.display_name,runtime_kind=excluded.runtime_kind`, t.ID, t.Key, t.DisplayName, t.RuntimeKind, now); err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, `SELECT id FROM device_templates WHERE template_key=?`, t.Key).Scan(&t.ID); err != nil {
			return err
		}
		for _, v := range t.Versions {
			if v.ID == "" {
				v.ID = domain.NewID()
			}
			defaults, _ := json.Marshal(v.Defaults)
			caps, _ := json.Marshal(v.Capabilities)
			nics, _ := json.Marshal(v.NICDrivers)
			consoles, _ := json.Marshal(v.ConsoleModes)
			options, _ := json.Marshal(v.RuntimeOptions)
			_, err := tx.ExecContext(ctx, `INSERT INTO template_versions(id,template_id,manifest_version,version,image_version_id,defaults_json,capabilities_json,nic_drivers_json,console_modes_json,runtime_options_json,enabled,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(template_id,version) DO UPDATE SET manifest_version=excluded.manifest_version,image_version_id=excluded.image_version_id,defaults_json=excluded.defaults_json,capabilities_json=excluded.capabilities_json,nic_drivers_json=excluded.nic_drivers_json,console_modes_json=excluded.console_modes_json,runtime_options_json=excluded.runtime_options_json,enabled=excluded.enabled`, v.ID, t.ID, v.ManifestVersion, v.Version, nullable(string(v.ImageVersionID)), defaults, caps, nics, consoles, options, v.Enabled, now)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
func (r *TemplateRepository) ListTemplates(ctx context.Context) ([]domain.DeviceTemplate, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT id,template_key,display_name,runtime_kind,created_at FROM device_templates ORDER BY display_name`)
	if err != nil {
		return nil, err
	}
	var values []domain.DeviceTemplate
	for rows.Next() {
		var t domain.DeviceTemplate
		var created string
		if err = rows.Scan(&t.ID, &t.Key, &t.DisplayName, &t.RuntimeKind, &created); err != nil {
			return nil, err
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		values = append(values, t)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	for index := range values {
		values[index].Versions, err = r.listVersions(ctx, values[index].ID)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}
func (r *TemplateRepository) listVersions(ctx context.Context, templateID domain.ID) ([]domain.TemplateVersion, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT id,version,manifest_version,COALESCE(image_version_id,''),defaults_json,capabilities_json,nic_drivers_json,console_modes_json,runtime_options_json,enabled,created_at FROM template_versions WHERE template_id=? ORDER BY version`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []domain.TemplateVersion
	for rows.Next() {
		var v domain.TemplateVersion
		var defaults, caps, nics, consoles, options []byte
		var created string
		if err = rows.Scan(&v.ID, &v.Version, &v.ManifestVersion, &v.ImageVersionID, &defaults, &caps, &nics, &consoles, &options, &v.Enabled, &created); err != nil {
			return nil, err
		}
		v.TemplateID = templateID
		_ = json.Unmarshal(defaults, &v.Defaults)
		_ = json.Unmarshal(caps, &v.Capabilities)
		_ = json.Unmarshal(nics, &v.NICDrivers)
		_ = json.Unmarshal(consoles, &v.ConsoleModes)
		_ = json.Unmarshal(options, &v.RuntimeOptions)
		v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		values = append(values, v)
	}
	return values, rows.Err()
}
func (r *TemplateRepository) CreateImage(ctx context.Context, image domain.ImageVersion) error {
	validation, _ := json.Marshal(image.Validation)
	_, err := r.database.DB.ExecContext(ctx, `INSERT INTO image_versions(id,runtime_kind,name,version,digest,source_type,source_reference,format,size_bytes,availability,license_status,license_notes,validation_json,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, image.ID, image.RuntimeKind, image.Name, image.Version, image.Digest, image.SourceType, image.SourceReference, image.Format, image.SizeBytes, image.Availability, image.LicenseStatus, image.LicenseNotes, validation, image.CreatedAt.Format(time.RFC3339Nano))
	return err
}
func (r *TemplateRepository) ListImages(ctx context.Context) ([]domain.ImageVersion, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT id,runtime_kind,name,version,digest,source_type,source_reference,format,size_bytes,availability,license_status,license_notes,validation_json,created_at FROM image_versions ORDER BY name,version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []domain.ImageVersion
	for rows.Next() {
		var image domain.ImageVersion
		var validation []byte
		var created string
		if err = rows.Scan(&image.ID, &image.RuntimeKind, &image.Name, &image.Version, &image.Digest, &image.SourceType, &image.SourceReference, &image.Format, &image.SizeBytes, &image.Availability, &image.LicenseStatus, &image.LicenseNotes, &validation, &created); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(validation, &image.Validation)
		image.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		values = append(values, image)
	}
	return values, rows.Err()
}

func (r *TemplateRepository) HasImageDigest(ctx context.Context, digest string) (bool, error) {
	var count int
	err := r.database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM image_versions WHERE digest=? AND availability='available' AND license_status='reviewed'`, digest).Scan(&count)
	return count > 0, err
}

func (r *TemplateRepository) GetTemplateVersion(ctx context.Context, id domain.ID) (domain.DeviceTemplate, domain.TemplateVersion, error) {
	var template domain.DeviceTemplate
	var version domain.TemplateVersion
	var defaults, capabilities, drivers, consoles, options []byte
	var created string
	err := r.database.DB.QueryRowContext(ctx, `SELECT t.id,t.template_key,t.display_name,t.runtime_kind,t.created_at,v.id,v.version,v.manifest_version,COALESCE(v.image_version_id,''),v.defaults_json,v.capabilities_json,v.nic_drivers_json,v.console_modes_json,v.runtime_options_json,v.enabled,v.created_at FROM template_versions v JOIN device_templates t ON t.id=v.template_id WHERE v.id=?`, id).Scan(&template.ID, &template.Key, &template.DisplayName, &template.RuntimeKind, &created, &version.ID, &version.Version, &version.ManifestVersion, &version.ImageVersionID, &defaults, &capabilities, &drivers, &consoles, &options, &version.Enabled, &created)
	if err == sql.ErrNoRows {
		return template, version, ErrNotFound
	}
	if err != nil {
		return template, version, err
	}
	version.TemplateID = template.ID
	_ = json.Unmarshal(defaults, &version.Defaults)
	_ = json.Unmarshal(capabilities, &version.Capabilities)
	_ = json.Unmarshal(drivers, &version.NICDrivers)
	_ = json.Unmarshal(consoles, &version.ConsoleModes)
	_ = json.Unmarshal(options, &version.RuntimeOptions)
	return template, version, nil
}

func (r *TemplateRepository) GetImage(ctx context.Context, id domain.ID) (domain.ImageVersion, error) {
	var image domain.ImageVersion
	var validation []byte
	var created string
	err := r.database.DB.QueryRowContext(ctx, `SELECT id,runtime_kind,name,version,digest,source_type,source_reference,format,size_bytes,availability,license_status,license_notes,validation_json,created_at FROM image_versions WHERE id=?`, id).Scan(&image.ID, &image.RuntimeKind, &image.Name, &image.Version, &image.Digest, &image.SourceType, &image.SourceReference, &image.Format, &image.SizeBytes, &image.Availability, &image.LicenseStatus, &image.LicenseNotes, &validation, &created)
	if err == sql.ErrNoRows {
		return image, ErrNotFound
	}
	if err != nil {
		return image, err
	}
	_ = json.Unmarshal(validation, &image.Validation)
	image.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return image, nil
}
