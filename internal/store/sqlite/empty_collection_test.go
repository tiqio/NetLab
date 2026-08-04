package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func TestEmptySharedCollectionsAreNonNil(t *testing.T) {
	database, err := Open(context.Background(), filepath.Join(t.TempDir(), "netlab.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	laboratories, err := NewTopologyRepository(database).ListLaboratories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if laboratories == nil || len(laboratories) != 0 {
		t.Fatalf("expected empty laboratory collection, got %#v", laboratories)
	}
	ownershipRecords, err := NewRepositories(database).ListRuntimeOwnership(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if ownershipRecords == nil || len(ownershipRecords) != 0 {
		t.Fatalf("expected empty ownership collection, got %#v", ownershipRecords)
	}
}
