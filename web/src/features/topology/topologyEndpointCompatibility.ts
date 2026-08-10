export type UnifiedConnectionEndpointKind =
  "node_interface" | "network_object_port" | "network_object_access";

export type UnifiedEndpointAvailability =
  | "free"
  | "reserved"
  | "occupied"
  | "reconciling"
  | "unavailable"
  | "incompatible";

export type UnifiedConnectionBackingKind =
  "link" | "network_attachment" | "network_object_link";

export interface UnifiedConnectionEndpoint {
  kind: UnifiedConnectionEndpointKind;
  laboratoryId: string;
  resourceId: string;
  resourceKind?: string;
  portId?: string;
  portName?: string;
  displayName: string;
  capabilities: string[];
  availability: UnifiedEndpointAvailability;
  unavailableReason?: string;
}

export interface EndpointCompatibility {
  compatible: boolean;
  backingKind?: UnifiedConnectionBackingKind;
  reason?: string;
}

export function endpointKey(endpoint: UnifiedConnectionEndpoint) {
  if (endpoint.kind === "node_interface")
    return `${endpoint.kind}:${endpoint.portId || ""}`;
  if (endpoint.kind === "network_object_port")
    return `${endpoint.kind}:${endpoint.resourceId}:${endpoint.portName || ""}`;
  return `${endpoint.kind}:${endpoint.resourceId}`;
}

export function connectionBackingKind(
  source: UnifiedConnectionEndpoint,
  target: UnifiedConnectionEndpoint,
): UnifiedConnectionBackingKind | undefined {
  const sourceNode = source.kind === "node_interface";
  const targetNode = target.kind === "node_interface";
  const sourcePort = source.kind === "network_object_port";
  const targetPort = target.kind === "network_object_port";
  const sourceAccess = source.kind === "network_object_access";
  const targetAccess = target.kind === "network_object_access";
  if (sourceNode && targetNode) return "link";
  if (
    (sourceNode && (targetPort || targetAccess)) ||
    (targetNode && (sourcePort || sourceAccess))
  )
    return "network_attachment";
  if (sourcePort && targetPort) return "network_object_link";
  return undefined;
}

export function endpointsCompatible(
  source: UnifiedConnectionEndpoint,
  target: UnifiedConnectionEndpoint,
): EndpointCompatibility {
  if (source.laboratoryId !== target.laboratoryId)
    return { compatible: false, reason: "cross_laboratory_connection" };
  if (
    endpointKey(source) === endpointKey(target) ||
    source.resourceId === target.resourceId
  )
    return { compatible: false, reason: "invalid_topology" };
  if (source.availability !== "free" || target.availability !== "free")
    return { compatible: false, reason: "endpoint_occupied" };
  const backingKind = connectionBackingKind(source, target);
  return backingKind
    ? { compatible: true, backingKind }
    : { compatible: false, reason: "endpoint_incompatible" };
}
