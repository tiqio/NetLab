package qemu

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type CapabilityProbe struct {
	adapter *Adapter
	timeout time.Duration
}

func NewCapabilityProbe(adapter *Adapter) *CapabilityProbe {
	return &CapabilityProbe{adapter: adapter, timeout: time.Second}
}

func (p *CapabilityProbe) Probe(ctx context.Context, node domain.Node) ([]domain.RuntimeCapabilityObservation, error) {
	if node.Kind != "qemu" {
		return nil, nil
	}
	if p == nil || p.adapter == nil {
		return nil, fmt.Errorf("QEMU adapter unavailable")
	}
	runtimeDirectory := p.adapter.RuntimeDir(node.ID)
	observations := []domain.RuntimeCapabilityObservation{
		fileObservation(node.ID, domain.CapabilityImage, imagePath(node, runtimeDirectory), true),
		fileObservation(node.ID, domain.CapabilityBootstrap, stringConfig(node.Config, "seed_iso"), bootstrapRequired(node)),
		fileObservation(node.ID, domain.CapabilitySerial, filepath.Join(runtimeDirectory, "serial.sock"), false),
		fileObservation(node.ID, domain.CapabilityVNC, filepath.Join(runtimeDirectory, "vnc.sock"), false),
	}
	qmp := socketProbe(ctx, node.ID, domain.CapabilityQMP, filepath.Join(runtimeDirectory, "qmp.sock"), true, func() error {
		monitor, err := ConnectQMP(filepath.Join(runtimeDirectory, "qmp.sock"), p.timeout)
		if err != nil {
			return err
		}
		defer monitor.Close()
		_, err = monitor.Run("query-status", nil)
		return err
	})
	qga := socketProbe(ctx, node.ID, domain.CapabilityQGA, filepath.Join(runtimeDirectory, "qga.sock"), false, func() error {
		agent, err := ConnectGuestAgent(filepath.Join(runtimeDirectory, "qga.sock"), p.timeout)
		if err != nil {
			return err
		}
		defer agent.Close()
		probeContext, cancel := context.WithTimeout(ctx, p.timeout)
		defer cancel()
		_, err = agent.RunContext(probeContext, "guest-ping", map[string]any{})
		return err
	})
	observations = append(observations, qmp, qga)
	observations = append(observations,
		derivedObservation(node.ID, domain.CapabilityHotplug, qmp, true),
		derivedObservation(node.ID, domain.CapabilityGuestExec, qga, false),
		domain.RuntimeCapabilityObservation{NodeID: node.ID, Capability: domain.CapabilityPortMapping, State: domain.CapabilityReady, Required: false, Details: map[string]any{"mechanism": "host nftables"}},
	)
	return observations, nil
}

func imagePath(node domain.Node, runtimeDirectory string) string {
	if path := stringConfig(node.Config, "image_path"); path != "" {
		return path
	}
	return filepath.Join(runtimeDirectory, "disk.qcow2")
}

func bootstrapRequired(node domain.Node) bool {
	key := stringConfig(node.Config, "template_key")
	return key == "ubuntu" || key == "vyos" || key == "fancywan"
}

func stringConfig(config map[string]any, key string) string {
	value, _ := config[key].(string)
	return value
}

func fileObservation(nodeID domain.ID, capability domain.RuntimeCapability, path string, required bool) domain.RuntimeCapabilityObservation {
	observation := domain.RuntimeCapabilityObservation{NodeID: nodeID, Capability: capability, State: domain.CapabilityUnavailable, Required: required, Details: map[string]any{"path": path}}
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			observation.State = domain.CapabilityReady
		} else {
			observation.Problem = &domain.Problem{Code: string(capability) + "_unavailable", Message: err.Error(), Retryable: true, ResourceType: "node", ResourceID: nodeID, Phase: "capability_probe", Cleanup: "runtime unchanged", OperatorHint: "provide or start the declared capability prerequisite"}
		}
	}
	return observation
}

func socketProbe(ctx context.Context, nodeID domain.ID, capability domain.RuntimeCapability, path string, required bool, probe func() error) domain.RuntimeCapabilityObservation {
	observation := fileObservation(nodeID, capability, path, required)
	if observation.State != domain.CapabilityReady || ctx.Err() != nil {
		return observation
	}
	if err := probe(); err != nil {
		observation.State = domain.CapabilityUnavailable
		observation.Problem = &domain.Problem{Code: string(capability) + "_unavailable", Message: err.Error(), Retryable: true, ResourceType: "node", ResourceID: nodeID, Phase: "capability_probe", Cleanup: "runtime unchanged", OperatorHint: "verify the guest/runtime service and retry"}
	}
	return observation
}

func derivedObservation(nodeID domain.ID, capability domain.RuntimeCapability, source domain.RuntimeCapabilityObservation, required bool) domain.RuntimeCapabilityObservation {
	return domain.RuntimeCapabilityObservation{NodeID: nodeID, Capability: capability, State: source.State, Required: required, Details: map[string]any{"depends_on": source.Capability}, Problem: source.Problem}
}
