package integration

import (
	"context"
	dockeradapter "github.com/netlab/netlab/internal/runtime/docker"
	qemuadapter "github.com/netlab/netlab/internal/runtime/qemu"
	"os"
	"testing"
)

func TestRuntimeAvailability(t *testing.T) {
	if os.Getenv("NETLAB_PRIVILEGED_TESTS") != "1" {
		t.Skip("set NETLAB_PRIVILEGED_TESTS=1")
	}
	if _, err := qemuadapter.NewAdapter(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	docker, err := dockeradapter.NewAdapter()
	if err != nil {
		t.Fatal(err)
	}
	_ = docker
	_ = context.Background()
}
