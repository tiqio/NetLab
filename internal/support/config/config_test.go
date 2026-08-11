package config

import (
	"strings"
	"testing"
)

func TestDefaultsWarnAndValidate(t *testing.T) {
	c := Defaults()
	if c.Captures.Concurrent != 16 {
		t.Fatalf("capture concurrency=%d", c.Captures.Concurrent)
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.SecurityWarning() == "" {
		t.Fatal("expected warning")
	}
	c.Listen = "127.0.0.1:8080"
	if c.SecurityWarning() != "" {
		t.Fatal("unexpected warning")
	}
}
func TestInvalidCaptureLimits(t *testing.T) {
	c := Defaults()
	c.Captures.Concurrent = 0
	if c.Validate() == nil {
		t.Fatal("expected validation error")
	}
}

func TestQEMUCountLimitAllowsZeroForUnlimited(t *testing.T) {
	c := Defaults()
	if c.Resources.MaxRunningQEMU != 0 {
		t.Fatalf("default max running QEMU=%d", c.Resources.MaxRunningQEMU)
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	c.Resources.MaxRunningQEMU = -1
	if c.Validate() == nil {
		t.Fatal("negative QEMU limit accepted")
	}
}

func TestReleaseIdentityRequired(t *testing.T) {
	c := Defaults()
	c.Release.CandidateID = ""
	if c.Validate() == nil {
		t.Fatal("expected release identity validation error")
	}
}

func TestAuthoritativeDeploymentRejectsPlaceholderReleaseIdentity(t *testing.T) {
	c := Defaults()
	c.Deployment.Role = "authoritative"
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "authoritative release") {
		t.Fatalf("expected authoritative identity error, got %v", err)
	}
	c.Release.Version = "1.0.0"
	c.Release.CandidateID = "candidate-20260729"
	c.Release.BinaryDigest = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	c.Release.ContractDigest = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	c.Release.BuiltAt = "2026-07-29T06:00:00Z"
	if err := c.Validate(); err != nil {
		t.Fatalf("valid authoritative identity rejected: %v", err)
	}
}
