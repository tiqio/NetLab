package reconcile

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

type deletionInterfaceCleanup struct{ names []string }

func (c *deletionInterfaceCleanup) Delete(_ context.Context, name string) error {
	c.names = append(c.names, name)
	return nil
}

func TestLaboratoryDeletionRemovesArtifactsAndRows(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:delete-reconcile?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repository := storesqlite.NewTopologyRepository(database)
	repositories := storesqlite.NewRepositories(database)
	lab, err := command.NewLaboratoryService(repository).Create(ctx, "delete-me", "", domain.RecoveryRemainStopped)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "artifact")
	if err = os.WriteFile(path, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err = repositories.CreateArtifact(ctx, domain.Artifact{ID: "artifact", Kind: "export", Path: path, MediaType: "application/json", SizeBytes: 4, SHA256: "test", OwnerType: "laboratory", OwnerID: lab.ID, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err = database.Write(ctx, func(tx *sql.Tx) error {
		_, execErr := tx.ExecContext(ctx, `INSERT INTO nodes(id,laboratory_id,name,kind,revision,desired_state,observed_state,cpu_count,cpu_quota_micros,memory_mib,storage_gib,interface_limit,process_limit,config_json,created_at,updated_at) VALUES(?,?,?,?,1,'stopped','stopped',1,0,512,1,4,64,'{}',?,?)`, "node", lab.ID, "node", "qemu", now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
		return execErr
	}); err != nil {
		t.Fatal(err)
	}
	if err = repository.AddInterface(ctx, domain.Interface{ID: "interface-long-identifier", NodeID: "node", Slot: 0, Name: "eth0", Driver: "virtio-net-pci", MACAddress: "02:00:00:00:00:01", OperationalState: "down", Revision: 1}); err != nil {
		t.Fatal(err)
	}
	if err = command.NewLaboratoryService(repository).Delete(ctx, lab.ID, lab.Revision); err != nil {
		t.Fatal(err)
	}
	cleanup := &deletionInterfaceCleanup{}
	reconciler := NewLaboratoryDeletionReconciler(repository, RuntimeDispatch{}, nil, nil, nil, cleanup)
	if err = reconciler.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.GetLaboratory(ctx, lab.ID); err == nil {
		t.Fatal("laboratory row remains")
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("artifact remains: %v", err)
	}
	if len(cleanup.names) != 1 || cleanup.names[0] != "nltinterface-lo" {
		t.Fatalf("interface cleanup names=%v", cleanup.names)
	}
}
