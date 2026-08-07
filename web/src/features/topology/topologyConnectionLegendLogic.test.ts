import { describe, expect, it } from "vitest";
import type { ConnectionPresentation } from "./interactionTypes";
import { buildConnectionLegend } from "./topologyConnectionLegend";

function connection(
  id: string,
  markers: ConnectionPresentation["semanticMarkers"],
): ConnectionPresentation {
  return {
    id,
    persistedKind: "node_link",
    source: {
      resourceId: "a",
      resourceType: "node",
      resourceKind: "docker",
      resourceName: "a",
      portId: "a0",
      portName: "eth0",
      endpointKey: "a/a0",
    },
    target: {
      resourceId: "b",
      resourceType: "node",
      resourceKind: "docker",
      resourceName: "b",
      portId: "b0",
      portName: "eth0",
      endpointKey: "b/b0",
    },
    desiredState: "connected",
    actualState: "connected",
    statusVisual: {
      state: "connected",
      label: "已连接",
      colorToken: "var(--topology-connection-success)",
      lineType: "solid",
      width: 2,
      cue: "normal",
    },
    semanticMarkers: markers,
    routeGroupKey: "a:b",
    routeIndex: 0,
    routeCount: 1,
    label: id,
    capabilities: {
      selectable: true,
      deletable: true,
      capturable: true,
      trafficFilterable: true,
    },
    accessibilityLabel: id,
  };
}

describe("buildConnectionLegend", () => {
  it("only emits semantic markers present in the topology", () => {
    const items = buildConnectionLegend([
      connection("a", ["managed-nat-uplink"]),
      connection("b", ["managed-nat-uplink"]),
      connection("c", []),
    ]);
    expect(items).toHaveLength(1);
    expect(items[0]).toMatchObject({
      key: "managed-nat-uplink",
      count: 2,
      connectionIds: ["a", "b"],
    });
  });
});
