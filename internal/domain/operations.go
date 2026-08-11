package domain

import "time"

type IdempotencyRecord struct {
	Scope              string
	Key                string
	RequestFingerprint string
	State              string
	StatusCode         int
	Response           []byte
	Error              []byte
	CreatedAt          time.Time
	ExpiresAt          time.Time
}

type PortMapping struct {
	ID            ID        `json:"id"`
	NodeID        ID        `json:"node_id"`
	Protocol      string    `json:"protocol"`
	HostAddress   string    `json:"host_address"`
	HostPort      int       `json:"host_port"`
	GuestAddress  string    `json:"guest_address"`
	GuestPort     int       `json:"guest_port"`
	Revision      Revision  `json:"revision"`
	ObservedState string    `json:"observed_state"`
	LastError     *Problem  `json:"last_error,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type Capture struct {
	ID               ID         `json:"id"`
	LaboratoryID     ID         `json:"laboratory_id,omitempty"`
	SourceType       string     `json:"source_type"`
	SourceID         ID         `json:"source_id"`
	Purpose          string     `json:"purpose,omitempty"`
	ParentResourceID ID         `json:"parent_resource_id,omitempty"`
	Filter           string     `json:"filter,omitempty"`
	Format           string     `json:"format"`
	State            string     `json:"state"`
	Retain           bool       `json:"retain"`
	MaxBytes         int64      `json:"max_bytes"`
	BytesWritten     int64      `json:"bytes_written"`
	Packets          int64      `json:"packets"`
	Truncated        bool       `json:"truncated"`
	ArtifactID       ID         `json:"artifact_id,omitempty"`
	ArtifactURL      string     `json:"artifact_url,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	CompletionReason string     `json:"completion_reason,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
	LastError        *Problem   `json:"last_error,omitempty"`
}

type TrafficObservation struct {
	Fingerprint         string    `json:"fingerprint"`
	ResourceType        string    `json:"resource_type,omitempty"`
	ResourceID          ID        `json:"resource_id,omitempty"`
	InterfaceID         ID        `json:"interface_id"`
	LinkID              ID        `json:"link_id,omitempty"`
	NetworkObjectLinkID ID        `json:"network_object_link_id,omitempty"`
	Direction           string    `json:"direction"`
	SourceAddress       string    `json:"source_address,omitempty"`
	DestinationAddress  string    `json:"destination_address,omitempty"`
	SourceMAC           string    `json:"source_mac,omitempty"`
	DestinationMAC      string    `json:"destination_mac,omitempty"`
	PacketRole          string    `json:"packet_role,omitempty"`
	FirstSeen           time.Time `json:"first_seen"`
	LastSeen            time.Time `json:"last_seen"`
	Count               int64     `json:"count"`
	Bytes               int64     `json:"bytes"`
}

type TrafficFilter struct {
	ID                   ID                   `json:"id"`
	LaboratoryID         ID                   `json:"laboratory_id"`
	Expression           string               `json:"expression"`
	Color                string               `json:"color"`
	State                string               `json:"state"`
	MaxObservations      int                  `json:"max_observations"`
	InterfaceIDs         []ID                 `json:"interface_ids,omitempty"`
	LinkIDs              []ID                 `json:"link_ids,omitempty"`
	NetworkObjectLinkIDs []ID                 `json:"network_object_link_ids,omitempty"`
	Observations         []TrafficObservation `json:"observations"`
	FingerprintCount     int64                `json:"fingerprint_count"`
	MatchedPackets       int64                `json:"matched_packets"`
	MatchedBytes         int64                `json:"matched_bytes"`
	FirstMatchAt         *time.Time           `json:"first_match_at,omitempty"`
	LastMatchAt          *time.Time           `json:"last_match_at,omitempty"`
	CreatedAt            time.Time            `json:"created_at"`
	FinishedAt           *time.Time           `json:"finished_at,omitempty"`
	LastError            *Problem             `json:"last_error,omitempty"`
}

type NetworkObject struct {
	ID            ID             `json:"id"`
	LaboratoryID  ID             `json:"laboratory_id"`
	Name          string         `json:"name"`
	Kind          string         `json:"kind"`
	Revision      Revision       `json:"revision"`
	DesiredState  string         `json:"desired_state"`
	ObservedState string         `json:"observed_state"`
	Config        map[string]any `json:"config"`
	LastError     *Problem       `json:"last_error,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type NetworkAttachment struct {
	ID              ID             `json:"id"`
	NetworkObjectID ID             `json:"network_object_id"`
	InterfaceID     ID             `json:"interface_id"`
	PortName        string         `json:"port_name"`
	Config          map[string]any `json:"config,omitempty"`
	Revision        Revision       `json:"revision"`
	ObservedState   string         `json:"observed_state"`
	LastError       *Problem       `json:"last_error,omitempty"`
}

type NetworkObjectLink struct {
	ID            ID       `json:"id"`
	LaboratoryID  ID       `json:"laboratory_id"`
	ObjectAID     ID       `json:"object_a_id"`
	PortAName     string   `json:"port_a_name"`
	ObjectBID     ID       `json:"object_b_id"`
	PortBName     string   `json:"port_b_name"`
	Revision      Revision `json:"revision"`
	DesiredState  string   `json:"desired_state"`
	ObservedState string   `json:"observed_state"`
	LastError     *Problem `json:"last_error,omitempty"`
}
