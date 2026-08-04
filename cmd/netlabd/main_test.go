package main

import (
	"testing"

	"github.com/netlab/netlab/internal/support/config"
)

func TestBuildIdentityMakesPlaceholderAuthoritativeConfigValid(t *testing.T) {
	previousVersion, previousCandidate := version, candidateID
	previousBinary, previousContract, previousBuiltAt := binaryDigest, contractDigest, builtAt
	t.Cleanup(func() {
		version, candidateID = previousVersion, previousCandidate
		binaryDigest, contractDigest, builtAt = previousBinary, previousContract, previousBuiltAt
	})
	version, candidateID = "0.6.0", "candidate-1"
	binaryDigest = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	contractDigest = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	builtAt = "2026-08-04T00:00:00Z"
	value := config.Defaults()
	value.Deployment.Role = "authoritative"
	applyBuildIdentity(&value)
	if err := value.Validate(); err != nil {
		t.Fatal(err)
	}
}
