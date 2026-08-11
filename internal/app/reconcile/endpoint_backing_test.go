package reconcile

import (
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func TestClassifyNetworkObjectBacking(t *testing.T) {
	tests := []struct {
		kind string
		want domain.RuntimeBackingKind
	}{
		{domain.NetworkBridge, domain.RuntimeBackingHostBridge},
		{domain.NetworkNAT, domain.RuntimeBackingHostBridge},
		{domain.NetworkPC, domain.RuntimeBackingNamespace},
		{domain.NetworkSwitchL2, domain.RuntimeBackingNamespace},
		{domain.NetworkSwitchL3, domain.RuntimeBackingNamespace},
	}
	for _, test := range tests {
		backing, err := ClassifyNetworkObjectBacking(domain.NetworkObject{ID: "object", Kind: test.kind})
		if err != nil {
			t.Fatalf("classify %s: %v", test.kind, err)
		}
		if backing.Kind != test.want {
			t.Fatalf("classify %s: got %s want %s", test.kind, backing.Kind, test.want)
		}
	}
}

func TestClassifyNetworkObjectBackingRejectsUnknownKind(t *testing.T) {
	_, err := ClassifyNetworkObjectBacking(domain.NetworkObject{ID: "object", Kind: "unknown"})
	if err == nil {
		t.Fatal("expected unknown backing kind to fail")
	}
}
