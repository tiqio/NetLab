<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { Monitor, Plus, TerminalSquare, X } from "lucide-vue-next";
import { api, type ConsoleDescriptor, type Node } from "@/api";
import { Button } from "@/components/ui";
import { randomUUID } from "@/lib/uuid";
import ConsoleWorkspace from "./ConsoleWorkspace.vue";

const props = defineProps<{
  laboratoryId?: string;
  nodes: Node[];
  requestNodeId?: string;
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
const STORAGE_PREFIX = "netlab.console-workspaces.v1:";
let restoring = false;
const activeWorkspace = computed(() =>
  workspaces.value.find((workspace) => workspace.nodeId === activeNodeId.value),
);
const activeNode = computed(() =>
  props.nodes.find((node) => node.id === activeWorkspace.value?.nodeId),
);
const canAddSerial = computed(
  () =>
    activeNode.value?.kind === "docker" ||
    !activeWorkspace.value?.sessions.some(
      (session) => session.mode === "telnet",
    ),
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
  if (props.nodes.find((node) => node.id === nodeId)?.kind === "docker")
    return sessions;
  let foundTelnet = false;
  return sessions.filter((session) => {
    if (session.mode !== "telnet") return true;
    if (foundTelnet) return false;
    foundTelnet = true;
    return true;
  });
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
      : "ssh");
  if (
    resolvedMode === "telnet" &&
    props.nodes.find((node) => node.id === workspace.nodeId)?.kind !== "docker"
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
function openNode(nodeId: string) {
  let workspace = workspaces.value.find((item) => item.nodeId === nodeId);
  if (!workspace) {
    workspace = { nodeId, sessions: [], activeSessionId: "" };
    workspaces.value.push(workspace);
    addSession(workspace);
  } else if (
    props.nodes.find((node) => node.id === nodeId)?.kind !== "docker" &&
    !workspace.sessions.some((session) => session.mode === "ssh")
  ) {
    addSession(workspace, "ssh");
  }
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
    openNode(nodeId);
  },
  { immediate: true },
);
watch([workspaces, activeNodeId], persist, { deep: true, flush: "sync" });
watch(
  () => props.nodes.map((node) => node.id),
  (nodeIds) => {
    const available = new Set(nodeIds);
    workspaces.value = workspaces.value.filter((workspace) =>
      available.has(workspace.nodeId),
    );
    if (!available.has(activeNodeId.value))
      activeNodeId.value = workspaces.value[0]?.nodeId || "";
  },
  { immediate: true },
);

function label(nodeId: string) {
  return props.nodes.find((node) => node.id === nodeId)?.name || nodeId;
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
    void api.closeNodeConsoleSession(nodeId, session.mode, session.id);
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
  if (activeWorkspace.value) addSession(activeWorkspace.value);
}
function addVNCForActiveNode() {
  if (activeWorkspace.value) addSession(activeWorkspace.value, "vnc");
}
function addSerialForActiveNode() {
  if (activeWorkspace.value) addSession(activeWorkspace.value, "telnet");
}
</script>

<template>
  <div class="flex h-full min-h-[180px] flex-col" data-global-console-workspace>
    <nav
      v-if="workspaces.length"
      class="flex gap-1 overflow-x-auto border-b border-border bg-muted/20 p-1"
      aria-label="Node console workspaces"
    >
      <div
        v-for="workspace in workspaces"
        :key="workspace.nodeId"
        class="flex items-center"
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
          :aria-label="`Close ${label(workspace.nodeId)} console workspace`"
          @click="closeNode(workspace.nodeId)"
        >
          <X :size="12" />
        </Button>
      </div>
    </nav>
    <nav
      v-if="activeWorkspace"
      class="flex gap-1 overflow-x-auto border-b border-border bg-muted/10 p-1"
      aria-label="Console sessions"
    >
      <div
        v-for="session in activeWorkspace.sessions"
        :key="session.id"
        class="flex items-center"
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
          :aria-label="`Close ${modeLabel(session.mode)} ${session.sequence}`"
          @click="closeSession(activeWorkspace, session.id)"
        >
          <X :size="12" />
        </Button>
      </div>
      <Button
        class="ml-auto shrink-0"
        size="icon"
        variant="ghost"
        aria-label="Add terminal session"
        :title="
          activeNode?.kind === 'docker'
            ? 'Add another container terminal session'
            : 'Add another SSH terminal session'
        "
        @click="addForActiveNode"
      >
        <Plus :size="14" />
      </Button>
      <Button
        v-if="activeNode?.kind !== 'docker'"
        class="shrink-0"
        size="icon"
        variant="ghost"
        aria-label="Add serial console"
        :disabled="!canAddSerial"
        :title="
          canAddSerial
            ? 'Open the QEMU serial rescue console'
            : 'QEMU exposes one serial console; switch to the existing SERIAL tab'
        "
        @click="addSerialForActiveNode"
      >
        <TerminalSquare :size="14" />
      </Button>
      <Button
        v-if="activeNode?.kind !== 'docker'"
        class="shrink-0"
        size="icon"
        variant="ghost"
        aria-label="Add VNC session"
        title="Add a VNC session for the active node"
        @click="addVNCForActiveNode"
      >
        <Monitor :size="14" />
      </Button>
    </nav>
    <div v-if="workspaces.length" class="min-h-0 flex-1">
      <template v-for="workspace in workspaces" :key="workspace.nodeId">
        <ConsoleWorkspace
          v-for="session in workspace.sessions"
          v-show="
            activeNodeId === workspace.nodeId &&
            workspace.activeSessionId === session.id
          "
          :key="session.id"
          :node-id="workspace.nodeId"
          :session-id="session.id"
          auto-open
          :auto-mode="session.mode"
        />
      </template>
    </div>
    <div
      v-else
      class="grid flex-1 place-items-center text-xs text-muted-foreground"
    >
      Right-click a node and choose Terminal.
    </div>
  </div>
</template>
