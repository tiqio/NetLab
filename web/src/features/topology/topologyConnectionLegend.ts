import type {
  ConnectionPresentation,
  TopologyConnectionSemanticMarker,
} from "./interactionTypes";
import { zhCN } from "@/locales/zh-CN";

export interface TopologyConnectionLegendItem {
  key: TopologyConnectionSemanticMarker;
  label: string;
  description: string;
  count: number;
  connectionIds: string[];
}

const descriptions: Record<
  TopologyConnectionSemanticMarker,
  Pick<TopologyConnectionLegendItem, "label" | "description">
> = {
  "managed-nat-uplink": {
    label: zhCN.topologyConnection.managedNATUplink.label,
    description: zhCN.topologyConnection.managedNATUplink.description,
  },
  "shared-broadcast-domain": {
    label: zhCN.topologyConnection.sharedBroadcastDomain.label,
    description: zhCN.topologyConnection.sharedBroadcastDomain.description,
  },
};

export function buildConnectionLegend(
  connections: ConnectionPresentation[],
): TopologyConnectionLegendItem[] {
  const grouped = new Map<TopologyConnectionSemanticMarker, string[]>();
  for (const connection of connections) {
    for (const marker of connection.semanticMarkers) {
      const ids = grouped.get(marker) || [];
      ids.push(connection.id);
      grouped.set(marker, ids);
    }
  }
  return [...grouped.entries()]
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([key, connectionIds]) => ({
      key,
      ...descriptions[key],
      count: connectionIds.length,
      connectionIds: [...connectionIds].sort(),
    }));
}
