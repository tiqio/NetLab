package security_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/netlab/netlab/internal/compliance"
)

func TestTrackedComplianceEvidenceContainsNoProhibitedPayload(t *testing.T) {
	root := filepath.Join("..", "..", "compliance")
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if findings := compliance.ScanEvidence(path, body); len(findings) != 0 {
			t.Fatalf("%s: %v", path, findings)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
