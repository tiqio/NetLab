package security_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseConfigPreservesOperatorSettings(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "netlab.yaml")
	release := filepath.Join(directory, "release.json")
	output := filepath.Join(directory, "result.yaml")
	config := "listen: \"10.72.1.7:18082\"\nstate_dir: /custom/state\nrelease:\n  version: old\n  candidate_id: old\ndeployment:\n  role: authoritative\n"
	identity := `{"version":"0.6.0","candidate_id":"candidate-2","binary_digest":"sha256:` + strings.Repeat("1", 64) + `","contract_digest":"sha256:` + strings.Repeat("2", 64) + `","built_at":"2026-08-04T00:00:00Z"}`
	if err := os.WriteFile(input, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(release, []byte(identity), 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("bash", "../../deploy/scripts/prepare-release-config.sh", input, release, output)
	if body, err := command.CombinedOutput(); err != nil {
		t.Fatalf("prepare: %v: %s", err, body)
	}
	body, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, expected := range []string{"10.72.1.7:18082", "/custom/state", "candidate-2", "deployment:"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("result missing %q: %s", expected, text)
		}
	}
	if strings.Contains(text, "candidate_id: old") {
		t.Fatalf("old release remained: %s", text)
	}
}

func TestGeneratedReadinessCoversBuiltIns(t *testing.T) {
	directory := t.TempDir()
	release := filepath.Join(directory, "release.json")
	output := filepath.Join(directory, "readiness.json")
	identity := `{"candidate_id":"candidate-2"}`
	if err := os.WriteFile(release, []byte(identity), 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("bash", "../../deploy/scripts/generate-template-readiness.sh", release, output)
	if body, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generate: %v: %s", err, body)
	}
	body, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"fancywan", "ubuntu-qemu", "fortigate", "vyos", "ruijie-router", "ruijie-switch", "busybox-container", "ubuntu-container", "nginx-container"} {
		if !strings.Contains(string(body), `"template_key": "`+key+`"`) {
			t.Fatalf("readiness missing %s", key)
		}
	}
}

func TestInstallerWaitsForAuthorityAndRetiresPreviewUnit(t *testing.T) {
	installer, err := os.ReadFile("../../deploy/scripts/install.sh")
	if err != nil {
		t.Fatal(err)
	}
	guard, err := os.ReadFile("../../deploy/scripts/check-authority.sh")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(installer), "for _ in {1..3000}") || !strings.Contains(string(installer), "authoritative listener did not become ready within 600 seconds") {
		t.Fatal("installer does not wait for the authoritative listener")
	}
	if !strings.Contains(string(guard), "systemctl disable --now netlab-preview.service") {
		t.Fatal("authority guard does not retire the known preview unit")
	}
	for _, required := range []string{"deploy/scripts/maintenance.sh", "rollback service failed to reach application readiness", "check-authority.sh verify"} {
		if !strings.Contains(string(installer), required) {
			t.Fatalf("installer missing %q", required)
		}
	}
}

func TestInstallerInitializesCredentialMasterKeyWithoutOverwrite(t *testing.T) {
	installer, err := os.ReadFile("../../deploy/scripts/install.sh")
	if err != nil {
		t.Fatal(err)
	}
	text := string(installer)
	for _, required := range []string{
		`if [[ ! -e /etc/netlab/credential-master.key ]]`,
		`openssl rand -base64 32`,
		`chmod 0600 "$key_file"`,
		`mv -n "$key_file" /etc/netlab/credential-master.key`,
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("installer is missing credential key safeguard %q", required)
		}
	}
}

func TestSystemdAuthorityRequiresApplicationReadiness(t *testing.T) {
	body, err := os.ReadFile("../../deploy/systemd/netlab.service")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, required := range []string{"Type=notify", "NotifyAccess=main", "TimeoutStartSec=600s"} {
		if !strings.Contains(text, required) {
			t.Fatalf("systemd unit missing %q", required)
		}
	}
}

func TestAuthoritativeStartupRejectsMissingTemplateReadiness(t *testing.T) {
	body, err := os.ReadFile("../../cmd/netlabd/main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, required := range []string{`cfg.Deployment.Role == "authoritative"`, `template readiness validation failed`, `os.Exit(1)`} {
		if !strings.Contains(text, required) {
			t.Fatalf("authoritative startup guard missing %q", required)
		}
	}
}
