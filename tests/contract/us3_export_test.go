package contract

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestExportRedactionAndRoundTrip(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:export?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repository := storesqlite.NewTopologyRepository(database)
	labs := command.NewLaboratoryService(repository)
	nodes := command.NewNodeService(repository)
	lab, err := labs.Create(ctx, "source", "portable", domain.RecoveryRemainStopped)
	if err != nil {
		t.Fatal(err)
	}
	node, _, err := nodes.Create(ctx, lab.ID, "pc1", "pc", 1)
	if err != nil {
		t.Fatal(err)
	}
	node.Config["password"] = "must-not-export"
	exporter := command.NewExportService(repository, nil)
	bundle, err := exporter.Build(ctx, lab.ID)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(bundle)
	if string(body) == "" || json.Valid(body) == false {
		t.Fatal("invalid export JSON")
	}
	if bundle.Redaction.CredentialsExcluded != true || containsBytes(body, []byte("must-not-export")) {
		t.Fatalf("redaction failed: %s", body)
	}
	bundle.Laboratory.Name = "round-trip"
	imported, err := command.NewImportService(repository, nil).Import(ctx, bundle)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := repository.Snapshot(ctx, imported.ID)
	if err != nil || len(snapshot.Nodes) != 1 || snapshot.Nodes[0].Name != "pc1" {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
}

func containsBytes(body, fragment []byte) bool {
	for index := 0; index+len(fragment) <= len(body); index++ {
		if string(body[index:index+len(fragment)]) == string(fragment) {
			return true
		}
	}
	return false
}
