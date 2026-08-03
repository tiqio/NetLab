import { describe, expect, it } from "vitest";
import { interfaceFactory, linkFactory, nodeFactory } from "@/test/factories";
import {
  linkDisplayName,
  parallelLinkCurveness,
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
});
