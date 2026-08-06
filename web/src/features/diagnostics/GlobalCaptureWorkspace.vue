<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { Cable, Radio } from "lucide-vue-next";
import type {
  CaptureSession,
  Link,
  NetworkObject,
  NetworkObjectLink,
  Node,
  NodeInterface,
} from "@/api";
import { Button } from "@/components/ui";
import {
  linkDisplayName,
  networkObjectLinkDisplayName,
} from "@/features/topology/linkPresentation";
import CapturePanel from "./CapturePanel.vue";

const props = defineProps<{
  laboratoryId?: string;
  nodes: Node[];
  interfaces: NodeInterface[];
  links: Link[];
  networkObjects?: NetworkObject[];
  networkObjectLinks?: NetworkObjectLink[];
  requestInterfaceId?: string;
  requestLinkId?: string;
  requestObjectLinkId?: string;
}>();

type SourceType = "interface" | "link" | "network_object_link";

interface OpenSource {
  key: string;
  type: SourceType;
  id: string;
  groupId: string;
}

const LINK_GROUP_ID = "links";
const OBJECT_LINK_GROUP_ID = "object-links";
const activeGroupId = ref("");
const activeSourceKey = ref("");
const openSources = ref<OpenSource[]>([]);
const captureStates = ref<Record<string, string>>({});
const lastSourceByGroup = ref<Record<string, string>>({});

const interfacesByNode = computed(() => {
  const grouped = new Map<string, NodeInterface[]>();
  for (const item of props.interfaces) {
    const values = grouped.get(item.node_id) || [];
    values.push(item);
    grouped.set(item.node_id, values);
  }
  for (const values of grouped.values())
    values.sort((left, right) => left.slot - right.slot);
  return grouped;
});

const nodeGroups = computed(() =>
  props.nodes.filter(
    (node) => (interfacesByNode.value.get(node.id) || []).length,
  ),
);

const activeInterfaces = computed(() =>
  activeGroupId.value === LINK_GROUP_ID
    ? []
    : interfacesByNode.value.get(activeGroupId.value) || [],
);

function interfaceKey(id: string) {
  return `interface:${id}`;
}

function linkKey(id: string) {
  return `link:${id}`;
}

function objectLinkKey(id: string) {
  return `network_object_link:${id}`;
}

function ensureOpen(source: OpenSource) {
  if (!openSources.value.some((item) => item.key === source.key))
    openSources.value.push(source);
  activeGroupId.value = source.groupId;
  activeSourceKey.value = source.key;
  lastSourceByGroup.value[source.groupId] = source.key;
}

function openInterface(interfaceId: string) {
  const item = props.interfaces.find(
    (candidate) => candidate.id === interfaceId,
  );
  if (!item) return;
  ensureOpen({
    key: interfaceKey(item.id),
    type: "interface",
    id: item.id,
    groupId: item.node_id,
  });
}

function openLink(linkId: string) {
  if (!props.links.some((link) => link.id === linkId)) return;
  ensureOpen({
    key: linkKey(linkId),
    type: "link",
    id: linkId,
    groupId: LINK_GROUP_ID,
  });
}

function openObjectLink(linkId: string) {
  if (!props.networkObjectLinks?.some((link) => link.id === linkId)) return;
  ensureOpen({
    key: objectLinkKey(linkId),
    type: "network_object_link",
    id: linkId,
    groupId: OBJECT_LINK_GROUP_ID,
  });
}

function selectGroup(groupId: string) {
  activeGroupId.value = groupId;
  const remembered = lastSourceByGroup.value[groupId];
  if (remembered) {
    activeSourceKey.value = remembered;
    return;
  }
  if (groupId === LINK_GROUP_ID) {
    if (props.links[0]) openLink(props.links[0].id);
    return;
  }
  if (groupId === OBJECT_LINK_GROUP_ID) {
    if (props.networkObjectLinks?.[0])
      openObjectLink(props.networkObjectLinks[0].id);
    return;
  }
  const firstInterface = interfacesByNode.value.get(groupId)?.[0];
  if (firstInterface) openInterface(firstInterface.id);
}

function sourceState(sourceKey: string) {
  return captureStates.value[sourceKey] || "idle";
}

function stateClass(sourceKey: string) {
  const state = sourceState(sourceKey);
  if (["running", "streaming", "starting", "requested"].includes(state))
    return "bg-emerald-400";
  if (["failed", "cancelled"].includes(state)) return "bg-red-400";
  if (["stopping"].includes(state)) return "bg-amber-400";
  return "bg-muted-foreground/40";
}

function updateCapture(sourceKey: string, capture?: CaptureSession) {
  captureStates.value[sourceKey] = capture?.state || "idle";
}

function interfaceLabel(interfaceId: string) {
  return (
    props.interfaces.find((item) => item.id === interfaceId)?.name ||
    interfaceId
  );
}

function nodeLabel(nodeId: string) {
  return props.nodes.find((node) => node.id === nodeId)?.name || nodeId;
}

function linkLabel(link: Link) {
  return linkDisplayName(link, props.interfaces, props.nodes);
}

function sourceLabel(source: OpenSource) {
  if (source.type === "interface")
    return `${nodeLabel(source.groupId)} · ${interfaceLabel(source.id)}`;
  if (source.type === "link") {
    const link = props.links.find((item) => item.id === source.id);
    return link ? linkLabel(link) : source.id;
  }
  const link = props.networkObjectLinks?.find((item) => item.id === source.id);
  return link
    ? networkObjectLinkDisplayName(link, props.networkObjects || [])
    : source.id;
}

watch(
  () => props.laboratoryId,
  () => {
    activeGroupId.value = "";
    activeSourceKey.value = "";
    openSources.value = [];
    captureStates.value = {};
    lastSourceByGroup.value = {};
  },
);

watch(
  () =>
    [
      props.requestInterfaceId,
      props.requestLinkId,
      props.requestObjectLinkId,
    ] as const,
  ([interfaceId, linkId, objectLinkId]) => {
    if (objectLinkId) openObjectLink(objectLinkId);
    else if (linkId) openLink(linkId);
    else if (interfaceId) openInterface(interfaceId);
  },
  { immediate: true },
);

watch(
  () => [
    props.interfaces.map((item) => item.id),
    props.links.map((item) => item.id),
    (props.networkObjectLinks || []).map((item) => item.id),
  ],
  () => {
    const interfaceIds = new Set(props.interfaces.map((item) => item.id));
    const linkIds = new Set(props.links.map((item) => item.id));
    const objectLinkIds = new Set(
      (props.networkObjectLinks || []).map((item) => item.id),
    );
    openSources.value = openSources.value.filter((source) =>
      source.type === "interface"
        ? interfaceIds.has(source.id)
        : source.type === "link"
          ? linkIds.has(source.id)
          : objectLinkIds.has(source.id),
    );
    if (
      !openSources.value.some((source) => source.key === activeSourceKey.value)
    ) {
      const fallback = openSources.value[0];
      activeSourceKey.value = fallback?.key || "";
      activeGroupId.value = fallback?.groupId || "";
    }
  },
  { deep: true },
);
</script>

<template>
  <div class="flex h-full min-h-[180px] flex-col" data-global-capture-workspace>
    <nav
      class="netlab-scrollbar flex shrink-0 gap-1 overflow-x-auto border-b border-border bg-muted/20 p-1"
      data-layout-region="capture-resources"
      aria-label="节点抓包工作区"
    >
      <Button
        v-for="node in nodeGroups"
        :key="node.id"
        size="sm"
        :variant="activeGroupId === node.id ? 'default' : 'ghost'"
        @click="selectGroup(node.id)"
      >
        <Radio :size="12" /> {{ node.name }}
        <span class="text-[10px] opacity-70">
          {{ interfacesByNode.get(node.id)?.length || 0 }} 个接口
        </span>
      </Button>
      <Button
        v-if="links.length"
        size="sm"
        :variant="activeGroupId === LINK_GROUP_ID ? 'default' : 'ghost'"
        @click="selectGroup(LINK_GROUP_ID)"
      >
        <Cable :size="12" /> 链路
        <span class="text-[10px] opacity-70">{{ links.length }}</span>
      </Button>
      <Button
        v-if="networkObjectLinks?.length"
        size="sm"
        :variant="activeGroupId === OBJECT_LINK_GROUP_ID ? 'default' : 'ghost'"
        @click="selectGroup(OBJECT_LINK_GROUP_ID)"
      >
        <Cable :size="12" /> 网络对象链路
        <span class="text-[10px] opacity-70">{{
          networkObjectLinks.length
        }}</span>
      </Button>
    </nav>

    <nav
      v-if="activeGroupId"
      class="netlab-scrollbar flex shrink-0 gap-1 overflow-x-auto border-b border-border bg-muted/10 p-1"
      data-layout-region="capture-sources"
      aria-label="抓包来源"
    >
      <Button
        v-for="item in activeInterfaces"
        :key="item.id"
        size="sm"
        :variant="
          activeSourceKey === interfaceKey(item.id) ? 'default' : 'ghost'
        "
        @click="openInterface(item.id)"
      >
        <span
          class="size-2 rounded-full"
          :class="stateClass(interfaceKey(item.id))"
        />
        {{ item.name }}
      </Button>
      <Button
        v-for="link in activeGroupId === LINK_GROUP_ID ? links : []"
        :key="link.id"
        size="sm"
        :variant="activeSourceKey === linkKey(link.id) ? 'default' : 'ghost'"
        @click="openLink(link.id)"
      >
        <span
          class="size-2 rounded-full"
          :class="stateClass(linkKey(link.id))"
        />
        {{ linkLabel(link) }}
      </Button>
      <Button
        v-for="link in activeGroupId === OBJECT_LINK_GROUP_ID
          ? networkObjectLinks || []
          : []"
        :key="link.id"
        size="sm"
        :variant="
          activeSourceKey === objectLinkKey(link.id) ? 'default' : 'ghost'
        "
        @click="openObjectLink(link.id)"
      >
        <span
          class="size-2 rounded-full"
          :class="stateClass(objectLinkKey(link.id))"
        />
        {{ networkObjectLinkDisplayName(link, networkObjects || []) }}
      </Button>
    </nav>

    <div
      v-if="openSources.length"
      class="min-h-0 flex-1 overflow-auto netlab-scrollbar"
      data-layout-region="capture-content"
    >
      <CapturePanel
        v-for="source in openSources"
        v-show="source.key === activeSourceKey"
        :key="source.key"
        :laboratory-id="laboratoryId"
        :interface-id="source.type === 'interface' ? source.id : undefined"
        :link-id="source.type === 'link' ? source.id : undefined"
        :object-link-id="
          source.type === 'network_object_link' ? source.id : undefined
        "
        :source-label="sourceLabel(source)"
        @capture-change="(capture) => updateCapture(source.key, capture)"
      />
    </div>
    <div
      v-else
      class="grid min-h-0 flex-1 place-items-center px-4 text-center text-xs text-muted-foreground"
    >
      请在上方选择节点和接口，或右键单击拓扑节点并选择抓包。
    </div>
  </div>
</template>
