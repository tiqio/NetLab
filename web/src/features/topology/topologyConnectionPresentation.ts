import type {
  Link,
  NetworkAttachment,
  NetworkObject,
  NetworkObjectLink,
  Node,
  NodeInterface,
} from "@/api";
import type {
  ConnectionEndpointPresentation,
  ConnectionPresentation,
  ConnectionStatusVisual,
} from "./interactionTypes";
import { assignParallelRoutes } from "./topologyParallelRoutes";
import { connectionSemanticMarkers } from "./topologyConnectionSemantics";
import { connectionVisualSemantic } from "./topologyVisualSemantics";
import { connectionDisplayName } from "./linkPresentation";
import { networkAttachmentPortLabel } from "./networkAttachmentPresentation";

export interface ConnectionPresentationInput {
  nodes: Node[];
  interfaces: NodeInterface[];
  networkObjects: NetworkObject[];
  links: Link[];
  networkAttachments: NetworkAttachment[];
  networkObjectLinks: NetworkObjectLink[];
}

export function connectionStatusVisual(
  actualState: string,
): ConnectionStatusVisual {
  return connectionVisualSemantic(actualState);
}

function endpoint(
  resourceId: string,
  resourceType: "node" | "network_object",
  resourceKind: string,
  resourceName: string,
  portId: string,
  portName: string,
): ConnectionEndpointPresentation {
  return {
    resourceId,
    resourceType,
    resourceKind,
    resourceName,
    portId,
    portName,
    endpointKey: `${resourceId}/${portId}`,
    symbolRole:
      resourceKind === "nat_bridge"
        ? "nat"
        : ["bridge", "switch_l2"].includes(resourceKind)
          ? "shared-domain"
          : resourceKind === "switch_l3"
            ? "router"
            : undefined,
  };
}

function present(
  id: string,
  persistedKind: ConnectionPresentation["persistedKind"],
  source: ConnectionEndpointPresentation,
  target: ConnectionEndpointPresentation,
  desiredState: string,
  actualState: string,
): ConnectionPresentation {
  const statusVisual = connectionStatusVisual(actualState);
  const label =
    persistedKind === "network_attachment"
      ? `${source.portName} ↔ ${target.portName === "接入口" ? target.portName : `端口 ${target.portName}`}`
      : connectionDisplayName(source, target);
  const accessibilityLabel = `${source.resourceName}:${source.portName} ↔ ${target.resourceName}:${target.portName} · ${statusVisual.label}`;
  return {
    id,
    persistedKind,
    source,
    target,
    desiredState,
    actualState,
    statusVisual,
    semanticMarkers: connectionSemanticMarkers(
      source.resourceKind,
      target.resourceKind,
    ),
    routeGroupKey: "",
    routeIndex: 0,
    routeCount: 1,
    label,
    capabilities: {
      selectable: true,
      deletable: true,
      capturable: true,
      trafficFilterable: true,
    },
    accessibilityLabel,
  };
}

export function buildConnectionPresentations(
  input: ConnectionPresentationInput,
) {
  const nodes = new Map(input.nodes.map((value) => [value.id, value]));
  const interfaces = new Map(
    input.interfaces.map((value) => [value.id, value]),
  );
  const objects = new Map(
    input.networkObjects.map((value) => [value.id, value]),
  );
  const values: ConnectionPresentation[] = [];

  for (const link of input.links) {
    const interfaceA = interfaces.get(link.endpoint_a_id);
    const interfaceB = interfaces.get(link.endpoint_b_id);
    if (!interfaceA || !interfaceB) continue;
    const nodeA = nodes.get(interfaceA.node_id);
    const nodeB = nodes.get(interfaceB.node_id);
    if (!nodeA || !nodeB) continue;
    values.push(
      present(
        link.id,
        "node_link",
        endpoint(
          nodeA.id,
          "node",
          nodeA.kind,
          nodeA.name,
          interfaceA.id,
          interfaceA.name,
        ),
        endpoint(
          nodeB.id,
          "node",
          nodeB.kind,
          nodeB.name,
          interfaceB.id,
          interfaceB.name,
        ),
        link.desired_state,
        link.observed_state,
      ),
    );
  }
  for (const attachment of input.networkAttachments) {
    const interfaceValue = interfaces.get(attachment.interface_id);
    const node = interfaceValue ? nodes.get(interfaceValue.node_id) : undefined;
    const object = objects.get(attachment.network_object_id);
    if (!interfaceValue || !node || !object) continue;
    values.push(
      present(
        attachment.id,
        "network_attachment",
        endpoint(
          node.id,
          "node",
          node.kind,
          node.name,
          interfaceValue.id,
          interfaceValue.name,
        ),
        endpoint(
          object.id,
          "network_object",
          object.kind,
          object.name,
          `${object.id}/${attachment.port_name || "port"}`,
          networkAttachmentPortLabel(attachment.port_name),
        ),
        "connected",
        attachment.observed_state,
      ),
    );
  }
  for (const link of input.networkObjectLinks) {
    const objectA = objects.get(link.object_a_id);
    const objectB = objects.get(link.object_b_id);
    if (!objectA || !objectB) continue;
    values.push(
      present(
        link.id,
        "network_object_link",
        endpoint(
          objectA.id,
          "network_object",
          objectA.kind,
          objectA.name,
          `${objectA.id}/${link.port_a_name}`,
          link.port_a_name,
        ),
        endpoint(
          objectB.id,
          "network_object",
          objectB.kind,
          objectB.name,
          `${objectB.id}/${link.port_b_name}`,
          link.port_b_name,
        ),
        link.desired_state,
        link.observed_state,
      ),
    );
  }
  return assignParallelRoutes(values);
}
