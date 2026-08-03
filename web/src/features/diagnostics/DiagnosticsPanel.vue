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
} from "@/api";
import GlobalCaptureWorkspace from "./GlobalCaptureWorkspace.vue";
import TrafficFilterPanel from "./TrafficFilterPanel.vue";
import { linkDisplayName } from "@/features/topology/linkPresentation";
const props = withDefaults(
  defineProps<{
    nodeId?: string;
    laboratoryId?: string;
    interfaceId?: string;
    linkId?: string;
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
    const left = props.networkObjects?.find((item) => item.id === link.object_a_id);
    const right = props.networkObjects?.find((item) => item.id === link.object_b_id);
    labels[link.id] = `${left?.name || link.object_a_id}:${link.port_a_name} ↔ ${right?.name || link.object_b_id}:${link.port_b_name}`;
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
</script>
<template>
  <section class="h-full min-h-[180px]" aria-labelledby="diagnostics-title">
    <h2 id="diagnostics-title" class="sr-only">Diagnostics</h2>
    <GlobalConsoleWorkspace
      v-show="section === 'console'"
      :laboratory-id="laboratoryId"
      :nodes="nodes || []"
      :request-node-id="consoleRequestNodeId"
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
    /><TrafficFilterPanel
      v-if="trafficActivated"
      v-show="section === 'traffic-filter'"
      :laboratory-id="laboratoryId"
      :interface-id="interfaceId"
      :link-id="linkId"
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
