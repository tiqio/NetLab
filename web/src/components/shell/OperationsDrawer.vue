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
  consoleRequestKey?: number;
}>();
const emit = defineEmits<{
  "update:modelValue": [BottomTab];
  refreshTasks: [];
  navigate: [string, string];
  trafficOverlay: [TrafficObservation[], boolean, string];
}>();
const value = computed({
  get: () => props.modelValue,
  set: (item) => emit("update:modelValue", item as BottomTab),
});
</script>
<template>
  <Tabs
    v-model="value"
    :tabs="[
      { value: 'tasks', label: `Tasks (${tasks.length})` },
      { value: 'console', label: 'Console' },
      { value: 'captures', label: 'Capture' },
      { value: 'traffic-filter', label: 'Traffic Filter' },
    ]"
    class="flex h-full flex-col"
  >
    <template #default="{ value: active }">
      <div class="min-h-0 flex-1 overflow-auto">
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
          :console-request-key="consoleRequestKey"
          @traffic-overlay="(...args) => $emit('trafficOverlay', ...args)"
        />
      </div>
    </template>
  </Tabs>
</template>
