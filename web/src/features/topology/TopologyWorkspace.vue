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
  type TrafficObservation,
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
import { useLaboratoryStore } from "@/stores/laboratory";
import type { BottomTab } from "@/types/workspace";
import CreateTopologyResourceDialog from "./CreateTopologyResourceDialog.vue";
import DevicePalette, { type PaletteSelection } from "./DevicePalette.vue";
import LinkContextMenu from "./LinkContextMenu.vue";
import PortChooser from "./PortChooser.vue";
import TopologyCanvas from "./TopologyCanvas.vue";
import TopologyInspector from "./TopologyInspector.vue";
import { fitViewport } from "./topologyGeometry";
import { TopologyKeyboardController } from "./topologyKeyboardController";
import { resolvePlacements } from "./topologyLayout";
import { buildPlacementBatch } from "./topologyPlacementBatch";
import {
  boxSelect,
  cleanSelection,
  rangeSelect,
  selectOne,
  toggleSelected,
} from "./topologySelection";

const store = useLaboratoryStore();
const ACTIVE_LAB_STORAGE_KEY = "netlab.active-laboratory.v1";
const shell = ref<InstanceType<typeof LaboratoryShell>>();
const topologyCanvas = ref<InstanceType<typeof TopologyCanvas>>();
const initialized = ref(false);
const panEnabled = ref(false);
const resourceContext = ref<{
  id: string;
  type: "node" | "link" | "network_object";
  x: number;
  y: number;
}>();
const contextDeleteNode = ref<Node>();
const contextDeleteObject = ref<NetworkObject>();
const diagnosticsRequestKey = ref(0);
const trafficObservations = ref<TrafficObservation[]>([]);
const trafficOverlayActive = ref(false);
const trafficOverlayColor = ref("#f59e0b");
const consoleRequestNodeId = ref("");
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
const keyboardAnnouncement = ref("");
const keyboardController = new TopologyKeyboardController();
const createOpen = ref(false);
const paletteSelection = ref<PaletteSelection>();
const commandOpen = ref(false);
const showLightweight = ref(false);
const pendingEndpoint = ref("");
const pendingObjectPort = ref<{ objectId: string; portName: string }>();
const canvasStatus = ref("");
const portChooserOpen = ref(false);
const portChooserMode = ref<"source" | "target" | "reconnect" | "capture">(
  "source",
);
const portChooserInterfaces = ref<NodeInterface[]>([]);
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

function openResourceContext(
  id: string,
  type: "node" | "link" | "network_object",
  x: number,
  y: number,
) {
  selectResource(id, type, false);
  resourceContext.value = {
    id,
    type,
    x: Math.max(8, Math.min(x, window.innerWidth - 220)),
    y: Math.max(8, Math.min(y, window.innerHeight - 260)),
  };
}
function closeResourceContext() {
  resourceContext.value = undefined;
}
function openNodeTerminal(node: Node) {
  consoleRequestNodeId.value = node.id;
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
function openContextObjectDiagnostics() {
  const object = contextObject.value;
  closeResourceContext();
  shell.value?.openInspector();
  if (!object) return;
  diagnosticsRequestKey.value += 1;
  canvasStatus.value = `Loading diagnostics for ${object.name}…`;
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
  canvasStatus.value = `Ready to capture ${value.name}.`;
}
function requestContextNodeCapture(node: Node) {
  closeResourceContext();
  const candidates = nodeInterfaces(node.id);
  if (!candidates.length) {
    canvasStatus.value = "The selected node has no interfaces to capture.";
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

async function openLaboratory(id: string) {
  selectedIds.value = [];
  showLightweight.value = false;
  await store.open(id);
  await store.loadLabs();
  await store.loadTasks();
  localStorage.setItem(ACTIVE_LAB_STORAGE_KEY, id);
}

async function laboratoryDeleteAccepted(id: string) {
  const activeId = store.active?.laboratory.id;
  selectedIds.value = [];
  showLightweight.value = false;
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
  if (selection.name === "Lightweight") {
    showLightweight.value = true;
    selectedIds.value = [];
    shell.value?.openInspector();
    return;
  }
  paletteSelection.value = selection;
  createOpen.value = true;
}

async function created(value: {
  node?: Node;
  interfaces?: NodeInterface[];
  networkObject?: NetworkObject;
}) {
  if (!store.active) return;
  const center = topologyCanvas.value?.viewportCenter?.() || { x: 0, y: 0 };
  await refreshActive();
  if (
    value.node &&
    !store.active.nodes.some((item) => item.id === value.node!.id)
  )
    store.active.nodes.push(value.node);
  for (const item of value.interfaces || [])
    if (!store.active.interfaces.some((current) => current.id === item.id))
      store.active.interfaces.push(item);
  if (
    value.networkObject &&
    !store.active.network_objects.some(
      (item) => item.id === value.networkObject!.id,
    )
  )
    store.active.network_objects.push(value.networkObject);
  const resource = value.node || value.networkObject;
  if (resource) {
    const resourceType = value.node ? "node" : "network_object";
    const result = await api.updateTopologyPlacements(
      store.active.laboratory.id,
      store.active.laboratory.revision,
      [
        {
          resource_id: resource.id,
          resource_type: resourceType,
          x: center.x,
          y: center.y,
        },
      ],
    );
    store.active.laboratory.revision = result.laboratory_revision;
    for (const placement of result.placements) {
      const index = store.active.placements.findIndex(
        (item) => item.resource_id === placement.resource_id,
      );
      if (index >= 0) store.active.placements[index] = placement;
      else store.active.placements.push(placement);
    }
    selectResource(resource.id, resourceType, false);
  }
}

async function refreshActive() {
  if (store.active) await store.open(store.active.laboratory.id);
  await store.loadTasks();
}

async function lightweightCreated(id: string) {
  await refreshActive();
  selectResource(id, "network_object", false);
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
  showLightweight.value = false;
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
    canvasStatus.value =
      "Network attachment selected. Open Capture or Traffic Filter to observe this segment.";
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
  showLightweight.value = false;
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

async function objectPortClicked(objectId: string, portName: string) {
  if (!store.active) return;
  if (objectPortOccupied(objectId, portName)) {
    canvasStatus.value = `${portName} 已被占用，请选择空闲端口。`;
    return;
  }
  if (pendingEndpoint.value) {
    canvasStatus.value =
      "普通节点接口不能直接连接对象端口；请使用 Inspector 的 Attachment 操作。";
    return;
  }
  const source = pendingObjectPort.value;
  if (!source) {
    pendingObjectPort.value = { objectId, portName };
    canvasStatus.value = `已选择 ${portName}；请选择另一个网络对象的空闲端口。`;
    return;
  }
  if (source.objectId === objectId && source.portName === portName) {
    pendingObjectPort.value = undefined;
    canvasStatus.value = "对象链路创建已取消。";
    return;
  }
  if (source.objectId === objectId) {
    canvasStatus.value = "对象间链路必须连接两个不同的网络对象。";
    return;
  }
  try {
    const envelope = await api.createNetworkObjectLink(
      store.active.laboratory.id,
      {
        object_a_id: source.objectId,
        port_a_name: source.portName,
        object_b_id: objectId,
        port_b_name: portName,
      },
    );
    const index = store.tasks.findIndex((item) => item.id === envelope.task.id);
    if (index >= 0) store.tasks[index] = envelope.task;
    else store.tasks.unshift(envelope.task);
    pendingObjectPort.value = undefined;
    canvasStatus.value = `对象链路任务 ${envelope.task.id} 已提交。`;
    await refreshActive();
  } catch (value) {
    const message = value instanceof Error ? value.message : String(value);
    canvasStatus.value = message.includes("port_in_use")
      ? "端口已被其他客户端占用，拓扑已刷新，请重新选择。"
      : message;
    pendingObjectPort.value = undefined;
    await refreshActive();
  }
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

async function interfaceClicked(interfaceId: string) {
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
  if (!pendingEndpoint.value) {
    pendingEndpoint.value = interfaceId;
    canvasStatus.value = `Selected ${target.name}; choose another available port.`;
    return;
  }
  if (pendingEndpoint.value === interfaceId) {
    pendingEndpoint.value = "";
    canvasStatus.value =
      "Connection cancelled because the same port was selected twice.";
    return;
  }
  await api.connectLink(
    store.active.laboratory.id,
    pendingEndpoint.value,
    interfaceId,
  );
  canvasStatus.value = `Connected ${pendingEndpoint.value} to ${interfaceId}.`;
  pendingEndpoint.value = "";
  await refreshActive();
}

function availableInterfaces(nodeId: string) {
  return (store.active?.interfaces || []).filter(
    (item) =>
      item.node_id === nodeId &&
      !item.name.startsWith("internal") &&
      !item.desired_link_id,
  );
}

function startConnection(nodeId: string) {
  const candidates = availableInterfaces(nodeId);
  if (!candidates.length) {
    canvasStatus.value = "The selected node has no available interfaces.";
    return;
  }
  if (candidates.length === 1) {
    pendingEndpoint.value = candidates[0].id;
    canvasStatus.value = `Selected ${candidates[0].name}; choose a target node or interface.`;
    return;
  }
  portChooserMode.value = "source";
  portChooserInterfaces.value = candidates;
  portChooserOpen.value = true;
}

async function chooseTargetNode(nodeId: string) {
  const candidates = availableInterfaces(nodeId).filter(
    (item) => item.id !== pendingEndpoint.value,
  );
  if (!candidates.length) {
    canvasStatus.value = "The target node has no available interfaces.";
    return;
  }
  if (candidates.length === 1) {
    await connectPending(candidates[0]);
    return;
  }
  portChooserMode.value = "target";
  portChooserInterfaces.value = candidates;
  portChooserOpen.value = true;
}

async function connectPending(target: NodeInterface) {
  if (!store.active || !pendingEndpoint.value) return;
  const source = pendingEndpoint.value;
  await api.connectLink(store.active.laboratory.id, source, target.id);
  pendingEndpoint.value = "";
  portChooserOpen.value = false;
  canvasStatus.value = `Connected ${source} to ${target.id}.`;
  await refreshActive();
}

async function portChosen(value: NodeInterface) {
  if (portChooserMode.value === "capture") {
    openInterfaceCapture(value);
    return;
  }
  if (portChooserMode.value === "source") {
    pendingEndpoint.value = value.id;
    portChooserOpen.value = false;
    canvasStatus.value = `Selected ${value.name}; choose a target node or interface.`;
    return;
  }
  if (portChooserMode.value === "reconnect") {
    if (!reconnectingLink.value) return;
    await submitReconnect(
      reconnectingLink.value,
      reconnectRetainedEndpoint.value,
      value.id,
    );
    return;
  }
  await connectPending(value);
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
    const task = await api.cancelTask(activeReconnectTaskId.value);
    const index = store.tasks.findIndex((item) => item.id === task.id);
    if (index >= 0) store.tasks[index] = task;
    canvasStatus.value =
      "Reconnect cancellation requested; original endpoints remain authoritative.";
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
  pendingEndpoint.value = "";
  pendingObjectPort.value = undefined;
  portChooserOpen.value = false;
  portChooserMode.value = "source";
  canvasStatus.value = captureSelection
    ? "Capture interface selection cancelled."
    : "Connection cancelled; no topology mutation was sent.";
}

async function disconnectSelectedLink() {
  if (!selectedLink.value) return;
  await api.disconnectLink(selectedLink.value.id);
  canvasStatus.value = "Disconnect task submitted.";
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
    canvasStatus.value = "No available interface can replace this endpoint.";
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
  canvasStatus.value = "Created a browser-local visual group.";
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
  canvasStatus.value =
    "Drag the amber handle to adjust this browser-local route.";
}

function updateRoutePoint(linkId: string, point: { x: number; y: number }) {
  if (editingRouteLinkId.value === linkId) setLinkRoute(linkId, [point]);
}

function finishRouteEdit() {
  editingRouteLinkId.value = "";
  routeEditOriginal.value = [];
  canvasStatus.value = "Saved the browser-local link route.";
}

function cancelRouteEdit() {
  if (!editingRouteLinkId.value) return;
  setLinkRoute(editingRouteLinkId.value, routeEditOriginal.value);
  editingRouteLinkId.value = "";
  routeEditOriginal.value = [];
  canvasStatus.value = "Cancelled local route editing.";
}

function resetRouteEdit() {
  if (!editingRouteLinkId.value) return;
  setLinkRoute(editingRouteLinkId.value, []);
  editingRouteLinkId.value = "";
  routeEditOriginal.value = [];
  canvasStatus.value = "Reset the link to automatic routing.";
}

function cycleLabelDensity() {
  const values = ["comfortable", "compact", "minimal"] as const;
  const index = values.indexOf(preferences.value.labelDensity);
  setLabelDensity(values[(index + 1) % values.length]);
  canvasStatus.value = `Topology label density: ${preferences.value.labelDensity}.`;
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
  if (event.key === "Escape" && resourceContext.value) {
    closeResourceContext();
    return;
  }
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
    event.preventDefault();
    commandOpen.value = true;
  }
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
  if (document.visibilityState === "hidden") cancelWorkspaceTransient();
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
    if (nodeInterface) await interfaceClicked(nodeInterface.id);
    else {
      const object = store.active?.network_objects.find((item) =>
        action.interfaceId.startsWith(`${item.id}:`),
      );
      if (object)
        await objectPortClicked(
          object.id,
          action.interfaceId.slice(object.id.length + 1),
        );
    }
  }
  if (action.type === "begin_connection") startConnection(action.resourceId);
  if (action.type === "choose_connection_target") {
    const target = keyboardResources.value.find(
      (item) => item.id === action.resourceId,
    );
    if (target?.type === "node") await chooseTargetNode(target.id);
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
      canvasStatus.value = "Disconnect task submitted from keyboard.";
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
  window.addEventListener("blur", cancelWorkspaceTransient);
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
  window.removeEventListener("blur", cancelWorkspaceTransient);
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
      Loading NetLab workspace…
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
          :pan-enabled="panEnabled"
          :connection-source-interface-id="pendingEndpoint"
          :connection-source-object-port-id="
            pendingObjectPort
              ? `${pendingObjectPort.objectId}:${pendingObjectPort.portName}`
              : ''
          "
          :traffic="trafficObservations"
          :traffic-active="trafficOverlayActive"
          :traffic-color="trafficOverlayColor"
          @select="selectResource"
          @connector="startConnection"
          @move="moveResource"
          @viewport="setViewport"
          @interface="interfaceClicked"
          @object-port="objectPortClicked"
          @keyboard="topologyKeyboard"
          @box-select="selectBox"
          @route-point="updateRoutePoint"
          @transient-cancelled="
            keyboardAnnouncement = 'Transient interaction cancelled.'
          "
          @context="openResourceContext"
          @background="cancelOrClear"
        />
        <div class="absolute left-3 right-28 top-3 flex flex-wrap gap-2">
          <Button
            size="sm"
            :variant="panEnabled ? 'default' : 'ghost'"
            :aria-pressed="panEnabled"
            title="Toggle canvas pan mode"
            data-testid="pan-view"
            @click="panEnabled = !panEnabled"
          >
            <Hand :size="13" /> {{ panEnabled ? "Panning" : "Pan view" }}
          </Button>
          <Button
            size="sm"
            variant="secondary"
            title="Fit all resources"
            data-testid="fit-all"
            @click="fitResources()"
          >
            <Maximize2 :size="13" /> Fit all
          </Button>
          <Button
            size="sm"
            variant="secondary"
            title="Fit selected resources"
            data-testid="fit-selection"
            :disabled="!selectedIds.length"
            @click="fitResources(selectedIds)"
          >
            <Focus :size="13" /> Fit selection
          </Button>
          <Button
            size="sm"
            variant="ghost"
            title="Reset topology view"
            data-testid="reset-view"
            @click="resetViewport"
          >
            <RotateCcw :size="13" /> Reset
          </Button>
          <Button
            size="sm"
            variant="ghost"
            aria-label="Cycle topology label density"
            @click="cycleLabelDensity"
          >
            Labels: {{ preferences.labelDensity }}
          </Button>
          <Button
            v-if="selectedIds.length > 1"
            size="sm"
            variant="secondary"
            @click="groupSelection"
          >
            <Group :size="13" /> Group selection
          </Button>
          <LinkContextMenu
            v-if="selectedLink"
            @inspect="shell?.openInspector()"
            @reconnect="requestReconnect"
            @disconnect="disconnectSelectedLink"
            @route="toggleSelectedRoute"
          />
          <template v-if="editingRouteLinkId">
            <Button size="sm" variant="secondary" @click="finishRouteEdit">
              Save route
            </Button>
            <Button size="sm" variant="ghost" @click="cancelRouteEdit">
              Cancel route
            </Button>
            <Button size="sm" variant="ghost" @click="resetRouteEdit">
              Reset route
            </Button>
          </template>
          <Button
            v-for="group in preferences.groups"
            :key="group.id"
            size="sm"
            variant="ghost"
            @click="toggleGroup(group.id)"
          >
            {{ group.collapsed ? "Expand" : "Collapse" }} {{ group.label }}
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
            <strong>Reconnect {{ activeReconnectTask.state }}</strong>
            <span class="text-muted-foreground">
              {{ activeReconnectTask.progress_current }}/{{
                activeReconnectTask.progress_total
              }}
            </span>
          </div>
          <p class="mt-1 text-muted-foreground">
            Original endpoints remain authoritative until this task succeeds.
          </p>
          <p
            v-if="activeReconnectTask.error"
            class="mt-2 text-red-300"
            role="alert"
          >
            {{ activeReconnectTask.error.message }}
          </p>
          <div class="mt-2 flex justify-end gap-2">
            <Button
              v-if="['queued', 'running'].includes(activeReconnectTask.state)"
              size="sm"
              variant="ghost"
              aria-label="Cancel reconnect"
              @click="cancelReconnect"
            >
              <XCircle :size="13" /> Cancel
            </Button>
            <Button
              v-if="['failed', 'cancelled'].includes(activeReconnectTask.state)"
              size="sm"
              variant="secondary"
              aria-label="Retry reconnect"
              @click="retryReconnect"
            >
              <RotateCcw :size="13" /> Retry
            </Button>
          </div>
        </div>
        <div
          v-if="store.error"
          role="alert"
          class="absolute bottom-3 left-3 rounded border border-destructive/40 bg-card p-2 text-xs text-red-300"
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
          :show-lightweight="showLightweight"
          :diagnostics-request-key="diagnosticsRequestKey"
          @changed="refreshActive"
          @clear="clearSelection"
          @terminal="openNodeTerminal"
          @diagnostics-loaded="
            canvasStatus = `Diagnostics loaded for ${$event}.`
          "
          @lightweight-created="lightweightCreated"
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
              ? `Actions for ${contextNode.name}`
              : contextObject
                ? `Actions for ${contextObject.name}`
                : 'Link actions'
          "
          class="fixed w-52 rounded-md border border-border bg-popover p-1 shadow-2xl"
          :style="{
            left: `${resourceContext.x}px`,
            top: `${resourceContext.y}px`,
          }"
        >
          <template v-if="contextNode">
            <Button
              variant="ghost"
              size="sm"
              class="w-full justify-start"
              role="menuitem"
              @click="openNodeTerminal(contextNode)"
            >
              <TerminalSquare :size="13" /> Terminal
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="w-full justify-start"
              role="menuitem"
              :disabled="contextNode.observed_state !== 'running'"
              :title="
                contextNode.observed_state === 'running'
                  ? 'Choose one of this node’s interfaces for live capture'
                  : 'Start the node before capturing an interface'
              "
              @click="requestContextNodeCapture(contextNode)"
            >
              <Radio :size="13" /> Capture interface…
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
              {{ contextNode.desired_state === "running" ? "Stop" : "Start" }}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="w-full justify-start text-red-300"
              role="menuitem"
              @click="requestContextNodeDelete(contextNode)"
            >
              <Trash2 :size="13" /> Delete
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
                shell?.openInspector();
              "
            >
              Inspect
            </Button>
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
              <Cable :size="13" /> Reconnect endpoint
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
              Edit route
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="w-full justify-start text-red-300"
              role="menuitem"
              @click="
                closeResourceContext();
                disconnectSelectedLink();
              "
            >
              Disconnect
            </Button>
          </template>
          <template v-else-if="contextObject">
            <Button
              variant="ghost"
              size="sm"
              class="w-full justify-start"
              role="menuitem"
              @click="openContextObjectDiagnostics"
            >
              Inspect & diagnostics
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="w-full justify-start text-red-300"
              role="menuitem"
              @click="requestContextObjectDelete(contextObject)"
            >
              <Trash2 :size="13" /> Delete
            </Button>
          </template>
        </section>
      </div>
    </Teleport>
    <ConfirmationDialog
      :model-value="Boolean(contextDeleteNode)"
      title="Delete node"
      :resource="
        contextDeleteNode
          ? `${contextDeleteNode.name} · ${contextDeleteNode.id}`
          : ''
      "
      description="This stops the node if it is running, then removes the node and all owned runtime resources."
      impact="Every attached link is deleted, and active console or capture sessions for this node are closed."
      confirm-label="Delete node"
      @update:model-value="!$event && (contextDeleteNode = undefined)"
      @confirm="confirmContextNodeDelete"
    />
    <ConfirmationDialog
      :model-value="Boolean(contextDeleteObject)"
      title="Delete network object"
      :resource="
        contextDeleteObject
          ? `${contextDeleteObject.name} · ${contextDeleteObject.id}`
          : ''
      "
      description="This removes the network object and its owned host networking resources."
      impact="Every interface attachment to this object is detached before its bridge, DHCP helper, and NAT rules are removed."
      confirm-label="Delete network object"
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
          ? 'Choose source interface'
          : portChooserMode === 'target'
            ? 'Choose target interface'
            : portChooserMode === 'capture'
              ? 'Choose interface to capture'
              : 'Choose replacement interface'
      "
      :description="
        portChooserMode === 'capture'
          ? 'Select the node interface whose live packets should stream to Capture and Wireshark.'
          : undefined
      "
      :interfaces="portChooserInterfaces"
      @choose="portChosen"
      @cancel="cancelConnection"
    />
    <CreateTopologyResourceDialog
      v-if="store.active"
      v-model="createOpen"
      :laboratory-id="store.active.laboratory.id"
      :selection="paletteSelection"
      @created="created"
    />
    <CommandPalette
      v-model="commandOpen"
      :actions="commands"
      @run="runCommand"
    />
  </main>
</template>
