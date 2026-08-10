import type { UnifiedConnectionEndpoint } from "./topologyEndpointCompatibility";

export interface ConnectionSourceNode {
  id: string;
  name: string;
  kind: string;
}

export interface ConnectionSourceInterface {
  id: string;
  node_id: string;
  name: string;
  desired_link_id?: string;
}

export interface ConnectionSourceObject {
  id: string;
  name: string;
  kind: "bridge" | "nat_bridge" | "pc" | "switch_l2" | "switch_l3";
  config: Record<string, unknown>;
}

export interface ConnectionSourceContext {
  laboratoryId: string;
  nodes: ConnectionSourceNode[];
  interfaces: ConnectionSourceInterface[];
  networkObjects: ConnectionSourceObject[];
  occupiedObjectPorts: ReadonlySet<string>;
}

export function resolveConnectionSourceCandidates(
  resourceId: string,
  context: ConnectionSourceContext,
): UnifiedConnectionEndpoint[] {
  const node = context.nodes.find((item) => item.id === resourceId);
  if (node)
    return context.interfaces
      .filter(
        (item) =>
          item.node_id === resourceId &&
          !item.name.startsWith("internal") &&
          !item.desired_link_id,
      )
      .map((item) => ({
        kind: "node_interface" as const,
        laboratoryId: context.laboratoryId,
        resourceId: node.id,
        resourceKind: node.kind,
        portId: item.id,
        portName: item.name,
        displayName: `${node.name}:${item.name}`,
        capabilities: [],
        availability: "free" as const,
      }));

  const object = context.networkObjects.find((item) => item.id === resourceId);
  if (!object) return [];
  const rows =
    object.kind === "switch_l2"
      ? object.config.ports
      : object.kind === "switch_l3" || object.kind === "pc"
        ? object.config.interfaces
        : undefined;
  if (Array.isArray(rows))
    return rows.flatMap((item) => {
      const portName = String((item as { name?: string }).name || "");
      if (
        !portName ||
        context.occupiedObjectPorts.has(`${object.id}:${portName}`)
      )
        return [];
      return [
        {
          kind: "network_object_port" as const,
          laboratoryId: context.laboratoryId,
          resourceId: object.id,
          resourceKind: object.kind,
          portName,
          displayName: `${object.name}:${portName}`,
          capabilities: [],
          availability: "free" as const,
        },
      ];
    });
  if (object.kind !== "bridge" && object.kind !== "nat_bridge") return [];
  return [
    {
      kind: "network_object_access",
      laboratoryId: context.laboratoryId,
      resourceId: object.id,
      resourceKind: object.kind,
      displayName: object.name,
      capabilities: ["multi_access"],
      availability: "free",
    },
  ];
}
