import type { NetworkObject, Node } from "@/api/generated";
import type { Placement } from "@/types/workspace";

export interface LayoutResource {
  id: string;
  name: string;
  kind: string;
}
const hash = (value: string) =>
  [...value].reduce(
    (result, character) =>
      ((result << 5) - result + character.charCodeAt(0)) | 0,
    0,
  );

export function deterministicPlacement(
  id: string,
  occupied: Array<{ x: number; y: number }> = [],
) {
  const seed = Math.abs(hash(id));
  let angle = ((seed % 360) * Math.PI) / 180;
  let radius = 100 + (seed % 5) * 55;
  for (let attempt = 0; attempt < 50; attempt += 1) {
    const point = {
      x: Math.round(Math.cos(angle) * radius),
      y: Math.round(Math.sin(angle) * radius),
    };
    if (
      occupied.every(
        (item) => Math.hypot(item.x - point.x, item.y - point.y) > 80,
      )
    )
      return point;
    angle += 0.75;
    radius += 10;
  }
  return { x: radius, y: radius };
}

export function resolvePlacements(
  resources: Array<Node | NetworkObject>,
  stored: Record<string, Placement>,
  previous: Record<string, { x: number; y: number }> = {},
) {
  const resolved: Record<string, { x: number; y: number }> = {};
  const ordered = [...resources].sort((a, b) => a.id.localeCompare(b.id));
  for (const resource of ordered) {
    if (stored[resource.id]) resolved[resource.id] = stored[resource.id];
  }
  for (const resource of ordered) {
    if (!stored[resource.id] && previous[resource.id])
      resolved[resource.id] = previous[resource.id];
  }
  for (const resource of ordered) {
    if (!resolved[resource.id])
      resolved[resource.id] = deterministicPlacement(
        resource.id,
        Object.values(resolved),
      );
  }
  return resolved;
}
