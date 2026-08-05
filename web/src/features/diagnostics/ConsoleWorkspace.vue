<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import {
  Maximize2,
  Monitor,
  PlugZap,
  RefreshCw,
  TerminalSquare,
  X,
} from "lucide-vue-next";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import RFB from "@novnc/novnc";
import "@xterm/xterm/css/xterm.css";
import { api, type ConsoleDescriptor } from "@/api";
import { Button } from "@/components/ui";
import { randomUUID } from "@/lib/uuid";

type ConsoleMode = ConsoleDescriptor["mode"];

const props = withDefaults(
  defineProps<{
    nodeId: string;
    resourceType?: "node" | "network_object";
    sessionId?: string;
    autoOpen?: boolean;
    autoMode?: ConsoleMode;
  }>(),
  {
    autoOpen: false,
    autoMode: "telnet",
    resourceType: "node",
  },
);
const descriptors = ref<ConsoleDescriptor[]>([]);
const active = ref<ConsoleMode>();
const activeSessionId = ref("");
const sessions = ref<
  Array<{ id: string; mode: ConsoleMode; label: string; state: string }>
>([]);
const state = ref("idle");
const error = ref("");
const terminalHost = ref<HTMLElement>();
const vncHost = ref<HTMLElement>();
let terminal: Terminal | undefined;
let fit: FitAddon | undefined;
let socket: WebSocket | undefined;
let rfb: RFB | undefined;
let observer: ResizeObserver | undefined;
let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
let intentionalClose = false;
let reconnects = 0;

async function load() {
  try {
    descriptors.value =
      props.resourceType === "network_object"
        ? await api.listNetworkObjectConsoles(props.nodeId)
        : await api.listNodeConsoles(props.nodeId);
    if (props.autoOpen && !sessions.value.length && descriptors.value[0]) {
      const descriptor = descriptors.value.find(
        (item) => item.mode === props.autoMode,
      );
      if (!descriptor) {
        state.value = "failed";
        error.value = `${props.autoMode.toUpperCase()} console is not available for this node`;
        return;
      }
      await createSession(descriptor.mode);
    }
  } catch (value) {
    error.value = value instanceof Error ? value.message : String(value);
  }
}
function modeUnavailableReason(mode: ConsoleMode) {
  if (descriptors.value.length === 0) return undefined;
  return descriptors.value.some((item) => item.mode === mode)
    ? undefined
    : `${mode.toUpperCase()} console is not supported by this node`;
}
function closeRenderer() {
  intentionalClose = true;
  clearTimeout(reconnectTimer);
  observer?.disconnect();
  observer = undefined;
  socket?.close();
  socket = undefined;
  rfb?.disconnect();
  rfb = undefined;
  terminal?.dispose();
  terminal = undefined;
  fit = undefined;
  terminalHost.value?.replaceChildren();
  vncHost.value?.replaceChildren();
  state.value = "closed";
}
function updateSessionState(value: string) {
  state.value = value;
  const session = sessions.value.find(
    (item) => item.id === activeSessionId.value,
  );
  if (session) session.state = value;
}
async function createSession(mode: ConsoleMode) {
  const session = {
    id: randomUUID(),
    mode,
    label: `${mode.toUpperCase()} ${sessions.value.filter((item) => item.mode === mode).length + 1}`,
    state: "connecting",
  };
  sessions.value.push(session);
  activeSessionId.value = session.id;
  await open(mode);
}
async function switchSession(id: string) {
  const session = sessions.value.find((item) => item.id === id);
  if (!session || id === activeSessionId.value) return;
  activeSessionId.value = id;
  await open(session.mode);
}
function closeSession(id: string) {
  const index = sessions.value.findIndex((item) => item.id === id);
  if (index < 0) return;
  const wasActive = activeSessionId.value === id;
  sessions.value.splice(index, 1);
  if (wasActive) {
    closeRenderer();
    const next = sessions.value[Math.max(0, index - 1)];
    activeSessionId.value = next?.id || "";
    active.value = undefined;
    if (next) void open(next.mode);
  }
}
async function open(mode: ConsoleMode) {
  closeRenderer();
  intentionalClose = false;
  active.value = mode;
  updateSessionState("connecting");
  error.value = "";
  await nextTick();
  const stream =
    descriptors.value.find((item) => item.mode === mode)?.stream_url ||
    (props.resourceType === "network_object"
      ? api.streamNetworkObjectConsole(props.nodeId, mode)
      : api.streamNodeConsole(props.nodeId, mode));
  const url = new URL(stream, window.location.origin);
  url.protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const stableSessionId = props.sessionId || activeSessionId.value;
  if (stableSessionId) url.searchParams.set("session_id", stableSessionId);
  try {
    if (mode !== "vnc") {
      terminal = new Terminal({
        cursorBlink: true,
        convertEol: true,
        theme: {
          background: "#050a0f",
          foreground: "#d9e5ef",
          cursor: "#5eead4",
        },
        fontSize: 13,
      });
      fit = new FitAddon();
      terminal.loadAddon(fit);
      terminal.open(terminalHost.value!);
      fit.fit();
      socket = new WebSocket(url);
      socket.binaryType = "arraybuffer";
      socket.onopen = () => {
        updateSessionState("connected");
        reconnects = 0;
        fit?.fit();
      };
      socket.onmessage = (event) =>
        terminal?.write(
          typeof event.data === "string"
            ? event.data
            : new Uint8Array(event.data),
        );
      socket.onclose = () => {
        if (!intentionalClose) {
          updateSessionState("reconnecting");
          reconnects += 1;
          reconnectTimer = setTimeout(
            () => open(mode),
            Math.min(1000 * reconnects, 5000),
          );
        }
      };
      terminal.onData(
        (data) =>
          socket?.readyState === WebSocket.OPEN &&
          socket.send(new TextEncoder().encode(data)),
      );
      observer = new ResizeObserver(() => fit?.fit());
      observer.observe(terminalHost.value!);
    } else {
      rfb = new RFB(vncHost.value!, url.toString(), { shared: true });
      rfb.scaleViewport = true;
      rfb.resizeSession = true;
      rfb.addEventListener("connect", () => updateSessionState("connected"));
      rfb.addEventListener("disconnect", () => {
        if (!intentionalClose) updateSessionState("reconnecting");
      });
      observer = new ResizeObserver(() => {
        if (rfb) rfb.scaleViewport = true;
      });
      observer.observe(vncHost.value!);
    }
  } catch (value) {
    updateSessionState("failed");
    error.value = value instanceof Error ? value.message : String(value);
  }
}
watch(
  () => props.nodeId,
  async () => {
    closeRenderer();
    sessions.value = [];
    activeSessionId.value = "";
    await load();
  },
);
onMounted(load);
onBeforeUnmount(closeRenderer);
</script>
<template>
  <div class="flex h-full min-h-[180px] flex-col">
    <header class="flex items-center gap-2 border-b border-border p-2">
      <template v-if="!autoOpen">
        <Button
          v-for="mode in ['ssh', 'telnet', 'vnc'] as const"
          :key="mode"
          size="sm"
          variant="secondary"
          :disabled="
            descriptors.length > 0 &&
            !descriptors.some((item) => item.mode === mode)
          "
          :title="modeUnavailableReason(mode)"
          @click="createSession(mode)"
        >
          <TerminalSquare v-if="mode !== 'vnc'" :size="14" /><Monitor
            v-else
            :size="14"
          />
          Open
          {{ mode === "ssh" ? "SSH" : mode === "telnet" ? "Serial" : "VNC" }}
        </Button>
      </template>
      <Button
        variant="ghost"
        size="sm"
        :disabled="!active"
        :title="
          active
            ? 'Reconnect the active console session'
            : 'Open a console session before reconnecting'
        "
        @click="active && open(active)"
      >
        <RefreshCw :size="13" /> Reconnect </Button
      ><span
        role="status"
        class="ml-auto inline-flex items-center gap-1 text-xs text-muted-foreground"
        ><PlugZap :size="13" />{{ state }}</span
      >
    </header>
    <nav
      v-if="sessions.length && !autoOpen"
      class="flex gap-1 overflow-x-auto border-b border-border bg-muted/20 p-1"
      aria-label="Console sessions"
    >
      <div
        v-for="session in sessions"
        :key="session.id"
        class="flex min-w-40 items-center gap-1"
      >
        <Button
          size="sm"
          :variant="activeSessionId === session.id ? 'default' : 'ghost'"
          class="min-w-32 flex-1 justify-start"
          :aria-pressed="activeSessionId === session.id"
          @click="switchSession(session.id)"
        >
          {{ session.label }} · {{ session.state }}
        </Button>
        <Button
          size="icon"
          variant="ghost"
          :aria-label="`Close ${session.label}`"
          :title="`Close ${session.label}`"
          @click="closeSession(session.id)"
        >
          <X :size="12" />
        </Button>
      </div>
    </nav>
    <p
      v-if="error"
      role="alert"
      class="border-b border-destructive/30 bg-destructive/10 p-2 text-xs text-red-300"
    >
      {{ error }}
    </p>
    <div class="relative min-h-0 flex-1 bg-[#050a0f]">
      <div
        v-show="active !== 'vnc'"
        ref="terminalHost"
        class="absolute inset-0 p-1"
      />
      <div
        v-show="active === 'vnc'"
        ref="vncHost"
        class="absolute inset-0 overflow-hidden"
      />
      <div
        v-if="!active"
        class="grid h-full place-items-center text-xs text-muted-foreground"
      >
        <span
          ><Maximize2 :size="15" class="mr-1 inline" />Open a supported console.
          Sessions are browser-local and restore with automatic reconnect after
          refresh.</span
        >
      </div>
    </div>
  </div>
</template>
