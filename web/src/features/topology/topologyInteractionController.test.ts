import { describe, expect, it } from "vitest";
import { TopologyInteractionController } from "./topologyInteractionController";
import type { UnifiedConnectionEndpoint } from "./topologyEndpointCompatibility";

const sample = (x: number, y: number, pointerId = 1) => ({
  x,
  y,
  pointerId,
  button: 0,
  time: 1,
});

const endpoint = (resourceId: string): UnifiedConnectionEndpoint => ({
  kind: "node_interface",
  laboratoryId: "lab",
  resourceId,
  portId: `${resourceId}:eth0`,
  portName: "eth0",
  displayName: `${resourceId}:eth0`,
  capabilities: [],
  availability: "free",
});

describe("TopologyInteractionController", () => {
  it("distinguishes clicks from node drags and captures the pointer", () => {
    const controller = new TopologyInteractionController();
    expect(
      controller.pointerDown(sample(0, 0), {
        kind: "resource",
        id: "node-1",
        resourceType: "node",
      }),
    ).toEqual([{ type: "capture_pointer", pointerId: 1 }]);
    expect(controller.pointerMove(sample(3, 3))).toEqual([]);
    expect(controller.pointerUp(sample(3, 3))[0]).toMatchObject({
      type: "select",
      id: "node-1",
    });
  });

  it("pans blank canvas and commits only on release", () => {
    const controller = new TopologyInteractionController();
    controller.pointerDown(sample(10, 10), { kind: "background" });
    expect(controller.pointerMove(sample(30, 20))).toEqual([
      { type: "pan_preview", dx: 20, dy: 10 },
    ]);
    expect(controller.pointerUp(sample(30, 20))[0]).toEqual({
      type: "pan_commit",
      dx: 20,
      dy: 10,
    });
  });

  it("drags the selected group and cancels on Escape or focus loss", () => {
    const controller = new TopologyInteractionController();
    controller.pointerDown(
      sample(0, 0),
      { kind: "resource", id: "node-1", resourceType: "node" },
      ["node-1", "node-2"],
    );
    expect(controller.pointerMove(sample(8, 4))[0]).toEqual({
      type: "drag_preview",
      ids: ["node-1", "node-2"],
      dx: 8,
      dy: 4,
    });
    expect(controller.cancel().map((item) => item.type)).toEqual([
      "cancel",
      "release_pointer",
    ]);
    expect(controller.state.mode).toBe("idle");
  });

  it("anchors wheel zoom and ignores wheel during an active drag", () => {
    const controller = new TopologyInteractionController();
    const action = controller.wheel(
      { centerX: 0, centerY: 0, zoom: 1 },
      { x: 100, y: 50 },
      -100,
    )[0];
    expect(action.type).toBe("viewport");
    controller.pointerDown(sample(0, 0), { kind: "background" });
    controller.pointerMove(sample(10, 0));
    expect(
      controller.wheel({ centerX: 0, centerY: 0, zoom: 1 }, { x: 0, y: 0 }, 10),
    ).toEqual([]);
  });

  it("transitions through connecting, target chooser, ready, and cancellation", () => {
    const controller = new TopologyInteractionController();
    expect(controller.beginConnection("a", { x: 1, y: 2 })[0]).toMatchObject({
      type: "connection_preview",
    });
    expect(controller.state.mode).toBe("connecting");
    expect(
      controller.chooseConnectionTarget("node-b", [
        { id: "b1", ownerId: "node-b", name: "eth0", available: true },
        { id: "b2", ownerId: "node-b", name: "eth1", available: true },
      ])[0],
    ).toMatchObject({ type: "connection_choose_port" });
    expect(controller.state.mode).toBe("choosing_target_port");
    expect(
      controller.chooseConnectionPort({
        id: "b2",
        ownerId: "node-b",
        name: "eth1",
        available: true,
      }),
    ).toEqual([
      {
        type: "connection_ready",
        sourceInterfaceId: "a",
        targetInterfaceId: "b2",
      },
    ]);
    controller.beginConnection("a", { x: 0, y: 0 });
    expect(controller.cancel()[0]).toEqual({ type: "cancel" });
    expect(controller.state.mode).toBe("idle");
  });

  it("previews, commits, and cancels thresholded box selection", () => {
    const controller = new TopologyInteractionController();
    controller.pointerDown(sample(40, 30), { kind: "background" }, ["a"], true);
    expect(controller.pointerMove(sample(10, 5))).toEqual([
      { type: "box_preview", left: 10, top: 5, right: 40, bottom: 30 },
    ]);
    expect(controller.pointerUp(sample(10, 5))[0]).toEqual({
      type: "box_commit",
      left: 10,
      top: 5,
      right: 40,
      bottom: 30,
    });
    controller.pointerDown(sample(0, 0), { kind: "background" }, [], true);
    controller.pointerMove(sample(20, 20));
    expect(controller.cancel()[0]).toEqual({ type: "cancel" });
  });

  it("captures a connection pointer and excludes move, pan, and box gestures", () => {
    const controller = new TopologyInteractionController();
    expect(
      controller.beginConnectionGesture(sample(20, 30, 7), endpoint("a")),
    ).toEqual([
      { type: "capture_pointer", pointerId: 7 },
      expect.objectContaining({ type: "connection_preview" }),
    ]);
    expect(controller.state.mode).toBe("connecting");
    expect(
      controller.pointerDown(
        sample(20, 30, 8),
        { kind: "background" },
        [],
        true,
      ),
    ).toEqual([]);
    expect(controller.pointerMove(sample(23, 33, 7))).toEqual([]);
    expect(controller.pointerMove(sample(29, 38, 7))[0]).toMatchObject({
      type: "connection_preview",
    });
  });

  it("releases captured connection pointers on pointer cancel", () => {
    const controller = new TopologyInteractionController();
    controller.beginConnectionGesture(sample(0, 0, 12), endpoint("a"));
    expect(controller.pointerCancel(12)).toEqual([
      { type: "cancel" },
      { type: "release_pointer", pointerId: 12 },
    ]);
    expect(controller.state.mode).toBe("idle");
  });
});
