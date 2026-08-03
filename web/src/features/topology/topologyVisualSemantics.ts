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
  qemu: "QEMU virtual machine",
  docker: "Docker container",
  pc: "PC endpoint",
  bridge: "Layer 2 bridge",
  nat_bridge: "NAT bridge",
  switch_l2: "Layer 2 switch",
  switch_l3: "Layer 3 switch",
  fallback: "Generic network device",
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
  let stateLabel = "Stopped or unknown";
  let color = "#64748b";
  let symbol: VisualSemantic["symbol"] = "roundRect";
  let pattern: VisualSemantic["pattern"] = "solid";
  if (
    observedState === "running" ||
    observedState === "connected" ||
    observedState === "active"
  ) {
    stateLabel = "Running";
    color = "#22c55e";
    symbol = "circle";
  } else if (observedState === "failed") {
    stateLabel = "Failed";
    color = "#ef4444";
    symbol = "diamond";
    pattern = "dotted";
  } else if (
    ["starting", "stopping", "provisioning", "queued", "pending"].includes(
      observedState,
    )
  ) {
    stateLabel = "Transitioning";
    color = "#f59e0b";
    pattern = "dashed";
  }
  const desiredStateLabel = desiredState
    ? desiredState.charAt(0).toUpperCase() + desiredState.slice(1)
    : undefined;
  return {
    kind: normalizedKind,
    kindLabel: kindLabels[normalizedKind],
    stateLabel,
    label: desiredStateLabel
      ? `${kindLabels[normalizedKind]} · Desired ${desiredStateLabel} · Actual ${stateLabel}`
      : `${kindLabels[normalizedKind]} · ${stateLabel}`,
    color,
    borderColor: selected ? "#f8fafc" : traffic ? "#f59e0b" : color,
    symbol,
    pattern,
    selected,
    traffic,
    desiredStateLabel,
  };
}
