import { describe, expect, it } from "vitest";
import type { ConnectionPresentation } from "./interactionTypes";
import { assignParallelRoutes } from "./topologyParallelRoutes";

function connection(
  id: string,
  source: string,
  target: string,
): ConnectionPresentation {
  return {
    id,
    persistedKind: "node_link",
    source: {
      resourceId: source,
      resourceType: "node",
      resourceKind: "docker",
      resourceName: source,
      portId: `${source}-0`,
      portName: "eth0",
      endpointKey: `${source}/${source}-0`,
    },
    target: {
      resourceId: target,
      resourceType: "node",
      resourceKind: "docker",
      resourceName: target,
      portId: `${target}-0`,
      portName: "eth0",
      endpointKey: `${target}/${target}-0`,
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
    semanticMarkers: [],
    routeGroupKey: "",
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

describe("assignParallelRoutes", () => {
  it("groups every connection kind by unordered resource pair", () => {
    const values = [
      connection("b", "right", "left"),
      connection("a", "left", "right"),
      connection("c", "left", "right"),
    ];
    values[1].persistedKind = "network_attachment";
    values[2].persistedKind = "network_object_link";
    const routed = assignParallelRoutes(values);
    expect(routed.map((item) => item.id)).toEqual(["b", "a", "c"]);
    expect(routed.map((item) => item.routeCount)).toEqual([3, 3, 3]);
    expect(routed.map((item) => item.routeIndex).sort()).toEqual([0, 1, 2]);
    expect(new Set(routed.map((item) => item.routeGroupKey)).size).toBe(1);
  });

  it("returns stable symmetric curveness", () => {
    const routed = assignParallelRoutes([
      connection("c", "a", "b"),
      connection("a", "a", "b"),
      connection("b", "a", "b"),
    ]);
    expect(routed.map((item) => item.curveness)).toEqual([0.22, -0.22, 0]);
  });
});
