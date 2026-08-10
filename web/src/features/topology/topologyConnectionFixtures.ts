import type {
  Link,
  NetworkAttachment,
  NetworkObject,
  NetworkObjectLink,
  Node,
  NodeInterface,
  TrafficObservation,
} from "@/api/generated";

export function topologyNode(
  id: string,
  name = id,
  kind: Node["kind"] = "docker",
): Node {
  return {
    id,
    laboratory_id: "lab-1",
    name,
    kind,
    revision: 1,
    desired_state: "running",
    observed_state: "running",
    cpu_count: 1,
    memory_mib: 256,
    interface_limit: 8,
    process_limit: 128,
    config: {},
  } as Node;
}

export function topologyInterface(
  id: string,
  nodeId: string,
  name: string,
  slot = 0,
): NodeInterface {
  return {
    id,
    node_id: nodeId,
    slot,
    name,
    driver: "virtio-net-pci",
    mac_address: `02:00:00:00:${String(slot).padStart(2, "0")}:01`,
    operational_state: "up",
    revision: 1,
  };
}

export function topologyLink(
  id: string,
  endpointAId: string,
  endpointBId: string,
  observedState = "connected",
): Link {
  return {
    id,
    laboratory_id: "lab-1",
    endpoint_a_id: endpointAId,
    endpoint_b_id: endpointBId,
    revision: 1,
    desired_state: "connected",
    observed_state: observedState,
  };
}

export function topologyNetworkObject(
  id: string,
  name = id,
  kind: NetworkObject["kind"] = "bridge",
): NetworkObject {
  return {
    id,
    laboratory_id: "lab-1",
    name,
    kind,
    revision: 1,
    desired_state: "running",
    observed_state: "running",
    config: {},
  };
}

export function topologyAttachment(
  id: string,
  networkObjectId: string,
  interfaceId: string,
  portName = "port0",
  observedState = "connected",
): NetworkAttachment {
  return {
    id,
    network_object_id: networkObjectId,
    interface_id: interfaceId,
    port_name: portName,
    revision: 1,
    observed_state: observedState,
  };
}

export function topologyObjectLink(
  id: string,
  objectAId: string,
  portAName: string,
  objectBId: string,
  portBName: string,
  observedState = "connected",
): NetworkObjectLink {
  return {
    id,
    laboratory_id: "lab-1",
    object_a_id: objectAId,
    port_a_name: portAName,
    object_b_id: objectBId,
    port_b_name: portBName,
    revision: 1,
    desired_state: "connected",
    observed_state: observedState,
  };
}

export function topologyObservation(
  connectionId: string,
  direction: TrafficObservation["direction"] = "a_to_b",
): TrafficObservation {
  const now = new Date().toISOString();
  return {
    fingerprint: `fixture:${connectionId}:${direction}`,
    resource_type: "link",
    resource_id: connectionId,
    interface_id: "",
    link_id: connectionId,
    direction,
    first_seen: now,
    last_seen: now,
    count: 1,
    bytes: 64,
  };
}
