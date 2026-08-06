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

export function normalizeTopologyKind(kind: string): TopologySymbolKind {
  return kind in kindLabels ? (kind as TopologySymbolKind) : "fallback";
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
  let color = "var(--topology-port-muted)";
  let symbol: VisualSemantic["symbol"] = "roundRect";
  let pattern: VisualSemantic["pattern"] = "solid";
  if (
    observedState === "running" ||
    observedState === "connected" ||
    observedState === "active"
  ) {
    stateLabel = "运行中";
    color = "var(--topology-running)";
    symbol = "circle";
  } else if (observedState === "failed") {
    stateLabel = "失败";
    color = "var(--topology-failed)";
    symbol = "diamond";
    pattern = "dotted";
  } else if (
    ["starting", "stopping", "provisioning", "queued", "pending"].includes(
      observedState,
    )
  ) {
    stateLabel = "状态转换中";
    color = "var(--topology-transition)";
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
    color,
    borderColor: selected
      ? "var(--topology-selected)"
      : traffic
        ? "var(--topology-traffic)"
        : color,
    symbol,
    pattern,
    selected,
    traffic,
    desiredStateLabel,
  };
}
