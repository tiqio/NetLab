export type TopologySymbolKind =
  | "qemu"
  | "docker"
  | "pc"
  | "bridge"
  | "nat_bridge"
  | "switch_l2"
  | "switch_l3"
  | "fallback";

export interface VisualSemantic {
  kind: TopologySymbolKind;
  kindLabel: string;
  stateLabel: string;
  desiredStateLabel?: string;
  label: string;
  color: string;
  borderColor: string;
  symbol: "circle" | "roundRect" | "diamond";
  pattern: "solid" | "dashed" | "dotted";
  selected: boolean;
  traffic: boolean;
}

export function connectionVisualSemantic(actualState: string) {
  if (["connected", "running", "active"].includes(actualState))
    return { state: "connected" as const, label: zhCN.topologyConnection.connected, colorToken: "var(--topology-connection-success)", lineType: "solid" as const, width: 2, cue: "normal" as const };
  if (["queued", "pending", "provisioning", "starting", "stopping"].includes(actualState))
    return { state: "pending" as const, label: zhCN.topologyConnection.transitioning, colorToken: "var(--topology-connection-transition)", lineType: "dashed" as const, width: 2, cue: "transition" as const };
  if (["disconnecting", "deleting"].includes(actualState))
    return { state: "disconnecting" as const, label: zhCN.topologyConnection.disconnecting, colorToken: "var(--topology-connection-disconnecting)", lineType: "dashed" as const, width: 2, cue: "removing" as const };
  if (["failed", "error", "degraded", "missing"].includes(actualState))
    return { state: "failed" as const, label: zhCN.topologyConnection.failed, colorToken: "var(--topology-connection-danger)", lineType: "dotted" as const, width: 2, cue: "warning" as const };
  return { state: "unknown" as const, label: zhCN.topologyConnection.unknown, colorToken: "var(--topology-connection-neutral)", lineType: "solid" as const, width: 2, cue: "unknown" as const };
}

const kindLabels: Record<TopologySymbolKind, string> = {
  qemu: "QEMU 虚拟机",
  docker: "Docker 容器",
  pc: "PC 端点",
  bridge: "二层网桥",
  nat_bridge: "NAT 网桥",
  switch_l2: "二层交换机",
  switch_l3: "三层交换机",
  fallback: "通用网络设备",
};

const kindColors: Record<TopologySymbolKind, string> = {
  qemu: "var(--topology-kind-qemu)",
  docker: "var(--topology-kind-docker)",
  pc: "var(--topology-kind-lightweight)",
  switch_l2: "var(--topology-kind-lightweight)",
  switch_l3: "var(--topology-kind-lightweight)",
  bridge: "var(--topology-kind-network)",
  nat_bridge: "var(--topology-kind-network)",
  fallback: "var(--topology-kind-network)",
};

export function normalizeTopologyKind(kind: string): TopologySymbolKind {
  return kind in kindLabels ? (kind as TopologySymbolKind) : "fallback";
}

export function topologyCategoryIndex(kind: string): number {
  const normalizedKind = normalizeTopologyKind(kind);
  if (normalizedKind === "qemu") return 0;
  if (normalizedKind === "docker") return 1;
  if (
    normalizedKind === "pc" ||
    normalizedKind === "switch_l2" ||
    normalizedKind === "switch_l3"
  )
    return 2;
  return 3;
}

export function lifecycleSemantic(state: string): VisualSemantic {
  return resourceVisualSemantic("fallback", state);
}

export function resourceVisualSemantic(
  kind: string,
  observedState: string,
  selected = false,
  traffic = false,
  desiredState?: string,
): VisualSemantic {
  const normalizedKind = normalizeTopologyKind(kind);
  let stateLabel = "已停止或状态未知";
  let stateColor = "var(--topology-port-muted)";
  let symbol: VisualSemantic["symbol"] = "roundRect";
  let pattern: VisualSemantic["pattern"] = "solid";
  if (
    observedState === "running" ||
    observedState === "connected" ||
    observedState === "active"
  ) {
    stateLabel = "运行中";
    stateColor = "var(--topology-running)";
    symbol = "circle";
  } else if (observedState === "failed") {
    stateLabel = "失败";
    stateColor = "var(--topology-failed)";
    symbol = "diamond";
    pattern = "dotted";
  } else if (
    ["starting", "stopping", "provisioning", "queued", "pending"].includes(
      observedState,
    )
  ) {
    stateLabel = "状态转换中";
    stateColor = "var(--topology-transition)";
    pattern = "dashed";
  }
  const desiredStateLabel = desiredState
    ? ({
        running: "运行中",
        stopped: "已停止",
        active: "活动",
        inactive: "未活动",
      }[desiredState] ?? desiredState)
    : undefined;
  return {
    kind: normalizedKind,
    kindLabel: kindLabels[normalizedKind],
    stateLabel,
    label: desiredStateLabel
      ? `${kindLabels[normalizedKind]} · 期望 ${desiredStateLabel} · 实际 ${stateLabel}`
      : `${kindLabels[normalizedKind]} · ${stateLabel}`,
    color: kindColors[normalizedKind],
    borderColor: selected
      ? "var(--topology-selected)"
      : traffic
        ? "var(--topology-traffic)"
        : stateColor,
    symbol,
    pattern,
    selected,
    traffic,
    desiredStateLabel,
  };
}
import { zhCN } from "@/locales/zh-CN";
