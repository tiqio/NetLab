import { describe, expect, it } from "vitest";
import { TopologyConnectionController } from "./topologyConnectionController";

const port = (id: string, available = true) => ({
  id,
  ownerId: id.split(":")[0],
  name: id,
  available,
});

describe("TopologyConnectionController", () => {
  it("retains the source while previewing and supports direct-port drop", () => {
    const controller = new TopologyConnectionController();
    expect(controller.begin("a:eth0", { x: 1, y: 2 })).toMatchObject({
      type: "preview",
      sourceInterfaceId: "a:eth0",
    });
    expect(controller.move({ x: 20, y: 30 })).toMatchObject({
      pointer: { x: 20, y: 30 },
    });
    expect(controller.dropOnPort(port("b:eth0"))).toEqual({
      type: "ready",
      sourceInterfaceId: "a:eth0",
      targetInterfaceId: "b:eth0",
    });
  });

  it("bypasses the chooser for one port and opens it for several", () => {
    const controller = new TopologyConnectionController();
    controller.begin("a:eth0", { x: 0, y: 0 });
    expect(controller.dropOnResource("b", [port("b:eth0")]).type).toBe("ready");
    const choice = controller.dropOnResource("c", [
      port("c:eth0"),
      port("c:eth1"),
    ]);
    expect(choice).toMatchObject({
      type: "choose_port",
      targetResourceId: "c",
    });
  });

  it("rejects occupied targets and cancels without a mutation", () => {
    const controller = new TopologyConnectionController();
    controller.begin("a:eth0", { x: 0, y: 0 });
    expect(controller.dropOnResource("b", [port("b:eth0", false)])).toEqual({
      type: "invalid",
      reason: "target has no available interfaces",
    });
    expect(controller.cancel()).toEqual({ type: "cancelled" });
    expect(controller.sourceInterfaceId).toBe("");
  });

  it("offers only free named ports for network objects", () => {
    const controller = new TopologyConnectionController();
    controller.begin("switch-a:swp1", { x: 10, y: 20 });
    const choice = controller.dropOnResource("switch-b", [
      {
        id: "switch-b:swp1",
        ownerId: "switch-b",
        name: "swp1",
        available: false,
      },
      {
        id: "switch-b:swp2",
        ownerId: "switch-b",
        name: "swp2",
        available: true,
      },
      {
        id: "switch-b:swp3",
        ownerId: "switch-b",
        name: "swp3",
        available: true,
      },
    ]);
    expect(choice).toMatchObject({
      type: "choose_port",
      candidates: [
        { id: "switch-b:swp2", name: "swp2" },
        { id: "switch-b:swp3", name: "swp3" },
      ],
    });
  });
});
