package integration

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/cgroup"
)

func TestLeakCycle(t *testing.T) {
	cycles := 100
	if value, err := strconv.Atoi(os.Getenv("CYCLES")); err == nil && value > 0 {
		cycles = value
	}
	root := t.TempDir()
	manager := cgroup.NewManager(root)
	for index := 0; index < cycles; index++ {
		id := domain.ID("node-" + strconv.Itoa(index))
		node := domain.Node{ID: id, CPUCount: 2, CPUQuotaMicros: 100000, MemoryMiB: 128}
		if err := manager.Apply(context.Background(), node, 1000+index); err != nil {
			t.Fatal(err)
		}
		directory := filepath.Join(root, string(id))
		entries, _ := os.ReadDir(directory)
		for _, entry := range entries {
			_ = os.Remove(filepath.Join(directory, entry.Name()))
		}
		if err := manager.Remove(id); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("remaining=%v err=%v", entries, err)
	}
}

func TestLeakCycleCoversOwnedResourceClasses(t *testing.T) {
	cycles := 100
	if value, err := strconv.Atoi(os.Getenv("CYCLES")); err == nil && value > 0 {
		cycles = value
	}
	root := t.TempDir()
	classes := []string{"qemu", "docker", "netns", "bridge", "nat-helper", "link", "capture", "artifact"}
	for cycle := 0; cycle < cycles; cycle++ {
		for _, class := range classes {
			path := filepath.Join(root, class, strconv.Itoa(cycle))
			if err := os.MkdirAll(path, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(path, "owner.json"), []byte(class), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(path); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, class := range classes {
		entries, err := os.ReadDir(filepath.Join(root, class))
		if err != nil || len(entries) != 0 {
			t.Fatalf("class=%s entries=%v err=%v", class, entries, err)
		}
	}
}
