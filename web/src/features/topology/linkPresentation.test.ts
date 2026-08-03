import { describe, expect, it } from "vitest";
import type { NetworkObject, NetworkObjectLink } from "@/api";
import { interfaceFactory, linkFactory, nodeFactory } from "@/test/factories";
import {
  linkDisplayName,
  networkObjectLinkDisplayName,
  parallelLinkCurveness,
  parallelNetworkObjectLinkCurveness,
} from "./linkPresentation";

describe("link presentation", () => {
  it("uses node and interface names instead of endpoint IDs", () => {
    const nodes = [
      nodeFactory({ id: "node-a", name: "BusyBox1" }),
      nodeFactory({ id: "node-b", name: "BusyBox2" }),
    ];
    const interfaces = [
      interfaceFactory({ id: "if-a", node_id: "node-a", name: "eth0" }),
      interfaceFactory({ id: "if-b", node_id: "node-b", name: "eth1" }),
    ];
    expect(
      linkDisplayName(
        linkFactory({ endpoint_a_id: "if-a", endpoint_b_id: "if-b" }),
        interfaces,
        nodes,
      ),
    ).toBe("BusyBox1:eth0 ↔ BusyBox2:eth1");
  });

  it("assigns opposite curves to parallel links", () => {
    const owners = {
      "if-a0": "node-a",
      "if-a1": "node-a",
      "if-b0": "node-b",
      "if-b1": "node-b",
    };
    const links = [
      linkFactory({
        id: "link-a",
        endpoint_a_id: "if-a0",
        endpoint_b_id: "if-b0",
      }),
      linkFactory({
        id: "link-b",
        endpoint_a_id: "if-a1",
        endpoint_b_id: "if-b1",
      }),
    ];
    const first = parallelLinkCurveness(links[0], links, owners);
    const second = parallelLinkCurveness(links[1], links, owners);
    expect(first).toBeLessThan(0);
    expect(second).toBeGreaterThan(0);
    expect(Math.abs(first)).toBeCloseTo(Math.abs(second));
  });

  it("labels and separates parallel network-object links", () => {
    const objects: NetworkObject[] = [
      {
        id: "a",
        name: "Switch A",
        kind: "switch_l2",
        laboratory_id: "lab",
        revision: 1,
        desired_state: "active",
        observed_state: "active",
        config: {},
      },
      {
        id: "b",
        name: "Switch B",
        kind: "switch_l2",
        laboratory_id: "lab",
        revision: 1,
        desired_state: "active",
        observed_state: "active",
        config: {},
      },
    ];
    const links: NetworkObjectLink[] = [
      {
        id: "link-1",
        laboratory_id: "lab",
        object_a_id: "a",
        port_a_name: "swp1",
        object_b_id: "b",
        port_b_name: "swp1",
        revision: 1,
        desired_state: "connected",
        observed_state: "connected",
      },
      {
        id: "link-2",
        laboratory_id: "lab",
        object_a_id: "a",
        port_a_name: "swp2",
        object_b_id: "b",
        port_b_name: "swp2",
        revision: 1,
        desired_state: "connected",
        observed_state: "connected",
      },
    ];
    expect(networkObjectLinkDisplayName(links[0], objects)).toBe(
      "Switch A:swp1 ↔ Switch B:swp1",
    );
    expect(parallelNetworkObjectLinkCurveness(links[0], links)).not.toBe(
      parallelNetworkObjectLinkCurveness(links[1], links),
    );
  });
});
