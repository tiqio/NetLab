<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import {
  ApiError,
  type Link,
  type NetworkAttachment,
  type NetworkObject,
  type NetworkObjectLink,
  type Node,
  type NodeInterface,
  type OperationTask,
  type PlacementIntent,
  type TrafficObservation,
  type TopologyConnectionConfig,
} from "@/api";
import {
  Cable,
  Focus,
  Group,
  Hand,
  Maximize2,
  Play,
  Radio,
  RotateCcw,
  Square,
  TerminalSquare,
  Trash2,
  XCircle,
} from "lucide-vue-next";
import { api } from "@/api";
import { Button } from "@/components/ui";
import ConfirmationDialog from "@/components/common/ConfirmationDialog.vue";
import CommandPalette from "@/components/shell/CommandPalette.vue";
import LaboratoryShell from "@/components/shell/LaboratoryShell.vue";
import OperationsDrawer from "@/components/shell/OperationsDrawer.vue";
import {
  pruneLinkRoutes,
  useWorkspacePreferences,
} from "@/composables/useWorkspacePreferences";
import LaboratoryToolbar from "@/features/laboratories/LaboratoryToolbar.vue";
import { randomUUID } from "@/lib/uuid";
import { useLaboratoryStore } from "@/stores/laboratory";
import type { BottomTab } from "@/types/workspace";
import CreateTopologyResourceDrawer from "./CreateTopologyResourceDrawer.vue";
import DevicePalette, { type PaletteSelection } from "./DevicePalette.vue";
import LinkContextMenu from "./LinkContextMenu.vue";
import PortChooser from "./PortChooser.vue";
import TopologyCanvas from "./TopologyCanvas.vue";
import TopologyInspector from "./TopologyInspector.vue";
import { fitViewport } from "./topologyGeometry";
import { resolveConnectionSourceCandidates } from "./topologyConnectionSources";
import {
  endpointKey,
  endpointsCompatible,
  type UnifiedConnectionEndpoint,
} from "./topologyEndpointCompatibility";
import { TopologyKeyboardController } from "./topologyKeyboardController";
import { resolvePlacements } from "./topologyLayout";
import { buildPlacementBatch } from "./topologyPlacementBatch";
import { runObjectLinkDeletion } from "./objectLinkDeletion";
import { useTemporaryPanShortcut } from "./useTemporaryPanShortcut";
import { waitForTopologyTaskFinal } from "./topologyTaskFinalization";
import {
  applyAuthoritativeCreation,
  type AuthoritativeCreation,
} from "./authoritativeCreation";
import {
  captureTopologyCreateWorkspace,
  openTopologyCreateDrawer,
} from "./topologyCreateDrawerState";
import {
  boxSelect,
  captureSelection,
  cleanSelection,
  rangeSelect,
  selectOne,
  toggleSelected,
  restoreSelection,
} from "./topologySelection";

const store = useLaboratoryStore();
const ACTIVE_LAB_STORAGE_KEY = "netlab.active-laboratory.v1";
const shell = ref<InstanceType<typeof LaboratoryShell>>();
const topologyCanvas = ref<InstanceType<typeof TopologyCanvas>>();
const initialized = ref(false);
const panEnabled = ref(false);
const {
  temporaryPanHeld,
  handleTemporaryPanKeyDown,
  handleTemporaryPanKeyUp,
  releaseTemporaryPan,
} = useTemporaryPanShortcut();
const effectivePanEnabled = computed(
  () => panEnabled.value || temporaryPanHeld.value,
);
const resourceContext = ref<{
  id: string;
  type: "node" | "link" | "network_object" | "network_object_link";
  x: number;
  y: number;
}>();
const contextDeleteNode = ref<Node>();
const contextDeleteObject = ref<NetworkObject>();
const deletingObjectLinkIds = ref<string[]>([]);
const deletingAttachmentIds = ref<string[]>([]);
const failedObjectLinkDelete = ref<NetworkObjectLink>();
const trafficObservations = ref<TrafficObservation[]>([]);
const trafficOverlayActive = ref(false);
const trafficOverlayColor = ref("#f59e0b");
const consoleRequestNodeId = ref("");
const consoleRequestNetworkObjectId = ref("");
const consoleRequestKey = ref(0);
const activeId = computed(() => store.active?.laboratory.id);
const {
  preferences,
  setPanel,
  setViewport,
  setActiveBottomTab,
  createGroup,
  toggleGroup,
  setLinkRoute,
  setLabelDensity,
  persist: persistWorkspacePreferences,
} = useWorkspacePreferences(activeId);
const selectedIds = ref<string[]>([]);
const selectedType = ref<
  | "node"
  | "link"
  | "network_object"
  | "network_attachment"
  | "network_object_link"
>();
const selectionAnchor = ref("");
const focusedResourceId = ref("");
const captureOverlay = ref<{ connectionIds: string[]; interfaceIds: string[] }>(
  {
    connectionIds: [],
    interfaceIds: [],
  },
);
const createPlacementIntent = ref<PlacementIntent>();
const keyboardAnnouncement = ref("");
const keyboardController = new TopologyKeyboardController();
const createOpen = ref(false);
const createSucceeded = ref(false);
const createSnapshot = ref<{
  inspector: { collapsed: boolean; size: number };
  selectedIds: string[];
  selectedType: typeof selectedType.value;
  focusedResourceId: string;
  activeElement: HTMLElement | null;
}>();
const paletteSelection = ref<PaletteSelection>();
const createDrawer = ref<{
  isDirty: () => boolean;
  requestExternalDiscard: (action: () => void) => void;
}>();
const commandOpen = ref(false);
type TopologyConnectionEntryPoint =
  "port_click" | "port_drag" | "resource_plus" | "keyboard";
const connectionEntryPoint = ref<TopologyConnectionEntryPoint>("port_click");
const connectionDraft = ref<{
  source: UnifiedConnectionEndpoint;
  target?: UnifiedConnectionEndpoint;
  entryPoint: TopologyConnectionEntryPoint;
}>();
const connectionSelectionSnapshot = ref<{
  selectedIds: string[];
  selectedType: typeof selectedType.value;
  focusedResourceId: string;
  activeElement: HTMLElement | null;
}>();
const pendingEndpoint = computed(() =>
  connectionDraft.value?.source.kind === "node_interface"
    ? connectionDraft.value.source.portId || ""
    : "",
);
const pendingObjectPort = computed(() =>
  connectionDraft.value?.source.kind === "network_object_port"
    ? {
        objectId: connectionDraft.value.source.resourceId,
        portName: connectionDraft.value.source.portName || "",
      }
    : undefined,
);
const canvasStatus = ref("");
const portChooserOpen = ref(false);
const portChooserMode = ref<"source" | "target" | "reconnect" | "capture">(
  "source",
);
const portChooserInterfaces = ref<NodeInterface[]>([]);
const portChooserEndpoints = ref<UnifiedConnectionEndpoint[]>([]);
const selectedInterfaceId = ref("");
const reconnectingLink = ref<Link>();
const editingRouteLinkId = ref("");
const routeEditOriginal = ref<Array<{ x: number; y: number }>>([]);
const reconnectRetainedEndpoint = ref("");
const activeReconnectTaskId = ref("");
const lastReconnectRequest = ref<{
  linkId: string;
  retainedEndpointId: string;
  replacementEndpointId: string;
}>();

const selectedNode = computed<Node | undefined>(() =>
  selectedType.value === "node"
    ? store.active?.nodes.find((item) => item.id === selectedIds.value[0])
    : undefined,
);
const selectedLink = computed<Link | undefined>(() =>
  selectedType.value === "link"
    ? store.active?.links.find((item) => item.id === selectedIds.value[0])
    : undefined,
);
const selectedAttachment = computed<NetworkAttachment | undefined>(() =>
  selectedType.value === "network_attachment"
    ? store.active?.network_attachments?.find(
        (item) => item.id === selectedIds.value[0],
      )
    : undefined,
);
const selectedObjectLink = computed<NetworkObjectLink | undefined>(() =>
  selectedType.value === "network_object_link"
    ? store.active?.network_object_links?.find(
        (item) => item.id === selectedIds.value[0],
      )
    : undefined,
);
const contextNode = computed(() =>
  resourceContext.value?.type === "node"
    ? store.active?.nodes.find((item) => item.id === resourceContext.value?.id)
    : undefined,
);
const contextLink = computed(() =>
  resourceContext.value?.type === "link"
    ? store.active?.links.find((item) => item.id === resourceContext.value?.id)
    : undefined,
);
const contextObject = computed(() =>
  resourceContext.value?.type === "network_object"
    ? store.active?.network_objects.find(
        (item) => item.id === resourceContext.value?.id,
      )
    : undefined,
);
const contextObjectLink = computed(() =>
  resourceContext.value?.type === "network_object_link"
    ? store.active?.network_object_links?.find(
        (item) => item.id === resourceContext.value?.id,
      )
    : undefined,
);
const resourceContextStyle = computed(() => {
  if (!resourceContext.value) return {};
  const width = 224;
  const estimatedHeight = contextNode.value ? 248 : 184;
  return {
    left: `${Math.max(8, Math.min(resourceContext.value.x, window.innerWidth - width - 8))}px`,
    top: `${Math.max(8, Math.min(resourceContext.value.y, window.innerHeight - estimatedHeight - 8))}px`,
    maxHeight: `${Math.max(120, window.innerHeight - 16)}px`,
  };
});

function openResourceContext(
  id: string,
  type: "node" | "link" | "network_object" | "network_object_link",
  x: number,
  y: number,
) {
  selectResource(id, type, false);
  resourceContext.value = {
    id,
    type,
    x,
    y,
  };
}

async function deleteObjectLink(link: NetworkObjectLink) {
  if (deletingObjectLinkIds.value.includes(link.id)) return;
  failedObjectLinkDelete.value = undefined;
  deletingObjectLinkIds.value.push(link.id);
  closeResourceContext();
  canvasStatus.value = `正在删除对象链路 ${link.id}…`;
  try {
    const envelope = await runObjectLinkDeletion(link, {
      hide: (id) => store.hideNetworkObjectLink(id),
      clearSelection: () => {
        if (selectedIds.value.includes(link.id)) clearSelection();
      },
      submit: (value) => api.deleteNetworkObjectLink(value),
      recordTask: (task) => {
        const index = store.tasks.findIndex((item) => item.id === task.id);
        if (index >= 0) store.tasks[index] = task;
        else store.tasks.unshift(task);
      },
      unhide: (id) => store.unhideNetworkObjectLink(id),
      reload: async () => {
        if (store.active) await store.open(store.active.laboratory.id);
      },
    });
    canvasStatus.value = `对象链路删除任务 ${envelope.task.id} 已提交。`;
  } catch (value) {
    failedObjectLinkDelete.value = link;
    canvasStatus.value = `对象链路删除失败：${value instanceof Error ? value.message : String(value)}`;
  } finally {
    deletingObjectLinkIds.value = deletingObjectLinkIds.value.filter(
      (id) => id !== link.id,
    );
  }
}

async function retryObjectLinkDelete() {
  const link = failedObjectLinkDelete.value;
  if (link) await deleteObjectLink(link);
}
async function deleteAttachment(attachment: NetworkAttachment) {
  if (deletingAttachmentIds.value.includes(attachment.id)) return;
  deletingAttachmentIds.value.push(attachment.id);
  closeResourceContext();
  canvasStatus.value = `正在解除网络附件 ${attachment.id}…`;
  try {
    const envelope = await api.deleteTopologyConnection(
      attachment.id,
      attachment.revision,
    );
    store.recordTopologyConnectionTask(envelope);
    if (selectedIds.value.includes(attachment.id)) clearSelection();
    canvasStatus.value = `网络附件删除任务 ${envelope.task.id} 已提交。`;
    await refreshActive();
  } catch (value) {
    canvasStatus.value = `网络附件删除失败：${value instanceof Error ? value.message : String(value)}`;
  } finally {
    deletingAttachmentIds.value = deletingAttachmentIds.value.filter(
      (id) => id !== attachment.id,
    );
  }
}
function closeResourceContext() {
  resourceContext.value = undefined;
}
function openNodeTerminal(node: Node) {
  consoleRequestNodeId.value = node.id;
  consoleRequestNetworkObjectId.value = "";
  consoleRequestKey.value += 1;
  setActiveBottomTab("console");
  shell.value?.openBottom();
  closeResourceContext();
}
function openNetworkObjectTerminal(object: NetworkObject) {
  if (object.kind !== "pc") return;
  consoleRequestNodeId.value = "";
  consoleRequestNetworkObjectId.value = object.id;
  consoleRequestKey.value += 1;
  setActiveBottomTab("console");
  shell.value?.openBottom();
  closeResourceContext();
}
async function setContextNodeState(node: Node) {
  const desired = node.desired_state === "running" ? "stopped" : "running";
  closeResourceContext();
  await api.setNodeState(node, desired);
  await refreshActive();
}
function requestContextNodeDelete(node: Node) {
  contextDeleteNode.value = node;
  closeResourceContext();
}
function requestContextObjectDelete(object: NetworkObject) {
  contextDeleteObject.value = object;
  closeResourceContext();
}
function nodeInterfaces(nodeId: string) {
  return (store.active?.interfaces || []).filter(
    (item) => item.node_id === nodeId && !item.name.startsWith("internal"),
  );
}
function openInterfaceCapture(value: NodeInterface) {
  selectedInterfaceId.value = value.id;
  selectedIds.value = selectOne(value.node_id);
  selectedType.value = "node";
  selectionAnchor.value = value.node_id;
  focusedResourceId.value = value.node_id;
  portChooserOpen.value = false;
  setActiveBottomTab("captures");
  shell.value?.openBottom();
  canvasStatus.value = `已选择接口 ${value.name}，可以开始抓包。`;
}
function requestContextNodeCapture(node: Node) {
  closeResourceContext();
  const candidates = nodeInterfaces(node.id);
  if (!candidates.length) {
    canvasStatus.value = "所选节点没有可抓包的接口。";
    return;
  }
  if (candidates.length === 1) {
    openInterfaceCapture(candidates[0]);
    return;
  }
  portChooserMode.value = "capture";
  portChooserInterfaces.value = candidates;
  portChooserOpen.value = true;
}
function openSelectedConnectionCapture() {
  if (selectedAttachment.value)
    selectedInterfaceId.value = selectedAttachment.value.interface_id;
  setActiveBottomTab("captures");
  shell.value?.openBottom();
  canvasStatus.value = "已打开所选连接的抓包工作区。";
}
function openSelectedConnectionTrafficFilter() {
  if (selectedAttachment.value)
    selectedInterfaceId.value = selectedAttachment.value.interface_id;
  setActiveBottomTab("traffic-filter");
  shell.value?.openBottom();
  canvasStatus.value = "已打开所选连接的流量过滤工作区。";
}
async function confirmContextNodeDelete() {
  if (!contextDeleteNode.value) return;
  await api.deleteNode(contextDeleteNode.value);
  contextDeleteNode.value = undefined;
  clearSelection();
  await refreshActive();
}
async function confirmContextObjectDelete() {
  if (!contextDeleteObject.value) return;
  await api.deleteNetworkObject(contextDeleteObject.value);
  contextDeleteObject.value = undefined;
  clearSelection();
  await refreshActive();
}
const activeReconnectTask = computed<OperationTask | undefined>(() => {
  const explicit = store.tasks.find(
    (item) => item.id === activeReconnectTaskId.value,
  );
  if (explicit) return explicit;
  const linkId =
    reconnectingLink.value?.id ||
    lastReconnectRequest.value?.linkId ||
    selectedLink.value?.id;
  if (!linkId) return undefined;
  return store.tasks
    .filter(
      (item) =>
        item.resource_type === "link" &&
        item.resource_id === linkId &&
        item.kind.includes("reconnect") &&
        ["queued", "running"].includes(item.state),
    )
    .sort((left, right) => right.created_at.localeCompare(left.created_at))[0];
});
const selectedObject = computed<NetworkObject | undefined>(() =>
  selectedType.value === "network_object"
    ? store.active?.network_objects.find(
        (item) => item.id === selectedIds.value[0],
      )
    : undefined,
);
const selectedInterface = computed(() =>
  selectedAttachment.value
    ? store.active?.interfaces.find(
        (item) => item.id === selectedAttachment.value?.interface_id,
      )
    : selectedNode.value
      ? store.active?.interfaces.find(
          (item) =>
            item.node_id === selectedNode.value!.id &&
            !item.name.startsWith("internal") &&
            item.id === selectedInterfaceId.value,
        ) ||
        store.active?.interfaces.find(
          (item) =>
            item.node_id === selectedNode.value!.id &&
            !item.name.startsWith("internal"),
        )
      : undefined,
);
const interfaceOwners = computed(() =>
  Object.fromEntries(
    (store.active?.interfaces || []).map((item) => [item.id, item.node_id]),
  ),
);
const coordinates = computed(() =>
  resolvePlacements(
    [...(store.active?.nodes || []), ...(store.active?.network_objects || [])],
    Object.fromEntries(
      (store.active?.placements || []).map((item) => [
        item.resource_id,
        { x: item.x, y: item.y, pinned: true, updatedAt: "" },
      ]),
    ),
  ),
);

const resourceTypes = computed(() =>
  Object.fromEntries([
    ...(store.active?.nodes || []).map((item) => [item.id, "node"] as const),
    ...(store.active?.network_objects || []).map(
      (item) => [item.id, "network_object"] as const,
    ),
  ]),
);
const keyboardResources = computed(() => [
  ...(store.active?.nodes || []).map((item) => ({
    id: item.id,
    type: "node" as const,
    x: coordinates.value[item.id]?.x || 0,
    y: coordinates.value[item.id]?.y || 0,
  })),
  ...(store.active?.network_objects || []).map((item) => ({
    id: item.id,
    type: "network_object" as const,
    x: coordinates.value[item.id]?.x || 0,
    y: coordinates.value[item.id]?.y || 0,
  })),
  ...(store.active?.links || []).map((item) => {
    const ownerA = interfaceOwners.value[item.endpoint_a_id];
    const ownerB = interfaceOwners.value[item.endpoint_b_id];
    const a = coordinates.value[ownerA] || { x: 0, y: 0 };
    const b = coordinates.value[ownerB] || { x: 0, y: 0 };
    return {
      id: item.id,
      type: "link" as const,
      x: (a.x + b.x) / 2,
      y: (a.y + b.y) / 2,
    };
  }),
  ...(store.active?.network_attachments || []).map((item) => {
    const interfaceValue = store.active?.interfaces.find(
      (candidate) => candidate.id === item.interface_id,
    );
    const source = coordinates.value[interfaceValue?.node_id || ""] || {
      x: 0,
      y: 0,
    };
    const target = coordinates.value[item.network_object_id] || { x: 0, y: 0 };
    return {
      id: item.id,
      type: "network_attachment" as const,
      x: (source.x + target.x) / 2,
      y: (source.y + target.y) / 2,
    };
  }),
  ...(store.active?.network_object_links || []).map((item) => {
    const a = coordinates.value[item.object_a_id] || { x: 0, y: 0 };
    const b = coordinates.value[item.object_b_id] || { x: 0, y: 0 };
    return {
      id: item.id,
      type: "network_object_link" as const,
      x: (a.x + b.x) / 2,
      y: (a.y + b.y) / 2,
    };
  }),
]);
function networkObjectPortNames(object: NetworkObject) {
  const rows =
    object.kind === "switch_l2"
      ? object.config?.ports
      : object.kind === "switch_l3" || object.kind === "pc"
        ? object.config?.interfaces
        : [];
  return Array.isArray(rows)
    ? rows
        .map((item) => String((item as { name?: string }).name || ""))
        .filter(Boolean)
    : [];
}

const keyboardPorts = computed(() => [
  ...(store.active?.interfaces || [])
    .filter((item) => !item.name.startsWith("internal"))
    .map((item) => ({
      id: item.id,
      ownerId: item.node_id,
      name: item.name,
      available: !item.desired_link_id,
    })),
  ...(store.active?.network_objects || []).flatMap((object) =>
    networkObjectPortNames(object).map((portName) => ({
      id: `${object.id}:${portName}`,
      ownerId: object.id,
      name: portName,
      available: !objectPortOccupied(object.id, portName),
    })),
  ),
]);

watch(
  () => keyboardResources.value.map((item) => item.id),
  (ids) => {
    selectedIds.value = cleanSelection(selectedIds.value, new Set(ids));
    if (!selectedIds.value.length) selectedType.value = undefined;
    if (focusedResourceId.value && !ids.includes(focusedResourceId.value))
      focusedResourceId.value = "";
  },
);
watch(
  () => (store.active?.links || []).map((item) => item.id),
  (ids) => {
    if (pruneLinkRoutes(preferences.value, ids)) persistWorkspacePreferences();
    const available = new Set(ids);
    if (editingRouteLinkId.value && !available.has(editingRouteLinkId.value)) {
      editingRouteLinkId.value = "";
      routeEditOriginal.value = [];
    }
  },
);

async function moveResource(id: string, x: number, y: number) {
  if (!store.active) return;
  const laboratoryId = store.active.laboratory.id;
  const laboratoryRevision = store.active.laboratory.revision;
  const placements = buildPlacementBatch(
    id,
    { x, y },
    selectedIds.value,
    coordinates.value,
    resourceTypes.value,
    store.active.placements,
  );
  if (!placements.length) return;
  for (const placement of placements) {
    const index = store.active.placements.findIndex(
      (item) => item.resource_id === placement.resource_id,
    );
    const optimistic = {
      laboratory_id: laboratoryId,
      resource_id: placement.resource_id,
      resource_type: placement.resource_type,
      x: placement.x,
      y: placement.y,
      revision:
        index >= 0
          ? store.active.placements[index].revision
          : (placement.revision ?? 0),
    };
    if (index >= 0) store.active.placements[index] = optimistic;
    else store.active.placements.push(optimistic);
  }
  try {
    const result = await api.updateTopologyPlacements(
      laboratoryId,
      laboratoryRevision,
      placements,
    );
    store.active.laboratory.revision = result.laboratory_revision;
    for (const placement of result.placements) {
      const index = store.active.placements.findIndex(
        (item) => item.resource_id === placement.resource_id,
      );
      if (index >= 0) store.active.placements[index] = placement;
      else store.active.placements.push(placement);
    }
  } catch (error) {
    await store.open(store.active.laboratory.id);
    canvasStatus.value =
      "Topology changed elsewhere; restored shared positions.";
    throw error;
  }
}

function canvasSize() {
  const canvas = document.querySelector<HTMLElement>(".topology-surface");
  return {
    width: Math.max(canvas?.clientWidth || window.innerWidth, 320),
    height: Math.max(canvas?.clientHeight || window.innerHeight, 320),
  };
}

function fitResources(ids?: string[]) {
  const selected = ids?.length ? new Set(ids) : undefined;
  const points = Object.entries(coordinates.value)
    .filter(([id]) => !selected || selected.has(id))
    .map(([, point]) => point);
  const size = canvasSize();
  setViewport(fitViewport(points, size.width, size.height));
}

function resetViewport() {
  fitResources();
}
const resourceIds = computed(() => [
  ...(store.active?.nodes || []).map((item) => item.id),
  ...(store.active?.links || []).map((item) => item.id),
  ...(store.active?.interfaces || []).map((item) => item.id),
  ...(store.active?.network_objects || []).map((item) => item.id),
  ...(store.active?.network_attachments || []).map((item) => item.id),
  ...(store.active?.network_object_links || []).map((item) => item.id),
]);

async function refresh() {
  await store.loadLabs();
  if (store.active) await store.open(store.active.laboratory.id);
  await store.loadTasks();
}

async function switchLaboratory(id: string) {
  selectedIds.value = [];
  await store.open(id);
  await store.loadLabs();
  await store.loadTasks();
  localStorage.setItem(ACTIVE_LAB_STORAGE_KEY, id);
}

async function openLaboratory(id: string) {
  if (createOpen.value && createDrawer.value?.isDirty()) {
    createDrawer.value.requestExternalDiscard(() => {
      createOpen.value = false;
      paletteSelection.value = undefined;
      void switchLaboratory(id);
    });
    return;
  }
  await switchLaboratory(id);
}

async function laboratoryDeleteAccepted(id: string) {
  const activeId = store.active?.laboratory.id;
  selectedIds.value = [];
  store.hideLaboratory(id);
  if (activeId && activeId !== id) {
    await store.loadTasks();
    return;
  }
  const fallback = store.labs[0];
  if (fallback) await openLaboratory(fallback.id);
  else {
    store.stopEvents();
    localStorage.removeItem(ACTIVE_LAB_STORAGE_KEY);
    await store.loadTasks();
  }
}

function choose(selection: PaletteSelection) {
  captureCreateSnapshot();
  createPlacementIntent.value = currentCreatePlacementIntent(selection.kind);
  const next = openTopologyCreateDrawer(selection);
  paletteSelection.value = next.selection;
  createOpen.value = next.open;
}

function openCreateDrawer() {
  captureCreateSnapshot();
  createPlacementIntent.value = currentCreatePlacementIntent();
  const next = openTopologyCreateDrawer();
  paletteSelection.value = next.selection;
  createOpen.value = next.open;
}

async function created(value: AuthoritativeCreation) {
  createSucceeded.value = true;
  if (!store.active) return;
  applyAuthoritativeCreation(value, {
    merge: (creation) => store.mergeAuthoritativeCreation(creation),
    select: (id, type) => selectResource(id, type, false),
    focus: (id) => (focusedResourceId.value = id),
    announce: (message) => (keyboardAnnouncement.value = message),
  });
}

function currentCreatePlacementIntent(
  kind?: PaletteSelection["kind"],
): PlacementIntent {
  const center = topologyCanvas.value?.viewportCenter?.() || { x: 0, y: 0 };
  return {
    preferred_x: center.x,
    preferred_y: center.y,
    footprint_class:
      kind === "qemu" || kind === "docker"
        ? "node-standard"
        : "network-object-standard",
  };
}

function captureCreateSnapshot() {
  if (createOpen.value || createSnapshot.value) return;
  createSucceeded.value = false;
  createSnapshot.value = captureTopologyCreateWorkspace({
    inspector: { ...preferences.value.panels.inspector },
    selectedIds: [...selectedIds.value],
    selectedType: selectedType.value,
    focusedResourceId: focusedResourceId.value,
    activeElement: document.activeElement as HTMLElement | null,
  });
}

function setCreateOpen(value: boolean) {
  createOpen.value = value;
  if (value) return;
  const snapshot = createSnapshot.value;
  createSnapshot.value = undefined;
  paletteSelection.value = undefined;
  if (!snapshot || createSucceeded.value) {
    createSucceeded.value = false;
    return;
  }
  setPanel("inspector", snapshot.inspector);
  selectedIds.value = snapshot.selectedIds;
  selectedType.value = snapshot.selectedType;
  focusedResourceId.value = snapshot.focusedResourceId;
  requestAnimationFrame(() => snapshot.activeElement?.focus());
}

async function refreshActive() {
  if (store.active) await store.open(store.active.laboratory.id);
  await store.loadTasks();
}

async function selectResource(
  id: string,
  type:
    | "node"
    | "link"
    | "network_object"
    | "network_attachment"
    | "network_object_link",
  additive: boolean,
) {
  if (pendingEndpoint.value && type === "node") {
    await chooseTargetNode(id);
    return;
  }
  selectedIds.value = additive
    ? toggleSelected(selectedIds.value, id)
    : selectOne(id);
  if (!additive) selectionAnchor.value = id;
  focusedResourceId.value = id;
  selectedType.value = type;
  if (type === "network_attachment") {
    selectedInterfaceId.value =
      store.active?.network_attachments?.find((item) => item.id === id)
        ?.interface_id || "";
    canvasStatus.value = "已选择网络附件，可打开“抓包”或“流量过滤”观察该网段。";
  } else if (type === "network_object_link") {
    selectedInterfaceId.value = "";
    canvasStatus.value =
      "对象间链路已选中，可在 Capture 或 Traffic Filter 中直接监控。";
  } else if (type !== "node") selectedInterfaceId.value = "";
  else if (
    !store.active?.interfaces.some(
      (item) => item.id === selectedInterfaceId.value && item.node_id === id,
    )
  )
    selectedInterfaceId.value = "";
  shell.value?.openInspector();
}

function clearSelection() {
  selectedIds.value = [];
  selectedType.value = undefined;
  selectedInterfaceId.value = "";
  selectionAnchor.value = "";
}

function cancelOrClear() {
  if (
    pendingEndpoint.value ||
    pendingObjectPort.value ||
    portChooserOpen.value
  ) {
    cancelConnection();
    return;
  }
  clearSelection();
}

function objectPortOccupied(objectId: string, portName: string) {
  const key = `${objectId}:${portName}`;
  return Boolean(
    store.active?.network_attachments?.some(
      (item) =>
        item.network_object_id === objectId && item.port_name === portName,
    ) ||
    store.active?.network_object_links?.some(
      (item) =>
        `${item.object_a_id}:${item.port_a_name}` === key ||
        `${item.object_b_id}:${item.port_b_name}` === key,
    ),
  );
}

function endpointForInterface(value: NodeInterface): UnifiedConnectionEndpoint {
  const node = store.active?.nodes.find((item) => item.id === value.node_id);
  return {
    kind: "node_interface",
    laboratoryId: store.active?.laboratory.id || "",
    resourceId: value.node_id,
    resourceKind: node?.kind,
    portId: value.id,
    portName: value.name,
    displayName: `${node?.name || value.node_id}:${value.name}`,
    capabilities: [],
    availability: value.desired_link_id ? "occupied" : "free",
  };
}

function endpointForObjectPort(
  objectId: string,
  portName: string,
): UnifiedConnectionEndpoint | undefined {
  const object = store.active?.network_objects.find(
    (item) => item.id === objectId,
  );
  if (!object) return undefined;
  return {
    kind: "network_object_port",
    laboratoryId: store.active?.laboratory.id || "",
    resourceId: objectId,
    resourceKind: object.kind,
    portName,
    displayName: `${object.name}:${portName}`,
    capabilities: [],
    availability: objectPortOccupied(objectId, portName) ? "occupied" : "free",
  };
}

function setConnectionSource(
  source: UnifiedConnectionEndpoint,
  entryPoint: TopologyConnectionEntryPoint = connectionEntryPoint.value,
) {
  if (!connectionDraft.value)
    connectionSelectionSnapshot.value = {
      ...(() => {
        const snapshot = captureSelection(
          selectedIds.value,
          selectedType.value,
          focusedResourceId.value,
        );
        return {
          selectedIds: snapshot.ids,
          selectedType: snapshot.type,
          focusedResourceId: snapshot.focusedResourceId,
        };
      })(),
      activeElement: document.activeElement as HTMLElement | null,
    };
  connectionEntryPoint.value = entryPoint;
  connectionDraft.value = { source, entryPoint };
  canvasStatus.value = `已选择 ${source.displayName}；请拖到或选择兼容目标。`;
}

async function submitUnifiedConnection(
  source: UnifiedConnectionEndpoint,
  target: UnifiedConnectionEndpoint,
  config?: TopologyConnectionConfig,
) {
  if (!store.active) return;
  const compatibility = endpointsCompatible(source, target);
  if (!compatibility.compatible) {
    canvasStatus.value = compatibility.reason || "连接端点不兼容。";
    return;
  }
  try {
    const envelope = await api.createTopologyConnection(
      store.active.laboratory.id,
      store.active.laboratory.revision,
      {
        source: {
          kind: source.kind,
          resource_id: source.resourceId,
          port_id: source.portId,
          port_name: source.portName,
        },
        target: {
          kind: target.kind,
          resource_id: target.resourceId,
          port_id: target.portId,
          port_name: target.portName,
        },
        config,
        entry_point:
          connectionDraft.value?.entryPoint || connectionEntryPoint.value,
      },
      randomUUID(),
    );
    store.recordTopologyConnectionTask(envelope);
    connectionDraft.value = undefined;
    connectionSelectionSnapshot.value = undefined;
    portChooserOpen.value = false;
    portChooserEndpoints.value = [];
    canvasStatus.value = `连接任务 ${envelope.task.id} 已提交。`;
    await refreshActive();
  } catch (error) {
    connectionDraft.value = undefined;
    if (
      error instanceof ApiError &&
      error.problem.code === "revision_conflict"
    ) {
      canvasStatus.value = "拓扑已被其他客户端更新，已刷新，请重新选择端点。";
      await refreshActive();
      return;
    }
    canvasStatus.value = error instanceof Error ? error.message : String(error);
    await refreshActive();
  }
}

async function handleUnifiedConnectionDrop(
  source: UnifiedConnectionEndpoint,
  target: UnifiedConnectionEndpoint | undefined,
  targetResourceId: string,
  candidates: UnifiedConnectionEndpoint[],
) {
  setConnectionSource(source, "port_drag");
  if (target) {
    await submitUnifiedConnection(source, target);
    return;
  }
  const compatible = candidates.filter(
    (candidate) => endpointsCompatible(source, candidate).compatible,
  );
  if (!compatible.length) {
    canvasStatus.value = "目标没有可用的兼容端点。";
    return;
  }
  if (compatible.length === 1) {
    await submitUnifiedConnection(source, compatible[0]);
    return;
  }
  portChooserMode.value = "target";
  portChooserEndpoints.value = compatible;
  portChooserInterfaces.value = [];
  portChooserOpen.value = true;
  canvasStatus.value = `请选择 ${targetResourceId} 上的目标端点。`;
}

async function objectPortClicked(
  objectId: string,
  portName: string,
  entryPoint: TopologyConnectionEntryPoint = "port_click",
) {
  if (!store.active) return;
  const endpoint = endpointForObjectPort(objectId, portName);
  if (!endpoint || endpoint.availability !== "free") {
    canvasStatus.value = `${portName} 已被占用，请选择空闲端口。`;
    return;
  }
  const source = connectionDraft.value?.source;
  if (!source) {
    setConnectionSource(endpoint, entryPoint);
    return;
  }
  if (endpointKey(source) === endpointKey(endpoint)) {
    connectionDraft.value = undefined;
    canvasStatus.value = "对象链路创建已取消。";
    return;
  }
  await submitUnifiedConnection(source, endpoint);
}

function selectBox(
  rectangle: { left: number; top: number; right: number; bottom: number },
  additive: boolean,
) {
  const bounds = keyboardResources.value
    .filter((item) => item.type !== "link")
    .map((item) => ({
      id: item.id,
      left: item.x - 32,
      top: item.y - 32,
      right: item.x + 32,
      bottom: item.y + 32,
    }));
  selectedIds.value = boxSelect(rectangle, bounds, selectedIds.value, additive);
  const last = keyboardResources.value.find(
    (item) => item.id === selectedIds.value.at(-1),
  );
  selectedType.value = last?.type;
  focusedResourceId.value = last?.id || "";
  keyboardAnnouncement.value = `Selected ${selectedIds.value.length} resources.`;
}

async function interfaceClicked(
  interfaceId: string,
  entryPoint: TopologyConnectionEntryPoint = "port_click",
) {
  if (!store.active) return;
  const target = store.active.interfaces.find(
    (item) => item.id === interfaceId,
  );
  if (!target) return;
  if (selectedLink.value) {
    canvasStatus.value =
      "Use Link actions → Reconnect endpoint. The original link remains unchanged until atomic reconnect is available.";
    return;
  }
  if (target.desired_link_id) {
    canvasStatus.value = `${target.name} is already connected. Select its link first to reconnect it.`;
    return;
  }
  const endpoint = endpointForInterface(target);
  const source = connectionDraft.value?.source;
  if (!source) {
    setConnectionSource(endpoint, entryPoint);
    return;
  }
  if (endpointKey(source) === endpointKey(endpoint)) {
    connectionDraft.value = undefined;
    canvasStatus.value =
      "Connection cancelled because the same port was selected twice.";
    return;
  }
  await submitUnifiedConnection(source, endpoint);
}

function availableInterfaces(nodeId: string) {
  return (store.active?.interfaces || []).filter(
    (item) =>
      item.node_id === nodeId &&
      !item.name.startsWith("internal") &&
      !item.desired_link_id,
  );
}

function connectionSourceCandidates(resourceId: string) {
  if (!store.active) return [];
  const occupied = new Set<string>();
  for (const attachment of store.active.network_attachments || [])
    if (attachment.port_name)
      occupied.add(`${attachment.network_object_id}:${attachment.port_name}`);
  for (const link of store.active.network_object_links || []) {
    occupied.add(`${link.object_a_id}:${link.port_a_name}`);
    occupied.add(`${link.object_b_id}:${link.port_b_name}`);
  }
  return resolveConnectionSourceCandidates(resourceId, {
    laboratoryId: store.active.laboratory.id,
    nodes: store.active.nodes,
    interfaces: store.active.interfaces,
    networkObjects: store.active.network_objects,
    occupiedObjectPorts: occupied,
  });
}

function startConnection(
  resourceId: string,
  entryPoint: TopologyConnectionEntryPoint = "resource_plus",
) {
  connectionEntryPoint.value = entryPoint;
  const candidates = connectionSourceCandidates(resourceId);
  if (!candidates.length) {
    canvasStatus.value = "所选资源没有可用连接端点。";
    return;
  }
  if (candidates.length === 1) {
    setConnectionSource(candidates[0], entryPoint);
    return;
  }
  portChooserMode.value = "source";
  portChooserInterfaces.value = [];
  portChooserEndpoints.value = candidates;
  portChooserOpen.value = true;
}

async function chooseTargetResource(resourceId: string) {
  const source = connectionDraft.value?.source;
  if (!source) return;
  const candidates = connectionSourceCandidates(resourceId).filter(
    (endpoint) => endpointsCompatible(source, endpoint).compatible,
  );
  if (!candidates.length) {
    canvasStatus.value = "目标资源没有兼容的可用端点。";
    return;
  }
  if (candidates.length === 1) {
    await submitUnifiedConnection(source, candidates[0]);
    return;
  }
  portChooserMode.value = "target";
  portChooserInterfaces.value = [];
  portChooserEndpoints.value = candidates;
  portChooserOpen.value = true;
}

async function chooseTargetNode(nodeId: string) {
  const candidates = availableInterfaces(nodeId).filter(
    (item) => item.id !== pendingEndpoint.value,
  );
  if (!candidates.length) {
    canvasStatus.value = "目标节点没有可用接口。";
    return;
  }
  if (candidates.length === 1) {
    await connectPending(candidates[0]);
    return;
  }
  portChooserMode.value = "target";
  portChooserInterfaces.value = candidates;
  portChooserEndpoints.value = candidates.map(endpointForInterface);
  portChooserOpen.value = true;
}

async function connectPending(target: NodeInterface) {
  const source = connectionDraft.value?.source;
  if (!source) return;
  await submitUnifiedConnection(source, endpointForInterface(target));
}

async function portChosen(value: NodeInterface | UnifiedConnectionEndpoint) {
  if (portChooserMode.value === "capture") {
    if ("id" in value) openInterfaceCapture(value);
    return;
  }
  if (portChooserMode.value === "source") {
    setConnectionSource("id" in value ? endpointForInterface(value) : value);
    portChooserOpen.value = false;
    return;
  }
  if (portChooserMode.value === "reconnect") {
    if (!reconnectingLink.value || !("id" in value)) return;
    await submitReconnect(
      reconnectingLink.value,
      reconnectRetainedEndpoint.value,
      value.id,
    );
    return;
  }
  const source = connectionDraft.value?.source;
  if (!source) return;
  await submitUnifiedConnection(
    source,
    "id" in value ? endpointForInterface(value) : value,
  );
}

async function submitReconnect(
  link: Link,
  retainedEndpointId: string,
  replacementEndpointId: string,
) {
  try {
    const envelope = await api.reconnectLink(
      link,
      retainedEndpointId,
      replacementEndpointId,
    );
    activeReconnectTaskId.value = envelope.task.id;
    lastReconnectRequest.value = {
      linkId: link.id,
      retainedEndpointId,
      replacementEndpointId,
    };
    const index = store.tasks.findIndex((item) => item.id === envelope.task.id);
    if (index >= 0) store.tasks[index] = envelope.task;
    else store.tasks.unshift(envelope.task);
    portChooserOpen.value = false;
    reconnectingLink.value = undefined;
    reconnectRetainedEndpoint.value = "";
    canvasStatus.value = `Atomic reconnect task ${envelope.task.id} submitted; original endpoints remain visible until success.`;
  } catch (error) {
    if (
      error instanceof ApiError &&
      error.problem.code === "revision_conflict"
    ) {
      canvasStatus.value =
        "The link changed in another client. Reloaded the shared topology; choose the endpoint again.";
      await refreshActive();
      return;
    }
    canvasStatus.value = error instanceof Error ? error.message : String(error);
  }
}

async function cancelReconnect() {
  if (!activeReconnectTaskId.value) return;
  try {
    const requested = await api.cancelTask(activeReconnectTaskId.value);
    const task = await waitForTopologyTaskFinal(api.getTask, requested.id);
    const index = store.tasks.findIndex((item) => item.id === task.id);
    if (index >= 0) store.tasks[index] = task;
    else store.tasks.unshift(task);
    await refreshActive();
    canvasStatus.value = `Reconnect cancellation resolved as ${task.state}; refreshed authoritative endpoints.`;
  } catch (error) {
    canvasStatus.value = error instanceof Error ? error.message : String(error);
  }
}

async function retryReconnect() {
  if (!store.active || !lastReconnectRequest.value) return;
  const request = lastReconnectRequest.value;
  const link = store.active.links.find((item) => item.id === request.linkId);
  if (!link) {
    canvasStatus.value =
      "The original link no longer exists. Reload the topology before retrying.";
    return;
  }
  await submitReconnect(
    link,
    request.retainedEndpointId,
    request.replacementEndpointId,
  );
}

function cancelConnection() {
  const captureSelection = portChooserMode.value === "capture";
  connectionDraft.value = undefined;
  connectionEntryPoint.value = "port_click";
  portChooserOpen.value = false;
  portChooserEndpoints.value = [];
  portChooserMode.value = "source";
  canvasStatus.value = captureSelection
    ? "Capture interface selection cancelled."
    : "Connection cancelled; no topology mutation was sent.";
  const snapshot = connectionSelectionSnapshot.value;
  connectionSelectionSnapshot.value = undefined;
  if (snapshot && !captureSelection) {
    const restored = restoreSelection({
      ids: snapshot.selectedIds,
      type: snapshot.selectedType,
      focusedResourceId: snapshot.focusedResourceId,
    });
    selectedIds.value = restored.ids;
    selectedType.value = restored.type;
    focusedResourceId.value = restored.focusedResourceId;
    requestAnimationFrame(() => snapshot.activeElement?.focus());
  }
}

async function disconnectSelectedLink() {
  if (!selectedLink.value) return;
  await api.disconnectLink(selectedLink.value.id);
  canvasStatus.value = "断开连接任务已提交。";
  await refreshActive();
}

function requestReconnect() {
  if (!store.active || !selectedLink.value) return;
  const candidates = store.active.interfaces.filter(
    (item) =>
      !item.name.startsWith("internal") &&
      !item.desired_link_id &&
      item.id !== selectedLink.value?.endpoint_a_id &&
      item.id !== selectedLink.value?.endpoint_b_id,
  );
  if (!candidates.length) {
    canvasStatus.value = "没有可用于替换该端点的接口。";
    return;
  }
  reconnectingLink.value = selectedLink.value;
  reconnectRetainedEndpoint.value = selectedLink.value.endpoint_a_id;
  portChooserMode.value = "reconnect";
  portChooserInterfaces.value = candidates;
  portChooserOpen.value = true;
}

function groupSelection() {
  createGroup(selectedIds.value);
  canvasStatus.value = "已创建当前浏览器专用的视觉分组。";
}

function toggleSelectedRoute() {
  if (!selectedLink.value) return;
  const link = selectedLink.value;
  const current = preferences.value.linkRoutes[link.id] || [];
  editingRouteLinkId.value = link.id;
  routeEditOriginal.value = current.map((point) => ({ ...point }));
  if (!current.length) {
    const ownerA = interfaceOwners.value[link.endpoint_a_id];
    const ownerB = interfaceOwners.value[link.endpoint_b_id];
    const a = coordinates.value[ownerA] || { x: 0, y: 0 };
    const b = coordinates.value[ownerB] || { x: 0, y: 0 };
    setLinkRoute(link.id, [{ x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 + 72 }]);
  }
  canvasStatus.value = "拖动琥珀色控制点可调整当前浏览器专用的链路路径。";
}

function updateRoutePoint(linkId: string, point: { x: number; y: number }) {
  if (editingRouteLinkId.value === linkId) setLinkRoute(linkId, [point]);
}

function finishRouteEdit() {
  editingRouteLinkId.value = "";
  routeEditOriginal.value = [];
  canvasStatus.value = "已保存当前浏览器专用的链路路径。";
}

function cancelRouteEdit() {
  if (!editingRouteLinkId.value) return;
  setLinkRoute(editingRouteLinkId.value, routeEditOriginal.value);
  editingRouteLinkId.value = "";
  routeEditOriginal.value = [];
  canvasStatus.value = "已取消本地链路路径编辑。";
}

function resetRouteEdit() {
  if (!editingRouteLinkId.value) return;
  setLinkRoute(editingRouteLinkId.value, []);
  editingRouteLinkId.value = "";
  routeEditOriginal.value = [];
  canvasStatus.value = "已将链路恢复为自动布线。";
}

function labelDensityText(value: "comfortable" | "compact" | "minimal") {
  return {
    comfortable: "舒适",
    compact: "紧凑",
    minimal: "精简",
  }[value];
}

function cycleLabelDensity() {
  const values = ["comfortable", "compact", "minimal"] as const;
  const index = values.indexOf(preferences.value.labelDensity);
  setLabelDensity(values[(index + 1) % values.length]);
  canvasStatus.value = `拓扑标签密度：${labelDensityText(preferences.value.labelDensity)}。`;
}

function navigate(type: string, id: string) {
  if (["node", "link", "network_object", "network_attachment"].includes(type))
    selectResource(
      id,
      type as "node" | "link" | "network_object" | "network_attachment",
      false,
    );
}

function runCommand(id: string) {
  if (id === "add-device") shell.value?.openPalette();
  if (id === "tasks") {
    setActiveBottomTab("tasks");
    shell.value?.openBottom();
  }
  if (id === "console" && selectedNode.value) {
    setActiveBottomTab("console");
    shell.value?.openBottom();
  }
  if (id === "fit") fitResources();
}

const commands = computed(() => [
  {
    id: "add-device",
    label: "Open device palette",
    keywords: "node qemu docker",
  },
  { id: "tasks", label: "Open task center" },
  {
    id: "console",
    label: "Open selected node console",
    disabled: !selectedNode.value,
  },
  { id: "fit", label: "Fit topology to view" },
]);

function keydown(event: KeyboardEvent) {
  handleTemporaryPanKeyDown(event);
  if (event.key === "Escape" && resourceContext.value) {
    closeResourceContext();
    return;
  }
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
    event.preventDefault();
    commandOpen.value = true;
  }
}

function keyup(event: KeyboardEvent) {
  handleTemporaryPanKeyUp(event);
}

function handleWorkspaceBlur() {
  releaseTemporaryPan();
  cancelWorkspaceTransient();
}

function cancelWorkspaceTransient() {
  if (editingRouteLinkId.value) {
    cancelRouteEdit();
    return true;
  }
  if (
    pendingEndpoint.value ||
    pendingObjectPort.value ||
    portChooserOpen.value
  ) {
    cancelConnection();
    return true;
  }
  return false;
}

function cancelOnVisibilityLoss() {
  if (document.visibilityState !== "hidden") return;
  releaseTemporaryPan();
  cancelWorkspaceTransient();
}

async function topologyKeyboard(event: KeyboardEvent) {
  if (event.key === "Escape" && cancelWorkspaceTransient()) {
    event.preventDefault();
    return;
  }
  keyboardController.update(keyboardResources.value, keyboardPorts.value);
  if (focusedResourceId.value)
    keyboardController.focusResource(focusedResourceId.value);
  const action = keyboardController.handle(
    {
      key: event.key,
      shiftKey: event.shiftKey,
      altKey: event.altKey,
      ctrlKey: event.ctrlKey,
      metaKey: event.metaKey,
    },
    selectedIds.value,
    {
      connection: Boolean(
        pendingEndpoint.value ||
        pendingObjectPort.value ||
        portChooserOpen.value,
      ),
    },
  );
  if (action.type === "none") return;
  event.preventDefault();
  if (action.type === "focus_resource") {
    focusedResourceId.value = action.resourceId;
    keyboardAnnouncement.value = action.announcement;
    if (action.extend) {
      const order = keyboardResources.value.map((item) => item.id);
      selectedIds.value = rangeSelect(
        order,
        selectionAnchor.value || selectedIds.value.at(-1) || action.resourceId,
        action.resourceId,
      );
    } else {
      selectedIds.value = selectOne(action.resourceId);
      selectionAnchor.value = action.resourceId;
    }
    selectedType.value = action.resourceType;
  }
  if (action.type === "toggle_resource") {
    selectedIds.value = toggleSelected(selectedIds.value, action.resourceId);
  }
  if (action.type === "focus_port")
    keyboardAnnouncement.value = action.announcement;
  if (action.type === "choose_port") {
    const nodeInterface = store.active?.interfaces.find(
      (item) => item.id === action.interfaceId,
    );
    if (nodeInterface) await interfaceClicked(nodeInterface.id, "keyboard");
    else {
      const object = store.active?.network_objects.find((item) =>
        action.interfaceId.startsWith(`${item.id}:`),
      );
      if (object)
        await objectPortClicked(
          object.id,
          action.interfaceId.slice(object.id.length + 1),
          "keyboard",
        );
    }
  }
  if (action.type === "begin_connection")
    startConnection(action.resourceId, "keyboard");
  if (action.type === "choose_connection_target") {
    const target = keyboardResources.value.find(
      (item) => item.id === action.resourceId,
    );
    if (target?.type === "node" || target?.type === "network_object")
      await chooseTargetResource(target.id);
  }
  if (action.type === "open_inspector") shell.value?.openInspector();
  if (action.type === "open_terminal") {
    const node = store.active?.nodes.find(
      (item) => item.id === action.resourceId,
    );
    if (node) openNodeTerminal(node);
  }
  if (action.type === "move_selection") {
    const id = focusedResourceId.value || selectedIds.value.at(-1);
    const point = id ? coordinates.value[id] : undefined;
    if (id && point)
      await moveResource(id, point.x + action.dx, point.y + action.dy);
  }
  if (action.type === "zoom_view") {
    const point = focusedResourceId.value
      ? coordinates.value[focusedResourceId.value]
      : undefined;
    setViewport({
      centerX: point?.x ?? preferences.value.viewport.centerX,
      centerY: point?.y ?? preferences.value.viewport.centerY,
      zoom: Math.min(
        8,
        Math.max(0.1, preferences.value.viewport.zoom * action.factor),
      ),
    });
    keyboardAnnouncement.value = action.announcement;
  }
  if (action.type === "select_all") {
    selectedIds.value = action.resourceIds;
    selectionAnchor.value = action.resourceIds[0] || "";
    focusedResourceId.value = action.resourceIds[0] || "";
    selectedType.value = keyboardResources.value.find(
      (item) => item.id === focusedResourceId.value,
    )?.type;
    keyboardAnnouncement.value = action.announcement;
  }
  if (action.type === "disconnect_link") {
    const link = store.active?.links.find(
      (item) => item.id === action.resourceId,
    );
    if (link) {
      await api.disconnectLink(link.id);
      canvasStatus.value = "已通过键盘提交断开连接任务。";
      await refreshActive();
    }
  }
  if (action.type === "cancel_connection") cancelConnection();
  if (action.type === "cancel_box_selection")
    keyboardAnnouncement.value = "Box selection cancelled.";
  if (action.type === "clear_selection") clearSelection();
}

onMounted(async () => {
  window.addEventListener("keydown", keydown);
  window.addEventListener("keyup", keyup);
  window.addEventListener("blur", handleWorkspaceBlur);
  document.addEventListener("visibilitychange", cancelOnVisibilityLoss);
  try {
    await store.loadLabs();
    await store.loadTasks();
    const savedLaboratoryId = localStorage.getItem(ACTIVE_LAB_STORAGE_KEY);
    const initialLaboratory =
      store.labs.find((laboratory) => laboratory.id === savedLaboratoryId) ||
      store.labs[0];
    if (initialLaboratory) await openLaboratory(initialLaboratory.id);
  } catch (error) {
    store.error = error instanceof Error ? error.message : String(error);
  } finally {
    initialized.value = true;
  }
});

onBeforeUnmount(() => {
  window.removeEventListener("keydown", keydown);
  window.removeEventListener("keyup", keyup);
  window.removeEventListener("blur", handleWorkspaceBlur);
  document.removeEventListener("visibilitychange", cancelOnVisibilityLoss);
  store.stopEvents();
});
</script>

<template>
  <main class="h-full min-h-0">
    <div
      v-if="!initialized"
      role="status"
      class="grid h-full place-items-center bg-background text-sm text-muted-foreground"
    >
      正在加载 NetLab 工作区…
    </div>
    <LaboratoryShell
      v-else-if="store.active"
      ref="shell"
      :preferences="preferences"
      @panel="(panel, value) => setPanel(panel, value)"
    >
      <template #toolbar="{ openPalette }">
        <LaboratoryToolbar
          :labs="store.labs"
          :active="store.active.laboratory"
          :event-status="store.eventStatus"
          :loading="store.loading"
          @select="openLaboratory"
          @delete-accepted="laboratoryDeleteAccepted"
          @refresh="refresh"
          @changed="refresh"
          @toggle-palette="openPalette"
          @open-create="openCreateDrawer"
          @open-commands="commandOpen = true"
        />
      </template>
      <template #palette>
        <DevicePalette @choose="choose" />
      </template>
      <template #canvas>
        <TopologyCanvas
          ref="topologyCanvas"
          :nodes="store.active.nodes"
          :interfaces="store.active.interfaces"
          :links="store.active.links"
          :network-objects="store.active.network_objects"
          :network-attachments="store.active.network_attachments || []"
          :network-object-links="store.active.network_object_links || []"
          :tasks="store.tasks"
          :preferences="preferences"
          :shared-placements="store.active.placements"
          :selected-ids="selectedIds"
          :focused-resource-id="focusedResourceId"
          :keyboard-announcement="keyboardAnnouncement"
          :editing-link-id="editingRouteLinkId"
          :pan-enabled="effectivePanEnabled"
          :laboratory-id="store.active.laboratory.id"
          :connection-source-interface-id="pendingEndpoint"
          :connection-source-object-port-id="
            pendingObjectPort
              ? `${pendingObjectPort.objectId}:${pendingObjectPort.portName}`
              : ''
          "
          :traffic="trafficObservations"
          :traffic-active="trafficOverlayActive"
          :traffic-color="trafficOverlayColor"
          :capture-connection-ids="captureOverlay.connectionIds"
          :capture-interface-ids="captureOverlay.interfaceIds"
          @select="selectResource"
          @connector="startConnection"
          @move="moveResource"
          @viewport="setViewport"
          @interface="interfaceClicked"
          @object-port="objectPortClicked"
          @connection-start="
            (source) => setConnectionSource(source, 'port_drag')
          "
          @connection-drop="handleUnifiedConnectionDrop"
          @connection-cancel="cancelConnection"
          @keyboard="topologyKeyboard"
          @box-select="selectBox"
          @route-point="updateRoutePoint"
          @transient-cancelled="keyboardAnnouncement = '已取消临时交互。'"
          @context="openResourceContext"
          @background="cancelOrClear"
        />
        <div class="absolute left-3 right-28 top-3 flex flex-wrap gap-2">
          <Button
            size="sm"
            :variant="effectivePanEnabled ? 'default' : 'ghost'"
            :aria-pressed="effectivePanEnabled"
            title="切换画布平移模式（按住 Ctrl 临时启用）"
            data-testid="pan-view"
            @click="panEnabled = !panEnabled"
          >
            <Hand :size="13" />
            {{ effectivePanEnabled ? "正在平移" : "平移视图" }}
          </Button>
          <Button
            size="sm"
            variant="secondary"
            title="适应全部资源"
            data-testid="fit-all"
            @click="fitResources()"
          >
            <Maximize2 :size="13" /> 适应全部
          </Button>
          <Button
            size="sm"
            variant="secondary"
            title="适应选中资源"
            data-testid="fit-selection"
            :disabled="!selectedIds.length"
            @click="fitResources(selectedIds)"
          >
            <Focus :size="13" /> 适应选中项
          </Button>
          <Button
            size="sm"
            variant="ghost"
            title="重置拓扑视图"
            data-testid="reset-view"
            @click="resetViewport"
          >
            <RotateCcw :size="13" /> 重置
          </Button>
          <Button
            size="sm"
            variant="ghost"
            aria-label="切换拓扑标签密度"
            @click="cycleLabelDensity"
          >
            标签：{{ labelDensityText(preferences.labelDensity) }}
          </Button>
          <Button
            v-if="selectedIds.length > 1"
            size="sm"
            variant="secondary"
            @click="groupSelection"
          >
            <Group :size="13" /> 将选中项分组
          </Button>
          <LinkContextMenu
            v-if="selectedLink"
            kind="node_link"
            @inspect="shell?.openInspector()"
            @reconnect="requestReconnect"
            @disconnect="disconnectSelectedLink"
            @route="toggleSelectedRoute"
            @capture="openSelectedConnectionCapture"
            @traffic-filter="openSelectedConnectionTrafficFilter"
          />
          <LinkContextMenu
            v-else-if="selectedObjectLink"
            kind="network_object_link"
            :pending="deletingObjectLinkIds.includes(selectedObjectLink.id)"
            @inspect="shell?.openInspector()"
            @delete="deleteObjectLink(selectedObjectLink)"
            @capture="openSelectedConnectionCapture"
            @traffic-filter="openSelectedConnectionTrafficFilter"
          />
          <LinkContextMenu
            v-else-if="selectedAttachment"
            kind="network_attachment"
            :pending="deletingAttachmentIds.includes(selectedAttachment.id)"
            @inspect="shell?.openInspector()"
            @delete="deleteAttachment(selectedAttachment)"
            @capture="openSelectedConnectionCapture"
            @traffic-filter="openSelectedConnectionTrafficFilter"
          />
          <Button
            v-if="failedObjectLinkDelete"
            size="sm"
            variant="destructive"
            @click="retryObjectLinkDelete"
          >
            重试删除链路
          </Button>
          <template v-if="editingRouteLinkId">
            <Button size="sm" variant="secondary" @click="finishRouteEdit">
              保存路径
            </Button>
            <Button size="sm" variant="ghost" @click="cancelRouteEdit">
              取消路径编辑
            </Button>
            <Button size="sm" variant="ghost" @click="resetRouteEdit">
              恢复自动布线
            </Button>
          </template>
          <Button
            v-for="group in preferences.groups"
            :key="group.id"
            size="sm"
            variant="ghost"
            @click="toggleGroup(group.id)"
          >
            {{ group.collapsed ? "展开" : "收起" }} {{ group.label }}
          </Button>
        </div>
        <p
          v-if="canvasStatus"
          role="status"
          class="absolute bottom-3 right-3 rounded border border-border bg-card/90 px-2 py-1 text-xs"
        >
          <Cable :size="12" class="mr-1 inline" />{{ canvasStatus }}
        </p>
        <div
          v-if="activeReconnectTask"
          class="absolute bottom-12 right-3 w-80 rounded border border-border bg-card/95 p-3 text-xs shadow-xl"
          data-testid="reconnect-task-feedback"
        >
          <div class="flex items-center justify-between gap-2">
            <strong>重新连接：{{ activeReconnectTask.state }}</strong>
            <span class="text-muted-foreground">
              {{ activeReconnectTask.progress_current }}/{{
                activeReconnectTask.progress_total
              }}
            </span>
          </div>
          <p class="mt-1 text-muted-foreground">
            任务成功前，原始端点仍是服务器权威状态。
          </p>
          <p
            v-if="activeReconnectTask.error"
            class="mt-2 text-destructive"
            role="alert"
          >
            {{ activeReconnectTask.error.message }}
          </p>
          <div class="mt-2 flex justify-end gap-2">
            <Button
              v-if="['queued', 'running'].includes(activeReconnectTask.state)"
              size="sm"
              variant="ghost"
              aria-label="取消重连"
              @click="cancelReconnect"
            >
              <XCircle :size="13" /> 取消
            </Button>
            <Button
              v-if="['failed', 'cancelled'].includes(activeReconnectTask.state)"
              size="sm"
              variant="secondary"
              aria-label="重试连接"
              @click="retryReconnect"
            >
              <RotateCcw :size="13" /> 重试
            </Button>
          </div>
        </div>
        <div
          v-if="store.error"
          role="alert"
          class="absolute bottom-3 left-3 rounded border border-destructive/40 bg-card p-2 text-xs text-destructive"
        >
          {{ store.error }}
        </div>
      </template>
      <template #inspector>
        <TopologyInspector
          :laboratory-id="store.active.laboratory.id"
          :node="selectedNode"
          :link="selectedLink"
          :attachment="selectedAttachment"
          :network-object-link="selectedObjectLink"
          :network-object="selectedObject"
          :interfaces="store.active.interfaces"
          :nodes="store.active.nodes"
          :network-objects="store.active.network_objects"
          :attachments="store.active.network_attachments || []"
          :network-object-links="store.active.network_object_links || []"
          @changed="refreshActive"
          @clear="clearSelection"
          @terminal="openNodeTerminal"
          @network-object-terminal="openNetworkObjectTerminal"
          @capture-connection="openSelectedConnectionCapture"
          @filter-connection="openSelectedConnectionTrafficFilter"
          @diagnostics-loaded="canvasStatus = `${$event} 的运行诊断已加载。`"
        />
      </template>
      <template #bottom>
        <OperationsDrawer
          :model-value="preferences.activeBottomTab"
          :tasks="store.tasks"
          :laboratory-id="store.active.laboratory.id"
          :selected-node-id="selectedNode?.id"
          :selected-interface="selectedInterface"
          :selected-link-id="selectedLink?.id"
          :selected-object-link-id="selectedObjectLink?.id"
          :interface-owners="interfaceOwners"
          :coordinates="coordinates"
          :resource-ids="resourceIds"
          :nodes="store.active.nodes"
          :interfaces="store.active.interfaces"
          :links="store.active.links"
          :attachments="store.active.network_attachments || []"
          :network-object-links="store.active.network_object_links || []"
          :network-objects="store.active.network_objects"
          :console-request-node-id="consoleRequestNodeId"
          :console-request-network-object-id="consoleRequestNetworkObjectId"
          :console-request-key="consoleRequestKey"
          @update:model-value="
            (value) => setActiveBottomTab(value as BottomTab)
          "
          @refresh-tasks="store.loadTasks"
          @navigate="navigate"
          @traffic-overlay="
            (observations, active, color) => {
              trafficObservations = observations;
              trafficOverlayActive = active;
              trafficOverlayColor = color || '#f59e0b';
            }
          "
          @capture-overlay="captureOverlay = $event"
        />
      </template>
    </LaboratoryShell>
    <Teleport to="body">
      <div
        v-if="resourceContext"
        class="fixed inset-0 z-[65]"
        @pointerdown.self="closeResourceContext"
        @contextmenu.prevent.self="closeResourceContext"
      >
        <section
          role="menu"
          :aria-label="
            contextNode
              ? `${contextNode.name} 的操作`
              : contextObject
                ? `${contextObject.name} 的操作`
                : contextObjectLink
                  ? `${contextObjectLink.id} 的操作`
                  : '链路操作'
          "
          class="netlab-scrollbar fixed w-56 max-w-[calc(100vw-1rem)] overflow-y-auto rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-2xl"
          :style="resourceContextStyle"
        >
          <template v-if="contextNode">
            <Button
              variant="ghost"
              size="sm"
              class="w-full justify-start"
              role="menuitem"
              @click="openNodeTerminal(contextNode)"
            >
              <TerminalSquare :size="13" class="shrink-0" /> 终端
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="w-full justify-start"
              role="menuitem"
              :disabled="contextNode.observed_state !== 'running'"
              :title="
                contextNode.observed_state === 'running'
                  ? '选择该节点的一个接口进行实时抓包'
                  : '请先启动节点，再选择接口抓包'
              "
              @click="requestContextNodeCapture(contextNode)"
            >
              <Radio :size="13" class="shrink-0" /> 抓取接口流量…
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="w-full justify-start"
              role="menuitem"
              @click="setContextNodeState(contextNode)"
            >
              <Play v-if="contextNode.desired_state !== 'running'" :size="13" />
              <Square v-else :size="13" />
              {{ contextNode.desired_state === "running" ? "停止" : "启动" }}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="w-full justify-start text-destructive"
              role="menuitem"
              @click="requestContextNodeDelete(contextNode)"
            >
              <Trash2 :size="13" class="shrink-0" /> 删除
            </Button>
          </template>
          <template v-else-if="contextLink">
            <Button
              variant="ghost"
              size="sm"
              class="w-full justify-start"
              role="menuitem"
              @click="
                closeResourceContext();
                requestReconnect();
              "
            >
              <Cable :size="13" class="shrink-0" /> 重新连接端点
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="w-full justify-start"
              role="menuitem"
              @click="
                closeResourceContext();
                toggleSelectedRoute();
              "
            >
              编辑路由
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="w-full justify-start text-destructive"
              role="menuitem"
              @click="
                closeResourceContext();
                disconnectSelectedLink();
              "
            >
              断开连接
            </Button>
          </template>
          <template v-else-if="contextObjectLink">
            <Button
              variant="ghost"
              size="sm"
              class="w-full justify-start text-destructive"
              role="menuitem"
              :disabled="deletingObjectLinkIds.includes(contextObjectLink.id)"
              @click="deleteObjectLink(contextObjectLink)"
            >
              <Trash2 :size="13" class="shrink-0" /> 删除链路
            </Button>
          </template>
          <template v-else-if="contextObject">
            <Button
              v-if="contextObject.kind === 'pc'"
              variant="ghost"
              size="sm"
              class="w-full justify-start"
              role="menuitem"
              @click="openNetworkObjectTerminal(contextObject)"
            >
              <TerminalSquare :size="13" class="shrink-0" /> 终端
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="w-full justify-start text-destructive"
              role="menuitem"
              @click="requestContextObjectDelete(contextObject)"
            >
              <Trash2 :size="13" class="shrink-0" /> 删除
            </Button>
          </template>
        </section>
      </div>
    </Teleport>
    <ConfirmationDialog
      :model-value="Boolean(contextDeleteNode)"
      title="删除节点"
      :resource="
        contextDeleteNode
          ? `${contextDeleteNode.name} · ${contextDeleteNode.id}`
          : ''
      "
      description="如果节点正在运行，将先停止节点，再删除节点及其拥有的全部运行时资源。"
      impact="所有相连链路都会删除，该节点的活动终端或抓包会话也会关闭。"
      confirm-label="删除节点"
      @update:model-value="!$event && (contextDeleteNode = undefined)"
      @confirm="confirmContextNodeDelete"
    />
    <ConfirmationDialog
      :model-value="Boolean(contextDeleteObject)"
      title="删除网络对象"
      :resource="
        contextDeleteObject
          ? `${contextDeleteObject.name} · ${contextDeleteObject.id}`
          : ''
      "
      description="这将删除网络对象及其拥有的宿主机网络资源。"
      impact="删除网桥、DHCP 辅助进程和 NAT 规则前，会先断开该对象上的所有接口连接。"
      confirm-label="删除网络对象"
      @update:model-value="!$event && (contextDeleteObject = undefined)"
      @confirm="confirmContextObjectDelete"
    />
    <div v-if="!store.active" class="flex h-full flex-col bg-background">
      <LaboratoryToolbar
        :labs="store.labs"
        event-status="disconnected"
        :loading="store.loading"
        :palette-available="false"
        @select="openLaboratory"
        @delete-accepted="laboratoryDeleteAccepted"
        @refresh="refresh"
        @changed="refresh"
        @toggle-palette="() => {}"
        @open-commands="commandOpen = true"
      />
      <div class="grid flex-1 place-items-center text-center">
        <p class="text-sm text-muted-foreground">
          {{ store.error || "No laboratory yet. Create one from the toolbar." }}
        </p>
      </div>
    </div>
    <PortChooser
      v-model="portChooserOpen"
      :title="
        portChooserMode === 'source'
          ? '选择源接口'
          : portChooserMode === 'target'
            ? '选择目标接口'
            : portChooserMode === 'capture'
              ? '选择抓包接口'
              : '选择替换接口'
      "
      :description="
        portChooserMode === 'capture'
          ? 'Select the node interface whose live packets should stream to Capture and Wireshark.'
          : undefined
      "
      :interfaces="portChooserInterfaces"
      :mode="portChooserMode"
      :endpoints="
        portChooserMode === 'source' || portChooserMode === 'target'
          ? portChooserEndpoints
          : undefined
      "
      @choose="portChosen"
      @cancel="cancelConnection"
    />
    <CreateTopologyResourceDrawer
      v-if="store.active"
      ref="createDrawer"
      :model-value="createOpen"
      :laboratory-id="store.active.laboratory.id"
      :laboratory-revision="store.active.laboratory.revision"
      :placement-intent="createPlacementIntent"
      :selection="paletteSelection"
      :node-names="store.active.nodes.map((node) => node.name)"
      :network-object-names="
        store.active.network_objects.map((item) => item.name)
      "
      @update:model-value="setCreateOpen"
      @selection-changed="paletteSelection = $event"
      @created="created"
    />
    <CommandPalette
      v-model="commandOpen"
      :actions="commands"
      @run="runCommand"
    />
  </main>
</template>
