import type {
  AuditEvent,
  CaptureSession,
  ConsoleDescriptor,
  CreateNodeResult,
  CreateNodeRequest,
  CreateNetworkObjectResult,
  DeviceTemplate,
  ImageVersion,
  Laboratory,
  Link,
  NetworkObject,
  NetworkObjectLink,
  NetworkObjectLinkTaskEnvelope,
  Node,
  NodeInterface,
  OperationTask,
  PortMapping,
  Problem,
  RuntimeOwnershipRecord,
  RuntimeCapabilityObservation,
  StartCaptureRequest,
  StartTrafficFilterRequest,
  TaskEnvelope,
  TopologySnapshot,
  TopologyPlacementResult,
  TopologyPlacementUpdate,
  TrafficFilter,
  UpdateNodeSettingsRequest,
} from "./generated";
import { randomUUID } from "@/lib/uuid";

export * from "./generated";

export class ApiError extends Error {
  constructor(
    public status: number,
    public problem: Problem,
  ) {
    super(problem.message);
  }
}

export interface RuijieConfigRequest {
  operation:
    "create_vlan" | "l2_access" | "l2_trunk" | "l3_address" | "admin_state";
  interface?: string;
  vlan_id?: number;
  vlan_name?: string;
  allowed_vlans?: string;
  address_cidr?: string;
  admin_up?: boolean;
  save?: boolean;
}

async function decode<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const problem = (await response.json().catch(() => ({
      code: "http_error",
      message: response.statusText,
    }))) as Problem;
    throw new ApiError(response.status, problem);
  }
  if (response.status === 204 || response.headers.get("content-length") === "0")
    return undefined as T;
  return response.json() as Promise<T>;
}

interface RequestOptions {
  body?: unknown;
  revision?: number;
  idempotencyKey?: string;
  contentType?: string;
  timeoutMs?: number;
}

function request<T>(
  path: string,
  method = "GET",
  options: RequestOptions = {},
): Promise<T> {
  const headers: Record<string, string> = {};
  if (options.body !== undefined)
    headers["Content-Type"] = options.contentType || "application/json";
  if (options.revision !== undefined)
    headers["If-Match"] = String(options.revision);
  if (!["GET", "HEAD"].includes(method))
    headers["Idempotency-Key"] = options.idempotencyKey || randomUUID();
  const controller = options.timeoutMs ? new AbortController() : undefined;
  const timeout = options.timeoutMs
    ? window.setTimeout(() => controller?.abort(), options.timeoutMs)
    : undefined;
  return fetch(`/api/v1${path}`, {
    method,
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    signal: controller?.signal,
  })
    .then(decode<T>)
    .catch((error: unknown) => {
      if (controller?.signal.aborted)
        throw new ApiError(504, {
          code: "request_timeout",
          message: `请求在 ${Math.round((options.timeoutMs || 0) / 1000)} 秒内未返回，请刷新映射列表确认最终状态。`,
          retryable: true,
        });
      throw error;
    })
    .finally(() => {
      if (timeout !== undefined) window.clearTimeout(timeout);
    });
}

export const generatedApi = {
  getCapabilities: () => request<Record<string, unknown>>("/capabilities"),
  listRuntimeOwnership: () =>
    request<RuntimeOwnershipRecord[]>("/runtime-ownership"),
  getNodeCapabilities: (nodeId: string) =>
    request<{ node_id: string; observations: RuntimeCapabilityObservation[] }>(
      `/nodes/${nodeId}/capabilities`,
    ),
  listTemplates: () => request<DeviceTemplate[]>("/templates"),
  listImages: () => request<ImageVersion[]>("/images"),
  importImage: (body: Record<string, unknown>, idempotencyKey?: string) =>
    request<ImageVersion>("/images", "POST", { body, idempotencyKey }),
  setImageConsoleCredentials: (
    imageId: string,
    body: { username: string; password: string },
  ) => request<void>(`/images/${imageId}/console-credentials`, "PUT", { body }),
  listLabs: () => request<Laboratory[]>("/labs"),
  createLab: (
    body: Pick<Laboratory, "name"> & Partial<Laboratory>,
    idempotencyKey?: string,
  ) => request<Laboratory>("/labs", "POST", { body, idempotencyKey }),
  getLab: (id: string) => request<TopologySnapshot>(`/labs/${id}`),
  updateTopologyPlacements: (
    laboratoryId: string,
    revision: number,
    placements: TopologyPlacementUpdate[],
    idempotencyKey?: string,
  ) =>
    request<TopologyPlacementResult>(
      `/labs/${laboratoryId}/placements`,
      "PUT",
      { revision, body: { placements }, idempotencyKey },
    ),
  updateLab: (lab: Laboratory) =>
    request<Laboratory>(`/labs/${lab.id}`, "PATCH", {
      body: lab,
      revision: lab.revision,
      contentType: "application/merge-patch+json",
    }),
  deleteLab: (lab: Pick<Laboratory, "id" | "revision">) =>
    request<TaskEnvelope>(`/labs/${lab.id}`, "DELETE", {
      revision: lab.revision,
    }),
  createNode: (
    labId: string,
    revision: number,
    body: CreateNodeRequest,
    idempotencyKey?: string,
  ) =>
    request<CreateNodeResult>(
      `/labs/${labId}/nodes`,
      "POST",
      { body, revision, idempotencyKey },
    ),
  getNode: (nodeId: string) => request<Node>(`/nodes/${nodeId}`),
  getNodeBootstrapCredentials: (nodeId: string) =>
    request<{ username: string; password: string; source: string }>(
      `/nodes/${nodeId}/bootstrap-credentials`,
    ),
  configureRuijie: (nodeId: string, body: RuijieConfigRequest) =>
    request<{ commands: string[]; console_mode: "telnet"; verified: boolean }>(
      `/nodes/${nodeId}/ruijie/configure`,
      "POST",
      { body },
    ),
  deleteNode: (node: Pick<Node, "id" | "revision">) =>
    request<TaskEnvelope>(`/nodes/${node.id}`, "DELETE", {
      revision: node.revision,
    }),
  setNodeState: (node: Pick<Node, "id" | "revision">, desiredState: string) =>
    request<TaskEnvelope>(`/nodes/${node.id}/state`, "PUT", {
      body: { desired_state: desiredState },
      revision: node.revision,
    }),
  addInterface: (nodeId: string, driver: string) =>
    request<NodeInterface | { interface: NodeInterface; task: OperationTask }>(
      `/nodes/${nodeId}/interfaces`,
      "POST",
      { body: { driver } },
    ),
  removeInterface: (interfaceId: string, revision: number) =>
    request<OperationTask | void>(`/interfaces/${interfaceId}`, "DELETE", {
      revision,
    }),
  connectLink: (labId: string, endpointAId: string, endpointBId: string) =>
    request<TaskEnvelope>(`/labs/${labId}/links`, "POST", {
      body: { endpoint_a_id: endpointAId, endpoint_b_id: endpointBId },
    }),
  disconnectLink: (linkId: string) =>
    request<TaskEnvelope>(`/links/${linkId}`, "DELETE"),
  reconnectLink: (
    link: Pick<Link, "id" | "revision">,
    retainedEndpointId: string,
    replacementEndpointId: string,
    idempotencyKey?: string,
  ) =>
    request<TaskEnvelope>(`/links/${link.id}/reconnect`, "POST", {
      revision: link.revision,
      idempotencyKey,
      body: {
        retained_endpoint_id: retainedEndpointId,
        replacement_endpoint_id: replacementEndpointId,
      },
    }),
  executeGuestCommand: (
    nodeId: string,
    body: { argv: string[]; timeout_seconds?: number; output_limit?: number },
  ) =>
    request<TaskEnvelope>(`/nodes/${nodeId}/guest-exec`, "POST", { body }).then(
      (value) => value.task,
    ),
  listNodePortMappings: (nodeId: string) =>
    request<PortMapping[]>(`/nodes/${nodeId}/port-mappings`),
  createPortMapping: (
    nodeId: string,
    body: {
      protocol: "tcp" | "udp";
      host_address?: string;
      host_port?: number;
      guest_address?: string;
      guest_port: number;
    },
  ) =>
    request<{ port_mapping: PortMapping; task: OperationTask }>(
      `/nodes/${nodeId}/port-mappings`,
      "POST",
      { body, timeoutMs: 15_000 },
    ),
  deletePortMapping: (mappingId: string) =>
    request<TaskEnvelope>(`/port-mappings/${mappingId}`, "DELETE").then(
      (value) => value.task,
    ),
  listNodeConsoles: (nodeId: string) =>
    request<ConsoleDescriptor[]>(`/nodes/${nodeId}/consoles`),
  listNetworkObjectConsoles: (objectId: string) =>
    request<ConsoleDescriptor[]>(`/network-objects/${objectId}/consoles`),
  streamNodeConsole: (nodeId: string, mode: "ssh" | "telnet" | "vnc") =>
    `/api/v1/nodes/${nodeId}/consoles/${mode}/stream`,
  streamNetworkObjectConsole: (
    objectId: string,
    mode: "ssh" | "telnet" | "vnc",
  ) => `/api/v1/network-objects/${objectId}/consoles/${mode}/stream`,
  closeNodeConsoleSession: (
    nodeId: string,
    mode: "ssh" | "telnet" | "vnc",
    sessionId: string,
  ) =>
    request<void>(
      `/nodes/${nodeId}/consoles/${mode}/sessions/${encodeURIComponent(sessionId)}`,
      "DELETE",
    ),
  closeNetworkObjectConsoleSession: (
    objectId: string,
    mode: "ssh" | "telnet" | "vnc",
    sessionId: string,
  ) =>
    request<void>(
      `/network-objects/${objectId}/consoles/${mode}/sessions/${encodeURIComponent(sessionId)}`,
      "DELETE",
    ),
  startCapture: (body: StartCaptureRequest) =>
    request<
      TaskEnvelope & {
        capture: CaptureSession;
        stream_url: string;
        wireshark: { mode: string; media_type: string };
      }
    >("/captures", "POST", { body }),
  listCaptures: (laboratoryId?: string) =>
    request<CaptureSession[]>(
      `/captures${laboratoryId ? `?laboratory_id=${encodeURIComponent(laboratoryId)}` : ""}`,
    ),
  getCapture: (captureId: string) =>
    request<CaptureSession>(`/captures/${captureId}`),
  stopCapture: (captureId: string) =>
    request<TaskEnvelope>(`/captures/${captureId}`, "DELETE"),
  streamCapture: (captureId: string) => `/api/v1/captures/${captureId}/stream`,
  downloadWiresharkHelper: (
    platform: "linux-amd64" | "windows-amd64" | "darwin-amd64" | "darwin-arm64",
  ) => `/api/v1/client-tools/wireshark-helper/${platform}`,
  startTrafficFilter: (body: StartTrafficFilterRequest) =>
    request<TaskEnvelope & { traffic_filter: TrafficFilter }>(
      "/traffic-filters",
      "POST",
      { body },
    ),
  listTrafficFilters: (laboratoryId?: string) =>
    request<Array<{ traffic_filter: TrafficFilter; ambiguous: boolean }>>(
      `/traffic-filters${laboratoryId ? `?laboratory_id=${encodeURIComponent(laboratoryId)}` : ""}`,
    ),
  getTrafficFilter: (filterId: string) =>
    request<{ traffic_filter: TrafficFilter; ambiguous: boolean }>(
      `/traffic-filters/${filterId}`,
    ),
  stopTrafficFilter: (filterId: string) =>
    request<TaskEnvelope>(`/traffic-filters/${filterId}`, "DELETE"),
  deleteTrafficFilterHistory: (filterId: string) =>
    request<{ traffic_filter: TrafficFilter }>(
      `/traffic-filters/${filterId}/history`,
      "DELETE",
    ),
  listTasks: (limit = 100) =>
    request<OperationTask[]>(`/tasks?limit=${encodeURIComponent(limit)}`),
  getTask: (taskId: string) => request<OperationTask>(`/tasks/${taskId}`),
  cancelTask: (taskId: string) =>
    request<OperationTask>(`/tasks/${taskId}/cancel`, "POST"),
  listAuditEvents: (limit = 100) =>
    request<AuditEvent[]>(`/audit-events?limit=${encodeURIComponent(limit)}`),
  listAudit: (limit = 100) =>
    request<AuditEvent[]>(`/audit-events?limit=${encodeURIComponent(limit)}`),
  exportLab: (labId: string) =>
    request<TaskEnvelope>(`/labs/${labId}/exports`, "POST"),
  importLab: (bundle: unknown) =>
    request<TaskEnvelope>("/lab-imports", "POST", { body: bundle }),
  duplicateLab: (labId: string, name?: string) =>
    request<TaskEnvelope>(`/labs/${labId}/duplicate`, "POST", {
      body: { name },
    }),
  downloadArtifact: (artifactId: string) => `/api/v1/artifacts/${artifactId}`,
  getNodeResources: (nodeId: string) =>
    request<Record<string, unknown>>(`/nodes/${nodeId}/resources`),
  updateNodeResources: (
    node: Pick<Node, "id" | "revision">,
    body: Pick<Node, "cpu_count" | "cpu_quota_micros" | "memory_mib">,
  ) =>
    request<Node>(`/nodes/${node.id}/resources`, "PUT", {
      body,
      revision: node.revision,
    }),
  updateNodeSettings: (
    node: Pick<Node, "id" | "revision">,
    body: UpdateNodeSettingsRequest,
  ) =>
    request<Node>(`/nodes/${node.id}/settings`, "PUT", {
      body,
      revision: node.revision,
    }),
  listNetworkObjects: (labId: string) =>
    request<NetworkObject[]>(`/labs/${labId}/network-objects`),
  createNetworkObject: (
    labId: string,
    revision: number,
    body: Pick<NetworkObject, "name" | "kind"> & {
      config?: Record<string, unknown>;
      placement_intent?: import("./generated").PlacementIntent;
    },
    idempotencyKey?: string,
  ) =>
    request<CreateNetworkObjectResult>(`/labs/${labId}/network-objects`, "POST", {
      body,
      revision,
      idempotencyKey,
    }),
  getNetworkObject: (objectId: string) =>
    request<NetworkObject>(`/network-objects/${objectId}`),
  updateNetworkObject: (
    value: Pick<NetworkObject, "id" | "revision">,
    body: Pick<NetworkObject, "name" | "config">,
  ) =>
    request<TaskEnvelope>(`/network-objects/${value.id}`, "PATCH", {
      body,
      revision: value.revision,
    }),
  deleteNetworkObject: (value: Pick<NetworkObject, "id" | "revision">) =>
    request<TaskEnvelope>(`/network-objects/${value.id}`, "DELETE", {
      revision: value.revision,
    }),
  attachNetworkObject: (
    objectId: string,
    body: {
      interface_id: string;
      port_name?: string;
      config?: Record<string, unknown>;
    },
  ) =>
    request<void>(`/network-objects/${objectId}/attachments`, "POST", {
      body,
    }),
  listNetworkObjectLinks: (labId: string) =>
    request<NetworkObjectLink[]>(`/labs/${labId}/network-object-links`),
  createNetworkObjectLink: (
    labId: string,
    body: {
      object_a_id: string;
      port_a_name: string;
      object_b_id: string;
      port_b_name: string;
    },
  ) =>
    request<NetworkObjectLinkTaskEnvelope>(
      `/labs/${labId}/network-object-links`,
      "POST",
      { body },
    ),
  getNetworkObjectLink: (linkId: string) =>
    request<NetworkObjectLink>(`/network-object-links/${linkId}`),
  deleteNetworkObjectLink: (
    value: Pick<NetworkObjectLink, "id" | "revision">,
    idempotencyKey?: string,
  ) =>
    request<NetworkObjectLinkTaskEnvelope>(
      `/network-object-links/${value.id}`,
      "DELETE",
      { revision: value.revision, idempotencyKey },
    ),
  getNetworkObjectDiagnostics: (objectId: string) =>
    request<Record<string, unknown>>(
      `/network-objects/${objectId}/diagnostics`,
    ),
};

export const api = generatedApi;
