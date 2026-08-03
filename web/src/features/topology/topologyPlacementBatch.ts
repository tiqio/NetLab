import type {
  TopologyPlacement,
  TopologyPlacementUpdate,
} from "@/api/generated";
import type { Point } from "@/types/workspace";

export function buildPlacementBatch(
  draggedId: string,
  target: Point,
  selectedIds: string[],
  coordinates: Record<string, Point>,
  resourceTypes: Record<string, "node" | "network_object">,
  placements: TopologyPlacement[],
): TopologyPlacementUpdate[] {
  const origin = coordinates[draggedId];
  if (!origin || !resourceTypes[draggedId]) return [];
  const ids = selectedIds.includes(draggedId) ? selectedIds : [draggedId];
  const dx = target.x - origin.x;
  const dy = target.y - origin.y;
  const revisions = new Map(
    placements.map((placement) => [placement.resource_id, placement.revision]),
  );
  return ids
    .filter((id) => coordinates[id] && resourceTypes[id])
    .slice(0, 100)
    .map((id) => ({
      resource_id: id,
      resource_type: resourceTypes[id],
      x: coordinates[id].x + dx,
      y: coordinates[id].y + dy,
      revision: revisions.get(id),
    }));
}
