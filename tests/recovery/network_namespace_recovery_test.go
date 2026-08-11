package recovery_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
	"golang.org/x/sys/unix"
)

func TestServiceRestartRecreatesInvalidOwnedNamespaceReference(t *testing.T) {
	if os.Getenv("NETLAB_PRIVILEGED_TESTS") != "1" {
		t.Skip("set NETLAB_PRIVILEGED_TESTS=1 to run namespace recovery")
	}
	if os.Geteuid() != 0 {
		t.Fatal("namespace recovery test requires root")
	}
	if _, err := exec.LookPath("ip"); err != nil {
		t.Fatal(err)
	}
	object := domain.NetworkObject{
		ID:   domain.NewID(),
		Kind: domain.NetworkSwitchL3,
		Config: map[string]any{
			"interfaces": []any{map[string]any{"name": "eth0", "addresses": []any{}}},
		},
	}
	namespace := linuxnet.SwitchL3NamespaceName(object.ID)
	path := filepath.Join("/run/netns", namespace)
	_ = exec.Command("ip", "netns", "delete", namespace).Run()
	t.Cleanup(func() { _ = exec.Command("ip", "netns", "delete", namespace).Run() })
	if output, err := exec.Command("ip", "netns", "add", namespace).CombinedOutput(); err != nil {
		t.Fatalf("create namespace: %v: %s", err, output)
	}
	if err := unix.Unmount(path, unix.MNT_DETACH); err != nil {
		t.Fatalf("invalidate namespace reference: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stale namespace reference missing: %v", err)
	}
	runtime, err := linuxnet.NewSwitchL3Runtime(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = runtime.Configure(context.Background(), object); err != nil {
		t.Fatalf("recreate invalid namespace: %v", err)
	}
	observation, err := runtime.InspectNetworkObject(context.Background(), object)
	if err != nil || !observation.Owned || !observation.Usable {
		t.Fatalf("observation=%+v err=%v", observation, err)
	}
}
