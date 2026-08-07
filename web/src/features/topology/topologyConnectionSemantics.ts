import type { TopologyConnectionSemanticMarker } from "./interactionTypes";

export function connectionSemanticMarkers(
  sourceKind: string,
  targetKind: string,
): TopologyConnectionSemanticMarker[] {
  const kinds = new Set([sourceKind, targetKind]);
  if (kinds.has("nat_bridge")) return ["managed-nat-uplink"];
  if (kinds.has("bridge") || kinds.has("switch_l2"))
    return ["shared-broadcast-domain"];
  return [];
}
