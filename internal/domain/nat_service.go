package domain

import "time"

type NATServiceObservation struct {
	NetworkObjectID ID        `json:"network_object_id"`
	ConfigDigest    string    `json:"config_digest"`
	UnitName        string    `json:"unit_name"`
	ConfigPath      string    `json:"config_path"`
	LeasePath       string    `json:"lease_path"`
	PID             int       `json:"pid,omitempty"`
	State           string    `json:"state"`
	Problem         *Problem  `json:"problem,omitempty"`
	ObservedAt      time.Time `json:"observed_at"`
}
