package contract_test

import (
	"os"
	"strings"
	"testing"
)

func TestRuntimeObservationEventContractsArePublishedAndDocumented(t *testing.T) {
	store, err := os.ReadFile("../../internal/store/sqlite/capability_repository.go")
	if err != nil {
		t.Fatal(err)
	}
	network, err := os.ReadFile("../../internal/store/sqlite/network_repository.go")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := os.ReadFile("../../specs/002-constitution-gap-closure/contracts/events.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, eventType := range []string{"node.capability_changed", "network_object.observed_state_changed"} {
		if !strings.Contains(string(store)+string(network), eventType) || !strings.Contains(string(contract), eventType) {
			t.Fatalf("missing event %s", eventType)
		}
	}
}
