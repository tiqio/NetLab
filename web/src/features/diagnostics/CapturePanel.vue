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
    status.value = `抓包任务已进入队列：${value.task.id}`;
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
    if (showStatus) status.value = "抓包状态已刷新";
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
    status.value = `停止任务已进入队列：${value.task.id}`;
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
    status.value = "Wireshark 命令已复制";
  } catch (error) {
    status.value = `无法复制 Wireshark 命令：${error instanceof Error ? error.message : String(error)}`;
  }
}

async function openWireshark() {
  if (!capture.value || !captureStreamable.value) return;
  busy.value = true;
  status.value = "正在检查本机 Wireshark 辅助程序…";
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
    status.value = "已使用 Wireshark 打开实时抓包流。";
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
              : "无法连接本机 Wireshark 辅助程序。";
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
  status.value = "本机 Wireshark 辅助程序不可用，请查看安装提示。";
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
          此抓包源使用独立控制，切换标签页后仍会继续运行。
        </p>
      </div>
      <StatusBadge v-if="capture" :state="capture.state" />
    </div>
    <div class="grid grid-cols-3 gap-2">
      <FormField label="抓包过滤表达式">
        <Input v-model="filter" placeholder="tcp port 443" /> </FormField
      ><FormField label="格式">
        <Select v-model="format">
          <option value="pcap">pcap</option>
          <option value="pcapng">pcapng</option>
        </Select> </FormField
      ><FormField label="最大字节数">
        <Input v-model="maxBytes" type="number" min="1048576" />
      </FormField>
    </div>
    <div class="flex flex-wrap gap-2">
      <Button
        size="sm"
        :disabled="(!interfaceId && !linkId && !objectLinkId) || busy"
        :title="
          !interfaceId && !linkId && !objectLinkId
            ? '开始抓包前请选择节点接口或链路'
            : busy
              ? '抓包请求正在处理中'
              : '开始抓取数据包'
        "
        @click="start"
      >
        <Radio :size="14" /> 开始抓包 </Button
      ><Button
        size="sm"
        variant="secondary"
        :disabled="!capture"
        :title="capture ? '刷新抓包状态' : '请先开始或发现一个抓包会话'"
        @click="refresh"
      >
        刷新 </Button
      ><Button
        size="sm"
        variant="destructive"
        :disabled="!capture"
        :title="capture ? '停止此抓包会话' : '请先开始或发现一个抓包会话'"
        @click="stop"
      >
        <Square :size="14" /> 停止 </Button
      ><Button
        size="sm"
        variant="outline"
        :disabled="!captureStreamable || busy"
        :title="
          captureStreamable
            ? '在本机 Wireshark 中打开实时流'
            : '请先启动抓包，再打开 Wireshark'
        "
        @click="openWireshark"
      >
        <ExternalLink :size="14" /> 使用 Wireshark 打开 </Button
      ><a
        v-if="capture"
        :href="safeCaptureStreamUrl(capture.id)"
        class="inline-flex items-center gap-1 rounded border border-border px-2 text-xs"
        ><Download :size="13" /> 实时流</a
      ><a
        v-if="capture?.artifact_id"
        :href="safeArtifactUrl(capture.artifact_id)"
        class="inline-flex items-center gap-1 rounded border border-border px-2 text-xs"
        ><Download :size="13" /> 保留文件</a
      >
    </div>
    <p
      v-if="!interfaceId && !linkId && !objectLinkId"
      class="text-xs text-amber-300"
    >
      开始抓包前请选择节点接口或链路。
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
        <dt>数据包</dt>
        <dd>{{ capture.packets }}</dd>
        <dt>字节数</dt>
        <dd>{{ capture.bytes_written }} / {{ capture.max_bytes }}</dd>
        <dt>保留方式</dt>
        <dd>
          {{
            capture.retain
              ? `已保留${capture.expires_at ? `，有效期至 ${capture.expires_at}` : ""}`
              : "仅实时流"
          }}
        </dd>
        <dt>完成状态</dt>
        <dd>{{ capture.completion_reason || "活动中" }}</dd>
        <dt>是否截断</dt>
        <dd>{{ capture.truncated ? "是——已达到配额" : "否" }}</dd>
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
      title="需要配置 Wireshark 集成"
      :description="helperMessage"
    >
      <div class="grid gap-3 text-sm">
        <p v-if="helperIssue === 'wireshark'" class="text-amber-300">
          请安装 Wireshark，然后重启 NetLab 辅助程序。
        </p>
        <p v-else-if="helperIssue === 'origin'" class="text-amber-300">
          请重启辅助程序，并明确允许当前 NetLab 地址。
        </p>
        <p v-else class="text-muted-foreground">
          仅安装 Wireshark 还不够。请下载 NetLab
          辅助程序，并在当前浏览器所在计算机上持续运行。Windows
          辅助程序可双击启动。
        </p>
        <p class="text-xs text-muted-foreground">
          诊断方法：打开本地辅助程序健康检查地址。如果没有显示
          JSON，说明辅助程序未运行或被操作系统拦截。
        </p>
        <a
          :href="`${helperBaseUrl}/health`"
          target="_blank"
          rel="noreferrer"
          class="text-primary underline"
          >测试本地辅助程序</a
        >
        <code
          class="overflow-x-auto rounded border border-border bg-background p-2 text-xs"
          >{{ helperCommand }}</code
        >
        <div class="flex flex-wrap gap-2">
          <a
            class="rounded border border-border px-2 py-1 text-xs hover:bg-accent"
            href="/api/v1/client-tools/wireshark-helper/windows-amd64"
            >Windows 辅助程序</a
          >
          <a
            class="rounded border border-border px-2 py-1 text-xs hover:bg-accent"
            href="/api/v1/client-tools/wireshark-helper/linux-amd64"
            >Linux 辅助程序</a
          >
          <a
            class="rounded border border-border px-2 py-1 text-xs hover:bg-accent"
            href="/api/v1/client-tools/wireshark-helper/darwin-amd64"
            >macOS Intel 辅助程序</a
          >
          <a
            class="rounded border border-border px-2 py-1 text-xs hover:bg-accent"
            href="/api/v1/client-tools/wireshark-helper/darwin-arm64"
            >macOS Apple Silicon 辅助程序</a
          >
        </div>
        <a
          href="https://www.wireshark.org/download.html"
          target="_blank"
          rel="noreferrer"
          class="text-primary underline"
          >安装 Wireshark</a
        >
      </div>
      <template #footer>
        <Button variant="secondary" @click="copyCommand">
          <Clipboard :size="14" /> 复制手动命令
        </Button>
        <Button
          variant="secondary"
          @click="
            helperDialogOpen = false;
            openWireshark();
          "
          >重试</Button
        >
        <Button @click="helperDialogOpen = false">关闭</Button>
      </template>
    </Dialog>
  </div>
</template>
