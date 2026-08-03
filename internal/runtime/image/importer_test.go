package image

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"github.com/netlab/netlab/internal/domain"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportChecksumAndLicense(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "image.raw")
	if err := os.WriteFile(source, []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	importer := NewImporter(root)
	importer.QEMUImg = ""
	if _, err := importer.Import(context.Background(), ImportRequest{Name: "u", Version: "1", RuntimeKind: domain.RuntimeQEMU, SourcePath: source}); err == nil {
		t.Fatal("expected license gate")
	}
	if _, err := importer.Import(context.Background(), ImportRequest{Name: "u", Version: "1", RuntimeKind: domain.RuntimeQEMU, SourcePath: source, LicenseNotes: "owned", ExpectedSHA256: "deadbeef"}); err == nil {
		t.Fatal("expected checksum mismatch")
	}
}

func TestDockerImportRequiresOCIReferenceAndDockerCLI(t *testing.T) {
	importer := NewImporter(t.TempDir())
	importer.Docker = ""
	if _, err := importer.Import(context.Background(), ImportRequest{RuntimeKind: domain.RuntimeDocker, LicenseNotes: "operator supplied"}); err == nil || !strings.Contains(err.Error(), "OCI image reference") {
		t.Fatalf("err=%v", err)
	}
	if _, err := importer.Import(context.Background(), ImportRequest{RuntimeKind: domain.RuntimeDocker, SourcePath: "busybox:1.36.1", LicenseNotes: "operator supplied"}); err == nil || !strings.Contains(err.Error(), "docker CLI") {
		t.Fatalf("err=%v", err)
	}
}

func TestLocalOCIReferenceValidation(t *testing.T) {
	importer := NewImporter(t.TempDir())
	importer.Docker = "docker"
	if _, err := importer.Import(context.Background(), ImportRequest{RuntimeKind: domain.RuntimeDocker, SourcePath: "local://", LicenseNotes: "operator built"}); err == nil || !strings.Contains(err.Error(), "local OCI image reference") {
		t.Fatalf("err=%v", err)
	}
}
func TestArchiveTraversalRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.tgz")
	file, _ := os.Create(path)
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "../escape.qcow2", Mode: 0o600, Size: 1, Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte("x"))
	_ = tw.Close()
	_ = gz.Close()
	_ = file.Close()
	if err := safeExtractTGZ(path, t.TempDir()); err == nil {
		t.Fatal("expected traversal rejection")
	}
}
