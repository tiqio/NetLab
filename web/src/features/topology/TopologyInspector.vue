<script setup lang="ts">
import { computed, ref, watch } from "vue";
import {
  Cable,
  ChevronDown,
  ChevronUp,
  Database,
  Filter,
  Info,
  Radio,
  Trash2,
} from "lucide-vue-next";
import {
  api,
  type CaptureSession,
  type Link,
  type NetworkAttachment,
  type NetworkObject,
  type NetworkObjectLink,
  type Node,
  type NodeInterface,
  type OperationTask,
} from "@/api";
import { Button, FormField, Input, Select } from "@/components/ui";
import EmptyState from "@/components/common/EmptyState.vue";
import ResourceIdentity from "@/components/common/ResourceIdentity.vue";
import StatusBadge from "@/components/common/StatusBadge.vue";
import StructuredProblem from "@/components/common/StructuredProblem.vue";
import NodeOperationsPanel from "@/features/nodes/NodeOperationsPanel.vue";
import FortiGateCredentialPanel from "@/features/nodes/FortiGateCredentialPanel.vue";
import { dockerRouteReadiness } from "@/features/nodes/dockerRouteReadiness";
import RuijieConfigurationPanel from "@/features/nodes/RuijieConfigurationPanel.vue";
import LightweightPCConfigurationPanel from "@/features/nodes/LightweightPCConfigurationPanel.vue";
import LightweightSwitchConfigurationPanel from "@/features/nodes/LightweightSwitchConfigurationPanel.vue";
import ResourceCharts from "@/features/analytics/ResourceCharts.vue";
import ConfirmationDialog from "@/components/common/ConfirmationDialog.vue";
import { linkDisplayName, linkEndpointName } from "./linkPresentation";
import { networkAttachmentPortLabel } from "./networkAttachmentPresentation";

const props = withDefaults(
  defineProps<{
    laboratoryId: string;
    node?: Node;
    link?: Link;
    attachment?: NetworkAttachment;
    networkObjectLink?: NetworkObjectLink;
    networkObject?: NetworkObject;
    interfaces: NodeInterface[];
    nodes?: Node[];
    networkObjects?: NetworkObject[];
    attachments?: NetworkAttachment[];
    networkObjectLinks?: NetworkObjectLink[];
    tasks?: OperationTask[];
  }>(),
  {
    nodes: () => [],
    networkObjects: () => [],
    attachments: () => [],
    networkObjectLinks: () => [],
    tasks: () => [],
  },
);
const emit = defineEmits<{
  changed: [];
  clear: [];
  diagnosticsLoaded: [string];
  terminal: [Node];
  networkObjectTerminal: [NetworkObject];
  captureConnection: [];
  filterConnection: [];
}>();
const attachNode = ref("");
const attachInterface = ref("");
const attachPortName = ref("");
const attachPVID = ref(1);
const attachTagged = ref("");
const attachmentStatus = ref("");
const objectLinkPeerId = ref("");
const objectLinkLocalPort = ref("");
const objectLinkPeerPort = ref("");
const objectLinkStatus = ref("");
const objectLinkCapture = ref<CaptureSession>();
const diagnostics = ref<Record<string, unknown>>();
const diagnosticsLoading = ref(false);
const diagnosticsExpanded = ref(false);
const error = ref("");
const confirmKind = ref<"link" | "network_object" | "">("");
const confirmOpen = computed({
  get: () => Boolean(confirmKind.value),
  set: (value) => {
    if (!value) confirmKind.value = "";
  },
});
const endpointAName = computed(() =>
  props.link
    ? linkEndpointName(props.link.endpoint_a_id, props.interfaces, props.nodes)
    : "",
);
const endpointBName = computed(() =>
  props.link
    ? linkEndpointName(props.link.endpoint_b_id, props.interfaces, props.nodes)
    : "",
);
const selectedLinkName = computed(() =>
  props.link ? linkDisplayName(props.link, props.interfaces, props.nodes) : "",
);
const attachmentInterface = computed(() =>
  props.interfaces.find((item) => item.id === props.attachment?.interface_id),
);
const attachmentNode = computed(() =>
  props.nodes.find((item) => item.id === attachmentInterface.value?.node_id),
);
const attachmentObject = computed(() =>
  props.networkObjects.find(
    (item) => item.id === props.attachment?.network_object_id,
  ),
);
const objectLinkA = computed(() =>
  props.networkObjects.find(
    (item) => item.id === props.networkObjectLink?.object_a_id,
  ),
);
const objectLinkB = computed(() =>
  props.networkObjects.find(
    (item) => item.id === props.networkObjectLink?.object_b_id,
  ),
);
const objectLinkTask = computed(
  () =>
    [...props.tasks]
      .filter(
        (item) =>
          item.resource_type === "network_object_link" &&
          item.resource_id === props.networkObjectLink?.id,
      )
      .sort((left, right) =>
        right.created_at.localeCompare(left.created_at),
      )[0],
);
const objectLinkCaptureStream = computed(() =>
  objectLinkCapture.value ? api.streamCapture(objectLinkCapture.value.id) : "",
);
const attachedInterfaceIds = computed(
  () => new Set(props.attachments.map((item) => item.interface_id)),
);
const availableInterfaces = computed(() =>
  props.interfaces.filter(
    (item) => !item.desired_link_id && !attachedInterfaceIds.value.has(item.id),
  ),
);
const attachNodeInterfaces = computed(() =>
  availableInterfaces.value.filter((item) => item.node_id === attachNode.value),
);
const selectedObjectAttachments = computed(() =>
  props.attachments.filter(
    (item) => item.network_object_id === props.networkObject?.id,
  ),
);
const configuredAttachmentPorts = computed(() => {
  const value = props.networkObject;
  if (!value)
    return [] as Array<{ name: string; pvid?: number; tagged?: number[] }>;
  const rows =
    value.kind === "switch_l2"
      ? value.config?.ports
      : value.kind === "switch_l3"
        ? value.config?.interfaces
        : [];
  if (!Array.isArray(rows)) return [];
  return rows
    .map((raw) => raw as { name?: string; pvid?: number; tagged?: number[] })
    .filter((row) => Boolean(row.name))
    .map((row) => ({
      name: String(row.name),
      pvid: row.pvid === undefined ? undefined : Number(row.pvid),
      tagged: Array.isArray(row.tagged) ? row.tagged.map(Number) : [],
    }));
});
function objectPorts(value?: NetworkObject) {
  if (!value) return [];
  const rows =
    value.kind === "switch_l2"
      ? value.config?.ports
      : value.kind === "switch_l3" || value.kind === "pc"
        ? value.config?.interfaces
        : [];
  return Array.isArray(rows)
    ? rows
        .map((raw) => String((raw as { name?: string }).name || ""))
        .filter(Boolean)
    : [];
}
const objectLinkPeers = computed(() =>
  props.networkObjects.filter(
    (item) =>
      item.id !== props.networkObject?.id &&
      ["pc", "switch_l2", "switch_l3"].includes(item.kind),
  ),
);
const objectLinkPeer = computed(() =>
  props.networkObjects.find((item) => item.id === objectLinkPeerId.value),
);
const objectLinkPeerPorts = computed(() => objectPorts(objectLinkPeer.value));
const objectLinkLocalPorts = computed(() => objectPorts(props.networkObject));
const attachmentRows = computed(() =>
  selectedObjectAttachments.value.map((attachment) => {
    const interfaceValue = props.interfaces.find(
      (item) => item.id === attachment.interface_id,
    );
    const nodeValue = props.nodes.find(
      (item) => item.id === interfaceValue?.node_id,
    );
    return {
      ...attachment,
      nodeName: nodeValue?.name || interfaceValue?.node_id || "未知节点",
      interfaceName: interfaceValue?.name || attachment.interface_id,
      driver: interfaceValue?.driver || "未知驱动",
    };
  }),
);
const natConfig = computed(() =>
  props.networkObject?.kind === "nat_bridge"
    ? props.networkObject.config
    : undefined,
);
const natGateway = computed(() => {
  const prefix = String(natConfig.value?.ipv4_prefix || "");
  const [address] = prefix.split("/");
  const octets = address.split(".").map(Number);
  if (
    octets.length !== 4 ||
    octets.some((value) => !Number.isInteger(value) || value < 0 || value > 255)
  )
    return "未配置";
  let numeric =
    ((octets[0] << 24) >>> 0) +
    (octets[1] << 16) +
    (octets[2] << 8) +
    octets[3] +
    1;
  numeric >>>= 0;
  return [24, 16, 8, 0].map((shift) => (numeric >>> shift) & 255).join(".");
});
const dhcpv4Summary = computed(() => {
  const value = natConfig.value?.dhcpv4 as
    { start?: string; end?: string; lease_time?: string } | undefined;
  if (!value?.start || !value.end) return "已禁用";
  return `${value.start} – ${value.end}${value.lease_time ? ` · ${value.lease_time}` : ""}`;
});
const dnsSummary = computed(() => {
  const values = natConfig.value?.dns_servers;
  return Array.isArray(values) && values.length
    ? values.join(", ")
    : "宿主机解析器";
});
const nodeInterfaces = computed(() =>
  props.node
    ? props.interfaces.filter((item) => item.node_id === props.node!.id)
    : [],
);
const selectedNodeRouteReadiness = computed(() =>
  props.node ? dockerRouteReadiness(props.node) : undefined,
);
const isRuijieNode = computed(() =>
  ["ruijie-router", "ruijie-switch"].includes(
    String(props.node?.config?.template_key || ""),
  ),
);
async function disconnect() {
  if (!props.link) return;
  await api.disconnectLink(props.link.id);
  emit("clear");
  emit("changed");
}
async function deleteObject() {
  if (!props.networkObject) return;
  await api.deleteNetworkObject(props.networkObject);
  emit("clear");
  emit("changed");
}
async function loadDiagnostics() {
  if (!props.networkObject) return;
  diagnosticsLoading.value = true;
  error.value = "";
  try {
    diagnostics.value = await api.getNetworkObjectDiagnostics(
      props.networkObject.id,
    );
    diagnosticsExpanded.value = true;
    emit("diagnosticsLoaded", props.networkObject.name);
  } catch (value) {
    error.value = value instanceof Error ? value.message : String(value);
  } finally {
    diagnosticsLoading.value = false;
  }
}
watch(
  () => props.networkObject?.id,
  () => {
    diagnostics.value = undefined;
    diagnosticsExpanded.value = false;
    error.value = "";
    attachNode.value = "";
    attachInterface.value = "";
    attachPortName.value = configuredAttachmentPorts.value[0]?.name || "";
    objectLinkPeerId.value = "";
    objectLinkLocalPort.value = objectLinkLocalPorts.value[0] || "";
    objectLinkPeerPort.value = "";
  },
);
watch(
  () => props.networkObjectLink?.id,
  async (id) => {
    objectLinkCapture.value = undefined;
    if (!id) return;
    try {
      objectLinkCapture.value = (await api.listCaptures(props.laboratoryId))
        .filter(
          (capture) =>
            capture.source_type === "network_object_link" &&
            capture.source_id === id,
        )
        .sort((left, right) =>
          right.created_at.localeCompare(left.created_at),
        )[0];
    } catch {
      objectLinkCapture.value = undefined;
    }
  },
  { immediate: true },
);
watch(objectLinkPeerId, () => {
  objectLinkPeerPort.value = objectLinkPeerPorts.value[0] || "";
});
watch(attachNode, () => {
  attachInterface.value = "";
});
watch(attachPortName, (name) => {
  const port = configuredAttachmentPorts.value.find(
    (item) => item.name === name,
  );
  attachPVID.value = port?.pvid ?? 1;
  attachTagged.value = (port?.tagged || []).join(",");
});
async function attach() {
  if (!props.networkObject || !attachInterface.value) return;
  const switchObject = ["switch_l2", "switch_l3"].includes(
    props.networkObject.kind,
  );
  if (switchObject && !attachPortName.value) return;
  await api.attachNetworkObject(props.networkObject.id, {
    interface_id: attachInterface.value,
    port_name: switchObject ? attachPortName.value : undefined,
    config:
      props.networkObject.kind === "switch_l2"
        ? {
            pvid: Number(attachPVID.value || 0),
            tagged: attachTagged.value
              .split(",")
              .map((item) => Number(item.trim()))
              .filter((item) => Number.isInteger(item) && item > 0),
          }
        : undefined,
  });
  attachNode.value = "";
  attachInterface.value = "";
  emit("changed");
}
async function createObjectLink() {
  if (
    !props.networkObject ||
    !objectLinkPeerId.value ||
    !objectLinkLocalPort.value ||
    !objectLinkPeerPort.value
  )
    return;
  error.value = "";
  try {
    objectLinkStatus.value = "正在提交对象链路任务…";
    const envelope = await api.createNetworkObjectLink(props.laboratoryId, {
      object_a_id: props.networkObject.id,
      port_a_name: objectLinkLocalPort.value,
      object_b_id: objectLinkPeerId.value,
      port_b_name: objectLinkPeerPort.value,
    });
    objectLinkStatus.value = `对象链路任务已提交 · ${envelope.task.id}`;
    emit("changed");
  } catch (value) {
    objectLinkStatus.value = "";
    error.value = value instanceof Error ? value.message : String(value);
  }
}
async function deleteAttachment() {
  if (!props.attachment) return;
  error.value = "";
  try {
    attachmentStatus.value = "正在提交附件删除任务…";
    const envelope = await api.deleteTopologyConnection(
      props.attachment.id,
      props.attachment.revision,
    );
    attachmentStatus.value = `附件删除任务已提交 · ${envelope.task.id}`;
    emit("clear");
    emit("changed");
  } catch (value) {
    attachmentStatus.value = "";
    error.value = value instanceof Error ? value.message : String(value);
  }
}
async function deleteObjectLink() {
  if (!props.networkObjectLink) return;
  error.value = "";
  try {
    objectLinkStatus.value = "正在提交对象链路删除任务…";
    const envelope = await api.deleteNetworkObjectLink(props.networkObjectLink);
    objectLinkStatus.value = `对象链路删除任务已提交 · ${envelope.task.id}`;
    emit("clear");
    emit("changed");
  } catch (value) {
    objectLinkStatus.value = "";
    error.value = value instanceof Error ? value.message : String(value);
  }
}
</script>
<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden">
    <header
      class="flex shrink-0 flex-wrap items-center gap-2 border-b border-border p-3"
      data-layout-region="inspector-header"
    >
      <Info :size="15" class="text-primary" />
      <h2 class="text-xs font-semibold uppercase tracking-wider">检查器</h2>
      <Button class="ml-auto" variant="ghost" size="sm" @click="$emit('clear')">
        清除选择
      </Button>
    </header>
    <div
      class="min-h-0 flex-1 overflow-x-hidden overflow-y-auto netlab-scrollbar"
      data-layout-region="inspector-content"
    >
      <template v-if="node">
        <section class="border-b border-border p-3">
          <div
            class="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-start gap-2"
          >
            <ResourceIdentity
              :id="node.id"
              type="节点"
              :name="node.name"
            /><StatusBadge :state="node.observed_state" />
          </div>
          <dl class="mt-3 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
            <dt>类型</dt>
            <dd>{{ node.kind }}</dd>
            <dt>期望状态</dt>
            <dd>{{ node.desired_state }}</dd>
            <dt>实际状态</dt>
            <dd>{{ node.observed_state }}</dd>
            <dt>修订版本</dt>
            <dd>{{ node.revision }}</dd>
            <dt>运行时标识</dt>
            <dd>{{ node.config?.runtime_id || "尚未分配" }}</dd>
          </dl>
          <StructuredProblem
            v-if="node.last_error"
            class="mt-3"
            :problem="node.last_error"
          />
          <div
            v-if="selectedNodeRouteReadiness?.state !== 'none'"
            class="mt-3 rounded-md border border-border bg-muted/30 p-3 text-xs"
            data-testid="inspector-docker-route-readiness"
          >
            <div class="flex items-center justify-between gap-2">
              <strong>Docker 路由</strong>
              <span
                :class="
                  selectedNodeRouteReadiness?.state === 'failed'
                    ? 'text-destructive'
                    : selectedNodeRouteReadiness?.state === 'applied'
                      ? 'text-[color:var(--success)]'
                      : 'text-[color:var(--warning)]'
                "
              >
                {{ selectedNodeRouteReadiness?.label }}
              </span>
            </div>
            <p class="mt-1 text-muted-foreground">
              {{ selectedNodeRouteReadiness?.routes.length }} 条声明：
              {{
                selectedNodeRouteReadiness?.routes
                  .map((route) => `${route.interfaceName}:${route.destination}`)
                  .join("、")
              }}
            </p>
            <p
              v-if="selectedNodeRouteReadiness?.state === 'failed'"
              class="mt-1 text-destructive"
            >
              请检查接口地址、网关可达性，并在修正后重试启动。
            </p>
          </div>
        </section>
        <RuijieConfigurationPanel
          v-if="isRuijieNode"
          :node="node"
          :interfaces="nodeInterfaces"
          @terminal="$emit('terminal', node)"
          @changed="$emit('changed')"
        />
        <FortiGateCredentialPanel
          v-if="
            String(node.config?.template_key || '').toLowerCase() ===
            'fortigate'
          "
          :node="node"
          @changed="$emit('changed')"
        />
        <div
          class="min-h-[240px] border-b border-border"
          data-layout-region="inspector-resource-chart"
        >
          <ResourceCharts :node="node" />
        </div>
        <NodeOperationsPanel
          :node="node"
          :interfaces="nodeInterfaces"
          @changed="$emit('changed')"
          @deleted="$emit('clear')"
        />
      </template>
      <template v-else-if="link">
        <section class="panel-section">
          <div class="flex items-center justify-between">
            <ResourceIdentity :id="link.id" type="链路" /><StatusBadge
              :state="link.observed_state"
            />
          </div>
          <dl>
            <dt>端点 A</dt>
            <dd :title="link.endpoint_a_id">{{ endpointAName }}</dd>
            <dt>端点 B</dt>
            <dd :title="link.endpoint_b_id">{{ endpointBName }}</dd>
            <dt>修订版本</dt>
            <dd>{{ link.revision }}</dd>
          </dl>
          <div class="mt-3 flex flex-wrap gap-2">
            <Button
              size="sm"
              variant="secondary"
              @click="emit('captureConnection')"
            >
              <Radio :size="14" /> 抓包
            </Button>
            <Button
              size="sm"
              variant="secondary"
              @click="emit('filterConnection')"
            >
              <Filter :size="14" /> 流量过滤
            </Button>
          </div>
          <Button
            class="mt-3"
            variant="destructive"
            size="sm"
            @click="confirmKind = 'link'"
          >
            <Cable :size="14" /> 实时断开链路
          </Button>
        </section>
      </template>
      <template v-else-if="networkObjectLink">
        <section class="panel-section">
          <div class="flex items-center justify-between gap-2">
            <ResourceIdentity :id="networkObjectLink.id" type="网络对象链路" />
            <StatusBadge :state="networkObjectLink.observed_state" />
          </div>
          <dl>
            <dt>端点 A</dt>
            <dd>
              {{ objectLinkA?.name || networkObjectLink.object_a_id }}:{{
                networkObjectLink.port_a_name
              }}
            </dd>
            <dt>端点 B</dt>
            <dd>
              {{ objectLinkB?.name || networkObjectLink.object_b_id }}:{{
                networkObjectLink.port_b_name
              }}
            </dd>
            <dt>期望状态</dt>
            <dd>{{ networkObjectLink.desired_state }}</dd>
            <dt>实际状态</dt>
            <dd>{{ networkObjectLink.observed_state }}</dd>
            <dt>修订版本</dt>
            <dd>{{ networkObjectLink.revision }}</dd>
            <template v-if="objectLinkTask">
              <dt>生命周期任务</dt>
              <dd>{{ objectLinkTask.id }} · {{ objectLinkTask.state }}</dd>
              <dt>任务进度</dt>
              <dd>
                {{ objectLinkTask.progress_current }} /
                {{ objectLinkTask.progress_total }}
              </dd>
            </template>
            <template v-if="objectLinkCapture">
              <dt>最近抓包</dt>
              <dd>
                {{ objectLinkCapture.id }} · {{ objectLinkCapture.state }}
              </dd>
              <dt>数据包 / 字节</dt>
              <dd>
                {{ objectLinkCapture.packets }} /
                {{ objectLinkCapture.bytes_written }}
              </dd>
              <dt>完成状态</dt>
              <dd>{{ objectLinkCapture.completion_reason || "进行中" }}</dd>
            </template>
          </dl>
          <div class="mt-3 flex flex-wrap gap-2">
            <Button
              size="sm"
              variant="secondary"
              @click="emit('captureConnection')"
            >
              <Radio :size="14" /> 抓包
            </Button>
            <Button
              size="sm"
              variant="secondary"
              @click="emit('filterConnection')"
            >
              <Filter :size="14" /> 流量过滤
            </Button>
          </div>
          <div
            v-if="objectLinkCapture"
            class="mt-3 flex flex-wrap gap-2 text-xs"
          >
            <a
              v-if="
                ['starting', 'running', 'streaming', 'requested'].includes(
                  objectLinkCapture.state,
                )
              "
              :href="objectLinkCaptureStream"
              class="text-primary underline"
              >实时流</a
            >
            <a
              v-if="objectLinkCapture.artifact_url"
              :href="objectLinkCapture.artifact_url"
              class="text-primary underline"
              >保留的抓包文件</a
            >
          </div>
          <StructuredProblem
            v-if="objectLinkTask?.error"
            class="mt-3"
            :problem="objectLinkTask.error"
          />
          <StructuredProblem
            v-if="networkObjectLink.last_error"
            class="mt-3"
            :problem="networkObjectLink.last_error"
          />
          <p
            class="mt-3 rounded border border-primary/30 bg-primary/5 p-2 text-xs text-muted-foreground"
          >
            该链路使用直接 veth pair 连接两个网络命名空间，可抓包、运行 Traffic
            Filter，并可在两端运行时实时删除。
          </p>
          <Button
            class="mt-3"
            variant="destructive"
            size="sm"
            @click="deleteObjectLink"
            ><Trash2 :size="14" /> 实时删除链路</Button
          >
        </section>
      </template>
      <template v-else-if="attachment">
        <section class="panel-section">
          <div class="flex items-center justify-between gap-2">
            <ResourceIdentity
              :id="attachment.id"
              type="网络附件"
              :name="`${attachmentNode?.name || '节点'}:${attachmentInterface?.name || attachment.interface_id} ↔ ${attachmentObject?.name || '轻量网络'}:${networkAttachmentPortLabel(attachment.port_name)}`"
            /><StatusBadge :state="attachment.observed_state" />
          </div>
          <dl>
            <dt>节点接口</dt>
            <dd>
              {{ attachmentNode?.name || "未知节点" }}:{{
                attachmentInterface?.name || attachment.interface_id
              }}
            </dd>
            <dt>网络端口</dt>
            <dd :title="attachment.port_name || undefined">
              {{ attachmentObject?.name || attachment.network_object_id }}:{{
                networkAttachmentPortLabel(attachment.port_name)
              }}
            </dd>
            <dt>监听接口</dt>
            <dd>{{ attachment.interface_id }}</dd>
          </dl>
          <div class="mt-3 flex flex-wrap gap-2">
            <Button
              size="sm"
              variant="secondary"
              @click="emit('captureConnection')"
            >
              <Radio :size="14" /> 抓包
            </Button>
            <Button
              size="sm"
              variant="secondary"
              @click="emit('filterConnection')"
            >
              <Filter :size="14" /> 流量过滤
            </Button>
          </div>
          <p
            class="mt-3 rounded border border-primary/30 bg-primary/5 p-2 text-xs text-muted-foreground"
          >
            此附件链路通过节点宿主接口监听。打开底部“抓包”或“流量过滤”，即可监控节点与轻量网络对象之间的流量。
          </p>
          <p v-if="attachmentStatus" class="mt-3 text-xs text-muted-foreground">
            {{ attachmentStatus }}
          </p>
          <Button
            class="mt-3"
            variant="destructive"
            size="sm"
            @click="deleteAttachment"
          >
            <Trash2 :size="14" /> 实时删除附件
          </Button>
        </section>
      </template>
      <template v-else-if="networkObject">
        <section class="panel-section">
          <div class="flex items-center justify-between">
            <ResourceIdentity
              :id="networkObject.id"
              type="网络对象"
              :name="networkObject.name"
            /><StatusBadge :state="networkObject.observed_state" />
          </div>
          <dl>
            <dt>类型</dt>
            <dd>{{ networkObject.kind }}</dd>
            <dt>期望状态</dt>
            <dd>{{ networkObject.desired_state }}</dd>
            <dt>实际状态</dt>
            <dd>{{ networkObject.observed_state }}</dd>
            <dt>修订版本</dt>
            <dd>{{ networkObject.revision }}</dd>
            <template v-if="networkObject.kind === 'nat_bridge'">
              <dt>IPv4 子网</dt>
              <dd>{{ natConfig?.ipv4_prefix || "未配置" }}</dd>
              <dt>网关</dt>
              <dd>{{ natGateway }}</dd>
              <dt>宿主机上联</dt>
              <dd>{{ natConfig?.uplink || "自动选择" }}</dd>
              <dt>DHCPv4</dt>
              <dd>{{ dhcpv4Summary }}</dd>
              <dt>DNS 上游</dt>
              <dd>{{ dnsSummary }}</dd>
              <dt>已连接接口</dt>
              <dd>{{ attachmentRows.length }}</dd>
            </template>
          </dl>
          <div
            v-if="networkObject.kind === 'nat_bridge'"
            class="mt-3 rounded-md border border-border bg-background/40 p-2"
          >
            <h3 class="text-[11px] font-semibold uppercase tracking-wide">
              已连接接口
            </h3>
            <p
              v-if="!attachmentRows.length"
              class="mt-2 text-xs text-muted-foreground"
            >
              尚未连接任何节点接口。
            </p>
            <div
              v-for="attachment in attachmentRows"
              :key="attachment.id"
              class="mt-2 flex items-start justify-between gap-2 rounded border border-border/70 p-2 text-xs"
            >
              <div class="min-w-0">
                <p class="truncate font-medium">{{ attachment.nodeName }}</p>
                <p class="truncate text-muted-foreground">
                  {{ attachment.interfaceName }} · {{ attachment.driver }}
                </p>
              </div>
              <StatusBadge :state="attachment.observed_state" />
            </div>
          </div>
          <div class="mt-3 flex gap-2">
            <Button
              v-if="networkObject.kind === 'pc'"
              size="sm"
              variant="secondary"
              @click="$emit('networkObjectTerminal', networkObject)"
            >
              终端
            </Button>
            <Button
              size="sm"
              :disabled="diagnosticsLoading"
              @click="loadDiagnostics"
            >
              <Database :size="14" />
              {{ diagnosticsLoading ? "正在加载…" : "运行诊断" }} </Button
            ><Button
              variant="destructive"
              size="sm"
              @click="confirmKind = 'network_object'"
            >
              <Trash2 :size="14" /> 删除
            </Button>
          </div>
          <section
            v-if="diagnostics"
            class="mt-3 overflow-hidden rounded border border-border bg-background/40"
          >
            <button
              type="button"
              class="flex w-full items-center justify-between px-3 py-2 text-left text-xs font-medium"
              :aria-expanded="diagnosticsExpanded"
              aria-controls="network-object-diagnostics"
              @click="diagnosticsExpanded = !diagnosticsExpanded"
            >
              <span>诊断结果</span>
              <ChevronUp v-if="diagnosticsExpanded" :size="14" />
              <ChevronDown v-else :size="14" />
            </button>
            <pre
              v-show="diagnosticsExpanded"
              id="network-object-diagnostics"
              aria-label="网络对象诊断结果"
              class="max-h-80 overflow-auto border-t border-border bg-background p-2 text-[10px]"
              >{{ JSON.stringify(diagnostics, null, 2) }}</pre>
          </section>
          <p v-if="error" role="alert" class="mt-2 text-xs text-destructive">
            {{ error }}
          </p>
        </section>
        <LightweightSwitchConfigurationPanel
          v-if="
            networkObject.kind === 'switch_l2' ||
            networkObject.kind === 'switch_l3'
          "
          :network-object="networkObject"
          @changed="$emit('changed')"
        />
        <LightweightPCConfigurationPanel
          v-if="networkObject.kind === 'pc'"
          :network-object="networkObject"
          @changed="$emit('changed')"
        />
        <section
          v-if="['pc', 'switch_l2', 'switch_l3'].includes(networkObject.kind)"
          class="panel-section"
        >
          <h3><Cable :size="13" class="inline" /> 连接轻量网络对象</h3>
          <div class="grid gap-2">
            <FormField label="本端端口"
              ><Select v-model="objectLinkLocalPort"
                ><option
                  v-for="port in objectLinkLocalPorts"
                  :key="port"
                  :value="port"
                >
                  {{ port }}
                </option></Select
              ></FormField
            >
            <FormField label="目标对象"
              ><Select v-model="objectLinkPeerId"
                ><option value="">选择目标</option>
                <option
                  v-for="item in objectLinkPeers"
                  :key="item.id"
                  :value="item.id"
                >
                  {{ item.name }} · {{ item.kind }}
                </option></Select
              ></FormField
            >
            <FormField label="目标端口"
              ><Select v-model="objectLinkPeerPort"
                ><option
                  v-for="port in objectLinkPeerPorts"
                  :key="port"
                  :value="port"
                >
                  {{ port }}
                </option></Select
              ></FormField
            >
            <Button
              size="sm"
              :disabled="
                !objectLinkPeerId || !objectLinkLocalPort || !objectLinkPeerPort
              "
              @click="createObjectLink"
              ><Cable :size="14" /> 创建对象间链路</Button
            >
            <p
              v-if="objectLinkStatus"
              role="status"
              class="text-xs text-muted-foreground"
            >
              {{ objectLinkStatus }}
            </p>
          </div>
        </section>
      </template>
      <EmptyState
        v-else
        title="尚未选择对象"
        description="请在拓扑图中选择节点、链路或网络对象，此处将显示详情和可执行操作。"
      />
      <ConfirmationDialog
        v-model="confirmOpen"
        :title="confirmKind === 'link' ? '断开链路' : '删除网络对象'"
        :resource="
          confirmKind === 'link'
            ? selectedLinkName
            : `${networkObject?.name || ''} · ${networkObject?.id || ''}`
        "
        :description="
          confirmKind === 'link'
            ? '实时链路将被移除，两端节点不会停止。'
            : '网络对象及其拥有的命名空间或网桥资源将被清理。'
        "
        impact="正在使用该资源的抓包、流量过滤或数据流可能会中断。"
        :confirm-label="confirmKind === 'link' ? '断开' : '删除对象'"
        @confirm="
          confirmKind === 'link' ? disconnect() : deleteObject();
          confirmKind = '';
        "
      />
      <section v-if="networkObject" class="panel-section">
        <h3>连接节点到此网络对象</h3>
        <p class="mb-2 text-xs text-muted-foreground">
          目标：{{ networkObject.name }}。用于连接 NAT、网桥或轻量交换机端口。
        </p>
        <FormField label="目标节点">
          <Select v-model="attachNode" aria-label="目标节点">
            <option value="">选择节点</option>
            <option v-for="item in nodes" :key="item.id" :value="item.id">
              {{ item.name }} · {{ item.kind }}
            </option>
          </Select> </FormField
        ><FormField label="目标接口">
          <Select
            v-model="attachInterface"
            aria-label="目标接口"
            :disabled="!attachNode"
          >
            <option value="">选择接口</option>
            <option
              v-for="item in attachNodeInterfaces"
              :key="item.id"
              :value="item.id"
            >
              {{ item.name }} · {{ item.driver }}
            </option>
          </Select> </FormField
        ><FormField
          v-if="
            networkObject?.kind === 'switch_l2' ||
            networkObject?.kind === 'switch_l3'
          "
          label="交换机端口"
        >
          <Select v-model="attachPortName" aria-label="交换机端口">
            <option value="">选择已配置端口</option>
            <option
              v-for="port in configuredAttachmentPorts"
              :key="port.name"
              :value="port.name"
            >
              {{ port.name }}
            </option>
          </Select> </FormField
        ><template v-if="networkObject?.kind === 'switch_l2'">
          <FormField label="接入 PVID">
            <Input v-model="attachPVID" type="number" min="0" max="4094" />
          </FormField>
          <FormField label="Tagged VLAN（带标签 VLAN）" hint="逗号分隔">
            <Input v-model="attachTagged" placeholder="10,20" />
          </FormField>
        </template>
        ><Button
          class="mt-2"
          size="sm"
          :disabled="
            !networkObject ||
            !attachNode ||
            !attachInterface ||
            ((networkObject.kind === 'switch_l2' ||
              networkObject.kind === 'switch_l3') &&
              !attachPortName)
          "
          @click="attach"
        >
          连接
        </Button>
      </section>
    </div>
  </div>
</template>
<style scoped>
.panel-section {
  border-bottom: 1px solid var(--border);
  padding: 1rem;
}
.panel-section h3 {
  margin-bottom: 0.65rem;
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--muted-foreground);
}
dl {
  margin-top: 0.75rem;
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 0.3rem 0.75rem;
  font-size: 0.75rem;
}
dd {
  min-width: 0;
  overflow-wrap: anywhere;
}
dt {
  color: var(--muted-foreground);
}
</style>
