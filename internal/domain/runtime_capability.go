package domain

import (
	"errors"
	"time"
)

type RuntimeCapability string

const (
	CapabilityImage       RuntimeCapability = "image"
	CapabilityBootstrap   RuntimeCapability = "bootstrap"
	CapabilityQMP         RuntimeCapability = "qmp"
	CapabilityQGA         RuntimeCapability = "qga"
	CapabilitySerial      RuntimeCapability = "serial"
	CapabilityVNC         RuntimeCapability = "vnc"
	CapabilityHotplug     RuntimeCapability = "hotplug"
	CapabilityGuestExec   RuntimeCapability = "guest_exec"
	CapabilityPortMapping RuntimeCapability = "port_mapping"
)

type CapabilityState string

const (
	CapabilityUnknown     CapabilityState = "unknown"
	CapabilityProbing     CapabilityState = "probing"
	CapabilityReady       CapabilityState = "ready"
	CapabilityUnavailable CapabilityState = "unavailable"
	CapabilityFailed      CapabilityState = "failed"
)

type RuntimeCapabilityObservation struct {
	NodeID     ID                `json:"node_id"`
	Capability RuntimeCapability `json:"capability"`
	Revision   Revision          `json:"revision"`
	State      CapabilityState   `json:"state"`
	Required   bool              `json:"required"`
	Details    map[string]any    `json:"details,omitempty"`
	Problem    *Problem          `json:"problem,omitempty"`
	ObservedAt time.Time         `json:"observed_at"`
}

func (o RuntimeCapabilityObservation) Validate() error {
	if o.NodeID == "" || o.Revision < 1 || o.ObservedAt.IsZero() {
		return errors.New("capability observation identity, revision, and observed time required")
	}
	validCapability := map[RuntimeCapability]bool{CapabilityImage: true, CapabilityBootstrap: true, CapabilityQMP: true, CapabilityQGA: true, CapabilitySerial: true, CapabilityVNC: true, CapabilityHotplug: true, CapabilityGuestExec: true, CapabilityPortMapping: true}
	validState := map[CapabilityState]bool{CapabilityUnknown: true, CapabilityProbing: true, CapabilityReady: true, CapabilityUnavailable: true, CapabilityFailed: true}
	if !validCapability[o.Capability] || !validState[o.State] {
		return errors.New("invalid capability or state")
	}
	if o.State == CapabilityFailed && o.Problem == nil {
		return errors.New("failed capability requires problem")
	}
	return nil
}
