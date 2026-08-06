import { describe, expect, it } from "vitest";
import {
  normalizeTopologyKind,
  resourceVisualSemantic,
} from "./topologyVisualSemantics";

describe("topology visual semantics", () => {
  it.each([
    ["qemu", "QEMU 虚拟机"],
    ["docker", "Docker 容器"],
    ["pc", "PC 端点"],
    ["bridge", "二层网桥"],
    ["nat_bridge", "NAT 网桥"],
    ["switch_l2", "二层交换机"],
    ["switch_l3", "三层交换机"],
  ])("labels %s without relying on color", (kind, label) => {
    expect(resourceVisualSemantic(kind, "running").label).toContain(label);
  });

  it.each([
    ["running", "运行中", "solid"],
    ["stopped", "已停止或状态未知", "solid"],
    ["starting", "状态转换中", "dashed"],
    ["failed", "失败", "dotted"],
  ] as const)("maps %s state", (state, label, pattern) => {
    expect(resourceVisualSemantic("qemu", state)).toMatchObject({
      stateLabel: label,
      pattern,
    });
  });

  it("uses a stable fallback and explicit selection/traffic borders", () => {
    expect(normalizeTopologyKind("vendor-unknown")).toBe("fallback");
    expect(
      resourceVisualSemantic("vendor-unknown", "failed", true, true),
    ).toMatchObject({
      kind: "fallback",
      selected: true,
      traffic: true,
      borderColor: "var(--topology-selected)",
    });
  });
  it("uses semantic palette tokens for every runtime state", () => {
    expect(resourceVisualSemantic("qemu", "running").color).toBe(
      "var(--topology-running)",
    );
    expect(resourceVisualSemantic("qemu", "failed").color).toBe(
      "var(--topology-failed)",
    );
    expect(resourceVisualSemantic("qemu", "starting").color).toBe(
      "var(--topology-transition)",
    );
    expect(
      resourceVisualSemantic("qemu", "stopped", false, true).borderColor,
    ).toBe("var(--topology-traffic)");
  });
  it("labels desired and actual state independently", () => {
    expect(
      resourceVisualSemantic("qemu", "stopped", false, false, "running").label,
    ).toContain("期望 运行中 · 实际 已停止或状态未知");
  });
});
