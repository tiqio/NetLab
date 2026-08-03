<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { Activity, CheckCheck, RefreshCw, Square, X } from "lucide-vue-next";
import {
  api,
  type Link,
  type NetworkAttachment,
  type NetworkObject,
  type NetworkObjectLink,
  type Node,
  type NodeInterface,
  type TrafficFilter,
  type TrafficObservation,
} from "@/api";
import { Button, FormField, Input, Select } from "@/components/ui";
import StatusBadge from "@/components/common/StatusBadge.vue";
import { linkDisplayName } from "@/features/topology/linkPresentation";
import { parseTrafficFilterMatch } from "./trafficFilterMatch";

const props = defineProps<{
  laboratoryId?: string;
  interfaceId?: string;
  linkId?: string;
  nodes?: Node[];
  interfaces?: NodeInterface[];
  links?: Link[];
  attachments?: NetworkAttachment[];
  networkObjects?: NetworkObject[];
  networkObjectLinks?: NetworkObjectLink[];
  interfaceOwners?: Record<string, string>;
  coordinates?: Record<string, { x: number; y: number }>;
  resourceLabels?: Record<string, string>;
}>();
const emit = defineEmits<{
  overlay: [TrafficObservation[], boolean, string];
}>();

interface FilterEntry {
  traffic_filter: TrafficFilter;
  ambiguous: boolean;
}

const expression = ref("icmp");
const selectedExample = ref("icmp");
const color = ref("#f59e0b");
const maximum = ref(1000);
const entries = ref<FilterEntry[]>([]);
const selectedFilterId = ref("");
const selectedInterfaceIds = ref<string[]>([]);
const selectedLinkIds = ref<string[]>([]);
const taskId = ref("");
const taskState = ref("");
const status = ref("");
const busy = ref(false);
let refreshTimer: ReturnType<typeof setTimeout> | undefined;
const ACTIVE_REFRESH_INTERVAL_MS = 100;
const filterExamples = [
  { label: "ICMP（IPv4 Ping）", value: "icmp" },
  { label: "ICMPv6（IPv6 Ping）", value: "icmp6" },
  { label: "ARP 地址解析", value: "arp" },
  { label: "全部 TCP", value: "tcp" },
  { label: "全部 UDP", value: "udp" },
  { label: "HTTP · TCP 80", value: "tcp port 80" },
  { label: "HTTPS · TCP 443", value: "tcp port 443" },
  { label: "SSH · TCP 22", value: "tcp dst port 22" },
  { label: "DNS · UDP 53", value: "udp port 53" },
  { label: "DHCP 请求 · UDP 67", value: "udp dst port 67" },
  { label: "源主机示例", value: "src host 192.0.2.10" },
  { label: "目标主机示例", value: "dst host 198.51.100.20" },
  { label: "源网段示例", value: "src net 10.0.0.0/8" },
  { label: "目标网段示例", value: "dst net 192.168.0.0/16" },
];

const currentEntry = computed(() =>
  entries.value.find(
    (entry) => entry.traffic_filter.id === selectedFilterId.value,
  ),
);
const filter = computed(() => currentEntry.value?.traffic_filter);
const ambiguous = computed(() => currentEntry.value?.ambiguous || false);
const active = computed(() =>
  ["starting", "running", "stopping"].includes(filter.value?.state || ""),
);
const selectedScopeCount = computed(
  () => selectedInterfaceIds.value.length + selectedLinkIds.value.length,
);
const colorValid = computed(() => /^#[0-9a-fA-F]{6}$/.test(color.value));
const interfacesByNode = computed(() =>
  (props.nodes || [])
    .map((node) => ({
      node,
      interfaces: (props.interfaces || [])
        .filter((item) => item.node_id === node.id)
        .sort((left, right) => left.slot - right.slot),
    }))
    .filter((group) => group.interfaces.length),
);
const interfaceById = computed(
  () => new Map((props.interfaces || []).map((item) => [item.id, item])),
);
const nodeById = computed(
  () => new Map((props.nodes || []).map((item) => [item.id, item])),
);
const networkObjectById = computed(
  () =>
    new Map((props.networkObjects || []).map((item) => [item.id, item])),
);
const attachmentRows = computed(() =>
  (props.attachments || [])
    .map((attachment) => {
      const interfaceValue = interfaceById.value.get(attachment.interface_id);
      if (!interfaceValue) return undefined;
      const nodeName =
        nodeById.value.get(interfaceValue.node_id)?.name ||
        interfaceValue.node_id;
      const objectName =
        networkObjectById.value.get(attachment.network_object_id)?.name ||
        attachment.network_object_id;
      return {
        ...attachment,
        label: `${nodeName}:${interfaceValue.name} ↔ ${objectName}:${attachment.port_name || "port"}`,
      };
    })
    .filter((item): item is NonNullable<typeof item> => Boolean(item)),
);
const objectLinkRows = computed(() =>
  (props.networkObjectLinks || []).map((link) => ({
    ...link,
    label: `${networkObjectById.value.get(link.object_a_id)?.name || link.object_a_id}:${link.port_a_name} ↔ ${networkObjectById.value.get(link.object_b_id)?.name || link.object_b_id}:${link.port_b_name}`,
  })),
);
const macOwners = computed(() => {
  const owners: Record<string, string> = {};
  for (const item of props.interfaces || []) {
    if (item.mac_address) owners[item.mac_address.toLowerCase()] = item.node_id;
  }
  return owners;
});
const scopeLinks = computed(() => {
  const selectedInterfaces = new Set(selectedInterfaceIds.value);
  const selectedLinks = new Set(selectedLinkIds.value);
  return (props.links || [])
    .filter(
      (link) =>
        selectedLinks.has(link.id) ||
        (selectedInterfaces.has(link.endpoint_a_id) &&
          selectedInterfaces.has(link.endpoint_b_id)),
    )
    .map((link) => {
      const left = interfaceById.value.get(link.endpoint_a_id);
      const right = interfaceById.value.get(link.endpoint_b_id);
      if (!left || !right) return undefined;
      return {
        id: link.id,
        source: left.node_id,
        target: right.node_id,
        label: linkLabel(link),
      };
    })
    .filter((link): link is NonNullable<typeof link> => Boolean(link));
});
const scopeNodes = computed(() => {
  const nodeIds = new Set<string>();
  for (const interfaceId of selectedInterfaceIds.value) {
    const item = interfaceById.value.get(interfaceId);
    if (item) nodeIds.add(item.node_id);
  }
  for (const link of scopeLinks.value) {
    nodeIds.add(link.source);
    nodeIds.add(link.target);
  }
  return [...nodeIds].map((id) => ({
    id,
    label: nodeName(id),
    x: props.coordinates?.[id]?.x,
    y: props.coordinates?.[id]?.y,
  }));
});

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}

function upsert(entry: FilterEntry) {
  const index = entries.value.findIndex(
    (item) => item.traffic_filter.id === entry.traffic_filter.id,
  );
  if (index >= 0) entries.value[index] = entry;
  else entries.value.unshift(entry);
  entries.value.sort((left, right) =>
    right.traffic_filter.created_at.localeCompare(
      left.traffic_filter.created_at,
    ),
  );
}

async function discover() {
  clearTimeout(refreshTimer);
  if (!props.laboratoryId) {
    entries.value = [];
    selectedFilterId.value = "";
    return;
  }
  try {
    entries.value = (await api.listTrafficFilters(props.laboratoryId)).sort(
      (left, right) =>
        right.traffic_filter.created_at.localeCompare(
          left.traffic_filter.created_at,
        ),
    );
    if (
      !entries.value.some(
        (entry) => entry.traffic_filter.id === selectedFilterId.value,
      )
    )
      selectedFilterId.value = entries.value[0]?.traffic_filter.id || "";
    scheduleRefresh();
  } catch (error) {
    status.value = errorMessage(error);
  }
}

async function waitForTask(id: string) {
  for (let attempt = 0; attempt < 120; attempt++) {
    const task = await api.getTask(id);
    taskState.value = task.state;
    if (["succeeded", "failed", "cancelled"].includes(task.state)) {
      if (task.state !== "succeeded")
        throw new Error(
          task.error?.message || `Traffic Filter task ${task.state}`,
        );
      return task;
    }
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  throw new Error("Traffic Filter 任务在 30 秒内未完成");
}

async function start() {
  if (!props.laboratoryId || !selectedScopeCount.value || busy.value) return;
  busy.value = true;
  clearTimeout(refreshTimer);
  taskState.value = "queued";
  try {
    const match = parseTrafficFilterMatch(expression.value);
    const value = await api.startTrafficFilter({
      laboratory_id: props.laboratoryId,
      match,
      max_observations: Number(maximum.value),
      interface_ids: selectedInterfaceIds.value,
      link_ids: selectedLinkIds.value,
      color: color.value,
    });
    selectedFilterId.value = value.traffic_filter.id;
    upsert({ traffic_filter: value.traffic_filter, ambiguous: false });
    taskId.value = value.task.id;
    status.value = `正在启动 Traffic Filter 任务 ${value.task.id}`;
    await waitForTask(value.task.id);
    await refresh(false);
    status.value = "Traffic Filter 已运行，并会自动刷新匹配结果。";
  } catch (error) {
    status.value = errorMessage(error);
    await discover();
  } finally {
    busy.value = false;
    scheduleRefresh();
  }
}

async function refresh(showStatus = true) {
  if (!selectedFilterId.value) return;
  try {
    const value = await api.getTrafficFilter(selectedFilterId.value);
    upsert(value);
    if (showStatus) status.value = "Traffic Filter 匹配结果已刷新";
  } catch (error) {
    status.value = errorMessage(error);
  } finally {
    scheduleRefresh();
  }
}

function scheduleRefresh() {
  clearTimeout(refreshTimer);
  if (!active.value) return;
  refreshTimer = setTimeout(() => void refresh(false), ACTIVE_REFRESH_INTERVAL_MS);
}

async function stop() {
  if (!filter.value || busy.value) return;
  busy.value = true;
  clearTimeout(refreshTimer);
  try {
    const value = await api.stopTrafficFilter(filter.value.id);
    taskId.value = value.task.id;
    taskState.value = value.task.state;
    status.value = `正在停止 Traffic Filter 任务 ${value.task.id}`;
    await waitForTask(value.task.id);
    await refresh(false);
    status.value = "Traffic Filter 已停止";
  } catch (error) {
    status.value = errorMessage(error);
  } finally {
    busy.value = false;
    scheduleRefresh();
  }
}

function toggleInterface(id: string, checked: boolean) {
  selectedInterfaceIds.value = checked
    ? [...new Set([...selectedInterfaceIds.value, id])]
    : selectedInterfaceIds.value.filter((value) => value !== id);
}

function toggleLink(id: string, checked: boolean) {
  selectedLinkIds.value = checked
    ? [...new Set([...selectedLinkIds.value, id])]
    : selectedLinkIds.value.filter((value) => value !== id);
}

function selectConnectedInterfaces() {
  selectedInterfaceIds.value = (props.interfaces || [])
    .filter(
      (item) =>
        item.operational_state === "up" || Boolean(item.desired_link_id),
    )
    .map((item) => item.id);
  selectedLinkIds.value = [];
}

function selectLinks() {
  selectedLinkIds.value = [...(props.links || []), ...(props.networkObjectLinks || [])]
    .filter((item) => item.observed_state === "connected")
    .map((item) => item.id);
  selectedInterfaceIds.value = attachmentRows.value
    .filter((item) => item.observed_state === "active")
    .map((item) => item.interface_id);
}

function clearScope() {
  selectedInterfaceIds.value = [];
  selectedLinkIds.value = [];
}

function nodeName(nodeId: string) {
  return (props.nodes || []).find((node) => node.id === nodeId)?.name || nodeId;
}

function linkLabel(link: Link) {
  return linkDisplayName(link, props.interfaces || [], props.nodes || []);
}

watch(
  () => props.laboratoryId,
  () => {
    selectedInterfaceIds.value = [];
    selectedLinkIds.value = [];
    selectedFilterId.value = "";
    void discover();
  },
  { immediate: true },
);
watch(
  () => props.interfaceId,
  (id) => {
    if (id) toggleInterface(id, true);
  },
  { immediate: true },
);
watch(
  () => props.linkId,
  (id) => {
    if (id) toggleLink(id, true);
  },
  { immediate: true },
);
watch(selectedFilterId, (id) => {
  const selected = entries.value.find(
    (entry) => entry.traffic_filter.id === id,
  )?.traffic_filter;
  if (selected) {
    selectedInterfaceIds.value = [...(selected.interface_ids || [])];
    selectedLinkIds.value = [...(selected.link_ids || [])];
    expression.value = selected.expression;
    color.value = selected.color || "#f59e0b";
    selectedExample.value = filterExamples.some(
      (item) => item.value === selected.expression,
    )
      ? selected.expression
      : "custom";
    maximum.value = selected.max_observations;
  }
  scheduleRefresh();
});
watch(
  [filter, active],
  ([value, running]) =>
    emit(
      "overlay",
      running ? value?.observations || [] : [],
      running,
      value?.color || color.value || "#f59e0b",
    ),
  { immediate: true },
);
onBeforeUnmount(() => {
  clearTimeout(refreshTimer);
  emit("overlay", [], false, color.value);
});

function applyExample(value: string | number | undefined) {
  const selected = String(value || "custom");
  selectedExample.value = selected;
  if (selected !== "custom") expression.value = selected;
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <header class="grid shrink-0 gap-3 border-b border-border p-3">
      <div class="grid grid-cols-2 gap-2 xl:grid-cols-5">
        <FormField label="过滤模板">
          <Select
            :model-value="selectedExample"
            aria-label="过滤模板"
            @update:model-value="applyExample"
          >
            <option value="custom">自定义表达式</option>
            <option
              v-for="example in filterExamples"
              :key="example.value"
              :value="example.value"
            >
              {{ example.label }}
            </option>
          </Select>
        </FormField>
        <FormField label="pcap 过滤表达式" class="xl:col-span-2">
          <Input
            v-model="expression"
            placeholder="icmp, tcp dst port 443, src host 192.0.2.20"
            aria-label="pcap 过滤表达式"
            @input="selectedExample = 'custom'"
          />
        </FormField>
        <FormField label="高亮颜色">
          <div class="flex items-center gap-2">
            <input
              v-model="color"
              type="color"
              aria-label="过滤器高亮颜色"
              class="h-9 w-12 cursor-pointer rounded border border-border bg-background p-1"
            />
            <Input
              v-model="color"
              aria-label="过滤器颜色值"
              maxlength="7"
              placeholder="#f59e0b"
            />
          </div>
          <p v-if="!colorValid" class="mt-1 text-xs text-destructive">
            请输入六位十六进制颜色，例如 #f59e0b。
          </p>
        </FormField>
        <FormField label="最大记录数">
          <Input v-model="maximum" type="number" min="1" max="10000" />
        </FormField>
        <FormField label="Traffic Filter 会话" class="xl:col-span-2">
          <Select v-model="selectedFilterId">
            <option value="">未选择会话</option>
            <option
              v-for="entry in entries"
              :key="entry.traffic_filter.id"
              :value="entry.traffic_filter.id"
            >
              {{ entry.traffic_filter.state }} ·
              {{ entry.traffic_filter.expression }} ·
              {{
                new Date(entry.traffic_filter.created_at).toLocaleTimeString()
              }}
            </option>
          </Select>
        </FormField>
      </div>
      <p class="text-xs text-muted-foreground">
        使用 pcap/tcpdump 风格：协议可写
        <code>icmp</code>、<code>tcp</code>、<code>udp</code>；端口可写
        <code>tcp port 443</code>、<code>udp dst port 53</code>；地址可写
        <code>src host 192.0.2.10</code> 或
        <code>dst net 10.0.0.0/8</code>。模板选中后仍可继续编辑。
      </p>

      <section
        class="grid gap-2 rounded border border-border bg-background/40 p-2"
      >
        <div class="flex flex-wrap items-center gap-2">
          <strong class="text-xs">监听范围</strong>
          <span class="text-xs text-muted-foreground">
            已选择
            {{ selectedScopeCount }} 个监听源；每个接口或链路占用一个抓包槽位。
          </span>
          <Button
            size="sm"
            variant="secondary"
            @click="selectConnectedInterfaces"
          >
            <CheckCheck :size="13" /> 已连接接口
          </Button>
          <Button size="sm" variant="secondary" @click="selectLinks">
            <CheckCheck :size="13" /> 已连接链路
          </Button>
          <Button size="sm" variant="ghost" @click="clearScope">
            <X :size="13" /> 清空范围
          </Button>
        </div>
        <div class="grid max-h-32 grid-cols-2 gap-3 overflow-auto text-xs">
          <div class="grid content-start gap-1">
            <strong class="text-muted-foreground">接口</strong>
            <template v-for="group in interfacesByNode" :key="group.node.id">
              <span class="mt-1 font-medium">{{ group.node.name }}</span>
              <label
                v-for="item in group.interfaces"
                :key="item.id"
                class="flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-accent"
              >
                <input
                  type="checkbox"
                  :checked="selectedInterfaceIds.includes(item.id)"
                  @change="
                    toggleInterface(
                      item.id,
                      ($event.target as HTMLInputElement).checked,
                    )
                  "
                />
                <span>{{ item.name }}</span>
                <span class="ml-auto text-muted-foreground">{{
                  item.operational_state
                }}</span>
              </label>
            </template>
          </div>
          <div class="grid content-start gap-1">
            <strong class="text-muted-foreground">链路</strong>
            <label
              v-for="link in links || []"
              :key="link.id"
              class="flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-accent"
            >
              <input
                type="checkbox"
                :checked="selectedLinkIds.includes(link.id)"
                @change="
                  toggleLink(
                    link.id,
                    ($event.target as HTMLInputElement).checked,
                  )
                "
              />
              <span>{{ linkLabel(link) }}</span>
              <span class="ml-auto text-muted-foreground">{{
                link.observed_state
              }}</span>
            </label>
            <label
              v-for="objectLink in objectLinkRows"
              :key="objectLink.id"
              class="flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-accent"
            >
              <input
                type="checkbox"
                :checked="selectedLinkIds.includes(objectLink.id)"
                @change="toggleLink(objectLink.id, ($event.target as HTMLInputElement).checked)"
              />
              <span>{{ objectLink.label }}</span>
              <span class="ml-auto text-muted-foreground">对象链路 · {{ objectLink.observed_state }}</span>
            </label>
            <label
              v-for="attachment in attachmentRows"
              :key="attachment.id"
              class="flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-accent"
            >
              <input
                type="checkbox"
                :checked="selectedInterfaceIds.includes(attachment.interface_id)"
                @change="
                  toggleInterface(
                    attachment.interface_id,
                    ($event.target as HTMLInputElement).checked,
                  )
                "
              />
              <span>{{ attachment.label }}</span>
              <span class="ml-auto text-muted-foreground">
                附件 · {{ attachment.observed_state }}
              </span>
            </label>
          </div>
        </div>
      </section>

      <div class="flex flex-wrap items-center gap-2">
        <Button
          size="sm"
          :disabled="!laboratoryId || !selectedScopeCount || !colorValid || busy"
          :title="
            !selectedScopeCount
              ? '请至少选择一个接口或链路'
              : !colorValid
                ? '请先修正高亮颜色'
                : '启动 Traffic Filter'
          "
          @click="start"
        >
          <Activity :size="14" /> 启动
        </Button>
        <Button
          size="sm"
          variant="secondary"
          :disabled="!filter || busy"
          @click="refresh"
        >
          <RefreshCw :size="14" /> 刷新
        </Button>
        <Button
          size="sm"
          variant="destructive"
          :disabled="!filter || !active || busy"
          @click="stop"
        >
          <Square :size="14" /> 停止
        </Button>
        <StatusBadge v-if="filter" :state="filter.state" />
        <span v-if="taskId" class="font-mono text-[10px] text-muted-foreground">
          任务 {{ taskId }} · {{ taskState }}
        </span>
      </div>
      <p
        v-if="status"
        role="status"
        class="rounded border border-border bg-background/60 px-2 py-1 text-xs text-muted-foreground"
      >
        {{ status }}
      </p>
    </header>

    <div class="min-h-0 flex-1 overflow-auto p-3">
      <section class="rounded-md border border-cyan-500/30 bg-cyan-500/5 p-3">
        <div class="flex items-center gap-2">
          <Activity :size="16" class="text-cyan-300" />
          <h3 class="text-sm font-semibold">拓扑流量高亮</h3>
          <StatusBadge
            :state="active ? 'running' : filter?.state || 'stopped'"
          />
        </div>
        <p class="mt-2 text-xs text-muted-foreground">
          匹配的数据包直接在主拓扑链路上流动显示。单向流量展示发送方 →
          接收方箭头；活动会话约每 100 ms 刷新。停止流量后粒子会先消失，方向箭头约保留 4 秒，便于确认最近的数据流向。
        </p>
        <dl class="mt-3 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
          <dt>表达式</dt>
          <dd>{{ filter?.expression || expression }}</dd>
          <dt>流量指纹</dt>
          <dd>
            {{
              new Set(
                (filter?.observations || []).map((item) => item.fingerprint),
              ).size
            }}
          </dd>
          <dt>匹配包数</dt>
          <dd>
            {{
              (filter?.observations || []).reduce(
                (total, item) => total + item.count,
                0,
              )
            }}
          </dd>
          <dt>匹配字节</dt>
          <dd>
            {{
              (filter?.observations || []).reduce(
                (total, item) => total + item.bytes,
                0,
              )
            }}
          </dd>
        </dl>
        <p
          v-if="active && !(filter?.observations || []).length"
          class="mt-3 rounded border border-dashed border-border p-2 text-xs text-muted-foreground"
        >
          正在等待匹配的数据包。请在节点中产生流量并观察上方主拓扑。
        </p>
      </section>
    </div>
    <footer class="border-t border-border p-2 text-xs text-muted-foreground">
      <span v-if="filter">
        {{ filter.state }} · {{ (filter.observations || []).length }}/{{
          filter.max_observations
        }}
        条记录
      </span>
    </footer>
  </div>
</template>
