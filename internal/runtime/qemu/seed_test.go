package qemu

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func TestSeedManagerBuildsPrivateNodeScopedISO(t *testing.T) {
	root := t.TempDir()
	xorriso := filepath.Join(root, "xorriso")
	script := `#!/bin/sh
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-output" ]; then shift; printf seed > "$1"; exit 0; fi
  shift
done
exit 1
`
	if err := os.WriteFile(xorriso, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	manager := &SeedManager{Root: filepath.Join(root, "bootstrap"), Xorriso: xorriso}
	path, err := manager.Build(context.Background(), domain.ID("lab-a"), domain.ID("node-a"), SeedSpec{UserData: "#cloud-config\npassword: private"})
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%v err=%v", info.Mode(), err)
	}
	if _, err = os.Stat(filepath.Join(filepath.Dir(path), "user-data")); !os.IsNotExist(err) {
		t.Fatal("plaintext user-data retained")
	}
}

func TestSeedManagerIncludesAndRemovesVendorData(t *testing.T) {
	root := t.TempDir()
	xorriso := filepath.Join(root, "xorriso")
	script := `#!/bin/sh
output=""
vendor=0
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-output" ]; then shift; output="$1"
  elif [ "$(basename "$1")" = "vendor-data" ]; then vendor=1
  fi
  shift
done
[ "$vendor" -eq 1 ] || exit 2
printf seed > "$output"
`
	if err := os.WriteFile(xorriso, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	manager := &SeedManager{Root: filepath.Join(root, "bootstrap"), Xorriso: xorriso}
	path, err := manager.Build(context.Background(), domain.ID("lab-a"), domain.ID("node-a"), SeedSpec{UserData: "#cloud-config\n{}", VendorData: "#cloud-config\npackages: [qemu-guest-agent]"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(filepath.Dir(path), "vendor-data")); !os.IsNotExist(err) {
		t.Fatal("plaintext vendor-data retained")
	}
}

func TestBuildArgsAttachesSeedISO(t *testing.T) {
	adapter := &Adapter{Root: t.TempDir()}
	args, _ := adapter.BuildArgs(domain.Node{ID: "node-a", CPUCount: 1, MemoryMiB: 512, Config: map[string]any{"seed_iso": "/run/netlab/seed.iso"}}, "/var/lib/netlab/base.qcow2")
	found := false
	forcedNoCloud := false
	for _, value := range args {
		if value == "file=/run/netlab/seed.iso,media=cdrom,readonly=on" {
			found = true
		}
		if value == "type=1,serial=ds=nocloud" {
			forcedNoCloud = true
		}
	}
	if !found || !forcedNoCloud {
		t.Fatalf("seed drive missing: %v", args)
	}
}

func TestSeedManagerRecoversUbuntuCredentials(t *testing.T) {
	root := t.TempDir()
	bootstrapRoot := filepath.Join(root, "bootstrap")
	seedPath := filepath.Join(bootstrapRoot, "lab-a", "node-a", "seed.iso")
	if err := os.MkdirAll(filepath.Dir(seedPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(seedPath, []byte("seed"), 0o600); err != nil {
		t.Fatal(err)
	}
	xorriso := filepath.Join(root, "xorriso-read")
	script := `#!/bin/sh
last=""
for value in "$@"; do last="$value"; done
cat > "$last" <<'EOF'
#cloud-config
{"chpasswd":{"users":[{"name":"ubuntu","password":"RecoveredPassword123","type":"text"}]}}
EOF
`
	if err := os.WriteFile(xorriso, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	manager := &SeedManager{Root: bootstrapRoot, Xorriso: xorriso}
	credentials, err := manager.Credentials(context.Background(), seedPath)
	if err != nil {
		t.Fatal(err)
	}
	if credentials.Username != "ubuntu" || credentials.Password != "RecoveredPassword123" {
		t.Fatalf("credentials=%+v", credentials)
	}
	if _, err = manager.Credentials(context.Background(), filepath.Join(root, "outside", "seed.iso")); err == nil {
		t.Fatal("expected unmanaged seed path rejection")
	}
}
