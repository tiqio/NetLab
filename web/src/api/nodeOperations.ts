import { ApiError } from "./index";
import type { OperationTask, Problem } from "./generated";
import { randomUUID } from "@/lib/uuid";

export interface NodeInterface {
  id: string;
  node_id: string;
  slot: number;
  name: string;
  driver: string;
  mac_address: string;
  revision: number;
  desired_link_id?: string;
}

export interface PortMapping {
  id: string;
  node_id: string;
  protocol: "tcp" | "udp";
  host_address: string;
  host_port: number;
  guest_address: string;
  guest_port: number;
  observed_state: string;
}

async function decode<T>(response: Response): Promise<T> {
  if (!response.ok)
    throw new ApiError(response.status, (await response.json()) as Problem);
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

function headers(revision?: number): Record<string, string> {
  const value: Record<string, string> = {
    "Content-Type": "application/json",
    "Idempotency-Key": randomUUID(),
  };
  if (revision !== undefined) value["If-Match"] = String(revision);
  return value;
}

export const nodeOperationsApi = {
  addInterface: (nodeId: string, driver: string) =>
    fetch(`/api/v1/nodes/${nodeId}/interfaces`, {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({ driver }),
    }).then(decode<{ interface: NodeInterface; task?: OperationTask }>),
  removeInterface: (value: NodeInterface) =>
    fetch(`/api/v1/interfaces/${value.id}`, {
      method: "DELETE",
      headers: headers(value.revision),
    }).then(decode<{ task: OperationTask } | undefined>),
  guestExec: (
    nodeId: string,
    argv: string[],
    timeoutSeconds = 30,
    outputLimit = 1 << 20,
  ) =>
    fetch(`/api/v1/nodes/${nodeId}/guest-exec`, {
      method: "POST",
      headers: headers(),
      body: JSON.stringify({
        argv,
        timeout_seconds: timeoutSeconds,
        output_limit: outputLimit,
      }),
    }).then(decode<{ task: OperationTask }>),
  listMappings: (nodeId: string) =>
    fetch(`/api/v1/nodes/${nodeId}/port-mappings`).then(decode<PortMapping[]>),
  createMapping: (
    nodeId: string,
    value: Omit<PortMapping, "id" | "node_id" | "observed_state">,
  ) =>
    fetch(`/api/v1/nodes/${nodeId}/port-mappings`, {
      method: "POST",
      headers: headers(),
      body: JSON.stringify(value),
    }).then(decode<{ port_mapping: PortMapping; task: OperationTask }>),
  deleteMapping: (id: string) =>
    fetch(`/api/v1/port-mappings/${id}`, {
      method: "DELETE",
      headers: headers(),
    }).then(decode<{ task: OperationTask }>),
  updateResources: (
    nodeId: string,
    revision: number,
    cpuCount: number,
    cpuQuotaMicros: number,
    memoryMiB: number,
  ) =>
    fetch(`/api/v1/nodes/${nodeId}/resources`, {
      method: "PUT",
      headers: headers(revision),
      body: JSON.stringify({
        cpu_count: cpuCount,
        cpu_quota_micros: cpuQuotaMicros,
        memory_mib: memoryMiB,
      }),
    }).then(decode<Record<string, unknown>>),
};
