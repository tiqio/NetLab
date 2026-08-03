package recovery

import (
	"context"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
	"os"
	"os/exec"
	"testing"
)

func TestNamespaceAdoption(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("requires root")
	}
	if _, err := exec.LookPath("ip"); err != nil {
		t.Skip("ip unavailable")
	}
	ctx := context.Background()
	db, err := storesqlite.Open(ctx, "file:recovery-us1?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := storesqlite.NewTopologyRepository(db)
	runtime, err := linuxnet.NewEndpointRuntime()
	if err != nil {
		t.Fatal(err)
	}
	node := domain.Node{ID: domain.NewID(), Name: "pc", Kind: "pc", DesiredState: domain.DesiredRunning, ObservedState: domain.ObservedStopped}
	if err = runtime.Start(ctx, node); err != nil {
		t.Fatal(err)
	}
	defer runtime.Delete(ctx, node)
	actual, err := runtime.Inspect(ctx, node)
	if err != nil || actual.State != domain.ObservedRunning {
		t.Fatalf("actual=%+v err=%v", actual, err)
	}
	_ = reconcile.NewRecovery(repo, reconcile.NewTopologyReconciler(repo, runtime))
}
