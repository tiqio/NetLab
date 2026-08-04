package security_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAuthorityGuardRejectsLegacyExternalInstance(t *testing.T) {
	root := t.TempDir()
	fixture := filepath.Join(root, "ss.txt")
	if err := os.WriteFile(fixture, []byte("LISTEN 0 4096 *:18082 *:* users:((\"netlabd\",pid=200,fd=10))\nLISTEN 0 4096 *:18080 *:* users:((\"netlabd\",pid=100,fd=11))\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "100"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/opt/netlab/netlabd", filepath.Join(root, "100", "exe")); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("bash", "../../deploy/scripts/check-authority.sh", "preflight")
	command.Env = append(os.Environ(), "NETLAB_SS_FIXTURE="+fixture, "NETLAB_PROC_ROOT="+root)
	if err := command.Run(); err == nil {
		t.Fatal("expected conflicting legacy authority to fail")
	}
	command = exec.Command("bash", "../../deploy/scripts/check-authority.sh", "preflight")
	command.Env = append(os.Environ(), "NETLAB_SS_FIXTURE="+fixture, "NETLAB_PROC_ROOT="+root, "NETLAB_RETIRE_LEGACY=1", "NETLAB_AUTHORITY_DRY_RUN=1")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("retire legacy: %v: %s", err, output)
	}
}
