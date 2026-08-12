<script setup lang="ts">
import { computed, ref, watch } from "vue";
import GlobalConsoleWorkspace from "./GlobalConsoleWorkspace.vue";
import type {
  Link,
  NetworkAttachment,
  NetworkObject,
  NetworkObjectLink,
  Node,
  NodeInterface,
  TrafficObservation,
  NetworkObjectDiagnostics,
  NodeNetworkDiagnostics,
  Problem,
} from "@/api";
import { api } from "@/api";
import { Button } from "@/components/ui";
import GlobalCaptureWorkspace from "./GlobalCaptureWorkspace.vue";
import TrafficFilterPanel from "./TrafficFilterPanel.vue";
import DeviceReadinessPanel from "./DeviceReadinessPanel.vue";
import { linkDisplayName } from "@/features/topology/linkPresentation";
const props = withDefaults(
  defineProps<{
    nodeId?: string;
    laboratoryId?: string;
    interfaceId?: string;
    linkId?: string;
    objectLinkId?: string;
    networkObjectId?: string;
    interfaceOwners?: Record<string, string>;
    coordinates?: Record<string, { x: number; y: number }>;
    initialSection?: string;
    nodes?: Node[];
    interfaces?: NodeInterface[];
    links?: Link[];
    attachments?: NetworkAttachment[];
    networkObjectLinks?: NetworkObjectLink[];
    networkObjects?: NetworkObject[];
    consoleRequestNodeId?: string;
    consoleRequestNetworkObjectId?: string;
    consoleRequestKey?: number;
  }>(),
  {
    initialSection: "console",
    laboratoryId: undefined,
    interfaceId: undefined,
    linkId: undefined,
  },
);
const emit = defineEmits<{
  trafficOverlay: [TrafficObservation[], boolean, string];
  captureOverlay: [{ connectionIds: string[]; interfaceIds: string[] }];
  reconcileNetworkObject: [NetworkObject];
  reconcileNetworkObjectLink: [NetworkObjectLink];
  deleteNetworkObjectLink: [NetworkObjectLink];
  setNodeForwarding: [Node, boolean, boolean];
}>();
const section = computed(() =>
  props.initialSection === "captures"
    ? "capture"
    : props.initialSection === "traffic-filter"
      ? "traffic-filter"
      : "console",
);
const captureActivated = ref(false);
const trafficActivated = ref(false);
const recoveryDiagnostics = ref<NetworkObjectDiagnostics>();
const recoveryLoading = ref(false);
const recoveryLoadError = ref("");
const nodeNetworkDiagnostics = ref<NodeNetworkDiagnostics>();
const nodeDiagnosticsLoading = ref(false);
const selectedNode = computed(() =>
  props.nodes?.find((item) => item.id === props.nodeId),
);
const recoveryObject = computed(() =>
  props.networkObjects?.find((item) => item.id === props.networkObjectId),
);
const recoveryObjectLink = computed(() =>
  props.networkObjectLinks?.find((item) => item.id === props.objectLinkId),
);
const recoveryProblem = computed<Problem | undefined>(
  () =>
    recoveryDiagnostics.value?.backing?.problem ||
    recoveryObject.value?.last_error ||
    recoveryObjectLink.value?.last_error,
);
const networkObjectMismatches = computed(() => {
  const runtime = recoveryDiagnostics.value?.runtime as
    { mismatches?: string[] } | undefined;
  return runtime?.mismatches || [];
});
const vlanDiagnostics = computed(() => {
  if (recoveryObject.value?.kind !== "switch_l2") return undefined;
  const runtime = recoveryDiagnostics.value?.runtime as
    | {
        desired?: {
          vlan_filtering?: boolean;
          ports?: Array<{ name: string; pvid?: number; tagged?: number[] }>;
        };
        observed?: {
          ports?: Array<{ name: string; pvid?: number; tagged?: number[] }>;
        };
      }
    | undefined;
  return runtime;
});
async function loadRecoveryDiagnostics() {
  recoveryDiagnostics.value = undefined;
  recoveryLoadError.value = "";
  if (!props.networkObjectId) return;
  recoveryLoading.value = true;
  try {
    recoveryDiagnostics.value = await api.getNetworkObjectDiagnostics(
      props.networkObjectId,
    );
  } catch (error) {
    recoveryLoadError.value =
      error instanceof Error ? error.message : String(error);
  } finally {
    recoveryLoading.value = false;
  }
}
async function loadNodeNetworkDiagnostics() {
  nodeNetworkDiagnostics.value = undefined;
  if (!props.nodeId || selectedNode.value?.kind !== "docker") return;
  nodeDiagnosticsLoading.value = true;
  try {
    nodeNetworkDiagnostics.value = await api.getNodeNetworkDiagnostics(
      props.nodeId,
    );
  } catch {
    nodeNetworkDiagnostics.value = undefined;
  } finally {
    nodeDiagnosticsLoading.value = false;
  }
}
const resourceLabels = computed(() => {
  const labels: Record<string, string> = {};
  for (const node of props.nodes || []) labels[node.id] = node.name;
  for (const link of props.links || []) {
    labels[link.id] = linkDisplayName(
      link,
      props.interfaces || [],
      props.nodes || [],
    );
  }
  for (const link of props.networkObjectLinks || []) {
    const left = props.networkObjects?.find(
      (item) => item.id === link.object_a_id,
    );
    const right = props.networkObjects?.find(
      (item) => item.id === link.object_b_id,
    );
    labels[link.id] =
      `${left?.name || link.object_a_id}:${link.port_a_name} ↔ ${right?.name || link.object_b_id}:${link.port_b_name}`;
  }
  return labels;
});
watch(
  section,
  (value) => {
    if (value === "capture") captureActivated.value = true;
    if (value === "traffic-filter") trafficActivated.value = true;
  },
  { immediate: true },
);
watch(
  () => [props.networkObjectId, recoveryObject.value?.observed_state],
  loadRecoveryDiagnostics,
  { immediate: true },
);
watch(
  () => [props.nodeId, selectedNode.value?.observed_state],
  loadNodeNetworkDiagnostics,
  { immediate: true },
);
</script>
<template>
  <section class="h-full min-h-[180px]" aria-labelledby="diagnostics-title">
    <h2 id="diagnostics-title" class="sr-only">诊断</h2>
    <DeviceReadinessPanel v-if="selectedNode && selectedNode.kind === 'qemu'" :node="selectedNode" :interfaces="interfaces" />
    <section
      v-if="
        recoveryObject || recoveryObjectLink || selectedNode?.kind === 'docker'
      "
      data-testid="recovery-diagnostics"
      class="m-3 rounded-lg border border-border bg-card p-3 text-sm"
      aria-label="恢复诊断"
    >
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div>
          <strong>{{
            recoveryObject?.name || selectedNode?.name || "对象链路"
          }}</strong>
          <p class="text-xs text-muted-foreground">
            期望：{{
              recoveryObject?.desired_state || recoveryObjectLink?.desired_state
            }}
            · 实际：{{
              recoveryObject?.observed_state ||
              recoveryObjectLink?.observed_state
            }}
          </p>
        </div>
        <div class="flex gap-2">
          <Button
            v-if="recoveryObject"
            size="sm"
            variant="secondary"
            :disabled="recoveryLoading"
            @click="emit('reconcileNetworkObject', recoveryObject)"
            >重试恢复</Button
          >
          <Button
            v-if="recoveryObjectLink"
            size="sm"
            variant="secondary"
            @click="emit('reconcileNetworkObjectLink', recoveryObjectLink)"
            >重试连线</Button
          >
          <Button
            v-if="recoveryObjectLink"
            size="sm"
            variant="destructive"
            @click="emit('deleteNetworkObjectLink', recoveryObjectLink)"
            >删除连线</Button
          >
          <Button
            v-if="selectedNode?.kind === 'docker'"
            size="sm"
            variant="secondary"
            :disabled="
              nodeDiagnosticsLoading ||
              selectedNode.observed_state !== 'stopped'
            "
            @click="emit('setNodeForwarding', selectedNode, true, true)"
            >启用双栈转发</Button
          >
          <Button
            v-if="selectedNode?.kind === 'docker'"
            size="sm"
            variant="ghost"
            :disabled="
              nodeDiagnosticsLoading ||
              selectedNode.observed_state !== 'stopped'
            "
            @click="emit('setNodeForwarding', selectedNode, false, false)"
            >关闭转发</Button
          >
        </div>
      </div>
      <dl
        v-if="recoveryDiagnostics?.backing"
        class="mt-3 grid gap-1 text-xs sm:grid-cols-2"
      >
        <div>运行承载：{{ recoveryDiagnostics.backing.backing_kind }}</div>
        <div>名称：{{ recoveryDiagnostics.backing.runtime_name || "—" }}</div>
        <div>
          归属：{{ recoveryDiagnostics.backing.owned ? "已确认" : "未确认" }}
        </div>
        <div>可用：{{ recoveryDiagnostics.backing.usable ? "是" : "否" }}</div>
      </dl>
      <dl
        v-if="nodeNetworkDiagnostics"
        class="mt-3 grid gap-1 text-xs sm:grid-cols-2"
      >
        <div>
          IPv4 转发：{{ nodeNetworkDiagnostics.desired?.forward_ipv4 }} /
          {{ nodeNetworkDiagnostics.observed?.forward_ipv4 }}
        </div>
        <div>
          IPv6 转发：{{ nodeNetworkDiagnostics.desired?.forward_ipv6 }} /
          {{ nodeNetworkDiagnostics.observed?.forward_ipv6 }}
        </div>
      </dl>
      <div
        v-if="vlanDiagnostics"
        class="mt-3 overflow-x-auto rounded border border-border"
        data-testid="vlan-diagnostics"
      >
        <p class="border-b border-border px-2 py-1 text-xs">
          VLAN Filtering：{{
            vlanDiagnostics.desired?.vlan_filtering ? "启用" : "关闭"
          }}
        </p>
        <table class="w-full text-left text-xs">
          <thead>
            <tr class="text-muted-foreground">
              <th class="px-2 py-1">端口</th>
              <th class="px-2 py-1">期望 PVID / Tagged</th>
              <th class="px-2 py-1">实际 PVID / Tagged</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="port in vlanDiagnostics.desired?.ports || []"
              :key="port.name"
              class="border-t border-border"
            >
              <td class="px-2 py-1">{{ port.name }}</td>
              <td class="px-2 py-1">
                {{ port.pvid || 0 }} / {{ port.tagged?.join(",") || "—" }}
              </td>
              <td class="px-2 py-1">
                {{
                  vlanDiagnostics.observed?.ports?.find(
                    (item) => item.name === port.name,
                  )?.pvid || 0
                }}
                /
                {{
                  vlanDiagnostics.observed?.ports
                    ?.find((item) => item.name === port.name)
                    ?.tagged?.join(",") || "—"
                }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <ul
        v-if="
          networkObjectMismatches.length ||
          nodeNetworkDiagnostics?.mismatches?.length
        "
        role="alert"
        class="mt-3 list-disc pl-5 text-xs text-destructive"
      >
        <li
          v-for="item in [
            ...networkObjectMismatches,
            ...(nodeNetworkDiagnostics?.mismatches || []),
          ]"
          :key="item"
        >
          {{ item }}
        </li>
      </ul>
      <div
        v-if="recoveryProblem"
        role="alert"
        class="mt-3 rounded border border-destructive/40 p-2 text-xs"
      >
        <strong>{{ recoveryProblem.message }}</strong>
        <p v-if="recoveryProblem.phase">阶段：{{ recoveryProblem.phase }}</p>
        <p v-if="recoveryProblem.cleanup">
          清理：{{ recoveryProblem.cleanup }}
        </p>
        <p v-if="recoveryProblem.operator_hint">
          建议：{{ recoveryProblem.operator_hint }}
        </p>
      </div>
      <p
        v-else-if="recoveryLoadError"
        role="alert"
        class="mt-2 text-xs text-destructive"
      >
        {{ recoveryLoadError }}
      </p>
    </section>
    <GlobalConsoleWorkspace
      v-show="section === 'console'"
      :laboratory-id="laboratoryId"
      :nodes="nodes || []"
      :network-objects="networkObjects || []"
      :request-node-id="consoleRequestNodeId"
      :request-network-object-id="consoleRequestNetworkObjectId"
      :request-key="consoleRequestKey"
    />
    <GlobalCaptureWorkspace
      v-if="captureActivated"
      v-show="section === 'capture'"
      :laboratory-id="laboratoryId"
      :nodes="nodes || []"
      :interfaces="interfaces || []"
      :links="links || []"
      :network-object-links="networkObjectLinks || []"
      :network-objects="networkObjects || []"
      :request-interface-id="section === 'capture' ? interfaceId : undefined"
      :request-link-id="section === 'capture' ? linkId : undefined"
      :request-object-link-id="section === 'capture' ? objectLinkId : undefined"
      @capture-overlay="$emit('captureOverlay', $event)"
    /><TrafficFilterPanel
      v-if="trafficActivated"
      v-show="section === 'traffic-filter'"
      :laboratory-id="laboratoryId"
      :interface-id="interfaceId"
      :link-id="linkId"
      :object-link-id="objectLinkId"
      :nodes="nodes || []"
      :interfaces="interfaces || []"
      :links="links || []"
      :attachments="attachments || []"
      :network-object-links="networkObjectLinks || []"
      :network-objects="networkObjects || []"
      :interface-owners="interfaceOwners"
      :coordinates="coordinates"
      :resource-labels="resourceLabels"
      @overlay="(...args) => $emit('trafficOverlay', ...args)"
    />
  </section>
</template>
