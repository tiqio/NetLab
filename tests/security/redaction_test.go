package security

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/app/audit"
)

func TestAuditRedactionAndRepositoryHygiene(t *testing.T) {
	redacted := audit.Redact(map[string]any{"password": "secret-value", "argv": []any{"sh", "-c", "private command"}, "nested": map[string]any{"api_token": "token-value", "safe": "visible"}, "cloud-init-user-data": "bootstrap"})
	if redacted["password"] != "[REDACTED]" || redacted["cloud-init-user-data"] != "[REDACTED]" {
		t.Fatalf("redacted=%v", redacted)
	}
	if redacted["argv"] != "[REDACTED]" {
		t.Fatalf("guest arguments leaked: %v", redacted["argv"])
	}
	nested := redacted["nested"].(map[string]any)
	if nested["api_token"] != "[REDACTED]" || nested["safe"] != "visible" {
		t.Fatalf("nested=%v", nested)
	}

	root := filepath.Join("..", "..")
	forbiddenExtensions := map[string]bool{".qcow2": true, ".vmdk": true, ".iso": true, ".pcap": true, ".pcapng": true, ".pem": true, ".key": true}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(root, path)
		if entry.IsDir() && (entry.Name() == "node_modules" || entry.Name() == "dist" || entry.Name() == ".git" || entry.Name() == "bin") {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		if forbiddenExtensions[strings.ToLower(filepath.Ext(entry.Name()))] {
			t.Errorf("forbidden binary/secret artifact: %s", relative)
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		if info.Size() > 2<<20 {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		privateKeyMarker := "-----BEGIN " + "PRIVATE KEY-----"
		if strings.Contains(string(body), privateKeyMarker) {
			t.Errorf("private key material: %s", relative)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSystemdCaptureCapabilitiesRemainBounded(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "deploy", "systemd", "netlab.service"))
	if err != nil {
		t.Fatal(err)
	}
	unit := string(body)
	capabilities := "CAP_CHOWN CAP_KILL CAP_SETGID CAP_SETUID CAP_NET_ADMIN CAP_NET_RAW CAP_SYS_ADMIN CAP_SYS_PTRACE"
	if !strings.Contains(unit, "AmbientCapabilities="+capabilities) || !strings.Contains(unit, "CapabilityBoundingSet="+capabilities) {
		t.Fatalf("capture capabilities are not explicitly bounded: %s", unit)
	}
	if !strings.Contains(unit, "NoNewPrivileges=true") {
		t.Fatal("NoNewPrivileges must remain enabled")
	}
	if !strings.Contains(unit, "ReadWritePaths=/var/lib/netlab /run/netlab /run/netns") {
		t.Fatal("network namespace mount directory must remain writable")
	}
}
