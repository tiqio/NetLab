package contract_test

import (
	"path/filepath"
	"testing"

	"github.com/netlab/netlab/internal/compliance"
)

func TestComplianceSchemasAndDocuments(t *testing.T) {
	root := filepath.Join("..", "..")
	if _, err := compliance.LoadContractSchemas(filepath.Join(root, "specs", "002-constitution-gap-closure", "contracts")); err != nil {
		t.Fatal(err)
	}
	documents, err := compliance.LoadDocuments(
		filepath.Join(root, "compliance", "constitution-ledger.json"),
		filepath.Join(root, "compliance", "deployment-authority.json"),
		filepath.Join(root, "compliance", "template-readiness.json"),
		filepath.Join(root, "compliance", "evidence"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := compliance.ValidateDocuments(documents); err != nil {
		t.Fatal(err)
	}
}
