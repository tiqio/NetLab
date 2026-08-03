package security_test

import (
	"os"
	"strings"
	"testing"
)

func TestManagementBoundaryPolicyCoversAllControlPorts(t *testing.T) {
	body, err := os.ReadFile("../../deploy/nftables/netlab-management.nft")
	if err != nil {
		t.Fatal(err)
	}
	policy := string(body)
	for _, required := range []string{"18082", "10.72.0.0/16", "drop"} {
		if !strings.Contains(policy, required) {
			t.Fatalf("policy missing %s", required)
		}
	}
	for _, unrelated := range []string{"8088", "18080"} {
		if strings.Contains(policy, unrelated) {
			t.Fatalf("policy must not claim unrelated host port %s", unrelated)
		}
	}
}
