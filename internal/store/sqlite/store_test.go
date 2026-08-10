package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func TestOpenMigrationAndAtomicOutbox(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, "file:testdb?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var foreignKeys int
	if err = db.DB.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		t.Fatalf("foreign keys=%d err=%v", foreignKeys, err)
	}
	repo := NewRepositories(db)
	now := time.Now().UTC()
	lab := domain.Laboratory{ID: domain.NewID(), Name: "lab", Revision: 1, RecoveryPolicy: domain.RecoveryAutoRestore, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	event := domain.OutboxEvent{Type: "laboratory.created", LaboratoryID: lab.ID, ResourceType: "laboratory", ResourceID: lab.ID, Revision: 1, OccurredAt: now}
	if err = repo.CreateLaboratory(ctx, lab, event); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.GetLaboratory(ctx, lab.ID); err != nil {
		t.Fatal(err)
	}
	events, err := repo.OutboxAfter(ctx, 0, 10)
	if err != nil || len(events) != 1 {
		t.Fatalf("events=%d err=%v", len(events), err)
	}
}

func TestMigrationIsIdempotent(t *testing.T) {
	db, err := Open(context.Background(), "file:migrate?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestNetworkAttachmentRevisionMigrationDefaultsLegacyRows(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "legacy-attachment.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	if _, err = raw.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
		t.Fatal(err)
	}
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") || entry.Name() >= "0014_" {
			continue
		}
		body, readErr := migrationFS.ReadFile("migrations/" + entry.Name())
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, err = raw.ExecContext(ctx, string(body)); err != nil {
			t.Fatalf("apply %s: %v", entry.Name(), err)
		}
		version, parseErr := strconv.Atoi(strings.SplitN(entry.Name(), "_", 2)[0])
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		sum := sha256.Sum256(body)
		if _, err = raw.ExecContext(ctx, "INSERT INTO schema_migrations(version,checksum,applied_at) VALUES(?,?,?)", version, hex.EncodeToString(sum[:]), time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	statements := []string{
		`INSERT INTO laboratories(id,name,created_at,updated_at) VALUES('lab','legacy',?,?)`,
		`INSERT INTO nodes(id,laboratory_id,name,kind,created_at,updated_at) VALUES('node','lab','node','docker',?,?)`,
		`INSERT INTO interfaces(id,node_id,slot,name,mac_address) VALUES('interface','node',0,'eth0','02:00:00:00:00:01')`,
		`INSERT INTO network_objects(id,laboratory_id,name,kind,created_at,updated_at) VALUES('object','lab','bridge','bridge',?,?)`,
		`INSERT INTO network_attachments(id,network_object_id,interface_id,port_name) VALUES('attachment','object','interface','eth0')`,
	}
	for _, statement := range statements {
		arguments := []any{}
		if strings.Count(statement, "?") == 2 {
			arguments = []any{now, now}
		}
		if _, err = raw.ExecContext(ctx, statement, arguments...); err != nil {
			t.Fatalf("legacy fixture: %v", err)
		}
	}
	database := &Database{DB: raw}
	if err = database.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var revision domain.Revision
	if err = raw.QueryRowContext(ctx, `SELECT revision FROM network_attachments WHERE id='attachment'`).Scan(&revision); err != nil {
		t.Fatal(err)
	}
	if revision != 1 {
		t.Fatalf("legacy attachment revision=%d want 1", revision)
	}
}
