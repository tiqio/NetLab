package domain

import "time"

type RuntimeBackingKind string

const (
	RuntimeBackingNamespace  RuntimeBackingKind = "namespace"
	RuntimeBackingHostBridge RuntimeBackingKind = "host_bridge"
	RuntimeBackingQEMUTap    RuntimeBackingKind = "qemu_tap"
	RuntimeBackingDockerVeth RuntimeBackingKind = "docker_veth"
)

type RuntimeRecoveryState string

const (
	RuntimeRecoveryUnknown    RuntimeRecoveryState = "unknown"
	RuntimeRecoveryInspecting RuntimeRecoveryState = "inspecting"
	RuntimeRecoveryAdopted    RuntimeRecoveryState = "adopted"
	RuntimeRecoveryRecreated  RuntimeRecoveryState = "recreated"
	RuntimeRecoveryFailed     RuntimeRecoveryState = "failed"
	RuntimeRecoveryDeleting   RuntimeRecoveryState = "deleting"
	RuntimeRecoveryDeleted    RuntimeRecoveryState = "deleted"
)

type RuntimeBackingObservation struct {
	Kind        RuntimeBackingKind `json:"backing_kind"`
	RuntimeName string             `json:"runtime_name,omitempty"`
	Owned       bool               `json:"owned"`
	Usable      bool               `json:"usable"`
	Adoptable   bool               `json:"adoptable"`
	Recreatable bool               `json:"recreatable"`
	ObservedAt  time.Time          `json:"observed_at"`
	Problem     *Problem           `json:"problem,omitempty"`
}

func (observation RuntimeBackingObservation) Healthy() bool {
	return observation.Owned && observation.Usable && observation.Problem == nil
}

func CanTransitionRuntimeRecovery(from, to RuntimeRecoveryState) bool {
	allowed := map[RuntimeRecoveryState]map[RuntimeRecoveryState]bool{
		RuntimeRecoveryUnknown:    {RuntimeRecoveryInspecting: true},
		RuntimeRecoveryInspecting: {RuntimeRecoveryAdopted: true, RuntimeRecoveryRecreated: true, RuntimeRecoveryFailed: true, RuntimeRecoveryDeleting: true},
		RuntimeRecoveryAdopted:    {RuntimeRecoveryInspecting: true, RuntimeRecoveryFailed: true, RuntimeRecoveryDeleting: true},
		RuntimeRecoveryRecreated:  {RuntimeRecoveryInspecting: true, RuntimeRecoveryFailed: true, RuntimeRecoveryDeleting: true},
		RuntimeRecoveryFailed:     {RuntimeRecoveryInspecting: true, RuntimeRecoveryDeleting: true},
		RuntimeRecoveryDeleting:   {RuntimeRecoveryDeleted: true, RuntimeRecoveryFailed: true},
	}
	return allowed[from][to]
}
