import { describe, expect, it } from "vitest";
import { TopologyConnectionController } from "./topologyConnectionController";
import type { UnifiedConnectionEndpoint } from "./topologyEndpointCompatibility";

const port = (id: string, available = true) => ({
  id,
  ownerId: id.split(":")[0],
  name: id,
  available,
});

const endpoint = (
  kind: UnifiedConnectionEndpoint["kind"],
  resourceId: string,
  portName?: string,
): UnifiedConnectionEndpoint => ({
  kind,
  laboratoryId: "lab",
  resourceId,
  portId: kind === "node_interface" ? `${resourceId}:${portName}` : undefined,
  portName,
  displayName: portName ? `${resourceId}:${portName}` : resourceId,
  capabilities: [],
  availability: "free",
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

  it("tracks normalized endpoint candidates through chooser and configuration", () => {
    const controller = new TopologyConnectionController();
    const source = endpoint("node_interface", "node-a", "eth0");
    const target = endpoint("network_object_port", "switch-b", "eth1");

    expect(controller.beginEndpoint(source, { x: 4, y: 8 })).toMatchObject({
      type: "preview",
      source,
      phase: "dragging",
    });
    expect(controller.setCandidate(target)).toMatchObject({
      type: "preview",
      candidate: target,
      valid: true,
      backingKind: "network_attachment",
    });
    expect(
      controller.dropOnResourceEndpoints("switch-b", [
        target,
        endpoint("network_object_port", "switch-b", "eth2"),
      ]),
    ).toMatchObject({ type: "choose_endpoint", phase: "choosing" });
    expect(controller.chooseEndpoint(target)).toMatchObject({
      type: "ready_endpoint",
      source,
      target,
      phase: "configuring",
    });
    expect(controller.markSubmitting()).toMatchObject({
      type: "submitting",
      source,
      target,
    });
  });

  it("resolves logical access, cancels same-source drops, and rejects incompatible targets", () => {
    const controller = new TopologyConnectionController();
    const source = endpoint("node_interface", "node-a", "eth0");
    controller.beginEndpoint(source, { x: 0, y: 0 });

    expect(controller.dropOnEndpoint(source)).toEqual({
      type: "cancelled",
      reason: "same_endpoint",
    });
    controller.beginEndpoint(source, { x: 0, y: 0 });
    expect(
      controller.dropOnEndpoint(endpoint("network_object_access", "bridge-a")),
    ).toMatchObject({
      type: "ready_endpoint",
      backingKind: "network_attachment",
    });
    controller.beginEndpoint(endpoint("network_object_access", "bridge-a"), {
      x: 0,
      y: 0,
    });
    expect(
      controller.dropOnEndpoint(endpoint("network_object_access", "nat-a")),
    ).toMatchObject({ type: "invalid", reason: "endpoint_incompatible" });
  });
});
