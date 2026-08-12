package domain

import "testing"

func TestTrafficWorkloadValidationAndAggregates(t *testing.T) {
	w := TrafficWorkload{Name: "ping", Source: TrafficWorkloadEndpoint{Kind: "node", ResourceID: "n"}, Protocol: "icmp", AddressFamily: "ipv4", Destination: TrafficWorkloadDestination{Address: "192.0.2.1"}, IntervalSeconds: 5, TimeoutSeconds: 2, Attempts: 2, Successes: 1, Failures: 1}
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
	w.Successes = 3
	if w.Validate() == nil {
		t.Fatal("invalid aggregates accepted")
	}
}
