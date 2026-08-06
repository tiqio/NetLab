<script setup lang="ts">
import { computed } from "vue";
import EChart from "@/components/charts/EChart.vue";
import type { TrafficObservation } from "@/api";

interface ScopeNode {
  id: string;
  label: string;
  x?: number;
  y?: number;
}

interface ScopeLink {
  id: string;
  source: string;
  target: string;
  label: string;
}

const props = defineProps<{
  observations: TrafficObservation[];
  ambiguous?: boolean;
  interfaceOwners?: Record<string, string>;
  macOwners?: Record<string, string>;
  coordinates?: Record<string, { x: number; y: number }>;
  resourceLabels?: Record<string, string>;
  scopeNodes?: ScopeNode[];
  scopeLinks?: ScopeLink[];
  listening?: boolean;
  expression?: string;
  sourceCount?: number;
  sessionStartedAt?: string;
  sessionFinishedAt?: string;
}>();

const displayedSourceCount = computed(
  () =>
    props.sourceCount ??
    (props.scopeNodes?.length || 0) + (props.scopeLinks?.length || 0),
);
const observedTimeRange = computed(() => {
  const timestamps = props.observations.flatMap((observation) => [
    observation.first_seen,
    observation.last_seen,
  ]);
  const valid = timestamps
    .map((value) => new Date(value).getTime())
    .filter((value) => Number.isFinite(value));
  if (!valid.length) return undefined;
  return {
    first: new Date(Math.min(...valid)).toISOString(),
    last: new Date(Math.max(...valid)).toISOString(),
  };
});

function formatTimestamp(value?: string) {
  if (!value) return "—";
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return value;
  const pad = (part: number, width = 2) => String(part).padStart(width, "0");
  const offsetMinutes = -date.getTimezoneOffset();
  const offsetSign = offsetMinutes >= 0 ? "+" : "-";
  const offsetHours = Math.floor(Math.abs(offsetMinutes) / 60);
  const offsetRemainder = Math.abs(offsetMinutes) % 60;
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}.${pad(date.getMilliseconds(), 3)} UTC${offsetSign}${pad(offsetHours)}:${pad(offsetRemainder)}`;
}

const option = computed(() => {
  const scopeNodes = props.scopeNodes || [];
  const scopeLinks = props.scopeLinks || [];
  const scopeLinkById = new Map(scopeLinks.map((link) => [link.id, link]));
  const resourceId = (item: TrafficObservation) =>
    props.interfaceOwners?.[item.interface_id] ||
    item.interface_id ||
    undefined;
  const macOwner = (value?: string) =>
    value ? props.macOwners?.[value.toLowerCase()] : undefined;
  const groups = new Map<string, TrafficObservation[]>();
  const directedLinkObservations = new Map<
    string,
    {
      linkId: string;
      source: string;
      target: string;
      fingerprints: Set<string>;
      observations: number;
      bytes: number;
      firstSeen: string;
      lastSeen: string;
    }
  >();
  const undirectedLinkObservations = new Map<
    string,
    {
      fingerprints: Set<string>;
      observations: number;
      bytes: number;
      firstSeen: string;
      lastSeen: string;
    }
  >();
  for (const observation of props.observations) {
    if (observation.link_id && scopeLinkById.has(observation.link_id)) {
      const link = scopeLinkById.get(observation.link_id)!;
      const source = macOwner(observation.source_mac);
      const destination = macOwner(observation.destination_mac);
      const target =
        destination ||
        (source === link.source
          ? link.target
          : source === link.target
            ? link.source
            : undefined);
      const directed = source && target && source !== target;
      if (directed) {
        const key = `${observation.link_id}:${source}>${target}`;
        const current = directedLinkObservations.get(key) || {
          linkId: observation.link_id,
          source,
          target,
          fingerprints: new Set<string>(),
          observations: 0,
          bytes: 0,
          firstSeen: observation.first_seen,
          lastSeen: observation.last_seen,
        };
        current.fingerprints.add(observation.fingerprint);
        current.observations += observation.count;
        current.bytes += observation.bytes;
        if (observation.first_seen < current.firstSeen)
          current.firstSeen = observation.first_seen;
        if (observation.last_seen > current.lastSeen)
          current.lastSeen = observation.last_seen;
        directedLinkObservations.set(key, current);
      } else {
        const current = undirectedLinkObservations.get(observation.link_id) || {
          fingerprints: new Set<string>(),
          observations: 0,
          bytes: 0,
          firstSeen: observation.first_seen,
          lastSeen: observation.last_seen,
        };
        current.fingerprints.add(observation.fingerprint);
        current.observations += observation.count;
        current.bytes += observation.bytes;
        if (observation.first_seen < current.firstSeen)
          current.firstSeen = observation.first_seen;
        if (observation.last_seen > current.lastSeen)
          current.lastSeen = observation.last_seen;
        undirectedLinkObservations.set(observation.link_id, current);
      }
    }
    if (!resourceId(observation)) continue;
    const values = groups.get(observation.fingerprint) || [];
    values.push(observation);
    groups.set(observation.fingerprint, values);
  }

  const ids = new Set(scopeNodes.map((node) => node.id));
  const observedNodeIds = new Set<string>();
  for (const observation of directedLinkObservations.values()) {
    observedNodeIds.add(observation.source);
    observedNodeIds.add(observation.target);
  }
  for (const linkId of undirectedLinkObservations.keys()) {
    const link = scopeLinkById.get(linkId);
    if (link) {
      observedNodeIds.add(link.source);
      observedNodeIds.add(link.target);
    }
  }
  const paths = new Set<string>();
  const edges = new Map<
    string,
    {
      source: string;
      target: string;
      fingerprints: Set<string>;
      observations: number;
      bytes: number;
      firstSeen: string;
      lastSeen: string;
      directions: Set<string>;
    }
  >();
  let repeatedResources = false;
  let multiHopFingerprints = 0;
  for (const [fingerprint, observations] of groups) {
    const ordered = [...observations].sort(
      (left, right) =>
        new Date(left.first_seen).getTime() -
        new Date(right.first_seen).getTime(),
    );
    let locations = ordered
      .map(resourceId)
      .filter((id): id is string => Boolean(id));
    const sourceNode = observations
      .map((item) => macOwner(item.source_mac))
      .find((value): value is string => Boolean(value));
    const destinationNode = observations
      .map((item) => macOwner(item.destination_mac))
      .find((value): value is string => Boolean(value));
    const uniqueLocations = [...new Set(locations)];
    if (
      sourceNode &&
      destinationNode &&
      sourceNode !== destinationNode &&
      uniqueLocations.includes(sourceNode) &&
      uniqueLocations.includes(destinationNode)
    ) {
      locations = [
        sourceNode,
        ...uniqueLocations.filter(
          (id) => id !== sourceNode && id !== destinationNode,
        ),
        destinationNode,
      ];
    }
    for (const id of locations) {
      ids.add(id);
      observedNodeIds.add(id);
    }
    if (new Set(locations).size < locations.length) repeatedResources = true;
    if (locations.length > 1) multiHopFingerprints++;
    if (locations.length) paths.add(locations.join(" → "));
    for (let index = 1; index < locations.length; index++) {
      const source = locations[index - 1];
      const target = locations[index];
      if (!source || !target || source === target) continue;
      const key = `${source}>${target}`;
      const item =
        observations.find(
          (observation) => resourceId(observation) === target,
        ) || ordered[Math.min(index, ordered.length - 1)];
      const edge = edges.get(key) || {
        source,
        target,
        fingerprints: new Set<string>(),
        observations: 0,
        bytes: 0,
        firstSeen: item.first_seen,
        lastSeen: item.last_seen,
        directions: new Set<string>(),
      };
      edge.fingerprints.add(fingerprint);
      edge.observations += item.count;
      edge.bytes += item.bytes;
      if (item.first_seen < edge.firstSeen) edge.firstSeen = item.first_seen;
      if (item.last_seen > edge.lastSeen) edge.lastSeen = item.last_seen;
      edge.directions.add(item.direction);
      edges.set(key, edge);
    }
  }

  const directedPaths = [...directedLinkObservations.values()].map(
    (observation) => `${observation.source} → ${observation.target}`,
  );
  const observedPaths = new Set([...paths, ...directedPaths]);
  const fingerprintCount = new Set(
    props.observations.map((observation) => observation.fingerprint),
  ).size;
  const confidence =
    props.ambiguous || repeatedResources
      ? "low"
      : multiHopFingerprints > 0 || directedLinkObservations.size > 0
        ? "high"
        : "insufficient";
  const edgeValues = [...edges.values()];
  const observedPairs = new Set(
    [
      ...edgeValues.map((edge) => [edge.source, edge.target]),
      ...[...directedLinkObservations.values()].map((edge) => [
        edge.source,
        edge.target,
      ]),
    ].map((pair) => pair.sort().join("|")),
  );
  const previewLinks = scopeLinks.filter(
    (link) =>
      !undirectedLinkObservations.has(link.id) &&
      !observedPairs.has([link.source, link.target].sort().join("|")),
  );
  const undirectedLinks = scopeLinks.filter((link) =>
    undirectedLinkObservations.has(link.id),
  );
  const scopeNodeById = new Map(scopeNodes.map((node) => [node.id, node]));

  return {
    tooltip: {
      trigger: "item",
      formatter: (value: { data: Record<string, unknown> }) =>
        String(value.data.tooltip || value.data.name || "Traffic observation"),
    },
    series: [
      {
        type: "graph",
        layout: props.coordinates ? "none" : "circular",
        roam: true,
        label: { show: true, color: "#e6edf3" },
        edgeLabel: {
          show: true,
          color: "#fbbf24",
          formatter: (value: { data: { label: string } }) => value.data.label,
        },
        data: [...ids].map((id) => {
          const scopeNode = scopeNodeById.get(id);
          const observed = observedNodeIds.has(id);
          return {
            id,
            name: scopeNode?.label || props.resourceLabels?.[id] || id,
            x: scopeNode?.x ?? props.coordinates?.[id]?.x,
            y: scopeNode?.y ?? props.coordinates?.[id]?.y,
            symbolSize: 52,
            tooltip: observed
              ? `${scopeNode?.label || props.resourceLabels?.[id] || id}<br/>已观测到匹配流量`
              : `${scopeNode?.label || props.resourceLabels?.[id] || id}<br/>正在监听，暂未观测到匹配流量`,
            itemStyle: observed
              ? {
                  color: "#164e63",
                  borderColor: "#38bdf8",
                  borderWidth: 2,
                }
              : {
                  color: "#273449",
                  borderColor: "#64748b",
                  borderWidth: 2,
                },
          };
        }),
        links: [
          ...previewLinks.map((link) => ({
            source: link.source,
            target: link.target,
            label: "监听中",
            tooltip: `${link.label}<br/>正在监听，暂未观测到匹配流量`,
            lineStyle: {
              width: 2,
              color: "#64748b",
              type: "dashed",
            },
            symbol: ["none", "none"],
          })),
          ...undirectedLinks.map((link) => {
            const stats = undirectedLinkObservations.get(link.id)!;
            return {
              source: link.source,
              target: link.target,
              label: `${stats.fingerprints.size} ${stats.fingerprints.size === 1 ? "packet" : "packets"}`,
              tooltip: `${link.label}<br/>此观测记录无法确定方向<br/>${stats.observations} packet sightings · ${stats.bytes} bytes<br/>first ${stats.firstSeen}<br/>last ${stats.lastSeen}`,
              lineStyle: {
                width: 3,
                color: props.ambiguous ? "#f59e0b" : "#2dd4bf",
                type: "solid",
              },
              symbol: ["none", "none"],
            };
          }),
          ...[...directedLinkObservations.values()].map((stats) => {
            const reverseKey = `${stats.linkId}:${stats.target}>${stats.source}`;
            return {
              source: stats.source,
              target: stats.target,
              label: `${stats.fingerprints.size} ${stats.fingerprints.size === 1 ? "packet" : "packets"}`,
              tooltip: `${props.resourceLabels?.[stats.source] || stats.source} → ${props.resourceLabels?.[stats.target] || stats.target}<br/>${stats.observations} packet sightings · ${stats.bytes} bytes<br/>first ${stats.firstSeen}<br/>last ${stats.lastSeen}`,
              lineStyle: {
                width: Math.min(7, 1 + stats.fingerprints.size / 4),
                color: props.ambiguous ? "#f59e0b" : "#2dd4bf",
                type: "solid",
                curveness: directedLinkObservations.has(reverseKey) ? 0.18 : 0,
              },
              symbol: ["none", "arrow"],
              symbolSize: 8,
            };
          }),
          ...edgeValues.map((item) => ({
            source: item.source,
            target: item.target,
            label: `${item.fingerprints.size} fingerprints · ${item.observations} observations`,
            tooltip: `${item.bytes} bytes<br/>directions ${[...item.directions].join(", ")}<br/>first ${item.firstSeen}<br/>last ${item.lastSeen}<br/>confidence ${confidence}`,
            lineStyle: {
              width: Math.min(7, 1 + item.fingerprints.size / 4),
              color: props.ambiguous ? "#f59e0b" : "#2dd4bf",
              type: "solid",
              curveness: edges.has(`${item.target}>${item.source}`) ? 0.18 : 0,
            },
            symbol: ["none", "arrow"],
            symbolSize: 8,
          })),
        ],
      },
    ],
    title: {
      text: `Path confidence: ${confidence} · ${fingerprintCount} fingerprints · ${observedPaths.size} observed paths${repeatedResources ? " · loop/revisit" : ""}`,
      left: 8,
      bottom: 6,
      textStyle: {
        color: confidence === "high" ? "#5eead4" : "#fbbf24",
        fontSize: 11,
      },
    },
  };
});
</script>

<template>
  <div class="flex h-full min-h-[220px] flex-col">
    <div class="grid shrink-0 gap-1 border-b border-border px-3 py-2 text-xs">
      <div class="flex flex-wrap items-center gap-x-4 gap-y-1">
        <strong>观测图</strong>
        <span class="flex items-center gap-1.5 text-muted-foreground">
          <span class="h-0 w-6 border-t-2 border-dashed border-slate-500" />
          灰色虚线表示监听范围
        </span>
        <span class="flex items-center gap-1.5 text-muted-foreground">
          <span class="h-0 w-6 border-t-2 border-teal-400" />
          青色实线表示匹配流量
        </span>
      </div>
      <div
        v-if="sessionStartedAt"
        class="flex flex-wrap gap-x-5 gap-y-1 text-[11px] text-muted-foreground"
      >
        <span>
          会话：
          <time :datetime="sessionStartedAt">{{
            formatTimestamp(sessionStartedAt)
          }}</time>
          →
          <span v-if="listening" class="text-teal-300">运行中</span>
          <time v-else-if="sessionFinishedAt" :datetime="sessionFinishedAt">
            {{ formatTimestamp(sessionFinishedAt) }}
          </time>
          <span v-else>已停止</span>
        </span>
        <span v-if="observedTimeRange">
          匹配流量：
          <time :datetime="observedTimeRange.first">{{
            formatTimestamp(observedTimeRange.first)
          }}</time>
          →
          <time :datetime="observedTimeRange.last">{{
            formatTimestamp(observedTimeRange.last)
          }}</time>
        </span>
        <span v-else>暂时没有匹配的数据包时间戳。</span>
      </div>
    </div>
    <div class="relative min-h-0 flex-1">
      <EChart :option="option" aria-label="流量过滤观测图" />
      <div
        v-if="!observations.length"
        class="pointer-events-none absolute inset-x-0 top-3 flex justify-center px-4 text-center text-xs text-muted-foreground"
      >
        <span v-if="listening">
          正在 {{ displayedSourceCount }} 个已选来源上监听
          {{
            expression || "匹配流量"
          }}。灰色节点和虚线表示监听范围；收到匹配数据包后会变为青色。
        </span>
        <span v-else-if="scopeNodes?.length || scopeLinks?.length">
          此会话尚未观测到匹配的数据包，灰色图形表示观测范围。
        </span>
        <span v-else> 请在上方选择接口或链路以预览观测范围。 </span>
      </div>
      <div
        v-if="ambiguous"
        class="absolute right-2 top-2 rounded border border-amber-500/40 bg-amber-500/10 px-2 py-1 text-xs text-amber-200"
      >
        观测到多条路径、重复项或环路
      </div>
    </div>
  </div>
</template>
