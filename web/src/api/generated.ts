export type RecoveryPolicy = "auto_restore" | "remain_stopped";
export type DesiredState = "stopped" | "running" | "deleted";
export type ObservedState =
  | "unknown"
  | "provisioning"
  | "starting"
  | "running"
  | "stopping"
  | "stopped"
  | "failed"
  | "deleting";
export type TaskState =
  "queued" | "running" | "cancelling" | "succeeded" | "failed" | "cancelled";

export interface Problem {
  code: string;
  message: string;
  retryable?: boolean;
  resource_type?: string;
  resource_id?: string;
  task_id?: string;
  phase?: string;
  cleanup?: string;
  operator_hint?: string;
  retry_after_seconds?: number;
  details?: Record<string, unknown>;
}

export interface RuntimeOwnershipRecord {
  resource_type: string;
  resource_id: string;
  object_kind: string;
  object_name: string;
  metadata?: Record<string, string>;
  cleanup_state: string;
}

export interface Laboratory {
  id: string;
  name: string;
  description: string;
  revision: number;
  recovery_policy: RecoveryPolicy;
  lifecycle_state: string;
}

export interface Node {
  id: string;
  laboratory_id: string;
  name: string;
  kind: "qemu" | "docker" | "pc" | "switch_l2" | "switch_l3";
  template_version_id?: string;
  revision: number;
  desired_state: DesiredState;
  observed_state: ObservedState;
  cpu_count: number;
  cpu_quota_micros: number;
  memory_mib: number;
  storage_gib: number;
  interface_limit: number;
  process_limit: number;
  config?: Record<string, unknown>;
  last_error?: Problem;
}

export interface NodeInterface {
  id: string;
  node_id: string;
  slot: number;
  name: string;
  driver: string;
  mac_address: string;
  desired_link_id?: string;
  operational_state: string;
  revision: number;
}

export interface Link {
  id: string;
  laboratory_id: string;
  endpoint_a_id: string;
  endpoint_b_id: string;
  revision: number;
  desired_state: string;
  observed_state: string;
}

export interface NetworkObject {
  id: string;
  laboratory_id: string;
  name: string;
  kind: "bridge" | "nat_bridge" | "pc" | "switch_l2" | "switch_l3";
  revision: number;
  desired_state: string;
  observed_state: string;
  config: Record<string, unknown>;
}

export interface NetworkObjectLink {
  id: string;
  laboratory_id: string;
  object_a_id: string;
  port_a_name: string;
  object_b_id: string;
  port_b_name: string;
  revision: number;
  desired_state: string;
  observed_state: string;
  last_error?: Problem;
}

export interface NetworkObjectLinkTaskEnvelope {
  network_object_link: NetworkObjectLink;
  task: OperationTask;
}

export interface TopologyPlacement {
  laboratory_id: string;
  resource_id: string;
  resource_type: "node" | "network_object";
  x: number;
  y: number;
  revision: number;
}

export interface TopologyPlacementUpdate {
  resource_id: string;
  resource_type: "node" | "network_object";
  x: number;
  y: number;
  revision?: number;
}

export interface TopologyPlacementResult {
  laboratory_revision: number;
  placements: TopologyPlacement[];
}

export interface TopologySnapshot {
  laboratory: Laboratory;
  nodes: Node[];
  interfaces: NodeInterface[];
  links: Link[];
  network_objects: NetworkObject[];
  network_attachments?: NetworkAttachment[];
  network_object_links?: NetworkObjectLink[];
  placements: TopologyPlacement[];
  event_sequence: number;
}

export interface NetworkAttachment {
  id: string;
  network_object_id: string;
  interface_id: string;
  port_name: string;
  config?: Record<string, unknown>;
  observed_state: string;
  last_error?: Problem;
}

export interface TemplateVersion {
  id: string;
  template_id: string;
  version: string;
  manifest_version: number;
  image_version_id?: string;
  compatible_image_version_ids?: string[];
  defaults: {
    cpu_count: number;
    cpu_quota_micros?: number;
    memory_mib: number;
    disk_gib?: number;
    interfaces: number;
    interface_name_format: string;
  };
  capabilities: string[];
  supported_nic_drivers: string[];
  console_modes: Array<"ssh" | "telnet" | "vnc">;
  runtime_options: Record<string, unknown>;
  enabled: boolean;
  readiness?: {
    status:
      | "unavailable"
      | "mechanics_validated"
      | "genuine_validated"
      | "blocked"
      | "accepted_exception";
    genuine_workload: boolean;
    checks?: Record<string, unknown>;
    exception_id?: string;
  };
  created_at: string;
}

export interface RuntimeCapabilityObservation {
  node_id: string;
  capability:
    | "image"
    | "bootstrap"
    | "qmp"
    | "qga"
    | "serial"
    | "vnc"
    | "hotplug"
    | "guest_exec"
    | "port_mapping";
  revision: number;
  state: "unknown" | "probing" | "ready" | "unavailable" | "failed";
  required: boolean;
  details?: Record<string, unknown>;
  problem?: Problem;
  observed_at: string;
}

export interface DeviceTemplate {
  id: string;
  template_key: string;
  display_name: string;
  runtime_kind: "qemu" | "docker";
  versions: TemplateVersion[];
  created_at: string;
}

export interface ImageVersion {
  id: string;
  name: string;
  version: string;
  runtime_kind: "qemu" | "docker";
  digest: string;
  source_type: string;
  source_reference: string;
  format: string;
  size_bytes: number;
  availability: string;
  license_status: string;
  license_notes: string;
  validation_result: Record<string, unknown>;
  created_at: string;
}

export interface OperationTask {
  id: string;
  kind: string;
  resource_type: string;
  resource_id: string;
  requested_revision?: number;
  state: TaskState;
  progress_current: number;
  progress_total: number;
  result?: Record<string, unknown>;
  error?: Problem;
  created_at: string;
  started_at?: string;
  finished_at?: string;
}

export interface TaskEnvelope {
  task: OperationTask;
  node?: Node;
  link?: Link;
  interface?: NodeInterface;
  port_mapping?: PortMapping;
  network_object?: NetworkObject;
  network_object_link?: NetworkObjectLink;
  traffic_filter?: TrafficFilter;
  capture?: CaptureSession;
  artifact?: Artifact;
}

export interface Artifact {
  id: string;
  kind: string;
  media_type: string;
  size_bytes: number;
  sha256: string;
  owner_type: string;
  owner_id: string;
  created_at: string;
  expires_at?: string;
}

export interface AuditEvent {
  id: string;
  actor_class: string;
  action: string;
  resource_type: string;
  resource_id: string;
  task_id?: string;
  outcome: string;
  correlation_id: string;
  details?: Record<string, unknown>;
  occurred_at: string;
}

export interface PortMapping {
  id: string;
  node_id: string;
  protocol: "tcp" | "udp";
  host_address: string;
  host_port: number;
  guest_address: string;
  guest_port: number;
  revision: number;
  observed_state: string;
  last_error?: Problem;
}

export interface ConsoleDescriptor {
  mode: "ssh" | "telnet" | "vnc";
  stream_url: string;
  idle_seconds: number;
  reconnectable?: boolean;
  subprotocol?: string;
}

export interface CaptureSession {
  id: string;
  laboratory_id?: string;
  source_type: "interface" | "link" | "network_object_link";
  source_id: string;
  purpose?: string;
  parent_resource_id?: string;
  filter?: string;
  format: "pcap" | "pcapng";
  state: string;
  retain: boolean;
  max_bytes: number;
  bytes_written: number;
  packets: number;
  truncated: boolean;
  artifact_id?: string;
  artifact_url?: string;
  expires_at?: string;
  completion_reason?: string;
  created_at: string;
  started_at?: string;
  finished_at?: string;
  last_error?: Problem;
}

export interface TrafficObservation {
  fingerprint: string;
  resource_type?: "interface" | "link" | "network_object_link" | string;
  resource_id?: string;
  interface_id: string;
  link_id?: string;
  network_object_link_id?: string;
  direction: "ingress" | "egress" | "a_to_b" | "b_to_a" | "ambiguous" | string;
  source_address?: string;
  destination_address?: string;
  source_mac?: string;
  destination_mac?: string;
  packet_role?: "request" | "reply" | string;
  first_seen: string;
  last_seen: string;
  count: number;
  bytes: number;
}

export interface TrafficFilter {
  id: string;
  laboratory_id: string;
  expression: string;
  color: string;
  state: string;
  max_observations: number;
  interface_ids?: string[];
  link_ids?: string[];
  network_object_link_ids?: string[];
  observations: TrafficObservation[];
  created_at: string;
  finished_at?: string;
  last_error?: Problem;
}

export interface StateEvent {
  sequence: number;
  type: string;
  laboratory_id?: string;
  resource_type: string;
  resource_id: string;
  revision: number;
  task_id?: string;
  data?: Record<string, unknown>;
  occurred_at?: string;
}

export interface CreateNodeRequest {
  name: string;
  kind?: Node["kind"];
  template_version_id?: string;
  image_version_id?: string;
  cpu_count?: number;
  cpu_quota_micros?: number;
  memory_mib?: number;
  storage_gib?: number;
  interface_limit?: number;
  process_limit?: number;
  interface_count?: number;
  config?: Record<string, unknown>;
  bootstrap?: {
    user_data?: string;
    meta_data?: string;
    network_config?: string;
  };
}

export interface DockerStaticRoute {
  destination: string;
  gateway?: string;
  metric?: number;
}

export interface NodeNetworkInterfaceSettings {
  id: string;
  name: string;
  driver: string;
  modes: string[];
  addresses: string[];
  routes: DockerStaticRoute[];
}

export interface UpdateNodeSettingsRequest {
  name: string;
  cpu_count: number;
  cpu_quota_micros: number;
  memory_mib: number;
  interface_limit: number;
  process_limit: number;
  network_interfaces?: NodeNetworkInterfaceSettings[];
}

export interface StartCaptureRequest {
  laboratory_id?: string;
  source_type: "interface" | "link" | "network_object_link";
  source_id: string;
  interface?: string;
  filter?: string;
  format?: "pcap" | "pcapng";
  retain?: boolean;
  duration_seconds?: number;
  max_bytes?: number;
}

export interface StartTrafficFilterRequest {
  laboratory_id: string;
  match: Record<string, unknown>;
  max_observations?: number;
  interface_ids?: string[];
  link_ids?: string[];
  network_object_link_ids?: string[];
  color?: string;
}
