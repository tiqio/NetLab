<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { Cable, Database, Info, Network, Trash2 } from "lucide-vue-next";
import {
  api,
  type Link,
  type NetworkAttachment,
  type NetworkObject,
  type NetworkObjectLink,
  type Node,
  type NodeInterface,
} from "@/api";
import { Button, FormField, Input, Select } from "@/components/ui";
import EmptyState from "@/components/common/EmptyState.vue";
import ResourceIdentity from "@/components/common/ResourceIdentity.vue";
import StatusBadge from "@/components/common/StatusBadge.vue";
import StructuredProblem from "@/components/common/StructuredProblem.vue";
import NodeOperationsPanel from "@/features/nodes/NodeOperationsPanel.vue";
import { dockerRouteReadiness } from "@/features/nodes/dockerRouteReadiness";
import RuijieConfigurationPanel from "@/features/nodes/RuijieConfigurationPanel.vue";
import LightweightNodeEditor from "@/features/nodes/LightweightNodeEditor.vue";
import LightweightSwitchConfigurationPanel from "@/features/nodes/LightweightSwitchConfigurationPanel.vue";
import ResourceCharts from "@/features/analytics/ResourceCharts.vue";
import ConfirmationDialog from "@/components/common/ConfirmationDialog.vue";
import { linkDisplayName, linkEndpointName } from "./linkPresentation";

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
    showLightweight?: boolean;
    diagnosticsRequestKey?: number;
  }>(),
  {
    nodes: () => [],
    networkObjects: () => [],
    attachments: () => [],
    networkObjectLinks: () => [],
  },
);
const emit = defineEmits<{
  changed: [];
  clear: [];
  lightweightCreated: [string];
  diagnosticsLoaded: [string];
  terminal: [Node];
}>();
const endpointA = ref("");
const endpointB = ref("");
const attachNode = ref("");
const attachInterface = ref("");
const attachPortName = ref("");
const attachPVID = ref(1);
const attachTagged = ref("");
const objectLinkPeerId = ref("");
const objectLinkLocalPort = ref("");
const objectLinkPeerPort = ref("");
const objectLinkStatus = ref("");
const diagnostics = ref<Record<string, unknown>>();
const diagnosticsLoading = ref(false);
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
      nodeName: nodeValue?.name || interfaceValue?.node_id || "Unknown node",
      interfaceName: interfaceValue?.name || attachment.interface_id,
      driver: interfaceValue?.driver || "unknown driver",
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
    return "not configured";
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
  if (!value?.start || !value.end) return "disabled";
  return `${value.start} – ${value.end}${value.lease_time ? ` · ${value.lease_time}` : ""}`;
});
const dnsSummary = computed(() => {
  const values = natConfig.value?.dns_servers;
  return Array.isArray(values) && values.length
    ? values.join(", ")
    : "host resolver";
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
async function connect() {
  if (
    !endpointA.value ||
    !endpointB.value ||
    endpointA.value === endpointB.value
  )
    return;
  try {
    await api.connectLink(props.laboratoryId, endpointA.value, endpointB.value);
    endpointA.value = "";
    endpointB.value = "";
    emit("changed");
  } catch (value) {
    error.value = value instanceof Error ? value.message : String(value);
  }
}
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
    emit("diagnosticsLoaded", props.networkObject.name);
  } catch (value) {
    error.value = value instanceof Error ? value.message : String(value);
  } finally {
    diagnosticsLoading.value = false;
  }
}
watch(
  () => props.diagnosticsRequestKey,
  (value, previous) => {
    if (value && value !== previous) void loadDiagnostics();
  },
);
watch(
  () => props.networkObject?.id,
  () => {
    diagnostics.value = undefined;
    error.value = "";
    attachNode.value = "";
    attachInterface.value = "";
    attachPortName.value = configuredAttachmentPorts.value[0]?.name || "";
    objectLinkPeerId.value = "";
    objectLinkLocalPort.value = objectLinkLocalPorts.value[0] || "";
    objectLinkPeerPort.value = "";
  },
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
async function deleteObjectLink() {
  if (!props.networkObjectLink) return;
  await api.deleteNetworkObjectLink(props.networkObjectLink.id);
  emit("clear");
  emit("changed");
}
</script>
<template>
  <div class="min-h-full">
    <header class="flex items-center gap-2 border-b border-border p-3">
      <Info :size="15" class="text-primary" />
      <h2 class="text-xs font-semibold uppercase tracking-wider">Inspector</h2>
      <Button class="ml-auto" variant="ghost" size="sm" @click="$emit('clear')">
        Clear
      </Button>
    </header>
    <LightweightNodeEditor
      v-if="showLightweight"
      :laboratory-id="laboratoryId"
      class="legacy-lightweight"
      @created="$emit('lightweightCreated', $event)"
    />
    <template v-else-if="node">
      <section class="border-b border-border p-3">
        <div class="flex items-center justify-between gap-2">
          <ResourceIdentity
            :id="node.id"
            type="node"
            :name="node.name"
          /><StatusBadge :state="node.observed_state" />
        </div>
        <dl class="mt-3 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
          <dt>Kind</dt>
          <dd>{{ node.kind }}</dd>
          <dt>Desired</dt>
          <dd>{{ node.desired_state }}</dd>
          <dt>Observed</dt>
          <dd>{{ node.observed_state }}</dd>
          <dt>Revision</dt>
          <dd>{{ node.revision }}</dd>
          <dt>Runtime</dt>
          <dd>{{ node.config?.runtime_id || "not assigned" }}</dd>
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
                    ? 'text-emerald-300'
                    : 'text-amber-300'
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
      <div class="h-56 border-b border-border">
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
          <ResourceIdentity :id="link.id" type="link" /><StatusBadge
            :state="link.observed_state"
          />
        </div>
        <dl>
          <dt>Endpoint A</dt>
          <dd :title="link.endpoint_a_id">{{ endpointAName }}</dd>
          <dt>Endpoint B</dt>
          <dd :title="link.endpoint_b_id">{{ endpointBName }}</dd>
          <dt>Revision</dt>
          <dd>{{ link.revision }}</dd>
        </dl>
        <Button
          class="mt-3"
          variant="destructive"
          size="sm"
          @click="confirmKind = 'link'"
        >
          <Cable :size="14" /> Disconnect live link
        </Button>
      </section>
    </template>
    <template v-else-if="networkObjectLink">
      <section class="panel-section">
        <div class="flex items-center justify-between gap-2">
          <ResourceIdentity
            :id="networkObjectLink.id"
            type="network object link"
          />
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
          <dt>Revision</dt>
          <dd>{{ networkObjectLink.revision }}</dd>
        </dl>
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
            type="network attachment"
            :name="`${attachmentNode?.name || 'Node'}:${attachmentInterface?.name || attachment.interface_id} ↔ ${attachmentObject?.name || 'Lightweight network'}:${attachment.port_name || 'port'}`"
          /><StatusBadge :state="attachment.observed_state" />
        </div>
        <dl>
          <dt>节点接口</dt>
          <dd>
            {{ attachmentNode?.name || "Unknown node" }}:{{
              attachmentInterface?.name || attachment.interface_id
            }}
          </dd>
          <dt>网络端口</dt>
          <dd>
            {{ attachmentObject?.name || attachment.network_object_id }}:{{
              attachment.port_name || "port"
            }}
          </dd>
          <dt>监听接口</dt>
          <dd>{{ attachment.interface_id }}</dd>
        </dl>
        <p
          class="mt-3 rounded border border-primary/30 bg-primary/5 p-2 text-xs text-muted-foreground"
        >
          此附件链路通过节点宿主接口监听。打开底部 Capture 或 Traffic
          Filter，即可监控节点与 Lightweight 网络对象之间的流量。
        </p>
      </section>
    </template>
    <template v-else-if="networkObject">
      <section class="panel-section">
        <div class="flex items-center justify-between">
          <ResourceIdentity
            :id="networkObject.id"
            type="network object"
            :name="networkObject.name"
          /><StatusBadge :state="networkObject.observed_state" />
        </div>
        <dl>
          <dt>Kind</dt>
          <dd>{{ networkObject.kind }}</dd>
          <dt>Desired</dt>
          <dd>{{ networkObject.desired_state }}</dd>
          <dt>Runtime</dt>
          <dd>{{ networkObject.observed_state }}</dd>
          <dt>Revision</dt>
          <dd>{{ networkObject.revision }}</dd>
          <template v-if="networkObject.kind === 'nat_bridge'">
            <dt>IPv4 subnet</dt>
            <dd>{{ natConfig?.ipv4_prefix || "not configured" }}</dd>
            <dt>Gateway</dt>
            <dd>{{ natGateway }}</dd>
            <dt>Host uplink</dt>
            <dd>{{ natConfig?.uplink || "auto" }}</dd>
            <dt>DHCPv4</dt>
            <dd>{{ dhcpv4Summary }}</dd>
            <dt>DNS upstream</dt>
            <dd>{{ dnsSummary }}</dd>
            <dt>Attachments</dt>
            <dd>{{ attachmentRows.length }}</dd>
          </template>
        </dl>
        <div
          v-if="networkObject.kind === 'nat_bridge'"
          class="mt-3 rounded-md border border-border bg-background/40 p-2"
        >
          <h3 class="text-[11px] font-semibold uppercase tracking-wide">
            Attached interfaces
          </h3>
          <p
            v-if="!attachmentRows.length"
            class="mt-2 text-xs text-muted-foreground"
          >
            No node interfaces are attached.
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
            size="sm"
            :disabled="diagnosticsLoading"
            @click="loadDiagnostics"
          >
            <Database :size="14" />
            {{ diagnosticsLoading ? "Loading…" : "Diagnostics" }} </Button
          ><Button
            variant="destructive"
            size="sm"
            @click="confirmKind = 'network_object'"
          >
            <Trash2 :size="14" /> Delete
          </Button>
        </div>
        <pre
          v-if="diagnostics"
          aria-label="Network diagnostics"
          class="mt-3 overflow-auto rounded bg-background p-2 text-[10px]"
          >{{ JSON.stringify(diagnostics, null, 2) }}</pre>
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
      <section
        v-if="['pc', 'switch_l2', 'switch_l3'].includes(networkObject.kind)"
        class="panel-section"
      >
        <h3><Cable :size="13" class="inline" /> 连接 Lightweight 网络对象</h3>
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
      title="Nothing selected"
      description="Select a node, link, or network object on the topology. Details and valid actions appear here."
    />
    <section class="panel-section">
      <h3><Network :size="13" class="inline" /> Create link</h3>
      <div class="grid gap-2">
        <FormField label="Interface A">
          <Select v-model="endpointA" aria-label="Endpoint A">
            <option value="">Choose an available interface</option>
            <option
              v-for="item in availableInterfaces"
              :key="item.id"
              :value="item.id"
            >
              {{ item.name }} · {{ item.node_id }}
            </option>
          </Select> </FormField
        ><FormField label="Interface B">
          <Select v-model="endpointB" aria-label="Endpoint B">
            <option value="">Choose a different interface</option>
            <option
              v-for="item in availableInterfaces"
              :key="item.id"
              :value="item.id"
              :disabled="item.id === endpointA"
            >
              {{ item.name }} · {{ item.node_id }}
            </option>
          </Select> </FormField
        ><Button
          size="sm"
          :disabled="!endpointA || !endpointB || endpointA === endpointB"
          @click="connect"
        >
          Connect live
        </Button>
      </div>
      <p v-if="error" role="alert" class="mt-2 text-xs text-destructive">
        {{ error }}
      </p>
    </section>
    <ConfirmationDialog
      v-model="confirmOpen"
      :title="
        confirmKind === 'link' ? 'Disconnect link' : 'Delete network object'
      "
      :resource="
        confirmKind === 'link'
          ? selectedLinkName
          : `${networkObject?.name || ''} · ${networkObject?.id || ''}`
      "
      :description="
        confirmKind === 'link'
          ? 'The live connection will be removed without stopping either endpoint.'
          : 'The network object and owned namespace or bridge resources will be cleaned up.'
      "
      impact="Active captures, Traffic Filters, or streams using this resource may be interrupted."
      :confirm-label="confirmKind === 'link' ? 'Disconnect' : 'Delete object'"
      @confirm="
        confirmKind === 'link' ? disconnect() : deleteObject();
        confirmKind = '';
      "
    />
    <section class="panel-section">
      <h3>Attach network object</h3>
      <p class="mb-2 text-xs text-muted-foreground">
        Target: {{ networkObject?.name || "Select a network object first" }}
      </p>
      <FormField label="Target node">
        <Select
          v-model="attachNode"
          aria-label="Target node"
          :disabled="!networkObject"
        >
          <option value="">Select node</option>
          <option v-for="item in nodes" :key="item.id" :value="item.id">
            {{ item.name }} · {{ item.kind }}
          </option>
        </Select> </FormField
      ><FormField label="Target interface">
        <Select
          v-model="attachInterface"
          aria-label="Target interface"
          :disabled="!attachNode"
        >
          <option value="">Select interface</option>
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
        <Select v-model="attachPortName" aria-label="Switch port">
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
        <FormField label="Tagged VLAN" hint="逗号分隔">
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
        Attach
      </Button>
    </section>
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
dt {
  color: var(--muted-foreground);
}
.legacy-lightweight {
  padding: 1rem;
}
.legacy-lightweight :deep(label),
.legacy-lightweight :deep(fieldset) {
  display: grid;
  gap: 0.35rem;
  margin: 0.5rem 0;
}
.legacy-lightweight :deep(input),
.legacy-lightweight :deep(select) {
  height: 2rem;
  border: 1px solid var(--input);
  border-radius: 0.3rem;
  background: var(--background);
  padding: 0 0.5rem;
}
.legacy-lightweight :deep(button) {
  margin: 0.25rem;
  padding: 0.4rem 0.6rem;
  border-radius: 0.3rem;
  background: var(--secondary);
}
</style>
