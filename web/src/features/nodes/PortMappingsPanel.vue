<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import {
  ChevronDown,
  Copy,
  ExternalLink,
  LoaderCircle,
  Plus,
  Radar,
  RefreshCw,
  Trash2,
} from "lucide-vue-next";
import {
  api,
  ApiError,
  type OperationTask,
  type PortMapping,
  type Problem,
} from "@/api";
import { Button, FormField, Input, Select } from "@/components/ui";
import StatusBadge from "@/components/common/StatusBadge.vue";
import StructuredProblem from "@/components/common/StructuredProblem.vue";

const props = defineProps<{ nodeId: string }>();
const protocol = ref<"tcp" | "udp">("tcp");
const hostAddress = ref(defaultHostAddress());
const hostPort = ref(0);
const guestAddress = ref("");
const guestPort = ref(22);
const preset = ref("ssh");
const exposure = ref<"management" | "local" | "all">("management");
const advanced = ref(false);
const mappings = ref<PortMapping[]>([]);
const status = ref("");
const problem = ref<Problem>();
const busy = ref(false);
const detecting = ref(false);
const createForm = ref<HTMLElement>();

const presets = [
  { value: "ssh", label: "SSH", protocol: "tcp", guest: 22 },
  { value: "http", label: "HTTP", protocol: "tcp", guest: 80 },
  { value: "https", label: "HTTPS", protocol: "tcp", guest: 443 },
  { value: "fortigate", label: "FortiGate HTTPS", protocol: "tcp", guest: 443 },
  { value: "ztp-http", label: "ZTP · HTTP", protocol: "tcp", guest: 80 },
  { value: "ztp-https", label: "ZTP · HTTPS", protocol: "tcp", guest: 443 },
  { value: "custom", label: "自定义 TCP/UDP", protocol: "tcp", guest: 10000 },
] as const;

const canCreate = computed(
  () =>
    !busy.value &&
    Boolean(hostAddress.value.trim()) &&
    validOptionalPort(hostPort.value) &&
    validPort(guestPort.value),
);

function defaultHostAddress() {
  if (typeof window === "undefined") return "0.0.0.0";
  const hostname = window.location.hostname;
  return hostname === "localhost" ? "127.0.0.1" : hostname || "0.0.0.0";
}

function validPort(value: number) {
  return (
    Number.isInteger(Number(value)) &&
    Number(value) >= 1 &&
    Number(value) <= 65535
  );
}

function validOptionalPort(value: number) {
  return Number(value) === 0 || validPort(value);
}

function errorProblem(error: unknown): Problem {
  return error instanceof ApiError
    ? error.problem
    : {
        code: "port_mapping_failed",
        message: error instanceof Error ? error.message : String(error),
      };
}

async function waitForTask(task: OperationTask) {
  let current = task;
  for (let attempt = 0; attempt < 120; attempt += 1) {
    if (["succeeded", "failed", "cancelled"].includes(current.state))
      return current;
    await new Promise((resolve) => setTimeout(resolve, 250));
    current = await api.getTask(current.id);
  }
  throw new Error(`任务 ${task.id} 在 30 秒内未完成`);
}

async function load(showStatus = false) {
  problem.value = undefined;
  try {
    const values = await api.listNodePortMappings(props.nodeId);
    mappings.value = Array.isArray(values) ? values : [];
    if (showStatus) status.value = `已刷新 ${mappings.value.length} 条映射`;
  } catch (error) {
    problem.value = errorProblem(error);
  }
}

function applyPreset(value: string | number | undefined) {
  const selected = presets.find((item) => item.value === String(value));
  if (!selected) return;
  preset.value = selected.value;
  protocol.value = selected.protocol;
  hostPort.value = 0;
  guestPort.value = selected.guest;
  if (selected.value === "custom") advanced.value = true;
}

function applyExposure(value: string | number | undefined) {
  exposure.value = String(value) as typeof exposure.value;
  if (exposure.value === "local") hostAddress.value = "127.0.0.1";
  else if (exposure.value === "all") hostAddress.value = "0.0.0.0";
  else hostAddress.value = defaultHostAddress();
}

async function create() {
  if (!canCreate.value) return;
  busy.value = true;
  problem.value = undefined;
  status.value = "正在创建端口映射…";
  try {
    const value = await api.createPortMapping(props.nodeId, {
      protocol: protocol.value,
      host_address: hostAddress.value.trim(),
      host_port: Number(hostPort.value),
      guest_address: guestAddress.value.trim(),
      guest_port: Number(guestPort.value),
    });
    mappings.value = [value.port_mapping, ...mappings.value];
    status.value = `正在发布端口 · 任务 ${value.task.id}`;
    const task = await waitForTask(value.task);
    if (task.state !== "succeeded")
      throw task.error || new Error(`任务状态：${task.state}`);
    await load();
    status.value = `端口映射已生效 · ${accessText(value.port_mapping)}`;
  } catch (error) {
    problem.value = errorProblem(error);
    await load();
  } finally {
    busy.value = false;
  }
}

async function remove(mapping: PortMapping) {
  if (busy.value) return;
  busy.value = true;
  problem.value = undefined;
  try {
    const task = await api.deletePortMapping(mapping.id);
    status.value = `正在移除 ${mapping.host_port} · 任务 ${task.id}`;
    const result = await waitForTask(task);
    if (result.state !== "succeeded")
      throw result.error || new Error(`任务状态：${result.state}`);
    await load();
    status.value = "端口映射已移除";
    await revealCreateForm();
  } catch (error) {
    problem.value = errorProblem(error);
  } finally {
    busy.value = false;
  }
}

async function revealCreateForm() {
  await nextTick();
  createForm.value?.scrollIntoView?.({ behavior: "smooth", block: "nearest" });
}

async function detectGuestAddress() {
  if (detecting.value) return;
  detecting.value = true;
  problem.value = undefined;
  try {
    const queued = await api.executeGuestCommand(props.nodeId, {
      argv: [
        "sh",
        "-lc",
        "ip -o -4 addr show scope global | awk '{split($4,address,\"/\"); print address[1]}'",
      ],
      timeout_seconds: 10,
      output_limit: 4096,
    });
    status.value = `正在通过 QGA 探测 IPv4 · 任务 ${queued.id}`;
    const task = await waitForTask(queued);
    if (task.state !== "succeeded")
      throw task.error || new Error(`任务状态：${task.state}`);
    const encoded = String(task.result?.stdout_base64 || "");
    const addresses = encoded
      ? atob(encoded)
          .split(/\s+/)
          .map((value) => value.trim())
          .filter(Boolean)
      : [];
    if (!addresses.length)
      throw new Error(
        "QGA 未发现全局 IPv4 地址，请先配置 NAT、DHCP 或静态地址",
      );
    guestAddress.value = addresses[0];
    status.value = `已选择 Guest IPv4：${addresses[0]}`;
  } catch (error) {
    problem.value = errorProblem(error);
  } finally {
    detecting.value = false;
  }
}

function accessHost(mapping: PortMapping) {
  if (!["0.0.0.0", "::"].includes(mapping.host_address))
    return mapping.host_address;
  return typeof window === "undefined"
    ? mapping.host_address
    : window.location.hostname;
}

function accessText(mapping: PortMapping) {
  const host = accessHost(mapping);
  const formattedHost = host.includes(":") ? `[${host}]` : host;
  if (mapping.protocol === "tcp" && mapping.guest_port === 22)
    return `ssh -p ${mapping.host_port} ubuntu@${host}`;
  if (mapping.protocol === "tcp" && [80, 443].includes(mapping.guest_port))
    return `${mapping.guest_port === 443 ? "https" : "http"}://${formattedHost}:${mapping.host_port}`;
  return `${mapping.protocol}://${formattedHost}:${mapping.host_port}`;
}

function serviceLabel(mapping: PortMapping) {
  if (mapping.protocol === "tcp" && mapping.guest_port === 22) return "SSH";
  if (mapping.protocol === "tcp" && mapping.guest_port === 80) return "HTTP";
  if (mapping.protocol === "tcp" && mapping.guest_port === 443) return "HTTPS";
  return `${mapping.protocol.toUpperCase()} ${mapping.guest_port}`;
}

function openAccess(mapping: PortMapping) {
  const value = accessText(mapping);
  if (value.startsWith("http://") || value.startsWith("https://"))
    window.open(value, "_blank", "noopener,noreferrer");
}

async function writeClipboard(value: string) {
  if (navigator.clipboard?.writeText) {
    const copied = await navigator.clipboard.writeText(value).then(
      () => true,
      () => false,
    );
    if (copied) return;
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

async function copyAccess(mapping: PortMapping) {
  problem.value = undefined;
  try {
    await writeClipboard(accessText(mapping));
    status.value = "访问地址已复制";
  } catch {
    status.value = `自动复制失败，请手动复制：${accessText(mapping)}`;
  }
}

watch(
  () => props.nodeId,
  () => void load(),
  { immediate: true },
);
</script>

<template>
  <section class="panel-section grid gap-3">
    <div class="flex items-center justify-between gap-2">
      <div>
        <h3>端口映射</h3>
        <p class="text-xs text-muted-foreground">
          选择服务后保存，系统自动识别节点地址、分配可用宿主机端口并立即生效。
        </p>
      </div>
      <div class="flex gap-1">
        <Button variant="secondary" size="sm" @click="revealCreateForm">
          <Plus :size="14" /> 新增映射
        </Button>
        <Button variant="ghost" size="sm" :disabled="busy" @click="load(true)">
          <RefreshCw :size="14" /> 刷新
        </Button>
      </div>
    </div>

    <div
      ref="createForm"
      class="grid gap-3 rounded-md border border-primary/30 bg-primary/5 p-3"
      aria-label="新增端口映射"
    >
      <div>
        <h4 class="text-sm font-semibold">新增端口映射</h4>
        <p class="mt-1 text-xs text-muted-foreground">
          无需停止节点。运行中的节点会立即生效；已停止节点将在启动并取得对应地址后可访问。
        </p>
      </div>
      <div class="grid grid-cols-2 gap-2">
        <FormField label="服务类型">
          <Select :model-value="preset" @update:model-value="applyPreset">
            <option
              v-for="item in presets"
              :key="item.value"
              :value="item.value"
            >
              {{ item.label }}
            </option>
          </Select>
        </FormField>
        <FormField label="访问范围">
          <Select :model-value="exposure" @update:model-value="applyExposure">
            <option value="management">当前管理地址</option>
            <option value="local">仅宿主机本地</option>
            <option value="all">所有宿主机 IPv4 地址</option>
          </Select>
        </FormField>
        <FormField label="节点端口">
          <Input v-model.number="guestPort" type="number" min="1" max="65535" />
        </FormField>
        <FormField label="宿主机端口" hint="填 0 自动从 20000–39999 分配">
          <Input v-model.number="hostPort" type="number" min="0" max="65535" />
        </FormField>
      </div>

      <Button
        variant="ghost"
        size="sm"
        class="w-fit"
        :aria-expanded="advanced"
        @click="advanced = !advanced"
      >
        <ChevronDown
          :size="14"
          :class="
            advanced
              ? 'rotate-180 transition-transform'
              : 'transition-transform'
          "
        />
        高级设置
      </Button>

      <div
        v-if="advanced"
        class="grid grid-cols-2 gap-2 rounded-md border border-border p-2"
      >
        <FormField label="协议">
          <Select v-model="protocol">
            <option value="tcp">TCP</option>
            <option value="udp">UDP</option>
          </Select>
        </FormField>
        <FormField label="宿主机监听地址">
          <Input v-model="hostAddress" placeholder="10.72.1.159" />
        </FormField>
        <FormField
          label="节点地址"
          hint="留空时自动从 NAT DHCP lease 或静态配置解析"
        >
          <div class="flex gap-1">
            <Input v-model="guestAddress" placeholder="自动识别" />
            <Button
              type="button"
              variant="secondary"
              size="sm"
              :disabled="detecting"
              title="通过 QEMU Guest Agent 自动探测 IPv4"
              @click="detectGuestAddress"
            >
              <LoaderCircle v-if="detecting" :size="14" class="animate-spin" />
              <Radar v-else :size="14" />
            </Button>
          </div>
        </FormField>
      </div>

      <div class="flex items-center gap-2">
        <Button size="sm" :disabled="!canCreate" @click="create">
          <Plus :size="14" /> 保存并生效
        </Button>
        <span class="text-xs text-muted-foreground">
          节点必须具有可达的 NAT DHCP 或静态地址。
        </span>
      </div>
    </div>

    <div v-if="mappings.length" class="grid gap-2">
      <article
        v-for="mapping in mappings"
        :key="mapping.id"
        class="grid gap-2 rounded-md border border-border bg-background/50 p-2 text-xs"
      >
        <div class="flex flex-wrap items-center gap-2">
          <StatusBadge :state="mapping.observed_state" />
          <strong>{{ serviceLabel(mapping) }}</strong>
          <span class="font-mono">
            {{ mapping.host_address }}:{{ mapping.host_port }} →
            {{ mapping.guest_address }}:{{ mapping.guest_port }}
          </span>
          <div class="ml-auto flex gap-1">
            <Button
              v-if="[80, 443].includes(mapping.guest_port)"
              variant="ghost"
              size="sm"
              title="在新窗口打开"
              @click="openAccess(mapping)"
            >
              <ExternalLink :size="13" /> Open
            </Button>
            <Button
              variant="ghost"
              size="sm"
              title="复制访问地址"
              @click="copyAccess(mapping)"
            >
              <Copy :size="13" /> Copy
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="text-red-300"
              :disabled="busy"
              @click="remove(mapping)"
            >
              <Trash2 :size="13" /> Remove
            </Button>
          </div>
        </div>
        <code class="select-all rounded bg-muted px-2 py-1 text-[11px]">{{
          accessText(mapping)
        }}</code>
        <StructuredProblem
          v-if="mapping.last_error"
          :problem="mapping.last_error"
        />
      </article>
    </div>
    <div
      v-else
      class="rounded-md border border-dashed border-border p-3 text-center"
    >
      <p class="text-xs text-muted-foreground">
        当前没有端口映射。新映射会属于该节点，并在刷新页面或切换客户端后保持共享状态。
      </p>
      <Button
        class="mt-2"
        size="sm"
        variant="secondary"
        @click="revealCreateForm"
      >
        <Plus :size="14" /> 添加端口映射
      </Button>
    </div>
    <p role="status" class="text-xs text-muted-foreground">{{ status }}</p>
    <StructuredProblem v-if="problem" :problem="problem" />
  </section>
</template>
