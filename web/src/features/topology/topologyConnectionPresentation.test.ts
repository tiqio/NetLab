import { describe, expect, it } from "vitest";
import {
  topologyAttachment,
  topologyInterface,
  topologyLink,
  topologyNetworkObject,
  topologyNode,
  topologyObjectLink,
} from "./topologyConnectionFixtures";
import { buildConnectionPresentations } from "./topologyConnectionPresentation";

describe("buildConnectionPresentations", () => {
  it("normalizes all persisted connection kinds into the same status language", () => {
    const nodes = [
      topologyNode("node-a", "BusyBox"),
      topologyNode("node-b", "Ubuntu"),
    ];
    const interfaces = [
      topologyInterface("if-a", "node-a", "eth0"),
      topologyInterface("if-b", "node-b", "ens0"),
      topologyInterface("if-c", "node-a", "eth1", 1),
    ];
    const objects = [
      topologyNetworkObject("bridge", "共享网桥", "bridge"),
      topologyNetworkObject("l2", "接入交换机", "switch_l2"),
    ];
    const result = buildConnectionPresentations({
      nodes,
      interfaces,
      networkObjects: objects,
      links: [topologyLink("link", "if-a", "if-b")],
      networkAttachments: [
        topologyAttachment("attachment", "bridge", "if-c", "port1"),
      ],
      networkObjectLinks: [
        topologyObjectLink("object-link", "bridge", "eth0", "l2", "eth0"),
      ],
    });

    expect(result.map((item) => item.persistedKind)).toEqual([
      "node_link",
      "network_attachment",
      "network_object_link",
    ]);
    expect(
      result.every((item) => item.statusVisual.state === "connected"),
    ).toBe(true);
    expect(result.map((item) => item.label)).toEqual([
      "eth0 ↔ ens0",
      "eth1 ↔ port1",
      "eth0 ↔ eth0",
    ]);
    expect(result[0].accessibilityLabel).toBe(
      "BusyBox:eth0 ↔ Ubuntu:ens0 · 已连接",
    );
  });

  it.each([
    ["queued", "pending", "状态转换中"],
    ["pending", "pending", "状态转换中"],
    ["connected", "connected", "已连接"],
    ["failed", "failed", "失败"],
    ["disconnecting", "disconnecting", "正在断开"],
  ])("maps %s to %s", (actualState, expectedState, label) => {
    const result = buildConnectionPresentations({
      nodes: [topologyNode("a"), topologyNode("b")],
      interfaces: [
        topologyInterface("a0", "a", "eth0"),
        topologyInterface("b0", "b", "eth0"),
      ],
      networkObjects: [],
      links: [topologyLink("link", "a0", "b0", actualState)],
      networkAttachments: [],
      networkObjectLinks: [],
    });
    expect(result[0].statusVisual).toMatchObject({
      state: expectedState,
      label,
    });
  });
});
