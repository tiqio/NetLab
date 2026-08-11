package domain

import "testing"

func TestRuntimeRecoveryTransitions(t *testing.T) {
	tests := []struct {
		from RuntimeRecoveryState
		to   RuntimeRecoveryState
		ok   bool
	}{
		{RuntimeRecoveryUnknown, RuntimeRecoveryInspecting, true},
		{RuntimeRecoveryInspecting, RuntimeRecoveryAdopted, true},
		{RuntimeRecoveryInspecting, RuntimeRecoveryRecreated, true},
		{RuntimeRecoveryInspecting, RuntimeRecoveryFailed, true},
		{RuntimeRecoveryFailed, RuntimeRecoveryInspecting, true},
		{RuntimeRecoveryAdopted, RuntimeRecoveryDeleted, false},
		{RuntimeRecoveryAdopted, RuntimeRecoveryDeleting, true},
		{RuntimeRecoveryDeleting, RuntimeRecoveryDeleted, true},
	}
	for _, test := range tests {
		if got := CanTransitionRuntimeRecovery(test.from, test.to); got != test.ok {
			t.Fatalf("transition %s -> %s: got %v want %v", test.from, test.to, got, test.ok)
		}
	}
}

func TestRuntimeBackingObservationHealthyRequiresOwnedUsableBacking(t *testing.T) {
	observation := RuntimeBackingObservation{Kind: RuntimeBackingNamespace, Owned: true, Usable: true}
	if !observation.Healthy() {
		t.Fatal("owned usable backing should be healthy")
	}
	observation.Owned = false
	if observation.Healthy() {
		t.Fatal("unowned backing must not be healthy")
	}
}
