package image

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/netlab/netlab/internal/domain"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ImportRequest struct {
	Name, Version                            string
	RuntimeKind                              domain.RuntimeKind
	SourcePath, ExpectedSHA256, LicenseNotes string
}
type Importer struct {
	Root    string
	QEMUImg string
	Docker  string
}

func NewImporter(root string) *Importer {
	qemuImg, _ := exec.LookPath("qemu-img")
	docker, _ := exec.LookPath("docker")
	return &Importer{Root: root, QEMUImg: qemuImg, Docker: docker}
}
func (i *Importer) Import(ctx context.Context, request ImportRequest) (domain.ImageVersion, error) {
	if request.LicenseNotes == "" {
		return domain.ImageVersion{}, fmt.Errorf("license notes required")
	}
	if request.RuntimeKind == domain.RuntimeDocker {
		return i.importOCI(ctx, request)
	}
	source, err := os.Open(request.SourcePath)
	if err != nil {
		return domain.ImageVersion{}, err
	}
	defer source.Close()
	if err = os.MkdirAll(filepath.Join(i.Root, "staging"), 0o700); err != nil {
		return domain.ImageVersion{}, err
	}
	temp, err := os.CreateTemp(filepath.Join(i.Root, "staging"), "image-*")
	if err != nil {
		return domain.ImageVersion{}, err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(temp, hash), source)
	closeErr := temp.Close()
	if err != nil {
		return domain.ImageVersion{}, err
	}
	if closeErr != nil {
		return domain.ImageVersion{}, closeErr
	}
	digest := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	if request.ExpectedSHA256 != "" && strings.TrimPrefix(request.ExpectedSHA256, "sha256:") != strings.TrimPrefix(digest, "sha256:") {
		return domain.ImageVersion{}, fmt.Errorf("checksum mismatch")
	}
	format := "qcow2"
	publishedSource := tempPath
	if strings.HasSuffix(request.SourcePath, ".tgz") || strings.HasSuffix(request.SourcePath, ".tar.gz") {
		extractDir := tempPath + ".d"
		if err = safeExtractTGZ(request.SourcePath, extractDir); err != nil {
			return domain.ImageVersion{}, err
		}
		entries, _ := filepath.Glob(filepath.Join(extractDir, "*.qcow2"))
		if len(entries) != 1 {
			return domain.ImageVersion{}, fmt.Errorf("archive must contain exactly one qcow2")
		}
		publishedSource = entries[0]
	}
	if request.RuntimeKind == domain.RuntimeQEMU && i.QEMUImg != "" {
		if output, err := exec.CommandContext(ctx, i.QEMUImg, "info", "--output=json", publishedSource).CombinedOutput(); err != nil {
			return domain.ImageVersion{}, fmt.Errorf("qemu-img validation: %s: %w", output, err)
		}
	}
	destination := filepath.Join(i.Root, "images", strings.TrimPrefix(digest, "sha256:")+"."+format)
	if err = os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return domain.ImageVersion{}, err
	}
	if err = os.Rename(publishedSource, destination); err != nil {
		input, openErr := os.Open(publishedSource)
		if openErr != nil {
			return domain.ImageVersion{}, err
		}
		defer input.Close()
		output, createErr := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if createErr != nil {
			return domain.ImageVersion{}, createErr
		}
		_, copyErr := io.Copy(output, input)
		closeErr = output.Close()
		if copyErr != nil {
			return domain.ImageVersion{}, copyErr
		}
		if closeErr != nil {
			return domain.ImageVersion{}, closeErr
		}
	}
	return domain.ImageVersion{ID: domain.NewID(), RuntimeKind: request.RuntimeKind, Name: request.Name, Version: request.Version, Digest: digest, SourceType: "local_import", SourceReference: filepath.Base(request.SourcePath), Format: format, SizeBytes: size, Availability: domain.ImageAvailable, LicenseStatus: domain.LicenseReviewed, LicenseNotes: request.LicenseNotes, Validation: map[string]any{"path": destination}, CreatedAt: time.Now().UTC()}, nil
}

func (i *Importer) importOCI(ctx context.Context, request ImportRequest) (domain.ImageVersion, error) {
	reference := strings.TrimSpace(request.SourcePath)
	if reference == "" {
		return domain.ImageVersion{}, fmt.Errorf("OCI image reference required")
	}
	if i.Docker == "" {
		return domain.ImageVersion{}, fmt.Errorf("docker CLI is required for OCI image import")
	}
	sourceType := "oci_registry"
	inspectReference := reference
	if strings.HasPrefix(reference, "local://") {
		sourceType = "oci_local"
		inspectReference = strings.TrimSpace(strings.TrimPrefix(reference, "local://"))
		if inspectReference == "" {
			return domain.ImageVersion{}, fmt.Errorf("local OCI image reference required")
		}
	} else if output, err := exec.CommandContext(ctx, i.Docker, "pull", reference).CombinedOutput(); err != nil {
		return domain.ImageVersion{}, fmt.Errorf("docker pull %s: %s: %w", reference, strings.TrimSpace(string(output)), err)
	}
	body, err := exec.CommandContext(ctx, i.Docker, "image", "inspect", inspectReference).Output()
	if err != nil {
		return domain.ImageVersion{}, fmt.Errorf("inspect OCI image %s: %w", inspectReference, err)
	}
	var inspections []struct {
		ID          string   `json:"Id"`
		RepoDigests []string `json:"RepoDigests"`
		Size        int64    `json:"Size"`
	}
	if err = json.Unmarshal(body, &inspections); err != nil || len(inspections) != 1 {
		return domain.ImageVersion{}, fmt.Errorf("decode OCI image inspection: %w", err)
	}
	digest := strings.TrimPrefix(inspections[0].ID, "docker-pullable://")
	if len(inspections[0].RepoDigests) > 0 {
		if _, value, ok := strings.Cut(inspections[0].RepoDigests[0], "@"); ok {
			digest = value
		}
	}
	if !strings.HasPrefix(digest, "sha256:") {
		return domain.ImageVersion{}, fmt.Errorf("OCI image did not expose a sha256 digest")
	}
	if request.ExpectedSHA256 != "" && strings.TrimPrefix(request.ExpectedSHA256, "sha256:") != strings.TrimPrefix(digest, "sha256:") {
		return domain.ImageVersion{}, fmt.Errorf("checksum mismatch")
	}
	sourceReference := reference
	validation := map[string]any{"repo_digests": inspections[0].RepoDigests}
	if sourceType == "oci_local" {
		sourceReference = inspections[0].ID
		validation["local_reference"] = inspectReference
	}
	return domain.ImageVersion{ID: domain.NewID(), RuntimeKind: domain.RuntimeDocker, Name: request.Name, Version: request.Version, Digest: digest, SourceType: sourceType, SourceReference: sourceReference, Format: "oci", SizeBytes: inspections[0].Size, Availability: domain.ImageAvailable, LicenseStatus: domain.LicenseReviewed, LicenseNotes: request.LicenseNotes, Validation: validation, CreatedAt: time.Now().UTC()}, nil
}
func safeExtractTGZ(path, destination string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	if err = os.MkdirAll(destination, 0o700); err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		clean := filepath.Clean(header.Name)
		if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") || strings.Contains(clean, "/../") {
			return fmt.Errorf("unsafe archive path %q", header.Name)
		}
		target := filepath.Join(destination, clean)
		if !strings.HasPrefix(target, filepath.Clean(destination)+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe archive path")
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err = os.MkdirAll(target, 0o700); err != nil {
				return err
			}
		case tar.TypeReg:
			if err = os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
}
