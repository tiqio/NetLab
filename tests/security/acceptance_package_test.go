package security_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/netlab/netlab/internal/compliance"
)

func TestAcceptancePackageRejectsSecretsAndBinaryArtifacts(t *testing.T) {
	if findings := compliance.ScanEvidence("run.json", []byte(`{"password":"eve"}`)); len(findings) == 0 {
		t.Fatal("secret-bearing evidence accepted")
	}
	if findings := compliance.ScanEvidence("capture.pcap", []byte("metadata")); len(findings) == 0 {
		t.Fatal("packet capture accepted")
	}
}

func TestAcceptancePackageDigestChangesWithMetadata(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "current-candidate.json")
	if err := os.WriteFile(path, []byte(`{"candidate_id":"one"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := compliance.DigestTree(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte(`{"candidate_id":"two"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := compliance.DigestTree(directory)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("evidence digest did not change")
	}
}
