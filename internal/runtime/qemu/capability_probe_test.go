package qemu

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func TestCapabilityProbeReportsImageAndMissingGuestAgentTruthfully(t *testing.T) {
	root := t.TempDir()
	image := filepath.Join(root, "ubuntu.qcow2")
	if err := os.WriteFile(image, []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	adapter := &Adapter{Root: root}
	probe := NewCapabilityProbe(adapter)
	observations, err := probe.Probe(context.Background(), domain.Node{ID: "node-1", Kind: "qemu", Config: map[string]any{"image_path": image, "template_key": "ubuntu"}})
	if err != nil {
		t.Fatal(err)
	}
	states := map[domain.RuntimeCapability]domain.CapabilityState{}
	for _, observation := range observations {
		states[observation.Capability] = observation.State
	}
	if states[domain.CapabilityImage] != domain.CapabilityReady {
		t.Fatalf("image state=%s", states[domain.CapabilityImage])
	}
	if states[domain.CapabilityQGA] != domain.CapabilityUnavailable || states[domain.CapabilityGuestExec] != domain.CapabilityUnavailable {
		t.Fatalf("qga=%s guest=%s", states[domain.CapabilityQGA], states[domain.CapabilityGuestExec])
	}
}

func TestBootstrapRequiredOnlyForValidatedUbuntuTemplate(t *testing.T) {
	tests := []struct {
		templateKey string
		required    bool
	}{
		{templateKey: "ubuntu-qemu", required: true},
		{templateKey: "vyos", required: false},
		{templateKey: "fancywan", required: false},
		{templateKey: "fortigate", required: false},
	}
	for _, test := range tests {
		t.Run(test.templateKey, func(t *testing.T) {
			node := domain.Node{Config: map[string]any{"template_key": test.templateKey}}
			if got := bootstrapRequired(node); got != test.required {
				t.Fatalf("bootstrapRequired()=%t, want %t", got, test.required)
			}
		})
	}
}
