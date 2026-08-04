package security_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestMaintenanceScriptUsesAtomicSQLiteSafetyAndExplicitReset(t *testing.T) {
	body, err := os.ReadFile("../../deploy/scripts/maintenance.sh")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, required := range []string{
		"NETLAB_RESET_CONFIRM=DELETE-ALL-NETLAB-LABORATORIES",
		"PRAGMA integrity_check",
		"PRAGMA foreign_key_check",
		"flock -n",
		"set -Eeuo pipefail",
		"atomic_replace",
		`"$sync_bin" -f`,
		"io.netlab.node_id",
		"startswith(\"netlab:\")",
		"foreign_observed",
		"VACUUM INTO",
		"NETLAB_RESET_DATABASE_MODE:-fresh",
		"INSERT INTO image_versions SELECT * FROM source.image_versions",
		"NETLAB_RESET_BACKUP",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("maintenance script missing %q", required)
		}
	}
	for _, forbidden := range []string{`cp -a "$database"`, `install -m 0600 "$backup" "$database"`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("maintenance script retains unsafe replacement %q", forbidden)
		}
	}
	command := exec.Command("bash", "-n", "../../deploy/scripts/maintenance.sh")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("maintenance syntax: %v: %s", err, output)
	}
}

func TestMaintenanceAtomicReplacementRollsBackOnFsyncFailure(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 CLI unavailable")
	}
	stateDirectory := t.TempDir()
	databasePath := filepath.Join(stateDirectory, "netlab.db")
	database, err := storesqlite.Open(context.Background(), databasePath)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err = database.DB.Exec(`INSERT INTO laboratories(id,name,created_at,updated_at) VALUES('keep','Keep',?,?)`, now, now); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if err = database.Close(); err != nil {
		t.Fatal(err)
	}
	failingSync := filepath.Join(stateDirectory, "sync-fail")
	if err = os.WriteFile(failingSync, []byte("#!/bin/sh\nexit 74\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("bash", "../../deploy/scripts/maintenance.sh", "vacuum")
	command.Env = append(os.Environ(),
		"NETLAB_STATE_DIR="+stateDirectory,
		"NETLAB_DATABASE="+databasePath,
		"NETLAB_SKIP_SERVICE_CHECK=1",
		"NETLAB_SKIP_IO_HEALTH_CHECK=1",
		"NETLAB_SYNC_BIN="+failingSync,
	)
	if output, runErr := command.CombinedOutput(); runErr == nil {
		t.Fatalf("vacuum unexpectedly succeeded: %s", output)
	}
	query := exec.Command("sqlite3", databasePath, `SELECT count(*) FROM laboratories WHERE id='keep'`)
	body, queryErr := query.CombinedOutput()
	if queryErr != nil || strings.TrimSpace(string(body)) != "1" {
		t.Fatalf("original database was not restored after fsync failure: %v: %s", queryErr, body)
	}
}

func TestMaintenanceBackupResetRestoreAndPrune(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 CLI unavailable")
	}
	stateDirectory := t.TempDir()
	databasePath := filepath.Join(stateDirectory, "netlab.db")
	database, err := storesqlite.Open(context.Background(), databasePath)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	old := time.Now().UTC().Add(-60 * 24 * time.Hour).Format(time.RFC3339Nano)
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO image_versions(id,runtime_kind,name,version,digest,source_type,source_reference,format,size_bytes,availability,license_status,license_notes,validation_json,created_at) VALUES('image','docker','busybox','1','sha256:test','registry','busybox','container',1,'available','approved','','{}',?)`, []any{now}},
		{`INSERT INTO device_templates(id,template_key,display_name,runtime_kind,created_at) VALUES('template','busybox','BusyBox','docker',?)`, []any{now}},
		{`INSERT INTO laboratories(id,name,created_at,updated_at) VALUES('laboratory','Maintenance',?,?)`, []any{now, now}},
		{`INSERT INTO operation_tasks(id,kind,resource_type,resource_id,state,progress_current,progress_total,created_at,finished_at) VALUES ('old-task','test','laboratory','laboratory','succeeded',1,1,?,?), ('running-task','test','laboratory','laboratory','running',0,1,?,NULL)`, []any{old, old, old}},
		{`INSERT INTO audit_events(id,actor_class,action,resource_type,resource_id,outcome,correlation_id,occurred_at) VALUES('audit','system','test','laboratory','laboratory','ok','correlation',?)`, []any{old}},
	}
	for _, statement := range statements {
		if _, err = database.DB.Exec(statement.query, statement.args...); err != nil {
			database.Close()
			t.Fatal(err)
		}
	}
	if err = database.Close(); err != nil {
		t.Fatal(err)
	}
	script := "../../deploy/scripts/maintenance.sh"
	baseEnvironment := append(os.Environ(),
		"NETLAB_STATE_DIR="+stateDirectory,
		"NETLAB_DATABASE="+databasePath,
		"NETLAB_SKIP_SERVICE_CHECK=1",
		"NETLAB_SKIP_IO_HEALTH_CHECK=1",
	)
	run := func(extra []string, arguments ...string) ([]byte, error) {
		command := exec.Command("bash", append([]string{script}, arguments...)...)
		command.Env = append(baseEnvironment, extra...)
		return command.CombinedOutput()
	}
	backupPath := filepath.Join(stateDirectory, "backups", "before-reset.db")
	if output, runErr := run(nil, "backup", backupPath); runErr != nil {
		t.Fatalf("backup: %v: %s", runErr, output)
	}
	if output, runErr := run(nil, "reset-labs", "--execute"); runErr == nil {
		t.Fatalf("reset succeeded without typed confirmation: %s", output)
	}
	if output, runErr := run([]string{"NETLAB_RESET_CONFIRM=DELETE-ALL-NETLAB-LABORATORIES"}, "reset-labs", "--execute"); runErr != nil {
		t.Fatalf("reset: %v: %s", runErr, output)
	}
	query := func(sql string) string {
		command := exec.Command("sqlite3", databasePath, sql)
		body, queryErr := command.CombinedOutput()
		if queryErr != nil {
			t.Fatalf("query %q: %v: %s", sql, queryErr, body)
		}
		return strings.TrimSpace(string(body))
	}
	if query("SELECT count(*) FROM laboratories") != "0" || query("SELECT count(*) FROM image_versions") != "1" || query("SELECT count(*) FROM device_templates") != "1" {
		t.Fatal("reset did not remove laboratories while preserving image and template metadata")
	}
	if output, runErr := run(nil, "restore", backupPath); runErr != nil {
		t.Fatalf("restore: %v: %s", runErr, output)
	}
	if query("SELECT count(*) FROM laboratories") != "1" {
		t.Fatal("restore did not recover the laboratory")
	}
	pruneOutput, runErr := run([]string{"NETLAB_RETENTION_DAYS=30"}, "prune", "--execute")
	if runErr != nil {
		t.Fatalf("prune: %v: %s", runErr, pruneOutput)
	}
	oldCount := query("SELECT count(*) FROM operation_tasks WHERE id='old-task'")
	runningCount := query("SELECT count(*) FROM operation_tasks WHERE id='running-task'")
	if oldCount != "0" || runningCount != "1" {
		t.Fatalf("prune counts old=%s running=%s output=%s rows=%s", oldCount, runningCount, pruneOutput, query("SELECT id||'|'||state||'|'||created_at||'|'||COALESCE(finished_at,'') FROM operation_tasks ORDER BY id"))
	}
	if output, runErr := run(nil, "vacuum"); runErr != nil {
		t.Fatalf("vacuum: %v: %s", runErr, output)
	}
	if query("PRAGMA integrity_check") != "ok" {
		t.Fatal("vacuum replacement failed integrity check")
	}
}
