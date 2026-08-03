import { describe, expect, it } from "vitest";
import { TopologyKeyboardController } from "./topologyKeyboardController";

const resources = [
  { id: "a", type: "node" as const, x: 0, y: 0 },
  { id: "b", type: "node" as const, x: 100, y: 0 },
  { id: "link", type: "link" as const, x: 50, y: 50 },
];
const ports = [
  { id: "a0", ownerId: "a", name: "eth0", available: true },
  { id: "a1", ownerId: "a", name: "eth1", available: true },
];

describe("TopologyKeyboardController", () => {
  it("navigates resources, extends selection, and announces focus", () => {
    const controller = new TopologyKeyboardController(resources, ports);
    expect(controller.handle({ key: "ArrowRight" }, [])).toMatchObject({
      type: "focus_resource",
      resourceId: "a",
      announcement: expect.stringContaining("node a"),
    });
    expect(
      controller.handle({ key: "ArrowRight", shiftKey: true }, ["a"]),
    ).toMatchObject({
      type: "focus_resource",
      resourceId: "b",
      extend: true,
    });
  });

  it("navigates ports and starts a keyboard connection", () => {
    const controller = new TopologyKeyboardController(resources, ports);
    controller.focusResource("a");
    expect(controller.handle({ key: "p" }, ["a"])).toMatchObject({
      type: "focus_port",
      interfaceId: "a0",
    });
    expect(controller.handle({ key: "ArrowDown" }, ["a"])).toMatchObject({
      type: "focus_port",
      interfaceId: "a1",
    });
    expect(controller.handle({ key: "Enter" }, ["a"])).toMatchObject({
      type: "choose_port",
      interfaceId: "a1",
    });
  });

  it("focuses the first available object port and returns to resources", () => {
    const controller = new TopologyKeyboardController(
      [{ id: "switch-a", type: "network_object", x: 0, y: 0 }],
      [
        {
          id: "switch-a:eth0",
          ownerId: "switch-a",
          name: "eth0",
          available: false,
        },
        {
          id: "switch-a:eth1",
          ownerId: "switch-a",
          name: "eth1",
          available: true,
        },
      ],
    );
    controller.focusResource("switch-a");
    expect(controller.handle({ key: "p" }, ["switch-a"])).toMatchObject({
      type: "focus_port",
      interfaceId: "switch-a:eth1",
    });
    controller.focusResource("switch-a");
    expect(controller.handle({ key: "Enter" }, ["switch-a"])).toEqual({
      type: "choose_port",
      interfaceId: "switch-a:eth1",
    });
    expect(
      controller.handle({ key: "ArrowRight" }, ["switch-a"]),
    ).toMatchObject({ type: "focus_resource", resourceId: "switch-a" });
  });

  it("prioritizes cancellation and supports movement and inspector activation", () => {
    const controller = new TopologyKeyboardController(resources, ports);
    controller.focusResource("b");
    expect(
      controller.handle({ key: "Escape" }, ["b"], {
        connection: true,
        boxSelection: true,
      }),
    ).toMatchObject({ type: "cancel_connection" });
    expect(
      controller.handle({ key: "ArrowLeft", altKey: true }, ["a", "b"]),
    ).toEqual({ type: "move_selection", dx: -10, dy: 0 });
    expect(controller.handle({ key: "Enter" }, ["b"])).toMatchObject({
      type: "open_inspector",
      resourceId: "b",
    });
    expect(controller.handle({ key: "t" }, ["b"])).toEqual({
      type: "open_terminal",
      resourceId: "b",
    });
    expect(
      controller.handle({ key: "Enter" }, ["b"], { connection: true }),
    ).toEqual({ type: "choose_connection_target", resourceId: "b" });
    controller.focusResource("link");
    expect(controller.handle({ key: "Delete" }, ["link"])).toEqual({
      type: "disconnect_link",
      resourceId: "link",
    });
  });

  it("zooms around focus and selects all resources", () => {
    const controller = new TopologyKeyboardController(resources, ports);
    expect(controller.handle({ key: "+" }, [])).toMatchObject({
      type: "zoom_view",
      factor: 1.2,
    });
    expect(controller.handle({ key: "-" }, [])).toMatchObject({
      type: "zoom_view",
    });
    expect(controller.handle({ key: "a", ctrlKey: true }, [])).toMatchObject({
      type: "select_all",
      resourceIds: ["a", "b", "link"],
    });
  });
});
