import type {
  ConnectionPresentation,
  TopologyConnectionSemanticMarker,
} from "./interactionTypes";

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
    label: "NAT 管理上联",
    description: "该连接接入 NetLab 管理的地址转换和互联网出口。",
  },
  "shared-broadcast-domain": {
    label: "共享广播域",
    description: "该连接进入可承载多个端点的二层共享网段。",
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
