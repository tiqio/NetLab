package contract_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/netlab/netlab/internal/compliance"
)

func TestExactlyOneExternallyReachableAuthority(t *testing.T) {
	body, err := os.ReadFile("../../compliance/deployment-authority.json")
	if err != nil {
		t.Fatal(err)
	}
	var authority compliance.DeploymentAuthority
	if err := json.Unmarshal(body, &authority); err != nil {
		t.Fatal(err)
	}
	if err := compliance.ValidateDeploymentAuthority(authority); err != nil {
		t.Fatal(err)
	}
	authority.Instances[1].Role = "candidate"
	authority.Instances[1].ExternallyReachable = true
	if compliance.ValidateDeploymentAuthority(authority) == nil {
		t.Fatal("expected preview exposure rejection")
	}
}
