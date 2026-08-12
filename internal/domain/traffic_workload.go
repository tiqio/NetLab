package domain

import (
	"fmt"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

type TrafficWorkloadEndpoint struct {
	Kind        string `json:"kind"`
	ResourceID  ID     `json:"resource_id"`
	InterfaceID ID     `json:"interface_id,omitempty"`
}
type TrafficWorkloadDestination struct {
	Address string `json:"address,omitempty"`
	URL     string `json:"url,omitempty"`
	Name    string `json:"name,omitempty"`
}
type TrafficWorkload struct {
	ID              ID                         `json:"id"`
	LaboratoryID    ID                         `json:"laboratory_id"`
	Name            string                     `json:"name"`
	Revision        Revision                   `json:"revision"`
	Source          TrafficWorkloadEndpoint    `json:"source"`
	Protocol        string                     `json:"protocol"`
	AddressFamily   string                     `json:"address_family"`
	Destination     TrafficWorkloadDestination `json:"destination"`
	IntervalSeconds int                        `json:"interval_seconds"`
	TimeoutSeconds  int                        `json:"timeout_seconds"`
	DesiredState    string                     `json:"desired_state"`
	ObservedState   string                     `json:"observed_state"`
	Attempts        int64                      `json:"attempts"`
	Successes       int64                      `json:"successes"`
	Failures        int64                      `json:"failures"`
	MatchedBytes    int64                      `json:"matched_bytes"`
	LastSuccessAt   *time.Time                 `json:"last_success_at,omitempty"`
	LastError       *Problem                   `json:"last_error,omitempty"`
	CreatedAt       time.Time                  `json:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at"`
}

func (w *TrafficWorkload) Validate() error {
	w.Name = strings.TrimSpace(w.Name)
	w.Protocol = strings.ToLower(strings.TrimSpace(w.Protocol))
	w.AddressFamily = strings.ToLower(strings.TrimSpace(w.AddressFamily))
	if w.Name == "" || w.Source.ResourceID == "" {
		return fmt.Errorf("workload name and source are required")
	}
	if w.Protocol != "icmp" && w.Protocol != "http" && w.Protocol != "dns" {
		return fmt.Errorf("unsupported workload protocol %q", w.Protocol)
	}
	if w.AddressFamily == "" {
		w.AddressFamily = "auto"
	}
	if w.AddressFamily != "auto" && w.AddressFamily != "ipv4" && w.AddressFamily != "ipv6" {
		return fmt.Errorf("unsupported address family %q", w.AddressFamily)
	}
	if w.IntervalSeconds < 1 || w.IntervalSeconds > 3600 || w.TimeoutSeconds < 1 || w.TimeoutSeconds > 60 || w.TimeoutSeconds > w.IntervalSeconds {
		return fmt.Errorf("workload interval or timeout is outside supported limits")
	}
	switch w.Protocol {
	case "icmp":
		if _, err := netip.ParseAddr(w.Destination.Address); err != nil {
			return fmt.Errorf("invalid ICMP destination")
		}
	case "http":
		u, err := url.Parse(w.Destination.URL)
		if err != nil || u.Scheme != "http" || u.Host == "" {
			return fmt.Errorf("invalid HTTP destination")
		}
	case "dns":
		if strings.TrimSpace(w.Destination.Name) == "" {
			return fmt.Errorf("DNS name is required")
		}
		if strings.TrimSpace(w.Destination.Address) != "" {
			address, err := netip.ParseAddr(w.Destination.Address)
			if err != nil {
				return fmt.Errorf("invalid DNS server address")
			}
			if (w.AddressFamily == "ipv4" && !address.Is4()) || (w.AddressFamily == "ipv6" && !address.Is6()) {
				return fmt.Errorf("DNS server address does not match address family")
			}
		}
	}
	if w.Attempts < 0 || w.Successes < 0 || w.Failures < 0 || w.Successes+w.Failures > w.Attempts || w.MatchedBytes < 0 {
		return fmt.Errorf("invalid workload aggregates")
	}
	return nil
}
