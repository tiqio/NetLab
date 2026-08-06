<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from "vue";
import EChart from "@/components/charts/EChart.vue";
import type {
  Link,
  NetworkAttachment,
  NetworkObject,
  NetworkObjectLink,
  Node,
  NodeInterface,
  TrafficObservation,
  TopologyPlacement,
} from "@/api";
import {
  networkObjectLinkDisplayName,
  parallelNetworkObjectLinkCurveness,
} from "./linkPresentation";
import type { WorkspacePreferences } from "@/types/workspace";
import { resolvePlacements } from "./topologyLayout";
import { screenToWorld } from "./topologyGeometry";
import {
  TopologyInteractionController,
  type InteractionAction,
} from "./topologyInteractionController";
import { topologySymbol } from "./topologySymbols";
import { resourceVisualSemantic } from "./topologyVisualSemantics";

const props = withDefaults(
  defineProps<{
    nodes: Node[];
    interfaces: NodeInterface[];
    links: Link[];
    networkAttachments?: NetworkAttachment[];
    networkObjectLinks?: NetworkObjectLink[];
    networkObjects: NetworkObject[];
    preferences: WorkspacePreferences;
    sharedPlacements?: TopologyPlacement[];
    selectedIds?: string[];
    focusedResourceId?: string;
    keyboardAnnouncement?: string;
    editingLinkId?: string;
    panEnabled?: boolean;
    connectionSourceInterfaceId?: string;
    connectionSourceObjectPortId?: string;
    traffic?: TrafficObservation[];
    trafficActive?: boolean;
    trafficColor?: string;
  }>(),
  {
    networkAttachments: () => [],
    networkObjectLinks: () => [],
    traffic: () => [],
    trafficActive: false,
    trafficColor: "#f59e0b",
  },
);
const emit = defineEmits<{
  select: [
    string,
    (
      | "node"
      | "link"
      | "network_object"
      | "network_attachment"
      | "network_object_link"
    ),
    boolean,
  ];
  interface: [string];
  objectPort: [string, string];
  connector: [string];
  move: [string, number, number];
  viewport: [{ centerX?: number; centerY?: number; zoom?: number }];
  background: [];
  keyboard: [KeyboardEvent];
  boxSelect: [
    { left: number; top: number; right: number; bottom: number },
    boolean,
  ];
  routePoint: [string, { x: number; y: number }];
  transientCancelled: [];
  context: [
    string,
    "node" | "link" | "network_object" | "network_object_link",
    number,
    number,
  ];
}>();
const interaction = new TopologyInteractionController();
const trafficClock = ref(Date.now());
const trafficActivity = new Map<string, { version: string; seenAt: number }>();
let trafficClockTimer: ReturnType<typeof setInterval> | undefined;
const TRAFFIC_PARTICLE_WINDOW_MS = 700;
const TRAFFIC_DIRECTION_LINGER_MS = 4000;
const GRAPH_WORLD_SIZE = 20000;
const chart = ref<InstanceType<typeof EChart>>();
const chartSize = ref({ width: 800, height: 600 });
const connectionTarget = ref<{ x: number; y: number }>();
const connectionPreview = ref<{
  sourceX: number;
  sourceY: number;
  targetX: number;
  targetY: number;
}>();
const trafficPathOverlays = ref<
  Array<{
    id: string;
    x1: number;
    y1: number;
    x2: number;
    y2: number;
    pathData: string;
    mode: "single" | "bidirectional" | "unknown";
    guideMode: "single" | "initiator" | "none";
    particleMode: "single" | "bidirectional" | "unknown";
    particlesActive: boolean;
    sourceId: string;
    targetId: string;
    count: number;
    bytes: number;
  }>
>([]);
const portOverlays = ref<
  Array<{
    id: string;
    ownerId: string;
    name: string;
    x: number;
    y: number;
    available: boolean;
    source: boolean;
    emphasized: boolean;
    state: string;
    kind: "node_interface" | "network_object_port";
  }>
>([]);
const connectorOverlay = ref<{ ownerId: string; x: number; y: number }>();
const draggingResource = ref(false);
const selectionRectangle = ref<{
  left: number;
  top: number;
  right: number;
  bottom: number;
}>();
const boxPointerId = ref<number>();
const panGesture = ref<{
  pointerId: number;
  startX: number;
  startY: number;
  centerX: number;
  centerY: number;
  zoom: number;
}>();
const hoveredResourceId = ref("");
const hoveredNode = computed(() =>
  props.nodes.find((item) => item.id === hoveredResourceId.value),
);
const hoveredPorts = computed(() =>
  hoveredNode.value ? interfacesByOwner.value[hoveredNode.value.id] || [] : [],
);
const ownerByInterface = computed(() =>
  Object.fromEntries(props.interfaces.map((item) => [item.id, item.node_id])),
);
const interfaceById = computed(() =>
  Object.fromEntries(props.interfaces.map((item) => [item.id, item])),
);
const nodeById = computed(() =>
  Object.fromEntries(props.nodes.map((item) => [item.id, item])),
);
const interfacesByOwner = computed(() => {
  const result: Record<string, NodeInterface[]> = {};
  for (const item of props.interfaces) {
    if (item.name.startsWith("internal")) continue;
    (result[item.node_id] ||= []).push(item);
  }
  return result;
});
let placementCacheKey = "";
let placementCache: Record<string, { x: number; y: number }> = {};
const placements = computed(() => {
  const resources = [...props.nodes, ...props.networkObjects];
  const shared = props.sharedPlacements || [];
  const cacheKey = [
    ...resources.map((item) => item.id).sort(),
    ...shared.map((item) => `${item.resource_id}:${item.x}:${item.y}`).sort(),
  ].join("|");
  if (cacheKey === placementCacheKey) return placementCache;
  placementCacheKey = cacheKey;
  placementCache = resolvePlacements(
    resources,
    Object.fromEntries(
      shared.map((item) => [
        item.resource_id,
        { x: item.x, y: item.y, pinned: true, updatedAt: "" },
      ]),
    ),
  );
  return placementCache;
});
const selected = computed(() => new Set(props.selectedIds || []));
const denseTopology = computed(
  () =>
    props.nodes.length >= 80 ||
    props.links.length +
      props.networkAttachments.length +
      props.networkObjectLinks.length >=
      200,
);
const effectiveLabelDensity = computed(() =>
  denseTopology.value && props.preferences.labelDensity === "comfortable"
    ? "compact"
    : props.preferences.labelDensity,
);
const selectedConnectorNode = computed(() => {
  if ((props.selectedIds || []).length !== 1) return undefined;
  return props.nodes.find((node) => node.id === props.selectedIds?.[0]);
});
function networkObjectPorts(value: NetworkObject) {
  const rows =
    value.kind === "switch_l2"
      ? value.config?.ports
      : value.kind === "switch_l3" || value.kind === "pc"
        ? value.config?.interfaces
        : [];
  return Array.isArray(rows)
    ? rows
        .map((item) => String((item as { name?: string }).name || ""))
        .filter(Boolean)
    : [];
}
function objectPortId(objectId: string, portName: string) {
  return `${objectId}:${portName}`;
}
const occupiedObjectPorts = computed(() => {
  const result = new Set<string>();
  for (const attachment of props.networkAttachments)
    if (attachment.port_name)
      result.add(
        objectPortId(attachment.network_object_id, attachment.port_name),
      );
  for (const link of props.networkObjectLinks) {
    result.add(objectPortId(link.object_a_id, link.port_a_name));
    result.add(objectPortId(link.object_b_id, link.port_b_name));
  }
  return result;
});
const connectionSourcePortId = computed(
  () =>
    props.connectionSourceInterfaceId ||
    props.connectionSourceObjectPortId ||
    "",
);
const availableInterfaceOwners = computed(
  () =>
    new Set(
      props.interfaces
        .filter((item) => !item.desired_link_id)
        .map((item) => item.node_id),
    ),
);
function resourceLabel(
  name: string,
  kind: string,
  observedState: string,
  desiredState?: string,
) {
  const semantic = resourceVisualSemantic(
    kind,
    observedState,
    false,
    false,
    desiredState,
  );
  if (effectiveLabelDensity.value === "minimal") return name;
  if (effectiveLabelDensity.value === "compact")
    return `${name}\n${semantic.stateLabel}`;
  return `${name}\n${semantic.kindLabel} · ${semantic.stateLabel}`;
}
function showPortDetails(nodeId: string) {
  return (
    selected.value.has(nodeId) ||
    hoveredResourceId.value === nodeId ||
    Boolean(props.connectionSourceInterfaceId)
  );
}
function showObjectPortDetails(objectId: string) {
  return (
    selected.value.has(objectId) ||
    hoveredResourceId.value === objectId ||
    Boolean(props.connectionSourceObjectPortId)
  );
}
function endpointLabel(interfaceId: string) {
  const interfaceValue = interfaceById.value[interfaceId];
  if (!interfaceValue) return interfaceId;
  const node = nodeById.value[interfaceValue.node_id];
  return `${node?.name || interfaceValue.node_id}:${interfaceValue.name}`;
}
const automaticLinkCurveness = computed(() => {
  const groups = new Map<string, Link[]>();
  const result = new Map<string, number>();
  for (const link of props.links) {
    const ownerA = ownerByInterface.value[link.endpoint_a_id];
    const ownerB = ownerByInterface.value[link.endpoint_b_id];
    if (!ownerA || !ownerB || ownerA === ownerB) {
      result.set(link.id, 0);
      continue;
    }
    const pair =
      ownerA <= ownerB ? `${ownerA}:${ownerB}` : `${ownerB}:${ownerA}`;
    const siblings = groups.get(pair) || [];
    siblings.push(link);
    groups.set(pair, siblings);
  }
  for (const siblings of groups.values()) {
    siblings.sort((left, right) => left.id.localeCompare(right.id));
    if (siblings.length < 2) {
      result.set(siblings[0].id, 0);
      continue;
    }
    const spacing = Math.min(0.24, 0.84 / (siblings.length - 1));
    siblings.forEach((link, index) => {
      const ownerA = ownerByInterface.value[link.endpoint_a_id];
      const ownerB = ownerByInterface.value[link.endpoint_b_id];
      const offset = (index - (siblings.length - 1) / 2) * spacing;
      result.set(link.id, ownerA <= ownerB ? offset : -offset);
    });
  }
  return result;
});
const macOwners = computed(() => {
  const result: Record<string, string> = {};
  for (const item of props.interfaces)
    if (item.mac_address) result[item.mac_address.toLowerCase()] = item.node_id;
  return result;
});
function trafficObservationKey(observation: TrafficObservation) {
  return [
    observation.fingerprint,
    observation.interface_id,
    observation.link_id,
    observation.network_object_link_id,
    observation.direction,
  ].join(":");
}
function trafficWithin(windowMs: number) {
  return props.traffic.filter((observation) => {
    const activity = trafficActivity.get(trafficObservationKey(observation));
    return Boolean(
      activity && trafficClock.value - activity.seenAt <= windowMs,
    );
  });
}
const recentTraffic = computed(() => trafficWithin(TRAFFIC_PARTICLE_WINDOW_MS));
const lingeringTraffic = computed(() =>
  trafficWithin(TRAFFIC_DIRECTION_LINGER_MS),
);
function aggregateTrafficLinks(observations: TrafficObservation[]) {
  const result = new Map<
    string,
    {
      count: number;
      bytes: number;
      directions: Set<string>;
      pairs: Set<string>;
      source?: string;
      target?: string;
      initiatorSource?: string;
      initiatorTarget?: string;
      initiatorSeenAt?: number;
    }
  >();
  for (const observation of observations) {
    const resourceId =
      observation.network_object_link_id ||
      (observation.resource_type === "network_object_link"
        ? observation.resource_id
        : observation.link_id);
    if (!resourceId) continue;
    const objectLink = props.networkObjectLinks.find(
      (item) => item.id === resourceId,
    );
    if (objectLink) {
      const current = result.get(resourceId) || {
        count: 0,
        bytes: 0,
        directions: new Set<string>(),
        pairs: new Set<string>(),
      };
      current.count += observation.count;
      current.bytes += observation.bytes;
      current.directions.add(observation.direction);
      if (observation.direction === "a_to_b") {
        current.source = objectLink.object_a_id;
        current.target = objectLink.object_b_id;
        current.pairs.add(`${current.source}>${current.target}`);
      } else if (observation.direction === "b_to_a") {
        current.source = objectLink.object_b_id;
        current.target = objectLink.object_a_id;
        current.pairs.add(`${current.source}>${current.target}`);
      }
      result.set(resourceId, current);
      continue;
    }
    const link = props.links.find((item) => item.id === resourceId);
    if (!link) continue;
    const ownerA = ownerByInterface.value[link.endpoint_a_id];
    const ownerB = ownerByInterface.value[link.endpoint_b_id];
    let source = observation.source_mac
      ? macOwners.value[observation.source_mac.toLowerCase()]
      : undefined;
    let target = observation.destination_mac
      ? macOwners.value[observation.destination_mac.toLowerCase()]
      : undefined;
    if (!source || !target || source === target) {
      const observedOwner = ownerByInterface.value[observation.interface_id];
      if (observedOwner === ownerA) {
        source = observation.direction === "ingress" ? ownerB : ownerA;
        target = observation.direction === "ingress" ? ownerA : ownerB;
      } else if (observedOwner === ownerB) {
        source = observation.direction === "ingress" ? ownerA : ownerB;
        target = observation.direction === "ingress" ? ownerB : ownerA;
      }
    }
    const current = result.get(resourceId) || {
      count: 0,
      bytes: 0,
      directions: new Set<string>(),
      pairs: new Set<string>(),
    };
    current.count += observation.count;
    current.bytes += observation.bytes;
    current.directions.add(observation.direction);
    if (source && target && source !== target) {
      current.source = source;
      current.target = target;
      current.pairs.add(`${source}>${target}`);
      if (observation.packet_role === "request") {
        const seenAt =
          trafficActivity.get(trafficObservationKey(observation))?.seenAt ||
          Date.parse(observation.last_seen);
        if (
          Number.isFinite(seenAt) &&
          seenAt >= (current.initiatorSeenAt || 0)
        ) {
          current.initiatorSource = source;
          current.initiatorTarget = target;
          current.initiatorSeenAt = seenAt;
        }
      }
    }
    result.set(resourceId, current);
  }
  return result;
}
const trafficLinks = computed(() =>
  aggregateTrafficLinks(lingeringTraffic.value),
);
const particleLinks = computed(() =>
  aggregateTrafficLinks(recentTraffic.value),
);
const trafficNodeIds = computed(() => {
  const result = new Set<string>();
  for (const observation of lingeringTraffic.value) {
    const owner = ownerByInterface.value[observation.interface_id];
    if (owner) result.add(owner);
    const objectLinkId =
      observation.network_object_link_id ||
      (observation.resource_type === "network_object_link"
        ? observation.resource_id
        : undefined);
    if (objectLinkId) {
      const objectLink = props.networkObjectLinks.find(
        (item) => item.id === objectLinkId,
      );
      if (objectLink) {
        result.add(objectLink.object_a_id);
        result.add(objectLink.object_b_id);
      }
      continue;
    }
    if (!observation.link_id) continue;
    const link = props.links.find((item) => item.id === observation.link_id);
    if (!link) continue;
    const ownerA = ownerByInterface.value[link.endpoint_a_id];
    const ownerB = ownerByInterface.value[link.endpoint_b_id];
    if (ownerA) result.add(ownerA);
    if (ownerB) result.add(ownerB);
  }
  return result;
});
const trafficInterfaceIds = computed(
  () => new Set(lingeringTraffic.value.map((item) => item.interface_id)),
);
const particleInterfaceIds = computed(
  () => new Set(recentTraffic.value.map((item) => item.interface_id)),
);
const groupGraphics = computed(() =>
  props.preferences.groups.flatMap((group) => {
    const points = group.memberResourceIds
      .map((id) => placements.value[id])
      .filter(Boolean);
    if (points.length < 2) return [];
    const left = Math.min(...points.map((point) => point.x)) - 55;
    const top = Math.min(...points.map((point) => point.y)) - 55;
    const right = Math.max(...points.map((point) => point.x)) + 55;
    const bottom = Math.max(...points.map((point) => point.y)) + 55;
    return [
      {
        type: "rect",
        silent: true,
        shape: {
          x: left,
          y: top,
          width: right - left,
          height: bottom - top,
          r: 8,
        },
        style: {
          fill: "rgba(14,116,144,.08)",
          stroke: "#0e7490",
          lineDash: [6, 4],
        },
        z: -1,
      },
      {
        type: "text",
        silent: true,
        x: left + 8,
        y: top + 8,
        style: {
          text: `${group.collapsed ? "▸" : "▾"} ${group.label}`,
          fill: "#67e8f9",
          fontSize: 11,
        },
      },
    ];
  }),
);
const routeGraphics = computed(() => {
  if (!props.editingLinkId) return [];
  const point = props.preferences.linkRoutes[props.editingLinkId]?.[0];
  if (!point) return [];
  return [
    {
      id: `route-handle:${props.editingLinkId}`,
      type: "circle",
      x: point.x,
      y: point.y,
      draggable: true,
      shape: { cx: 0, cy: 0, r: 8 },
      style: { fill: "#f59e0b", stroke: "#fef3c7", lineWidth: 2 },
      z: 30,
      ondrag: (event: { target?: { x?: number; y?: number } }) =>
        emit("routePoint", props.editingLinkId!, {
          x: Number(event.target?.x ?? point.x),
          y: Number(event.target?.y ?? point.y),
        }),
    },
  ];
});
const topologyGraphics = computed(() => [
  ...groupGraphics.value,
  ...routeGraphics.value,
]);
function routeCurveness(link: Link) {
  const point = props.preferences.linkRoutes[link.id]?.[0];
  if (!point) return automaticLinkCurveness.value.get(link.id) || 0;
  const source = placements.value[ownerByInterface.value[link.endpoint_a_id]];
  const target = placements.value[ownerByInterface.value[link.endpoint_b_id]];
  if (!source || !target) return 0.12;
  const dx = target.x - source.x;
  const dy = target.y - source.y;
  const length = Math.max(Math.hypot(dx, dy), 1);
  const midpoint = {
    x: (source.x + target.x) / 2,
    y: (source.y + target.y) / 2,
  };
  const signedOffset =
    ((point.x - midpoint.x) * -dy + (point.y - midpoint.y) * dx) / length;
  return Math.max(-0.5, Math.min(0.5, signedOffset / length));
}
function curvedPathData(
  source: { x: number; y: number },
  target: { x: number; y: number },
  curveness: number,
) {
  if (!curveness) return `M ${source.x} ${source.y} L ${target.x} ${target.y}`;
  const dx = target.x - source.x;
  const dy = target.y - source.y;
  const length = Math.max(Math.hypot(dx, dy), 1);
  const controlX =
    (source.x + target.x) / 2 + (-dy / length) * length * curveness;
  const controlY =
    (source.y + target.y) / 2 + (dx / length) * length * curveness;
  return `M ${source.x} ${source.y} Q ${controlX} ${controlY} ${target.x} ${target.y}`;
}
const option = computed(() => ({
  animation: false,
  backgroundColor: "transparent",
  tooltip: { trigger: "item", confine: true },
  legend: [
    {
      data: ["QEMU", "Docker", "Lightweight", "Network"],
      textStyle: { color: "#91a4b5", fontSize: 10 },
      right: 12,
      top: 10,
    },
  ],
  graphic: topologyGraphics.value,
  series: [
    {
      type: "graph",
      layout: "none",
      left: (chartSize.value.width - GRAPH_WORLD_SIZE) / 2,
      top: (chartSize.value.height - GRAPH_WORLD_SIZE) / 2,
      width: GRAPH_WORLD_SIZE,
      height: GRAPH_WORLD_SIZE,
      roam: "scale",
      center: [
        props.preferences.viewport.centerX,
        props.preferences.viewport.centerY,
      ],
      zoom: props.preferences.viewport.zoom,
      draggable: !props.panEnabled,
      animationDurationUpdate: 0,
      edgeSymbol: ["none", "none"],
      edgeLabel: {
        show: !denseTopology.value,
        color: "#91a4b5",
        fontSize: 9,
        formatter: (value: { data: { label?: string } }) =>
          value.data.label || "",
      },
      label: { show: true, position: "bottom", color: "#e6edf3", fontSize: 11 },
      emphasis: { focus: "adjacency" },
      categories: [
        { name: "QEMU" },
        { name: "Docker" },
        { name: "Lightweight" },
        { name: "Network" },
      ],
      data: [
        ...props.nodes.map((node) => {
          const trafficHit =
            props.trafficActive && trafficNodeIds.value.has(node.id);
          const semantic = resourceVisualSemantic(
            node.kind,
            node.observed_state,
            selected.value.has(node.id),
            false,
            node.desired_state,
          );
          return {
            id: node.id,
            name: resourceLabel(
              node.name,
              node.kind,
              node.observed_state,
              node.desired_state,
            ),
            value: node.observed_state,
            x: placements.value[node.id].x,
            y: placements.value[node.id].y,
            category: node.kind === "qemu" ? 0 : node.kind === "docker" ? 1 : 2,
            symbol: topologySymbol(node.kind),
            symbolSize: selected.value.has(node.id) ? 64 : 56,
            itemStyle: {
              color: semantic.color,
              borderColor: trafficHit
                ? props.trafficColor
                : props.focusedResourceId === node.id
                  ? "#fde047"
                  : semantic.borderColor,
              borderWidth: trafficHit ? 4 : selected.value.has(node.id) ? 4 : 2,
              borderType: semantic.pattern,
              shadowColor: trafficHit ? props.trafficColor : "transparent",
              shadowBlur: trafficHit ? 10 : 0,
            },
            tooltip: {
              formatter: `${node.name}<br/>${semantic.label}<br/>desired ${node.desired_state}<br/>revision ${node.revision}`,
            },
            resourceType: "node",
          };
        }),
        ...props.networkObjects.map((item) => {
          const semantic = resourceVisualSemantic(
            item.kind,
            item.observed_state,
            selected.value.has(item.id),
            false,
            item.desired_state,
          );
          return {
            id: item.id,
            name: resourceLabel(
              item.name,
              item.kind,
              item.observed_state,
              item.desired_state,
            ),
            x: placements.value[item.id].x,
            y: placements.value[item.id].y,
            category: 3,
            symbol: topologySymbol(item.kind),
            symbolSize: selected.value.has(item.id) ? 62 : 54,
            itemStyle: {
              color: semantic.color,
              borderColor:
                props.focusedResourceId === item.id
                  ? "#fde047"
                  : semantic.borderColor,
              borderWidth: selected.value.has(item.id) ? 4 : 2,
              borderType: semantic.pattern,
            },
            tooltip: { formatter: `${item.name}<br/>${semantic.label}` },
            resourceType: "network_object",
          };
        }),
        {
          id: "viewport-anchor-top-left",
          name: "",
          x: -GRAPH_WORLD_SIZE / 2,
          y: -GRAPH_WORLD_SIZE / 2,
          symbolSize: 0,
          silent: true,
          itemStyle: { opacity: 0 },
          label: { show: false },
          tooltip: { show: false },
          resourceType: "viewport_anchor",
        },
        {
          id: "viewport-anchor-bottom-right",
          name: "",
          x: GRAPH_WORLD_SIZE / 2,
          y: GRAPH_WORLD_SIZE / 2,
          symbolSize: 0,
          silent: true,
          itemStyle: { opacity: 0 },
          label: { show: false },
          tooltip: { show: false },
          resourceType: "viewport_anchor",
        },
      ],
      links: [
        ...props.links.map((link) => {
          const hit = trafficLinks.value.get(link.id);
          const trafficMode = !hit?.pairs.size
            ? "unknown"
            : hit.pairs.size === 1
              ? "single"
              : "bidirectional";
          const endpointA = endpointLabel(link.endpoint_a_id);
          const endpointB = endpointLabel(link.endpoint_b_id);
          return {
            id: link.id,
            source:
              ownerByInterface.value[link.endpoint_a_id] || link.endpoint_a_id,
            target:
              ownerByInterface.value[link.endpoint_b_id] || link.endpoint_b_id,
            label: `${endpointA} ↔ ${endpointB}`,
            resourceType: "link",
            symbol: ["none", "none"],
            symbolSize: props.trafficActive && hit ? 13 : 0,
            lineStyle: {
              color:
                props.trafficActive && hit
                  ? props.trafficColor
                  : link.observed_state === "connected"
                    ? "#64748b"
                    : link.observed_state === "pending"
                      ? "#f59e0b"
                      : "#ef4444",
              width: props.trafficActive && hit ? 4 : 2,
              opacity: props.trafficActive && hit ? 0.68 : 1,
              shadowColor:
                props.trafficActive && hit ? props.trafficColor : undefined,
              shadowBlur: props.trafficActive && hit ? 7 : 0,
              type: link.observed_state === "connected" ? "solid" : "dashed",
              curveness: routeCurveness(link),
            },
            tooltip: {
              formatter: `${endpointA} ↔ ${endpointB}<br/>${link.observed_state}${hit ? `<br/>${hit.count} packets · ${hit.bytes} bytes · ${trafficMode}${hit.initiatorSource && hit.initiatorTarget ? `<br/>initiated ${hit.initiatorSource} → ${hit.initiatorTarget}` : ""}` : ""}`,
            },
          };
        }),
        ...props.networkAttachments.flatMap((attachment) => {
          const attachedInterface =
            interfaceById.value[attachment.interface_id];
          if (!attachedInterface) return [];
          const node = nodeById.value[attachedInterface.node_id];
          const networkObject = props.networkObjects.find(
            (item) => item.id === attachment.network_object_id,
          );
          if (!node || !networkObject) return [];
          const healthy = !["failed", "missing"].includes(
            attachment.observed_state,
          );
          const trafficHit =
            props.trafficActive &&
            trafficInterfaceIds.value.has(attachment.interface_id);
          const attachmentSelected = selected.value.has(attachment.id);
          return [
            {
              id: attachment.id,
              source: node.id,
              target: networkObject.id,
              label: `${node.name}:${attachedInterface.name} → ${networkObject.name}`,
              resourceType: "network_attachment",
              lineStyle: {
                color: trafficHit
                  ? props.trafficColor
                  : attachmentSelected
                    ? "#fde047"
                    : healthy
                      ? "#22d3ee"
                      : "#ef4444",
                width: trafficHit || attachmentSelected ? 4 : 2,
                opacity: trafficHit ? 0.68 : 1,
                type: healthy ? "dashed" : "dotted",
                curveness: 0.06,
                shadowColor: trafficHit
                  ? props.trafficColor
                  : attachmentSelected
                    ? "#fde047"
                    : undefined,
                shadowBlur: trafficHit || attachmentSelected ? 7 : 0,
              },
              tooltip: {
                formatter: `${node.name}:${attachedInterface.name} ↔ ${networkObject.name}:${attachment.port_name || "port"}<br/>Network attachment · ${attachment.observed_state}<br/>Click to select for Capture or Traffic Filter`,
              },
            },
          ];
        }),
        ...props.networkObjectLinks.map((link) => {
          const hit = trafficLinks.value.get(link.id);
          const selectedLink = selected.value.has(link.id);
          return {
            id: link.id,
            source: link.object_a_id,
            target: link.object_b_id,
            label: networkObjectLinkDisplayName(link, props.networkObjects),
            resourceType: "network_object_link",
            lineStyle: {
              color: hit
                ? props.trafficColor
                : selectedLink
                  ? "#fde047"
                  : link.observed_state === "connected"
                    ? "#38bdf8"
                    : link.observed_state === "pending"
                      ? "#f59e0b"
                      : "#ef4444",
              width: hit || selectedLink ? 4 : 2,
              opacity: hit ? 0.72 : 1,
              type: link.observed_state === "connected" ? "solid" : "dashed",
              curveness: parallelNetworkObjectLinkCurveness(
                link,
                props.networkObjectLinks,
              ),
              shadowColor: hit
                ? props.trafficColor
                : selectedLink
                  ? "#fde047"
                  : undefined,
              shadowBlur: hit || selectedLink ? 8 : 0,
            },
            tooltip: {
              formatter: `${networkObjectLinkDisplayName(link, props.networkObjects)}<br/>对象间链路 · ${link.observed_state}<br/>可用于抓包和 Traffic Filter`,
            },
          };
        }),
      ],
    },
  ],
}));
function handleClick(event: unknown) {
  const value = event as {
    data?: {
      id?: string;
      resourceType?:
        | "node"
        | "link"
        | "network_object"
        | "network_attachment"
        | "network_object_link"
        | "interface"
        | "connector";
      ownerId?: string;
    };
    event?: { event?: MouseEvent & { offsetX?: number; offsetY?: number } };
  };
  if (!value.data?.id || !value.data.resourceType) {
    emit("background");
    return;
  }
  if (value.data.resourceType === "connector") {
    if (value.data.ownerId) emit("connector", value.data.ownerId);
    return;
  }
  if (value.data.resourceType === "interface") {
    emit("interface", value.data.id);
    return;
  }
  const pointer = pointerSample(value.event?.event);
  interaction.pointerDown(
    pointer,
    {
      kind: "resource",
      id: value.data.id,
      resourceType: value.data.resourceType,
    },
    props.selectedIds,
  );
  applyActions(
    interaction.pointerUp(pointer),
    Boolean(value.event?.event?.ctrlKey || value.event?.event?.metaKey),
  );
}
function handleContext(event: unknown) {
  const value = event as {
    data?: { id?: string; resourceType?: string };
    event?: { event?: MouseEvent };
  };
  const type = value.data?.resourceType;
  if (
    !value.data?.id ||
    !["node", "link", "network_object", "network_object_link"].includes(
      type || "",
    )
  )
    return;
  value.event?.event?.preventDefault?.();
  const pointer = value.event?.event;
  emit(
    "context",
    value.data.id,
    type as "node" | "link" | "network_object" | "network_object_link",
    pointer?.clientX || 0,
    pointer?.clientY || 0,
  );
}
function handleHover(event: unknown) {
  const value = event as {
    data?: { id?: string; resourceType?: string; ownerId?: string };
  };
  if (value.data?.resourceType === "node")
    hoveredResourceId.value = value.data.id || "";
  if (value.data?.resourceType === "network_object")
    hoveredResourceId.value = value.data.id || "";
  if (value.data?.resourceType === "interface")
    hoveredResourceId.value = value.data.ownerId || "";
  scheduleOverlayRefresh();
}
function clearHover() {
  hoveredResourceId.value = "";
  scheduleOverlayRefresh();
}
function handleRoam(event: unknown) {
  const value = event as {
    centerX?: number;
    centerY?: number;
    zoom?: number;
  };
  if (
    Number.isFinite(value.centerX) &&
    Number.isFinite(value.centerY) &&
    Number.isFinite(value.zoom)
  )
    emit("viewport", {
      centerX: value.centerX,
      centerY: value.centerY,
      zoom: value.zoom,
    });
  if (connectionSourcePortId.value) void nextTick(updateConnectionPreview);
}
function refreshOverlays() {
  if (draggingResource.value) {
    portOverlays.value = [];
    connectorOverlay.value = undefined;
    connectionPreview.value = undefined;
    trafficPathOverlays.value = [];
    return;
  }
  const nextPorts: typeof portOverlays.value = [];
  for (const node of props.nodes.filter((item) => showPortDetails(item.id))) {
    const ownerPixel = chart.value?.graphItemPixel?.(node.id);
    if (!ownerPixel) continue;
    const values = interfacesByOwner.value[node.id] || [];
    values.forEach((item, index) => {
      const angle = (Math.PI * 2 * index) / Math.max(values.length, 1);
      const available = !item.desired_link_id;
      nextPorts.push({
        id: item.id,
        ownerId: node.id,
        name: item.name,
        x: ownerPixel.x + Math.cos(angle) * 42,
        y: ownerPixel.y + Math.sin(angle) * 42,
        available,
        source: item.id === props.connectionSourceInterfaceId,
        emphasized: Boolean(
          props.connectionSourceInterfaceId &&
          available &&
          item.id !== props.connectionSourceInterfaceId,
        ),
        state: item.operational_state,
        kind: "node_interface",
      });
    });
  }
  for (const object of props.networkObjects.filter((item) =>
    showObjectPortDetails(item.id),
  )) {
    const ownerPixel = chart.value?.graphItemPixel?.(object.id);
    if (!ownerPixel) continue;
    const values = networkObjectPorts(object);
    values.forEach((name, index) => {
      const id = objectPortId(object.id, name);
      const angle = (Math.PI * 2 * index) / Math.max(values.length, 1);
      const available = !occupiedObjectPorts.value.has(id);
      nextPorts.push({
        id,
        ownerId: object.id,
        name,
        x: ownerPixel.x + Math.cos(angle) * 42,
        y: ownerPixel.y + Math.sin(angle) * 42,
        available,
        source: id === props.connectionSourceObjectPortId,
        emphasized: Boolean(
          props.connectionSourceObjectPortId &&
          available &&
          id !== props.connectionSourceObjectPortId,
        ),
        state: object.observed_state,
        kind: "network_object_port",
      });
    });
  }
  portOverlays.value = nextPorts;
  if (
    selectedConnectorNode.value &&
    availableInterfaceOwners.value.has(selectedConnectorNode.value.id)
  ) {
    const pixel = chart.value?.graphItemPixel?.(selectedConnectorNode.value.id);
    connectorOverlay.value = pixel
      ? { ownerId: selectedConnectorNode.value.id, x: pixel.x + 58, y: pixel.y }
      : undefined;
  } else connectorOverlay.value = undefined;
  const trafficPaths: typeof trafficPathOverlays.value = [];
  if (props.trafficActive) {
    for (const link of props.links) {
      const hit = trafficLinks.value.get(link.id);
      if (!hit) continue;
      const particleHit = particleLinks.value.get(link.id);
      const sourceId =
        hit.initiatorSource ||
        (hit.pairs.size === 1 && hit.source
          ? hit.source
          : ownerByInterface.value[link.endpoint_a_id]);
      const targetId =
        hit.initiatorTarget ||
        (hit.pairs.size === 1 && hit.target
          ? hit.target
          : ownerByInterface.value[link.endpoint_b_id]);
      if (!sourceId || !targetId) continue;
      const source = chart.value?.graphItemPixel?.(sourceId);
      const target = chart.value?.graphItemPixel?.(targetId);
      if (!source || !target) continue;
      const endpointAOwner = ownerByInterface.value[link.endpoint_a_id];
      const curveness =
        sourceId === endpointAOwner
          ? -routeCurveness(link)
          : routeCurveness(link);
      trafficPaths.push({
        id: `traffic:${link.id}`,
        x1: source.x,
        y1: source.y,
        x2: target.x,
        y2: target.y,
        pathData: curvedPathData(source, target, curveness),
        mode:
          hit.pairs.size === 1
            ? "single"
            : hit.pairs.size > 1
              ? "bidirectional"
              : "unknown",
        guideMode:
          hit.pairs.size === 1
            ? "single"
            : hit.initiatorSource && hit.initiatorTarget
              ? "initiator"
              : "none",
        particleMode: !particleHit?.pairs.size
          ? "unknown"
          : particleHit.pairs.size === 1
            ? "single"
            : "bidirectional",
        particlesActive: Boolean(particleHit),
        sourceId,
        targetId,
        count: hit.count,
        bytes: hit.bytes,
      });
    }
    for (const attachment of props.networkAttachments) {
      if (!trafficInterfaceIds.value.has(attachment.interface_id)) continue;
      const interfaceValue = interfaceById.value[attachment.interface_id];
      if (!interfaceValue) continue;
      const source = chart.value?.graphItemPixel?.(interfaceValue.node_id);
      const target = chart.value?.graphItemPixel?.(
        attachment.network_object_id,
      );
      if (!source || !target) continue;
      const observations = recentTraffic.value.filter(
        (item) => item.interface_id === attachment.interface_id,
      );
      trafficPaths.push({
        id: `traffic:${attachment.id}`,
        x1: source.x,
        y1: source.y,
        x2: target.x,
        y2: target.y,
        pathData: curvedPathData(source, target, 0),
        mode: "unknown",
        guideMode: "none",
        particleMode: "unknown",
        particlesActive: particleInterfaceIds.value.has(
          attachment.interface_id,
        ),
        sourceId: interfaceValue.node_id,
        targetId: attachment.network_object_id,
        count: observations.reduce((total, item) => total + item.count, 0),
        bytes: observations.reduce((total, item) => total + item.bytes, 0),
      });
    }
    for (const link of props.networkObjectLinks) {
      const hit = trafficLinks.value.get(link.id);
      if (!hit) continue;
      const particleHit = particleLinks.value.get(link.id);
      const sourceId = hit.source || link.object_a_id;
      const targetId = hit.target || link.object_b_id;
      const source = chart.value?.graphItemPixel?.(sourceId);
      const target = chart.value?.graphItemPixel?.(targetId);
      if (!source || !target) continue;
      const curveness =
        sourceId === link.object_a_id
          ? -parallelNetworkObjectLinkCurveness(link, props.networkObjectLinks)
          : parallelNetworkObjectLinkCurveness(link, props.networkObjectLinks);
      trafficPaths.push({
        id: `traffic:${link.id}`,
        x1: source.x,
        y1: source.y,
        x2: target.x,
        y2: target.y,
        pathData: curvedPathData(source, target, curveness),
        mode: hit.pairs.size === 1 ? "single" : "unknown",
        guideMode: hit.pairs.size === 1 ? "single" : "none",
        particleMode: particleHit?.pairs.size === 1 ? "single" : "unknown",
        particlesActive: Boolean(particleHit),
        sourceId,
        targetId,
        count: hit.count,
        bytes: hit.bytes,
      });
    }
  }
  trafficPathOverlays.value = trafficPaths;
  updateConnectionPreview();
}
function scheduleOverlayRefresh() {
  void nextTick(refreshOverlays);
}
function handleChartGeometryChange() {
  const size = chart.value?.canvasSize?.();
  if (size?.width && size?.height) chartSize.value = size;
  scheduleOverlayRefresh();
}
function updateConnectionPreview() {
  if (!connectionSourcePortId.value || !connectionTarget.value) {
    connectionPreview.value = undefined;
    return;
  }
  const sourcePixel = portOverlays.value.find(
    (item) => item.id === connectionSourcePortId.value,
  );
  if (!sourcePixel) {
    connectionPreview.value = undefined;
    return;
  }
  connectionPreview.value = {
    sourceX: sourcePixel.x,
    sourceY: sourcePixel.y,
    targetX: connectionTarget.value.x,
    targetY: connectionTarget.value.y,
  };
}
function handleConnectionPointer(event: MouseEvent) {
  if (!connectionSourcePortId.value) {
    connectionTarget.value = undefined;
    connectionPreview.value = undefined;
    return;
  }
  connectionTarget.value = { x: event.offsetX, y: event.offsetY };
  refreshOverlays();
}
function handleDragStart(event: unknown) {
  const value = event as {
    data?: { id?: string; resourceType?: "node" | "network_object" };
    event?: { offsetX?: number; offsetY?: number };
  };
  if (!value.data?.id || !value.data.resourceType) return;
  draggingResource.value = true;
  portOverlays.value = [];
  connectorOverlay.value = undefined;
  interaction.pointerDown(
    pointerSample(value.event),
    {
      kind: "resource",
      id: value.data.id,
      resourceType: value.data.resourceType,
    },
    props.selectedIds,
  );
}
function handleDrag(event: unknown) {
  const value = event as {
    data?: { id?: string };
    event?: { offsetX?: number; offsetY?: number };
    graphPoint?: { x: number; y: number };
  };
  if (
    value.data?.id &&
    Number.isFinite(value.event?.offsetX) &&
    Number.isFinite(value.event?.offsetY)
  ) {
    const pointer = pointerSample(value.event);
    interaction.pointerMove(pointer);
    const actions = interaction.pointerUp(pointer);
    const point =
      value.graphPoint ||
      screenToWorld(
        {
          x: Number(value.event?.offsetX),
          y: Number(value.event?.offsetY),
        },
        props.preferences.viewport,
      );
    if (actions.some((action) => action.type === "drag_commit"))
      emit("move", value.data.id, point.x, point.y);
  }
  draggingResource.value = false;
  scheduleOverlayRefresh();
}
function pointerSample(event?: {
  offsetX?: number;
  offsetY?: number;
  button?: number;
  pointerId?: number;
}) {
  return {
    x: Number(event?.offsetX || 0),
    y: Number(event?.offsetY || 0),
    pointerId: Number(event?.pointerId || 1),
    button: Number(event?.button || 0),
    time: Date.now(),
  };
}
function handleSurfacePointerDown(event: PointerEvent) {
  if (props.panEnabled && event.button === 0) {
    panGesture.value = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      centerX: props.preferences.viewport.centerX,
      centerY: props.preferences.viewport.centerY,
      zoom: props.preferences.viewport.zoom,
    };
    (event.currentTarget as HTMLElement).setPointerCapture?.(event.pointerId);
    event.preventDefault();
    event.stopPropagation();
    return;
  }
  if (boxPointerId.value !== undefined) return;
  if (!event.shiftKey || event.button !== 0) return;
  boxPointerId.value = event.pointerId;
  interaction.pointerDown(
    pointerSample(event),
    { kind: "background" },
    props.selectedIds || [],
    true,
  );
  selectionRectangle.value = {
    left: event.offsetX,
    top: event.offsetY,
    right: event.offsetX,
    bottom: event.offsetY,
  };
  (event.currentTarget as HTMLElement).setPointerCapture?.(event.pointerId);
  event.preventDefault();
  event.stopPropagation();
}
function handleSurfacePointerMove(event: PointerEvent) {
  const pan = panGesture.value;
  if (pan?.pointerId === event.pointerId) {
    emit("viewport", {
      centerX: pan.centerX - (event.clientX - pan.startX) / pan.zoom,
      centerY: pan.centerY - (event.clientY - pan.startY) / pan.zoom,
      zoom: pan.zoom,
    });
    event.preventDefault();
    event.stopPropagation();
    return;
  }
  if (boxPointerId.value !== event.pointerId) return;
  const action = interaction
    .pointerMove(pointerSample(event))
    .find((item) => item.type === "box_preview");
  if (action?.type === "box_preview") selectionRectangle.value = action;
}
function handleSurfacePointerUp(event: PointerEvent) {
  if (panGesture.value?.pointerId === event.pointerId) {
    panGesture.value = undefined;
    const surface = event.currentTarget as HTMLElement;
    if (surface.hasPointerCapture?.(event.pointerId))
      surface.releasePointerCapture?.(event.pointerId);
    event.preventDefault();
    event.stopPropagation();
    return;
  }
  if (boxPointerId.value !== event.pointerId) return;
  const action = interaction
    .pointerUp(pointerSample(event))
    .find((item) => item.type === "box_commit");
  if (action?.type === "box_commit") {
    const start = screenToWorld(
      { x: action.left, y: action.top },
      props.preferences.viewport,
    );
    const end = screenToWorld(
      { x: action.right, y: action.bottom },
      props.preferences.viewport,
    );
    emit(
      "boxSelect",
      { left: start.x, top: start.y, right: end.x, bottom: end.y },
      event.ctrlKey || event.metaKey,
    );
  }
  selectionRectangle.value = undefined;
  boxPointerId.value = undefined;
}
function cancelBoxSelection() {
  if (boxPointerId.value === undefined) return;
  interaction.cancel();
  draggingResource.value = false;
  selectionRectangle.value = undefined;
  boxPointerId.value = undefined;
}
function cancelSurfaceInteraction() {
  panGesture.value = undefined;
  cancelBoxSelection();
}
function cancelCanvasTransient() {
  const active =
    panGesture.value !== undefined ||
    boxPointerId.value !== undefined ||
    interaction.state.mode !== "idle";
  if (!active) return false;
  interaction.cancel();
  panGesture.value = undefined;
  selectionRectangle.value = undefined;
  boxPointerId.value = undefined;
  emit("transientCancelled");
  return true;
}
function applyActions(actions: InteractionAction[], additive = false) {
  for (const action of actions) {
    if (action.type === "select")
      emit("select", action.id, action.resourceType, additive);
    if (action.type === "viewport") emit("viewport", action.viewport);
  }
}
function handleKeyboard(event: KeyboardEvent) {
  if (event.key === "Escape" && cancelCanvasTransient()) {
    event.preventDefault();
    event.stopPropagation();
    return;
  }
  emit("keyboard", event);
}
function cancelOnVisibilityLoss() {
  if (document.visibilityState === "hidden") cancelCanvasTransient();
}
onMounted(() => {
  window.addEventListener("blur", cancelCanvasTransient);
  document.addEventListener("visibilitychange", cancelOnVisibilityLoss);
  trafficClockTimer = setInterval(() => {
    trafficClock.value = Date.now();
    scheduleOverlayRefresh();
  }, 100);
});
watch(
  () => props.traffic,
  (observations) => {
    const now = Date.now();
    const keys = new Set<string>();
    for (const observation of observations) {
      const key = trafficObservationKey(observation);
      const version = `${observation.last_seen}:${observation.count}:${observation.bytes}`;
      keys.add(key);
      const current = trafficActivity.get(key);
      if (!current || current.version !== version)
        trafficActivity.set(key, { version, seenAt: now });
    }
    for (const key of trafficActivity.keys())
      if (!keys.has(key)) trafficActivity.delete(key);
    trafficClock.value = now;
  },
  { deep: true, immediate: true },
);
watch(
  () => props.trafficActive,
  (active) => {
    if (!active) trafficActivity.clear();
  },
);
watch(
  () => [
    props.nodes,
    props.interfaces,
    props.networkObjects,
    props.networkAttachments,
    props.networkObjectLinks,
    props.sharedPlacements,
    props.selectedIds,
    props.connectionSourceInterfaceId,
    props.connectionSourceObjectPortId,
    props.traffic,
    props.trafficActive,
    props.trafficColor,
    props.preferences.viewport,
  ],
  scheduleOverlayRefresh,
  { deep: true, immediate: true },
);
onBeforeUnmount(() => {
  clearInterval(trafficClockTimer);
  window.removeEventListener("blur", cancelCanvasTransient);
  document.removeEventListener("visibilitychange", cancelOnVisibilityLoss);
});
defineExpose({
  viewportCenter: () => chart.value?.dataPointAtCanvasCenter?.(),
});
</script>
<template>
  <div
    class="topology-surface h-full min-h-[320px] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
    :data-pan-enabled="panEnabled"
    :data-dense-topology="denseTopology"
    :data-label-density="effectiveLabelDensity"
    :data-traffic-active="trafficActive"
    :data-traffic-observations="traffic.length"
    :data-traffic-recent="recentTraffic.length"
    :data-traffic-lingering="lingeringTraffic.length"
    data-traffic-pulse="false"
    tabindex="0"
    aria-label="Topology canvas keyboard area. Use arrow keys to traverse resources, Shift to extend selection, Enter to open actions, and Escape to clear."
    @keydown="handleKeyboard"
    @dblclick="$emit('background')"
    @pointerdown.capture="handleSurfacePointerDown"
    @pointermove.capture="handleSurfacePointerMove"
    @pointerup.capture="handleSurfacePointerUp"
    @pointercancel.capture="cancelSurfaceInteraction"
    @lostpointercapture="cancelSurfaceInteraction"
  >
    <EChart
      ref="chart"
      :option="option"
      :class="panEnabled ? 'cursor-grab active:cursor-grabbing' : ''"
      aria-label="Topology canvas. Use the inspector for equivalent resource details and actions."
      @chart-click="handleClick"
      @chart-over="handleHover"
      @chart-out="clearHover"
      @chart-context="handleContext"
      @ready="handleChartGeometryChange"
      @rendered="handleChartGeometryChange"
      @resized="handleChartGeometryChange"
      @canvas-pointer="handleConnectionPointer"
      @node-drag-start="handleDragStart"
      @node-drag="handleDrag"
      @graph-roam="handleRoam"
    />
    <svg
      class="pointer-events-none absolute inset-0 h-full w-full overflow-visible"
      aria-hidden="true"
    >
      <defs>
        <marker
          id="traffic-arrow"
          viewBox="0 0 10 10"
          refX="9"
          refY="5"
          markerWidth="7"
          markerHeight="7"
          orient="auto-start-reverse"
        >
          <path d="M 0 0 L 10 5 L 0 10 z" :fill="trafficColor" />
        </marker>
      </defs>
      <g
        v-if="trafficActive"
        data-traffic-path-overlay
        class="traffic-path-overlay"
        :style="{ '--traffic-color': trafficColor }"
      >
        <g
          v-for="path in trafficPathOverlays"
          :key="path.id"
          :data-traffic-path-id="path.id"
          :data-traffic-direction="path.mode"
          :data-traffic-guide="path.guideMode"
          :data-traffic-source="path.sourceId"
          :data-traffic-target="path.targetId"
        >
          <title>{{ path.count }} packets · {{ path.bytes }} bytes</title>
          <path class="traffic-flow-glow" :d="path.pathData" />
          <path class="traffic-flow-trace" :d="path.pathData" />
          <path
            v-if="path.guideMode !== 'none'"
            class="traffic-direction-guide"
            :d="path.pathData"
            marker-end="url(#traffic-arrow)"
          />
          <path
            v-if="path.particlesActive && path.particleMode === 'single'"
            class="traffic-flow-core"
            :d="path.pathData"
          />
          <template
            v-else-if="
              path.particlesActive && path.particleMode === 'bidirectional'
            "
          >
            <path
              class="traffic-flow-core traffic-flow-forward"
              :d="path.pathData"
            />
            <path
              class="traffic-flow-core traffic-flow-reverse"
              :d="path.pathData"
            />
          </template>
          <path
            v-else-if="path.particlesActive"
            class="traffic-flow-unknown-particles"
            :d="path.pathData"
          />
        </g>
      </g>
      <g v-if="connectionPreview" data-connection-preview>
        <line
          :x1="connectionPreview.sourceX"
          :y1="connectionPreview.sourceY"
          :x2="connectionPreview.targetX"
          :y2="connectionPreview.targetY"
          stroke="#5eead4"
          stroke-width="2"
          stroke-dasharray="7 5"
        />
        <circle
          :cx="connectionPreview.targetX"
          :cy="connectionPreview.targetY"
          r="4"
          fill="#5eead4"
        />
      </g>
      <g
        v-for="port in portOverlays"
        :key="port.id"
        class="pointer-events-auto cursor-crosshair"
        :data-interface-id="port.id"
        role="button"
        :aria-label="`${port.name}, ${port.available ? 'available' : 'connected'}, ${port.state}`"
        tabindex="0"
        @click.stop="
          port.kind === 'node_interface'
            ? $emit('interface', port.id)
            : $emit('objectPort', port.ownerId, port.name)
        "
        @keydown.enter.prevent="
          port.kind === 'node_interface'
            ? $emit('interface', port.id)
            : $emit('objectPort', port.ownerId, port.name)
        "
        @keydown.space.prevent="
          port.kind === 'node_interface'
            ? $emit('interface', port.id)
            : $emit('objectPort', port.ownerId, port.name)
        "
      >
        <title>
          {{ port.name }} · {{ port.available ? "available" : "connected" }}
        </title>
        <circle
          :cx="port.x"
          :cy="port.y"
          :r="port.emphasized ? 7.5 : 5.5"
          :fill="
            port.source ? '#f59e0b' : port.available ? '#5eead4' : '#64748b'
          "
          :stroke="port.emphasized ? '#f0fdfa' : '#08131d'"
          :stroke-width="port.emphasized || port.source ? 3 : 2"
        />
        <text :x="port.x + 10" :y="port.y + 3" fill="#ccfbf1" font-size="9">
          {{ port.name }}
        </text>
      </g>
      <g
        v-if="connectorOverlay"
        class="pointer-events-auto cursor-crosshair"
        data-topology-connector
        role="button"
        aria-label="Start connection"
        tabindex="0"
        @click.stop="$emit('connector', connectorOverlay.ownerId)"
        @keydown.enter.prevent="$emit('connector', connectorOverlay.ownerId)"
        @keydown.space.prevent="$emit('connector', connectorOverlay.ownerId)"
      >
        <circle
          :cx="connectorOverlay.x"
          :cy="connectorOverlay.y"
          r="11"
          fill="#0d9488"
          stroke="#99f6e4"
          stroke-width="2"
        />
        <text
          :x="connectorOverlay.x"
          :y="connectorOverlay.y + 4"
          text-anchor="middle"
          fill="#ffffff"
          font-size="14"
        >
          +
        </text>
      </g>
    </svg>
    <div
      v-if="selectionRectangle"
      data-selection-rectangle
      class="pointer-events-none absolute border border-cyan-300 bg-cyan-300/10"
      :style="{
        left: `${selectionRectangle.left}px`,
        top: `${selectionRectangle.top}px`,
        width: `${selectionRectangle.right - selectionRectangle.left}px`,
        height: `${selectionRectangle.bottom - selectionRectangle.top}px`,
      }"
    />
    <p class="sr-only" role="status" aria-live="polite">
      {{ keyboardAnnouncement }}
    </p>
    <p class="sr-only" aria-live="polite" data-testid="topology-hover-details">
      <template v-if="hoveredNode">
        {{ hoveredNode.name }} ports:
        <template v-for="item in hoveredPorts" :key="`hover:${item.id}`">
          {{ item.name }},
          {{ item.desired_link_id ? "connected" : "available" }},
          {{ item.operational_state }};
        </template>
      </template>
    </p>
    <ul class="sr-only" aria-live="polite" data-testid="topology-a11y-summary">
      <li v-for="node in nodes" :key="`a11y:${node.id}`">
        {{ node.name }}:
        {{
          resourceVisualSemantic(
            node.kind,
            node.observed_state,
            false,
            false,
            node.desired_state,
          ).label
        }}{{ selectedIds?.includes(node.id) ? ", selected" : "" }}
      </li>
      <li v-for="item in networkObjects" :key="`a11y:${item.id}`">
        {{ item.name }}:
        {{
          resourceVisualSemantic(
            item.kind,
            item.observed_state,
            false,
            false,
            item.desired_state,
          ).label
        }}{{ selectedIds?.includes(item.id) ? ", selected" : "" }}
      </li>
      <li v-for="link in networkObjectLinks" :key="`a11y:${link.id}`">
        {{ networkObjectLinkDisplayName(link, networkObjects) }}:
        {{ link.observed_state
        }}{{ selectedIds?.includes(link.id) ? ", selected" : "" }}
      </li>
    </ul>
    <div
      v-if="!nodes.length && !networkObjects.length"
      class="pointer-events-none absolute inset-0 grid place-items-center"
    >
      <div
        class="rounded-lg border border-dashed border-border bg-card/85 p-6 text-center"
      >
        <h2 class="font-medium">Empty laboratory</h2>
        <p class="mt-1 text-xs text-muted-foreground">
          Choose a device from the left palette to begin.
        </p>
      </div>
    </div>
  </div>
</template>
<style scoped>
.topology-surface {
  position: relative;
  background-color: var(--topology-canvas);
  background-image:
    linear-gradient(var(--topology-grid) 1px, transparent 1px),
    linear-gradient(90deg, var(--topology-grid) 1px, transparent 1px);
  background-size: 24px 24px;
}
.traffic-flow-glow {
  fill: none;
  stroke: var(--traffic-color);
  stroke-width: 9;
  stroke-linecap: round;
  opacity: 0.1;
}
.traffic-flow-trace {
  fill: none;
  stroke: var(--traffic-color);
  stroke-width: 2;
  stroke-linecap: round;
  opacity: 0.3;
}
.traffic-direction-guide {
  fill: none;
  stroke: color-mix(in srgb, var(--traffic-color) 70%, white);
  stroke-width: 1.8;
  stroke-linecap: round;
  opacity: 0.78;
  filter: drop-shadow(0 0 3px var(--traffic-color));
}
.traffic-flow-core {
  fill: none;
  stroke: color-mix(in srgb, var(--traffic-color) 38%, white);
  stroke-width: 3.5;
  stroke-linecap: round;
  stroke-dasharray: 2 15;
  filter: drop-shadow(0 0 4px var(--traffic-color));
  opacity: 0.88;
  animation: traffic-dash 0.72s linear infinite;
}
.traffic-flow-forward,
.traffic-flow-reverse {
  stroke-width: 2.8;
  stroke-dasharray: 2 18;
}
.traffic-flow-reverse {
  stroke-dashoffset: 10;
  animation: traffic-dash-reverse 0.72s linear infinite;
}
.traffic-flow-unknown-particles {
  fill: none;
  stroke: color-mix(in srgb, var(--traffic-color) 65%, white);
  stroke-width: 3;
  stroke-linecap: round;
  stroke-dasharray: 2 12;
  opacity: 0.72;
  filter: drop-shadow(0 0 4px var(--traffic-color));
}
@keyframes traffic-dash {
  to {
    stroke-dashoffset: -34;
  }
}
@keyframes traffic-dash-reverse {
  to {
    stroke-dashoffset: 44;
  }
}
.topology-surface[data-pan-enabled="true"] :deep(canvas) {
  cursor: grab !important;
}
.topology-surface[data-pan-enabled="true"]:active :deep(canvas) {
  cursor: grabbing !important;
}
</style>
