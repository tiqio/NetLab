package domain

import "testing"

func TestReleaseIdentityValidation(t *testing.T) {
	digest := DigestBytes([]byte("contract"))
	identity := ReleaseIdentity{Version: "1.0.0", CandidateID: "candidate-1", ContractDigest: digest}
	if err := identity.Validate(); err != nil {
		t.Fatal(err)
	}
	identity.ContractDigest = "invalid"
	if identity.Validate() == nil {
		t.Fatal("expected digest validation error")
	}
}

func TestTerminalProblemValidation(t *testing.T) {
	problem := Problem{Code: ProblemCodeCleanupFailed, Message: "cleanup failed", ResourceType: "node", ResourceID: "node-1", Phase: "delete", Cleanup: "partial", OperatorHint: "retry cleanup"}
	if err := problem.ValidateTerminal(); err != nil {
		t.Fatal(err)
	}
}
