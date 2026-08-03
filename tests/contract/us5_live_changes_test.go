package contract

import (
	"os"
	"strings"
	"testing"
)

func TestLiveChangeOperationsExistInOpenAPI(t *testing.T) {
	body, err := os.ReadFile("../../specs/001-network-simulator-platform/contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, operation := range []string{"addInterface", "removeInterface", "executeGuestCommand", "createPortMapping", "deletePortMapping"} {
		if !strings.Contains(text, "operationId: "+operation) {
			t.Fatalf("missing %s", operation)
		}
	}
}
