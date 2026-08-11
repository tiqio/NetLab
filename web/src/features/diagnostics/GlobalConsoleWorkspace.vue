<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { Monitor, Plus, TerminalSquare, X } from "lucide-vue-next";
import {
  api,
  type ConsoleDescriptor,
  type NetworkObject,
  type Node,
} from "@/api";
import { Button } from "@/components/ui";
import { randomUUID } from "@/lib/uuid";
import ConsoleWorkspace from "./ConsoleWorkspace.vue";

const props = defineProps<{
  laboratoryId?: string;
  nodes: Node[];
  networkObjects?: NetworkObject[];
  requestNodeId?: string;
  requestNetworkObjectId?: string;
  requestKey?: number;
}>();
interface TerminalSession {
  id: string;
  sequence: number;
  mode: ConsoleDescriptor["mode"];
}
interface NodeConsoleWorkspace {
  nodeId: string;
  sessions: TerminalSession[];
  activeSessionId: string;
}

const workspaces = ref<NodeConsoleWorkspace[]>([]);
const activeNodeId = ref("");
const descriptorsByNode = ref<Record<string, ConsoleDescriptor[]>>({});
const STORAGE_PREFIX = "netlab.console-workspaces.v1:";
let restoring = false;
const activeWorkspace = computed(() =>
  workspaces.value.find((workspace) => workspace.nodeId === activeNodeId.value),
);
const activeNode = computed(() =>
  props.nodes.find((node) => node.id === activeWorkspace.value?.nodeId),
);
const activePC = computed(() =>
  props.networkObjects?.find(
    (object) =>
      object.id === activeWorkspace.value?.nodeId && object.kind === "pc",
  ),
);
const activeKind = computed(
  () => activeNode.value?.kind || activePC.value?.kind,
);
const canAddSerial = computed(
  () =>
    activeKind.value === "docker" ||
    !activeWorkspace.value?.sessions.some(
      (session) => session.mode === "telnet",
    ),
);
const activeDescriptors = computed(
  () => descriptorsByNode.value[activeWorkspace.value?.nodeId || ""] || [],
);
const canAddSSH = computed(() =>
  activeDescriptors.value.some((descriptor) => descriptor.mode === "ssh"),
);
const canAddVNC = computed(() =>
  activeDescriptors.value.some((descriptor) => descriptor.mode === "vnc"),
);
const canUseSerial = computed(() =>
  activeDescriptors.value.some((descriptor) => descriptor.mode === "telnet"),
);

function storageKey(laboratoryId = props.laboratoryId) {
  return laboratoryId ? `${STORAGE_PREFIX}${laboratoryId}` : "";
}
function validSession(value: unknown): value is TerminalSession {
  if (!value || typeof value !== "object") return false;
  const session = value as Partial<TerminalSession>;
  return (
    typeof session.id === "string" &&
    Number.isInteger(session.sequence) &&
    Number(session.sequence) > 0 &&
    (session.mode === "ssh" ||
      session.mode === "telnet" ||
      session.mode === "vnc")
  );
}
function normalizeSessions(nodeId: string, sessions: TerminalSession[]) {
  if (
    props.nodes.find((node) => node.id === nodeId)?.kind === "docker" ||
    isNetworkObject(nodeId)
  )
    return sessions;
  let foundTelnet = false;
  return sessions.filter((session) => {
    if (session.mode !== "telnet") return true;
    if (foundTelnet) return false;
    foundTelnet = true;
    return true;
  });
}
function isNetworkObject(id: string) {
  return Boolean(
    props.networkObjects?.some(
      (object) => object.id === id && object.kind === "pc",
    ),
  );
}
function restore(laboratoryId?: string) {
  restoring = true;
  const key = storageKey(laboratoryId);
  if (!key) {
    workspaces.value = [];
    activeNodeId.value = "";
    restoring = false;
    return;
  }
  try {
    const raw = localStorage.getItem(key);
    if (!raw) {
      workspaces.value = [];
      activeNodeId.value = "";
      return;
    }
    const stored = JSON.parse(raw) as {
      activeNodeId?: string;
      workspaces?: Array<Partial<NodeConsoleWorkspace>>;
    };
    workspaces.value = (stored.workspaces || [])
      .filter(
        (workspace) =>
          typeof workspace.nodeId === "string" &&
          Array.isArray(workspace.sessions),
      )
      .slice(0, 32)
      .map((workspace) => {
        const nodeId = String(workspace.nodeId);
        const sessions = normalizeSessions(
          nodeId,
          (workspace.sessions || []).filter(validSession).slice(0, 32),
        );
        const activeSessionId = sessions.some(
          (session) => session.id === workspace.activeSessionId,
        )
          ? String(workspace.activeSessionId)
          : sessions[0]?.id || "";
        return {
          nodeId,
          sessions,
          activeSessionId,
        };
      })
      .filter((workspace) => workspace.sessions.length > 0);
    activeNodeId.value = workspaces.value.some(
      (workspace) => workspace.nodeId === stored.activeNodeId,
    )
      ? String(stored.activeNodeId)
      : workspaces.value[0]?.nodeId || "";
  } catch {
    workspaces.value = [];
    activeNodeId.value = "";
    localStorage.removeItem(key);
  } finally {
    restoring = false;
  }
}
function persist() {
  if (restoring) return;
  const key = storageKey();
  if (!key) return;
  try {
    if (!workspaces.value.length) {
      localStorage.removeItem(key);
      return;
    }
    localStorage.setItem(
      key,
      JSON.stringify({
        activeNodeId: activeNodeId.value,
        workspaces: workspaces.value,
      }),
    );
  } catch {
    return;
  }
}

function addSession(
  workspace: NodeConsoleWorkspace,
  mode?: ConsoleDescriptor["mode"],
) {
  const resolvedMode =
    mode ||
    (props.nodes.find((node) => node.id === workspace.nodeId)?.kind === "docker"
      ? "telnet"
      : "telnet");
  if (
    resolvedMode === "telnet" &&
    props.nodes.find((node) => node.id === workspace.nodeId)?.kind !==
      "docker" &&
    !isNetworkObject(workspace.nodeId)
  ) {
    const existing = workspace.sessions.find(
      (session) => session.mode === "telnet",
    );
    if (existing) {
      workspace.activeSessionId = existing.id;
      return;
    }
  }
  const sequence =
    Math.max(0, ...workspace.sessions.map((session) => session.sequence)) + 1;
  const session = {
    id: randomUUID(),
    sequence,
    mode: resolvedMode,
  };
  workspace.sessions.push(session);
  workspace.activeSessionId = session.id;
}
async function loadDescriptors(nodeId: string) {
  try {
    const descriptors = isNetworkObject(nodeId)
      ? await api.listNetworkObjectConsoles(nodeId)
      : await api.listNodeConsoles(nodeId);
    descriptorsByNode.value = {
      ...descriptorsByNode.value,
      [nodeId]: descriptors,
    };
    return descriptors;
  } catch {
    const fallback: ConsoleDescriptor[] = [];
    descriptorsByNode.value = {
      ...descriptorsByNode.value,
      [nodeId]: fallback,
    };
    return fallback;
  }
}
async function openNode(nodeId: string) {
  const descriptors = await loadDescriptors(nodeId);
  let workspace = workspaces.value.find((item) => item.nodeId === nodeId);
  if (!workspace) {
    workspace = { nodeId, sessions: [], activeSessionId: "" };
    workspaces.value.push(workspace);
  }
  const supportedModes = new Set(
    descriptors.map((descriptor) => descriptor.mode),
  );
  workspace.sessions = workspace.sessions.filter((session) =>
    supportedModes.has(session.mode),
  );
  if (!workspace.sessions.length) {
    const preferred =
      descriptors.find((descriptor) => descriptor.mode === "telnet") ||
      descriptors.find((descriptor) => descriptor.mode === "ssh") ||
      descriptors[0];
    if (preferred) addSession(workspace, preferred.mode);
  }
  if (
    !workspace.sessions.some(
      (session) => session.id === workspace.activeSessionId,
    )
  )
    workspace.activeSessionId = workspace.sessions[0]?.id || "";
  activeNodeId.value = nodeId;
}

watch(
  () => props.laboratoryId,
  (laboratoryId) => restore(laboratoryId),
  { immediate: true },
);
watch(
  () => [props.requestNodeId, props.requestKey] as const,
  ([nodeId]) => {
    if (!nodeId) return;
    void openNode(nodeId);
  },
  { immediate: true },
);
watch(
  () => [props.requestNetworkObjectId, props.requestKey] as const,
  ([objectId]) => {
    if (!objectId) return;
    void openNode(objectId);
  },
  { immediate: true },
);
watch([workspaces, activeNodeId], persist, { deep: true, flush: "sync" });
watch(
  () => [
    ...props.nodes.map((node) => node.id),
    ...(props.networkObjects || [])
      .filter((object) => object.kind === "pc")
      .map((object) => object.id),
  ],
  (resourceIds) => {
    const available = new Set(resourceIds);
    workspaces.value = workspaces.value.filter((workspace) =>
      available.has(workspace.nodeId),
    );
    if (!available.has(activeNodeId.value))
      activeNodeId.value = workspaces.value[0]?.nodeId || "";
  },
  { immediate: true },
);

function label(nodeId: string) {
  return (
    props.nodes.find((node) => node.id === nodeId)?.name ||
    props.networkObjects?.find((object) => object.id === nodeId)?.name ||
    nodeId
  );
}
function modeLabel(mode: ConsoleDescriptor["mode"]) {
  return mode === "telnet" ? "SERIAL" : mode.toUpperCase();
}
function closeNode(nodeId: string) {
  const index = workspaces.value.findIndex(
    (workspace) => workspace.nodeId === nodeId,
  );
  const workspace = workspaces.value[index];
  for (const session of workspace?.sessions || []) {
    if (isNetworkObject(nodeId))
      void api.closeNetworkObjectConsoleSession(
        nodeId,
        session.mode,
        session.id,
      );
    else void api.closeNodeConsoleSession(nodeId, session.mode, session.id);
  }
  workspaces.value = workspaces.value.filter(
    (workspace) => workspace.nodeId !== nodeId,
  );
  if (activeNodeId.value === nodeId)
    activeNodeId.value = workspaces.value[Math.max(0, index - 1)]?.nodeId || "";
}
function closeSession(workspace: NodeConsoleWorkspace, sessionId: string) {
  const index = workspace.sessions.findIndex(
    (session) => session.id === sessionId,
  );
  const session = workspace.sessions[index];
  if (session) {
    if (isNetworkObject(workspace.nodeId))
      void api.closeNetworkObjectConsoleSession(
        workspace.nodeId,
        session.mode,
        session.id,
      );
    else
      void api.closeNodeConsoleSession(
        workspace.nodeId,
        session.mode,
        session.id,
      );
  }
  workspace.sessions = workspace.sessions.filter(
    (session) => session.id !== sessionId,
  );
  if (!workspace.sessions.length) {
    closeNode(workspace.nodeId);
    return;
  }
  if (workspace.activeSessionId === sessionId)
    workspace.activeSessionId =
      workspace.sessions[Math.max(0, index - 1)]?.id || "";
}
function addForActiveNode() {
  if (!activeWorkspace.value) return;
  if (activeKind.value === "docker" || activeKind.value === "pc")
    addSession(activeWorkspace.value, "telnet");
  else if (canAddSSH.value) addSession(activeWorkspace.value, "ssh");
}
function addVNCForActiveNode() {
  if (activeWorkspace.value) addSession(activeWorkspace.value, "vnc");
}
function addSerialForActiveNode() {
  if (activeWorkspace.value) addSession(activeWorkspace.value, "telnet");
}
function isActiveSession(
  workspace: NodeConsoleWorkspace,
  session: TerminalSession,
) {
  return (
    activeNodeId.value === workspace.nodeId &&
    workspace.activeSessionId === session.id
  );
}
function shouldMountSession(
  workspace: NodeConsoleWorkspace,
  session: TerminalSession,
) {
  return session.mode !== "vnc" || isActiveSession(workspace, session);
}
</script>

<template>
  <div class="flex h-full min-h-[180px] flex-col" data-global-console-workspace>
    <nav
      v-if="workspaces.length"
      class="netlab-scrollbar flex shrink-0 gap-1 overflow-x-auto border-b border-border bg-muted/20 p-1"
      data-layout-region="console-resources"
      aria-label="节点终端工作区"
    >
      <div
        v-for="workspace in workspaces"
        :key="workspace.nodeId"
        class="flex shrink-0 items-center"
      >
        <Button
          size="sm"
          :variant="activeNodeId === workspace.nodeId ? 'default' : 'ghost'"
          @click="activeNodeId = workspace.nodeId"
        >
          {{ label(workspace.nodeId) }}
        </Button>
        <Button
          size="icon"
          variant="ghost"
          :aria-label="`关闭 ${label(workspace.nodeId)} 终端工作区`"
          @click="closeNode(workspace.nodeId)"
        >
          <X :size="12" />
        </Button>
      </div>
    </nav>
    <nav
      v-if="activeWorkspace"
      class="netlab-scrollbar flex shrink-0 gap-1 overflow-x-auto border-b border-border bg-muted/10 p-1"
      data-layout-region="console-sessions"
      aria-label="终端会话"
    >
      <div
        v-for="session in activeWorkspace.sessions"
        :key="session.id"
        class="flex shrink-0 items-center"
      >
        <Button
          size="sm"
          :variant="
            activeWorkspace.activeSessionId === session.id ? 'default' : 'ghost'
          "
          @click="activeWorkspace.activeSessionId = session.id"
        >
          {{ modeLabel(session.mode) }} {{ session.sequence }}
        </Button>
        <Button
          size="icon"
          variant="ghost"
          :aria-label="`关闭 ${modeLabel(session.mode)} ${session.sequence}`"
          @click="closeSession(activeWorkspace, session.id)"
        >
          <X :size="12" />
        </Button>
      </div>
      <Button
        class="ml-auto shrink-0"
        size="icon"
        variant="ghost"
        aria-label="新增终端会话"
        :title="
          activeKind === 'docker' || activeKind === 'pc'
            ? '新增容器终端会话'
            : canAddSSH
              ? '新增 SSH 终端会话'
              : 'QEMU 串口只有一个；如需多个独立终端，请将节点接入可达网络并配置 SSH 凭据'
        "
        :disabled="!['docker', 'pc'].includes(String(activeKind)) && !canAddSSH"
        @click="addForActiveNode"
      >
        <Plus :size="14" />
      </Button>
      <Button
        v-if="activeKind !== 'docker' && activeKind !== 'pc'"
        class="shrink-0"
        size="icon"
        variant="ghost"
        aria-label="新增串口终端"
        :disabled="!canAddSerial || !canUseSerial"
        :title="
          canAddSerial && canUseSerial
            ? '打开 QEMU 串口救援终端'
            : 'QEMU 仅提供一个串口终端；请切换到已有的串口标签页'
        "
        @click="addSerialForActiveNode"
      >
        <TerminalSquare :size="14" />
      </Button>
      <Button
        v-if="activeKind !== 'docker' && activeKind !== 'pc'"
        class="shrink-0"
        size="icon"
        variant="ghost"
        aria-label="新增 VNC 会话"
        :disabled="!canAddVNC"
        title="为当前节点新增 VNC 会话"
        @click="addVNCForActiveNode"
      >
        <Monitor :size="14" />
      </Button>
    </nav>
    <div
      v-if="workspaces.length"
      class="min-h-0 flex-1 overflow-hidden"
      data-layout-region="console-content"
    >
      <template v-for="workspace in workspaces" :key="workspace.nodeId">
        <template v-for="session in workspace.sessions" :key="session.id">
          <ConsoleWorkspace
            v-if="shouldMountSession(workspace, session)"
            v-show="isActiveSession(workspace, session)"
            :node-id="workspace.nodeId"
            :resource-type="
              isNetworkObject(workspace.nodeId) ? 'network_object' : 'node'
            "
            :session-id="session.id"
            auto-open
            :auto-mode="session.mode"
          />
        </template>
      </template>
    </div>
    <div
      v-else
      class="grid flex-1 place-items-center text-xs text-muted-foreground"
    >
      尚未打开终端会话。请右键节点并选择“终端”。
    </div>
  </div>
</template>
