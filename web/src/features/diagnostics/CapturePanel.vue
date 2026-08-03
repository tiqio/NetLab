<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import {
  Clipboard,
  Download,
  ExternalLink,
  Radio,
  Square,
} from "lucide-vue-next";
import { api, type CaptureSession } from "@/api";
import {
  safeArtifactUrl,
  safeCaptureStreamUrl,
  wiresharkCommand,
} from "@/api/diagnostics";
import { Button, Dialog, FormField, Input, Select } from "@/components/ui";
import StatusBadge from "@/components/common/StatusBadge.vue";
import CaptureVolumeChart from "@/components/charts/CaptureVolumeChart.vue";
const props = defineProps<{
  laboratoryId?: string;
  interfaceId?: string;
  linkId?: string;
  objectLinkId?: string;
  sourceLabel?: string;
}>();
const emit = defineEmits<{
  captureChange: [CaptureSession | undefined];
}>();
const filter = ref("");
const format = ref<"pcap" | "pcapng">("pcap");
const maxBytes = ref(67108864);
const capture = ref<CaptureSession>();
const taskId = ref("");
const status = ref("");
const busy = ref(false);
const helperDialogOpen = ref(false);
const helperIssue = ref<"missing" | "origin" | "wireshark" | "launch">(
  "missing",
);
const helperMessage = ref("");
const helperBaseUrl = "http://127.0.0.1:38765";
let captureRefreshTimer: ReturnType<typeof setTimeout> | undefined;
const captureStreamable = computed(() =>
  ["starting", "streaming", "running", "requested"].includes(
    capture.value?.state || "",
  ),
);
const helperCommand = computed(
  () => `netlab-wireshark-helper -allow-origin ${window.location.origin}`,
);
class HelperResponseError extends Error {
  constructor(
    readonly code: string,
    message: string,
  ) {
    super(message);
  }
}
async function start() {
  if (!props.interfaceId && !props.linkId && !props.objectLinkId) return;
  busy.value = true;
  try {
    const value = await api.startCapture({
      laboratory_id: props.laboratoryId,
      source_type: props.objectLinkId
        ? "network_object_link"
        : props.linkId
          ? "link"
          : "interface",
      source_id: props.objectLinkId || props.linkId || props.interfaceId || "",
      filter: filter.value || undefined,
      format: format.value,
      retain: true,
      max_bytes: maxBytes.value,
    });
    capture.value = value.capture;
    taskId.value = value.task.id;
    status.value = `Capture queued: ${value.task.id}`;
    scheduleCaptureRefresh();
  } catch (error) {
    status.value = error instanceof Error ? error.message : String(error);
  } finally {
    busy.value = false;
  }
}
async function discover() {
  if (!props.laboratoryId) return;
  try {
    const values = await api.listCaptures(props.laboratoryId);
    capture.value = values
      .filter(
        (item) =>
          item.source_id ===
          (props.objectLinkId || props.linkId || props.interfaceId),
      )
      .sort((left, right) =>
        right.created_at.localeCompare(left.created_at),
      )[0];
    scheduleCaptureRefresh();
  } catch (error) {
    status.value = error instanceof Error ? error.message : String(error);
  }
}
watch(
  () => [
    props.laboratoryId,
    props.interfaceId,
    props.linkId,
    props.objectLinkId,
  ],
  discover,
);
watch(capture, (value) => emit("captureChange", value), { immediate: true });
onMounted(discover);
onBeforeUnmount(() => clearTimeout(captureRefreshTimer));
async function refresh(showStatus = true) {
  if (!capture.value) return;
  try {
    capture.value = await api.getCapture(capture.value.id);
    if (showStatus) status.value = "Capture status refreshed";
    scheduleCaptureRefresh();
  } catch (error) {
    status.value = error instanceof Error ? error.message : String(error);
  }
}
function scheduleCaptureRefresh() {
  clearTimeout(captureRefreshTimer);
  if (
    !capture.value ||
    !["requested", "starting", "streaming", "running", "stopping"].includes(
      capture.value.state,
    )
  )
    return;
  captureRefreshTimer = setTimeout(() => void refresh(false), 1000);
}
async function stop() {
  if (!capture.value) return;
  try {
    const value = await api.stopCapture(capture.value.id);
    taskId.value = value.task.id;
    status.value = `Stop queued: ${value.task.id}`;
    await refresh(false);
  } catch (error) {
    status.value = error instanceof Error ? error.message : String(error);
  }
}
async function copyCommand() {
  if (!capture.value) return;
  try {
    const platform = /Windows/i.test(navigator.userAgent) ? "windows" : "unix";
    await writeClipboard(
      wiresharkCommand(capture.value.id, window.location.origin, platform),
    );
    status.value = "Wireshark command copied";
  } catch (error) {
    status.value = `Unable to copy Wireshark command: ${error instanceof Error ? error.message : String(error)}`;
  }
}

async function openWireshark() {
  if (!capture.value || !captureStreamable.value) return;
  busy.value = true;
  status.value = "Checking the local Wireshark helper…";
  try {
    const health = await helperRequest<{
      allowed_origin: string;
      wireshark_available: boolean;
    }>("/health", { method: "GET" });
    if (health.allowed_origin !== window.location.origin) {
      showHelperIssue(
        "origin",
        `The helper trusts ${health.allowed_origin || "no NetLab origin"}, not ${window.location.origin}.`,
      );
      return;
    }
    if (!health.wireshark_available) {
      showHelperIssue(
        "wireshark",
        "The local helper is running, but it cannot find Wireshark.",
      );
      return;
    }
    const streamUrl = new URL(
      safeCaptureStreamUrl(capture.value.id),
      window.location.origin,
    ).toString();
    await helperRequest("/launch", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ stream_url: streamUrl }),
    });
    status.value = "Wireshark opened with the live capture stream.";
  } catch (error) {
    if (error instanceof HelperResponseError) {
      showHelperIssue(
        error.code === "wireshark_not_found" ? "wireshark" : "launch",
        error.message,
      );
    } else {
      const message =
        error instanceof DOMException && error.name === "AbortError"
          ? "The local NetLab Wireshark helper did not respond. Wireshark may be installed, but the helper is not running."
          : error instanceof TypeError
            ? "The browser could not connect to the local NetLab Wireshark helper. Wireshark installation alone is not enough; start the downloaded helper first."
            : error instanceof Error
              ? error.message
              : "The local Wireshark helper could not be reached.";
      showHelperIssue("missing", message);
    }
  } finally {
    busy.value = false;
  }
}

async function helperRequest<T = Record<string, unknown>>(
  path: string,
  init: Parameters<typeof fetch>[1],
) {
  const controller = new AbortController();
  const timer = window.setTimeout(() => controller.abort(), 3500);
  try {
    const response = await fetch(`${helperBaseUrl}${path}`, {
      ...init,
      mode: "cors",
      signal: controller.signal,
    });
    const body = await response.json().catch(() => ({}));
    if (!response.ok) {
      const problem = body as { code?: string; message?: string };
      throw new HelperResponseError(
        problem.code || "helper_error",
        problem.message || `Helper returned HTTP ${response.status}.`,
      );
    }
    return body as T;
  } finally {
    window.clearTimeout(timer);
  }
}

function showHelperIssue(
  issue: "missing" | "origin" | "wireshark" | "launch",
  message: string,
) {
  helperIssue.value = issue;
  helperMessage.value = message;
  helperDialogOpen.value = true;
  status.value =
    "Local Wireshark helper unavailable. See the installation dialog.";
}

async function writeClipboard(value: string) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      return;
    } catch (error) {
      void error;
    }
  }
  const textarea = document.createElement("textarea");
  textarea.value = value;
  textarea.setAttribute("readonly", "");
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  textarea.style.pointerEvents = "none";
  document.body.appendChild(textarea);
  textarea.select();
  textarea.setSelectionRange(0, value.length);
  const copied = document.execCommand?.("copy") === true;
  textarea.remove();
  if (!copied) throw new Error("clipboard unavailable");
}
</script>
<template>
  <div class="grid gap-3 p-3">
    <div v-if="sourceLabel" class="flex items-center justify-between gap-3">
      <div>
        <strong class="text-sm">{{ sourceLabel }}</strong>
        <p class="text-[11px] text-muted-foreground">
          This source has independent capture controls and remains active while
          you switch tabs.
        </p>
      </div>
      <StatusBadge v-if="capture" :state="capture.state" />
    </div>
    <div class="grid grid-cols-3 gap-2">
      <FormField label="Capture filter">
        <Input v-model="filter" placeholder="tcp port 443" /> </FormField
      ><FormField label="Format">
        <Select v-model="format">
          <option value="pcap">pcap</option>
          <option value="pcapng">pcapng</option>
        </Select> </FormField
      ><FormField label="Maximum bytes">
        <Input v-model="maxBytes" type="number" min="1048576" />
      </FormField>
    </div>
    <div class="flex flex-wrap gap-2">
      <Button
        size="sm"
        :disabled="(!interfaceId && !linkId && !objectLinkId) || busy"
        :title="
          !interfaceId && !linkId && !objectLinkId
            ? 'Select a node interface or link before starting capture'
            : busy
              ? 'Capture request is in progress'
              : 'Start packet capture'
        "
        @click="start"
      >
        <Radio :size="14" /> Start capture </Button
      ><Button
        size="sm"
        variant="secondary"
        :disabled="!capture"
        :title="
          capture
            ? 'Refresh capture status'
            : 'Start or discover a capture before refreshing'
        "
        @click="refresh"
      >
        Refresh </Button
      ><Button
        size="sm"
        variant="destructive"
        :disabled="!capture"
        :title="
          capture
            ? 'Stop this capture'
            : 'Start or discover a capture before stopping'
        "
        @click="stop"
      >
        <Square :size="14" /> Stop </Button
      ><Button
        size="sm"
        variant="outline"
        :disabled="!captureStreamable || busy"
        :title="
          captureStreamable
            ? 'Open this live stream in local Wireshark'
            : 'Start an active capture before opening Wireshark'
        "
        @click="openWireshark"
      >
        <ExternalLink :size="14" /> Open Wireshark </Button
      ><a
        v-if="capture"
        :href="safeCaptureStreamUrl(capture.id)"
        class="inline-flex items-center gap-1 rounded border border-border px-2 text-xs"
        ><Download :size="13" /> Stream</a
      ><a
        v-if="capture?.artifact_id"
        :href="safeArtifactUrl(capture.artifact_id)"
        class="inline-flex items-center gap-1 rounded border border-border px-2 text-xs"
        ><Download :size="13" /> Retained file</a
      >
    </div>
    <p
      v-if="!interfaceId && !linkId && !objectLinkId"
      class="text-xs text-amber-300"
    >
      Select a node interface or link before starting capture.
    </p>
    <article
      v-if="capture"
      class="rounded border border-border bg-background/40 p-3"
    >
      <div class="flex items-center gap-2">
        <StatusBadge :state="capture.state" /><strong class="text-xs">{{
          capture.id
        }}</strong
        ><span class="ml-auto font-mono text-[10px] text-muted-foreground"
          >task {{ taskId }}</span
        >
      </div>
      <dl class="mt-2 grid grid-cols-[auto_1fr] gap-x-2 gap-y-1 text-xs">
        <dt>Packets</dt>
        <dd>{{ capture.packets }}</dd>
        <dt>Bytes</dt>
        <dd>{{ capture.bytes_written }} / {{ capture.max_bytes }}</dd>
        <dt>Retention</dt>
        <dd>
          {{
            capture.retain
              ? `retained${capture.expires_at ? ` until ${capture.expires_at}` : ""}`
              : "stream only"
          }}
        </dd>
        <dt>Completion</dt>
        <dd>{{ capture.completion_reason || "active" }}</dd>
        <dt>Truncated</dt>
        <dd>{{ capture.truncated ? "Yes — quota reached" : "No" }}</dd>
      </dl>
      <CaptureVolumeChart
        class="mt-2"
        :bytes="capture.bytes_written"
        :maximum="capture.max_bytes"
        :packets="capture.packets"
        :truncated="capture.truncated"
      />
    </article>
    <p role="status" class="text-xs text-muted-foreground">
      {{ status }}
    </p>
    <Dialog
      v-model="helperDialogOpen"
      title="Wireshark integration required"
      :description="helperMessage"
    >
      <div class="grid gap-3 text-sm">
        <p v-if="helperIssue === 'wireshark'" class="text-amber-300">
          Install Wireshark, then restart the NetLab helper.
        </p>
        <p v-else-if="helperIssue === 'origin'" class="text-amber-300">
          Restart the helper with this NetLab address explicitly allowed.
        </p>
        <p v-else class="text-muted-foreground">
          Wireshark installation alone is not enough. Download and keep the
          NetLab helper running on the same computer as this browser. The
          server-specific Windows helper can be started by double-clicking it.
        </p>
        <p class="text-xs text-muted-foreground">
          Diagnostic check: open the local helper health address. If it does not
          show JSON, the helper is not running or was blocked by the operating
          system.
        </p>
        <a
          :href="`${helperBaseUrl}/health`"
          target="_blank"
          rel="noreferrer"
          class="text-primary underline"
          >Test local helper</a
        >
        <code
          class="overflow-x-auto rounded border border-border bg-background p-2 text-xs"
          >{{ helperCommand }}</code
        >
        <div class="flex flex-wrap gap-2">
          <a
            class="rounded border border-border px-2 py-1 text-xs hover:bg-accent"
            href="/api/v1/client-tools/wireshark-helper/windows-amd64"
            >Windows helper</a
          >
          <a
            class="rounded border border-border px-2 py-1 text-xs hover:bg-accent"
            href="/api/v1/client-tools/wireshark-helper/linux-amd64"
            >Linux helper</a
          >
          <a
            class="rounded border border-border px-2 py-1 text-xs hover:bg-accent"
            href="/api/v1/client-tools/wireshark-helper/darwin-amd64"
            >macOS Intel</a
          >
          <a
            class="rounded border border-border px-2 py-1 text-xs hover:bg-accent"
            href="/api/v1/client-tools/wireshark-helper/darwin-arm64"
            >macOS Apple Silicon</a
          >
        </div>
        <a
          href="https://www.wireshark.org/download.html"
          target="_blank"
          rel="noreferrer"
          class="text-primary underline"
          >Install Wireshark</a
        >
      </div>
      <template #footer>
        <Button variant="secondary" @click="copyCommand">
          <Clipboard :size="14" /> Copy manual command
        </Button>
        <Button
          variant="secondary"
          @click="
            helperDialogOpen = false;
            openWireshark();
          "
          >Retry</Button
        >
        <Button @click="helperDialogOpen = false">Close</Button>
      </template>
    </Dialog>
  </div>
</template>
