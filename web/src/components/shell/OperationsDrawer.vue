<script setup lang="ts">
import { computed } from "vue";
import Tabs from "@/components/ui/tabs/Tabs.vue";
import TaskCenter from "@/features/tasks/TaskCenter.vue";
import DiagnosticsPanel from "@/features/diagnostics/DiagnosticsPanel.vue";
import type {
  Link,
  NetworkAttachment,
  NetworkObject,
  NetworkObjectLink,
  OperationTask,
  Node,
  NodeInterface,
  TrafficObservation,
} from "@/api";
import type { BottomTab } from "@/types/workspace";
const props = defineProps<{
  modelValue: BottomTab;
  tasks: OperationTask[];
  laboratoryId?: string;
  selectedNodeId?: string;
  selectedInterface?: NodeInterface;
  selectedLinkId?: string;
  selectedObjectLinkId?: string;
  selectedNetworkObjectId?: string;
  interfaceOwners?: Record<string, string>;
  coordinates?: Record<string, { x: number; y: number }>;
  resourceIds?: string[];
  nodes?: Node[];
  interfaces?: NodeInterface[];
  links?: Link[];
  attachments?: NetworkAttachment[];
  networkObjectLinks?: NetworkObjectLink[];
  networkObjects?: NetworkObject[];
  consoleRequestNodeId?: string;
  consoleRequestNetworkObjectId?: string;
  consoleRequestKey?: number;
}>();
const emit = defineEmits<{
  "update:modelValue": [BottomTab];
  refreshTasks: [];
  navigate: [string, string];
  trafficOverlay: [TrafficObservation[], boolean, string];
  captureOverlay: [{ connectionIds: string[]; interfaceIds: string[] }];
  reconcileNetworkObject: [NetworkObject];
  reconcileNetworkObjectLink: [NetworkObjectLink];
  deleteNetworkObjectLink: [NetworkObjectLink];
  setNodeForwarding: [Node, boolean, boolean];
}>();
const value = computed({
  get: () => props.modelValue,
  set: (item) => emit("update:modelValue", item as BottomTab),
});
function setNodeForwarding(node: Node, ipv4: boolean, ipv6: boolean) {
  emit("setNodeForwarding", node, ipv4, ipv6);
}
</script>
<template>
  <Tabs
    v-model="value"
    :tabs="[
      { value: 'tasks', label: `任务 (${tasks.length})` },
      { value: 'console', label: '终端' },
      { value: 'captures', label: '抓包' },
      { value: 'traffic-filter', label: '流量过滤' },
    ]"
    class="flex h-full min-h-0 flex-col overflow-hidden"
    data-layout-region="operations-drawer"
  >
    <template #default="{ value: active }">
      <div
        class="min-h-0 flex-1 overflow-auto overscroll-contain netlab-scrollbar"
        data-layout-region="operations-content"
      >
        <TaskCenter
          v-show="active === 'tasks'"
          :tasks="tasks"
          :laboratory-id="laboratoryId"
          :resource-ids="resourceIds"
          @refresh="$emit('refreshTasks')"
          @navigate="(type, id) => $emit('navigate', type, id)"
        /><DiagnosticsPanel
          v-show="active !== 'tasks'"
          :node-id="selectedNodeId"
          :laboratory-id="laboratoryId"
          :interface-id="selectedInterface?.id"
          :link-id="selectedLinkId"
          :object-link-id="selectedObjectLinkId"
          :network-object-id="selectedNetworkObjectId"
          :interface-owners="interfaceOwners"
          :coordinates="coordinates"
          :initial-section="active"
          :nodes="nodes"
          :interfaces="interfaces"
          :links="links"
          :attachments="attachments"
          :network-object-links="networkObjectLinks"
          :network-objects="networkObjects"
          :console-request-node-id="consoleRequestNodeId"
          :console-request-network-object-id="consoleRequestNetworkObjectId"
          :console-request-key="consoleRequestKey"
          @traffic-overlay="(...args) => $emit('trafficOverlay', ...args)"
          @capture-overlay="$emit('captureOverlay', $event)"
          @reconcile-network-object="$emit('reconcileNetworkObject', $event)"
          @reconcile-network-object-link="
            $emit('reconcileNetworkObjectLink', $event)
          "
          @delete-network-object-link="$emit('deleteNetworkObjectLink', $event)"
          @set-node-forwarding="setNodeForwarding"
        />
      </div>
    </template>
  </Tabs>
</template>
