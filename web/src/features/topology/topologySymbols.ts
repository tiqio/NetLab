import bridge from "@/assets/topology/bridge.svg?raw";
import docker from "@/assets/topology/docker.svg?raw";
import nat from "@/assets/topology/nat.svg?raw";
import pc from "@/assets/topology/pc.svg?raw";
import qemu from "@/assets/topology/qemu.svg?raw";
import switchL2 from "@/assets/topology/switch-l2.svg?raw";
import switchL3 from "@/assets/topology/switch-l3.svg?raw";

export type TopologyIconTheme = "light" | "dark";

export interface TopologySymbolOptions {
  theme: TopologyIconTheme;
  observedState: string;
  selected?: boolean;
  focused?: boolean;
  trafficColor?: string;
}

const symbols: Record<string, string> = {
  qemu,
  docker,
  pc,
  bridge,
  nat_bridge: nat,
  switch_l2: switchL2,
  switch_l3: switchL3,
};

const mainFill: Record<string, string> = {
  qemu: "#16324a",
  docker: "#123b52",
  pc: "#17354a",
  bridge: "#164e63",
  nat_bridge: "#164e63",
  switch_l2: "#17354a",
  switch_l3: "#17354a",
};

const primaryOutline: Record<string, string> = {
  qemu: "#7dd3fc",
  docker: "#67e8f9",
  pc: "#93c5fd",
  bridge: "#a5f3fc",
  nat_bridge: "#67e8f9",
  switch_l2: "#93c5fd",
  switch_l3: "#c4b5fd",
};

const typeFill = {
  dark: {
    qemu: "#8290a3",
    docker: "#7f988e",
    lightweight: "#968a9f",
    network: "#a58e70",
  },
  light: {
    qemu: "#708096",
    docker: "#6f877e",
    lightweight: "#877b90",
    network: "#927b5d",
  },
} as const;

const stateOutline = {
  dark: {
    running: "#86a087",
    stopped: "#77756f",
    transition: "#b29a72",
    failed: "#bf7773",
    selected: "#ece7dc",
    focused: "#c5ad79",
    traffic: "#b6905d",
    detail: "#f0ece4",
  },
  light: {
    running: "#657d68",
    stopped: "#817d75",
    transition: "#92744f",
    failed: "#a75f5d",
    selected: "#302e29",
    focused: "#88724f",
    traffic: "#8d6e49",
    detail: "#302e29",
  },
} as const;

function category(kind: string): keyof (typeof typeFill)["light"] {
  if (kind === "qemu") return "qemu";
  if (kind === "docker") return "docker";
  if (kind === "pc" || kind === "switch_l2" || kind === "switch_l3")
    return "lightweight";
  return "network";
}

function outlineFor(options: TopologySymbolOptions): string {
  const palette = stateOutline[options.theme];
  if (options.trafficColor)
    return options.trafficColor.startsWith("#")
      ? options.trafficColor
      : palette.traffic;
  if (options.focused) return palette.focused;
  if (options.selected) return palette.selected;
  if (options.observedState === "failed") return palette.failed;
  if (
    ["starting", "stopping", "provisioning", "queued", "pending"].includes(
      options.observedState,
    )
  )
    return palette.transition;
  if (["running", "connected", "active"].includes(options.observedState))
    return palette.running;
  return palette.stopped;
}

function replaceOnce(source: string, value: string, replacement: string) {
  const index = source.indexOf(value);
  if (index < 0) return source;
  return `${source.slice(0, index)}${replacement}${source.slice(index + value.length)}`;
}

export function topologySymbol(
  kind: string,
  options: TopologySymbolOptions,
): string {
  const normalizedKind = kind in symbols ? kind : "qemu";
  const fill = typeFill[options.theme][category(normalizedKind)];
  const outline = outlineFor(options);
  const detail = stateOutline[options.theme].detail;
  let svg = symbols[normalizedKind];
  svg = replaceOnce(svg, mainFill[normalizedKind], fill);
  svg = replaceOnce(svg, primaryOutline[normalizedKind], outline);
  for (const color of [
    "#67e8f9",
    "#a5f3fc",
    "#93c5fd",
    "#7dd3fc",
    "#c4b5fd",
    "#e0f2fe",
    "#ecfeff",
    "#dbeafe",
    "#bfdbfe",
    "#ede9fe",
    "#ddd6fe",
  ])
    svg = svg.replaceAll(color, detail);
  return `image://data:image/svg+xml;charset=UTF-8,${encodeURIComponent(svg)}`;
}
