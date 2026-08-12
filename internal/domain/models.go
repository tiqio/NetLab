package domain

import "time"

type RecoveryPolicy string

const (
	RecoveryAutoRestore   RecoveryPolicy = "auto_restore"
	RecoveryRemainStopped RecoveryPolicy = "remain_stopped"
)

type Laboratory struct {
	ID             ID             `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Revision       Revision       `json:"revision"`
	RecoveryPolicy RecoveryPolicy `json:"recovery_policy"`
	LifecycleState string         `json:"lifecycle_state"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type Node struct {
	ID                ID             `json:"id"`
	LaboratoryID      ID             `json:"laboratory_id"`
	Name              string         `json:"name"`
	Kind              string         `json:"kind"`
	TemplateVersionID ID             `json:"template_version_id,omitempty"`
	Revision          Revision       `json:"revision"`
	DesiredState      DesiredState   `json:"desired_state"`
	ObservedState     ObservedState  `json:"observed_state"`
	CPUCount          int            `json:"cpu_count"`
	CPUQuotaMicros    int64          `json:"cpu_quota_micros"`
	MemoryMiB         int            `json:"memory_mib"`
	StorageGiB        int            `json:"storage_gib"`
	InterfaceLimit    int            `json:"interface_limit"`
	ProcessLimit      int            `json:"process_limit"`
	Config            map[string]any `json:"config,omitempty"`
	LastError         *Problem       `json:"last_error,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type Interface struct {
	ID               ID       `json:"id"`
	NodeID           ID       `json:"node_id"`
	Slot             int      `json:"slot"`
	Name             string   `json:"name"`
	Driver           string   `json:"driver,omitempty"`
	MACAddress       string   `json:"mac_address"`
	DesiredLinkID    ID       `json:"desired_link_id,omitempty"`
	OperationalState string   `json:"operational_state"`
	Revision         Revision `json:"revision"`
}

type NodeNetworkInterfaceSettings struct {
	ID        ID            `json:"id"`
	Name      string        `json:"name"`
	Driver    string        `json:"driver"`
	Modes     []string      `json:"modes"`
	Addresses []string      `json:"addresses"`
	Routes    []RouteConfig `json:"routes,omitempty"`
}

type NodeSettings struct {
	Name              string                         `json:"name"`
	CPUCount          int                            `json:"cpu_count"`
	CPUQuotaMicros    int64                          `json:"cpu_quota_micros"`
	MemoryMiB         int                            `json:"memory_mib"`
	InterfaceLimit    int                            `json:"interface_limit"`
	ProcessLimit      int                            `json:"process_limit"`
	NetworkInterfaces []NodeNetworkInterfaceSettings `json:"network_interfaces,omitempty"`
	ForwardIPv4       *bool                          `json:"forward_ipv4,omitempty"`
	ForwardIPv6       *bool                          `json:"forward_ipv6,omitempty"`
	DeviceRoles       []DeviceInterfaceRole          `json:"device_roles,omitempty"`
}

type Link struct {
	ID            ID       `json:"id"`
	LaboratoryID  ID       `json:"laboratory_id"`
	EndpointAID   ID       `json:"endpoint_a_id"`
	EndpointBID   ID       `json:"endpoint_b_id"`
	Revision      Revision `json:"revision"`
	DesiredState  string   `json:"desired_state"`
	ObservedState string   `json:"observed_state"`
}

type OperationTask struct {
	ID                 ID             `json:"id"`
	Kind               string         `json:"kind"`
	ResourceType       string         `json:"resource_type"`
	ResourceID         ID             `json:"resource_id"`
	IdempotencyKey     string         `json:"idempotency_key,omitempty"`
	RequestFingerprint string         `json:"-"`
	Input              map[string]any `json:"-"`
	RequestedRevision  Revision       `json:"requested_revision,omitempty"`
	State              TaskState      `json:"state"`
	ProgressCurrent    int            `json:"progress_current"`
	ProgressTotal      int            `json:"progress_total"`
	Result             map[string]any `json:"result,omitempty"`
	Error              *Problem       `json:"error,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	StartedAt          *time.Time     `json:"started_at,omitempty"`
	FinishedAt         *time.Time     `json:"finished_at,omitempty"`
}

type AuditEvent struct {
	ID            ID             `json:"id"`
	ActorClass    string         `json:"actor_class"`
	Action        string         `json:"action"`
	ResourceType  string         `json:"resource_type"`
	ResourceID    ID             `json:"resource_id"`
	TaskID        ID             `json:"task_id,omitempty"`
	Outcome       string         `json:"outcome"`
	CorrelationID string         `json:"correlation_id"`
	Details       map[string]any `json:"details,omitempty"`
	OccurredAt    time.Time      `json:"occurred_at"`
}

type Artifact struct {
	ID        ID         `json:"id"`
	Kind      string     `json:"kind"`
	Path      string     `json:"-"`
	MediaType string     `json:"media_type"`
	SizeBytes int64      `json:"size_bytes"`
	SHA256    string     `json:"sha256"`
	OwnerType string     `json:"owner_type"`
	OwnerID   ID         `json:"owner_id"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type OutboxEvent struct {
	Sequence     int64          `json:"sequence"`
	Type         string         `json:"type"`
	LaboratoryID ID             `json:"laboratory_id,omitempty"`
	ResourceType string         `json:"resource_type"`
	ResourceID   ID             `json:"resource_id"`
	Revision     Revision       `json:"revision"`
	TaskID       ID             `json:"task_id,omitempty"`
	Data         map[string]any `json:"data,omitempty"`
	OccurredAt   time.Time      `json:"occurred_at"`
}

type TopologySnapshot struct {
	Laboratory         Laboratory          `json:"laboratory"`
	Nodes              []Node              `json:"nodes"`
	Interfaces         []Interface         `json:"interfaces"`
	Links              []Link              `json:"links"`
	NetworkObjects     []NetworkObject     `json:"network_objects"`
	Attachments        []NetworkAttachment `json:"network_attachments"`
	NetworkObjectLinks []NetworkObjectLink `json:"network_object_links"`
	Placements         []TopologyPlacement `json:"placements"`
	TrafficWorkloads   []TrafficWorkload   `json:"traffic_workloads,omitempty"`
	Sequence           int64               `json:"event_sequence"`
}
