import { describe, expect, it } from "vitest";
import { interfaceFactory, linkFactory, nodeFactory } from "@/test/factories";
import type {
  NetworkAttachment,
  NetworkObject,
  NetworkObjectLink,
} from "@/api";
import { organizeTopology } from "./topologyOrganizer";

function object(id: string, kind: NetworkObject["kind"]): NetworkObject {
  return {
    id,
    laboratory_id: "lab",
    name: id,
    kind,
    revision: 1,
    desired_state: "active",
    observed_state: "active",
    config: {},
  };
}

describe("organizeTopology", () => {
  it("places NAT highest and runtime or PC endpoints lowest", () => {
    const result = organizeTopology({
      nodes: [nodeFactory({ id: "qemu", kind: "qemu" })],
      interfaces: [],
      networkObjects: [
        object("nat", "nat_bridge"),
        object("router", "switch_l3"),
        object("switch", "switch_l2"),
        object("pc", "pc"),
      ],
      links: [],
      networkAttachments: [],
      networkObjectLinks: [],
      current: {},
    });
    expect(result.nat.y).toBeLessThan(result.router.y);
    expect(result.router.y).toBeLessThan(result.switch.y);
    expect(result.switch.y).toBeLessThan(result.pc.y);
    expect(result.qemu.y).toBe(result.pc.y);
  });

  it("uses connectivity ordering to avoid crossings between adjacent layers", () => {
    const interfaces = [
      interfaceFactory({ id: "a0", node_id: "a", name: "eth0" }),
      interfaceFactory({ id: "b0", node_id: "b", name: "eth0" }),
    ];
    const attachments: NetworkAttachment[] = [
      {
        id: "attach-a",
        network_object_id: "left",
        interface_id: "a0",
        port_name: "a",
        revision: 1,
        observed_state: "active",
      },
      {
        id: "attach-b",
        network_object_id: "right",
        interface_id: "b0",
        port_name: "b",
        revision: 1,
        observed_state: "active",
      },
    ];
    const result = organizeTopology({
      nodes: [nodeFactory({ id: "a" }), nodeFactory({ id: "b" })],
      interfaces,
      networkObjects: [
        object("left", "switch_l2"),
        object("right", "switch_l2"),
      ],
      links: [],
      networkAttachments: attachments,
      networkObjectLinks: [],
      current: {
        left: { x: -100, y: 0 },
        right: { x: 100, y: 0 },
        a: { x: 100, y: 0 },
        b: { x: -100, y: 0 },
      },
    });
    expect(Math.sign(result.left.x - result.right.x)).toBe(
      Math.sign(result.a.x - result.b.x),
    );
  });

  it("is deterministic and gives every resource a unique coordinate", () => {
    const input = {
      nodes: [
        nodeFactory({ id: "a", name: "A" }),
        nodeFactory({ id: "b", name: "B" }),
      ],
      interfaces: [
        interfaceFactory({ id: "a0", node_id: "a" }),
        interfaceFactory({ id: "b0", node_id: "b" }),
      ],
      networkObjects: [object("nat", "nat_bridge")],
      links: [linkFactory({ endpoint_a_id: "a0", endpoint_b_id: "b0" })],
      networkAttachments: [] as NetworkAttachment[],
      networkObjectLinks: [] as NetworkObjectLink[],
      current: {},
    };
    const first = organizeTopology(input);
    expect(organizeTopology(input)).toEqual(first);
    expect(
      new Set(Object.values(first).map(({ x, y }) => `${x}:${y}`)).size,
    ).toBe(Object.keys(first).length);
  });
});
