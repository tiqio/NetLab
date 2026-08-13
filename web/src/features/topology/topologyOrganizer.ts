import type {
  Link,
  NetworkAttachment,
  NetworkObject,
  NetworkObjectLink,
  Node,
  NodeInterface,
} from "@/api";
import type { Point } from "@/types/workspace";

export interface TopologyOrganizationInput {
  nodes: Node[];
  interfaces: NodeInterface[];
  networkObjects: NetworkObject[];
  links: Link[];
  networkAttachments: NetworkAttachment[];
  networkObjectLinks: NetworkObjectLink[];
  current: Record<string, Point>;
}

interface OrganizedResource {
  id: string;
  name: string;
  layer: number;
  previousX: number;
}

const LAYER_Y = [-420, -120, 180, 500];
const HORIZONTAL_SPACING = 260;
const TERMINAL_EXTRA_SPACING = 60;
const SWEEP_COUNT = 8;

function resourceLayer(resourceType: "node" | "network_object", kind: string) {
  if (resourceType === "node" || kind === "pc") return 3;
  if (kind === "nat_bridge") return 0;
  if (kind === "switch_l3") return 1;
  return 2;
}

function centeredPositions(count: number, spacing: number) {
  const start = -((count - 1) * spacing) / 2;
  return Array.from({ length: count }, (_, index) => start + index * spacing);
}

function normalizedIndex(index: number, count: number) {
  return count < 2 ? 0 : index / (count - 1);
}

export function organizeTopology(input: TopologyOrganizationInput) {
  const interfaceOwners = new Map(
    input.interfaces.map((value) => [value.id, value.node_id]),
  );
  const resources: OrganizedResource[] = [
    ...input.nodes.map((value) => ({
      id: value.id,
      name: value.name,
      layer: resourceLayer("node", value.kind),
      previousX: input.current[value.id]?.x ?? 0,
    })),
    ...input.networkObjects.map((value) => ({
      id: value.id,
      name: value.name,
      layer: resourceLayer("network_object", value.kind),
      previousX: input.current[value.id]?.x ?? 0,
    })),
  ];
  const resourceIds = new Set(resources.map((value) => value.id));
  const adjacency = new Map<string, Set<string>>(
    resources.map((value) => [value.id, new Set<string>()]),
  );
  const connect = (left?: string, right?: string) => {
    if (!left || !right || left === right) return;
    if (!resourceIds.has(left) || !resourceIds.has(right)) return;
    adjacency.get(left)?.add(right);
    adjacency.get(right)?.add(left);
  };
  for (const link of input.links)
    connect(
      interfaceOwners.get(link.endpoint_a_id),
      interfaceOwners.get(link.endpoint_b_id),
    );
  for (const attachment of input.networkAttachments)
    connect(
      interfaceOwners.get(attachment.interface_id),
      attachment.network_object_id,
    );
  for (const link of input.networkObjectLinks)
    connect(link.object_a_id, link.object_b_id);

  const layers = Array.from({ length: 4 }, (_, layer) =>
    resources
      .filter((value) => value.layer === layer)
      .sort(
        (left, right) =>
          left.previousX - right.previousX ||
          left.name.localeCompare(right.name) ||
          left.id.localeCompare(right.id),
      ),
  );

  const orderPositions = () => {
    const result = new Map<string, number>();
    for (const layer of layers)
      layer.forEach((resource, index) =>
        result.set(resource.id, normalizedIndex(index, layer.length)),
      );
    return result;
  };
  const sortLayer = (layerIndex: number, positions: Map<string, number>) => {
    const layer = layers[layerIndex];
    const stableIndex = new Map(layer.map((value, index) => [value.id, index]));
    layer.sort((left, right) => {
      const score = (resource: OrganizedResource) => {
        const neighbors = [...(adjacency.get(resource.id) || [])]
          .map((id) => positions.get(id))
          .filter((value): value is number => value !== undefined);
        return neighbors.length
          ? neighbors.reduce((sum, value) => sum + value, 0) / neighbors.length
          : normalizedIndex(stableIndex.get(resource.id) || 0, layer.length);
      };
      return (
        score(left) - score(right) ||
        (stableIndex.get(left.id) || 0) - (stableIndex.get(right.id) || 0)
      );
    });
  };

  for (let sweep = 0; sweep < SWEEP_COUNT; sweep += 1) {
    let positions = orderPositions();
    for (let layer = 1; layer < layers.length; layer += 1) {
      sortLayer(layer, positions);
      positions = orderPositions();
    }
    for (let layer = layers.length - 2; layer >= 0; layer -= 1) {
      sortLayer(layer, positions);
      positions = orderPositions();
    }
  }

  const widestUpperLayer = Math.max(
    0,
    ...layers
      .slice(0, 3)
      .map((layer) => Math.max(0, (layer.length - 1) * HORIZONTAL_SPACING)),
  );
  const result: Record<string, Point> = {};
  for (const [layerIndex, layer] of layers.entries()) {
    if (!layer.length) continue;
    let spacing = HORIZONTAL_SPACING;
    if (layerIndex === 3 && layer.length > 1) {
      spacing = Math.max(
        HORIZONTAL_SPACING + TERMINAL_EXTRA_SPACING,
        (widestUpperLayer + HORIZONTAL_SPACING * 2) / (layer.length - 1),
      );
    }
    const positions = centeredPositions(layer.length, spacing);
    layer.forEach((resource, index) => {
      result[resource.id] = {
        x: Math.round(positions[index]),
        y: LAYER_Y[layerIndex],
      };
    });
  }
  return result;
}
