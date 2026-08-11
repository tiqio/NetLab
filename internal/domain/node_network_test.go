package domain

import "testing"

func boolPointer(value bool) *bool { return &value }

func TestValidateNodeForwardingSettingsAllowsExplicitDockerDualStack(t *testing.T) {
	if err := ValidateNodeForwardingSettings(string(RuntimeDocker), boolPointer(true), boolPointer(false)); err != nil {
		t.Fatal(err)
	}
}

func TestValidateNodeForwardingSettingsRejectsNonDockerRuntime(t *testing.T) {
	err := ValidateNodeForwardingSettings(string(RuntimeQEMU), boolPointer(true), nil)
	problem, ok := err.(NetworkConfigError)
	if !ok || problem.Code != "forwarding_unsupported" {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateNodeForwardingSettingsAllowsOmittedValues(t *testing.T) {
	if err := ValidateNodeForwardingSettings(string(RuntimeQEMU), nil, nil); err != nil {
		t.Fatal(err)
	}
}
