<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { CheckCircle2, Copy } from "lucide-vue-next";
import {
  api,
  ApiError,
  type Node,
  type NodeInterface,
  type Problem,
} from "@/api";
import StructuredProblem from "@/components/common/StructuredProblem.vue";
import { Button, FormField, Input, Select } from "@/components/ui";

type InterfaceForm = {
  id: string;
  name: string;
  driver: string;
  ipv4Mode: "disabled" | "dhcp" | "static";
  ipv4Address: string;
  ipv6Mode: "disabled" | "dhcp" | "slaac" | "static";
  ipv6Address: string;
};

const props = defineProps<{ node: Node; interfaces: NodeInterface[] }>();
const emit = defineEmits<{ changed: [] }>();
const name = ref("");
const interfaceForms = ref<InterfaceForm[]>([]);
const initialValue = ref("");
const busy = ref(false);
const status = ref("");
const problem = ref<Problem>();
const credentials = ref<{ username: string; password: string }>();
const credentialsLoading = ref(false);
const credentialsMessage = ref("");
const copied = ref<"username" | "password">();
let copiedTimer: ReturnType<typeof setTimeout> | undefined;

const stopped = computed(
  () =>
    props.node.desired_state === "stopped" &&
    props.node.observed_state === "stopped",
);
const ubuntuQemu = computed(() => {
  const templateKey = String(props.node.config?.template_key || "");
  return (
    props.node.kind === "qemu" && templateKey.toLowerCase().includes("ubuntu")
  );
});
const dirty = computed(
  () =>
    name.value.trim() !== props.node.name ||
    JSON.stringify(interfaceForms.value) !== initialValue.value,
);

function configuredNetwork() {
  const values = Array.isArray(props.node.config?.network_interfaces)
    ? (props.node.config.network_interfaces as Array<Record<string, unknown>>)
    : [];
  return Object.fromEntries(
    values.map((value) => [String(value.name || ""), value]),
  );
}

function reset() {
  const configured = configuredNetwork();
  name.value = props.node.name;
  interfaceForms.value = props.interfaces.map((item) => {
    const value = configured[item.name] || {};
    const modes = Array.isArray(value.modes) ? value.modes.map(String) : [];
    const addresses = Array.isArray(value.addresses)
      ? value.addresses.map(String)
      : [];
    return {
      id: item.id,
      name: item.name,
      driver: item.driver || "virtio-net-pci",
      ipv4Mode: modes.includes("dhcpv4")
        ? "dhcp"
        : addresses.some((address) => !address.includes(":"))
          ? "static"
          : "disabled",
      ipv4Address: addresses.find((address) => !address.includes(":")) || "",
      ipv6Mode: modes.includes("dhcpv6")
        ? "dhcp"
        : modes.includes("slaac")
          ? "slaac"
          : addresses.some((address) => address.includes(":"))
            ? "static"
            : "disabled",
      ipv6Address: addresses.find((address) => address.includes(":")) || "",
    } as InterfaceForm;
  });
  initialValue.value = JSON.stringify(interfaceForms.value);
  status.value = "";
  problem.value = undefined;
}

async function loadCredentials() {
  credentials.value = undefined;
  credentialsMessage.value = "";
  if (!ubuntuQemu.value) return;
  credentialsLoading.value = true;
  try {
    credentials.value = await api.getNodeBootstrapCredentials(props.node.id);
  } catch (error) {
    credentialsMessage.value =
      error instanceof ApiError && error.status === 404
        ? "此节点没有可恢复的初始登录信息。"
        : error instanceof Error
          ? error.message
          : String(error);
  } finally {
    credentialsLoading.value = false;
  }
}

async function writeClipboard(value: string) {
  if (navigator.clipboard?.writeText) {
    let copiedWithAPI = false;
    try {
      await navigator.clipboard.writeText(value);
      copiedWithAPI = true;
    } catch {
      copiedWithAPI = false;
    }
    if (copiedWithAPI) return;
  }
  const textarea = document.createElement("textarea");
  textarea.value = value;
  textarea.setAttribute("readonly", "");
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.select();
  const successful = document.execCommand?.("copy") === true;
  textarea.remove();
  if (!successful) throw new Error("clipboard unavailable");
}

async function copyCredential(field: "username" | "password", value: string) {
  clearTimeout(copiedTimer);
  try {
    await writeClipboard(value);
    copied.value = field;
    status.value = field === "username" ? "用户名已复制。" : "密码已复制。";
    copiedTimer = setTimeout(() => (copied.value = undefined), 2500);
  } catch {
    status.value = "复制失败，请选中文本后手动复制。";
  }
}

async function save() {
  if (!stopped.value || busy.value || !dirty.value) return;
  busy.value = true;
  problem.value = undefined;
  try {
    await api.updateNodeSettings(props.node, {
      name: name.value.trim(),
      cpu_count: props.node.cpu_count,
      cpu_quota_micros: props.node.cpu_quota_micros,
      memory_mib: props.node.memory_mib,
      interface_limit: props.node.interface_limit,
      process_limit: props.node.process_limit,
      network_interfaces:
        props.node.kind === "qemu"
          ? interfaceForms.value.map((item) => ({
              id: item.id,
              name: item.name,
              driver: item.driver,
              modes: [
                ...(item.ipv4Mode === "dhcp" ? ["dhcpv4"] : []),
                ...(item.ipv4Mode === "static" ? ["static"] : []),
                ...(item.ipv6Mode === "dhcp" ? ["dhcpv6"] : []),
                ...(item.ipv6Mode === "slaac" ? ["slaac"] : []),
                ...(item.ipv6Mode === "static" ? ["static"] : []),
              ],
              addresses: [
                ...(item.ipv4Mode === "static" && item.ipv4Address.trim()
                  ? [item.ipv4Address.trim()]
                  : []),
                ...(item.ipv6Mode === "static" && item.ipv6Address.trim()
                  ? [item.ipv6Address.trim()]
                  : []),
              ],
            }))
          : undefined,
    });
    status.value = "节点配置已保存，将在下次启动时生效。";
    initialValue.value = JSON.stringify(interfaceForms.value);
    emit("changed");
  } catch (error) {
    problem.value =
      error instanceof ApiError
        ? error.problem
        : {
            code: "node_settings_failed",
            message: error instanceof Error ? error.message : String(error),
          };
  } finally {
    busy.value = false;
  }
}

watch(
  () => [props.node, props.interfaces] as const,
  () => {
    reset();
    void loadCredentials();
  },
  { immediate: true, deep: true },
);
onBeforeUnmount(() => clearTimeout(copiedTimer));
</script>

<template>
  <section class="panel-section grid gap-4" aria-label="节点配置">
    <div>
      <h3>基本设置</h3>
      <p class="text-xs text-muted-foreground">
        名称与启动网络需要节点完全停止后修改。
      </p>
    </div>
    <div
      v-if="!stopped"
      role="status"
      class="rounded-md border border-amber-500/40 bg-amber-500/10 p-2 text-xs text-amber-200"
    >
      当前节点正在运行。资源限制仍可热修改；名称、网卡驱动和 IP 预配置已锁定。
    </div>
    <FormField label="节点名称">
      <Input
        v-model="name"
        aria-label="节点名称"
        maxlength="128"
        :disabled="!stopped || busy"
        required
      />
    </FormField>

    <div
      v-if="node.kind === 'qemu' && interfaceForms.length"
      class="grid gap-3"
    >
      <div>
        <h3>启动网络</h3>
        <p class="text-xs text-muted-foreground">
          用于 cloud-init 初始化；修改后在下次启动时应用。
        </p>
      </div>
      <div
        v-for="item in interfaceForms"
        :key="item.id"
        class="grid gap-3 rounded-md border border-border/70 bg-background/30 p-3"
      >
        <strong class="text-sm">{{ item.name }}</strong>
        <FormField label="网卡驱动">
          <Select
            v-model="item.driver"
            :aria-label="`${item.name} 网卡驱动`"
            :disabled="!stopped || busy"
          >
            <option value="virtio-net-pci">virtio-net-pci（推荐）</option>
            <option value="e1000">e1000</option>
            <option value="e1000e">e1000e</option>
            <option value="vmxnet3">vmxnet3</option>
            <option value="rtl8139">rtl8139</option>
          </Select>
        </FormField>
        <div class="grid grid-cols-2 gap-2">
          <FormField label="IPv4 配置">
            <Select
              v-model="item.ipv4Mode"
              :aria-label="`${item.name} IPv4 配置`"
              :disabled="!stopped || busy"
            >
              <option value="disabled">不配置</option>
              <option value="dhcp">DHCP 自动获取</option>
              <option value="static">静态地址</option>
            </Select>
          </FormField>
          <FormField v-if="item.ipv4Mode === 'static'" label="IPv4 地址/CIDR">
            <Input
              v-model="item.ipv4Address"
              :aria-label="`${item.name} IPv4 地址`"
              placeholder="192.0.2.10/24"
              :disabled="!stopped || busy"
              required
            />
          </FormField>
        </div>
        <div class="grid grid-cols-2 gap-2">
          <FormField label="IPv6 配置">
            <Select
              v-model="item.ipv6Mode"
              :aria-label="`${item.name} IPv6 配置`"
              :disabled="!stopped || busy"
            >
              <option value="disabled">不配置</option>
              <option value="dhcp">DHCPv6</option>
              <option value="slaac">SLAAC</option>
              <option value="static">静态地址</option>
            </Select>
          </FormField>
          <FormField v-if="item.ipv6Mode === 'static'" label="IPv6 地址/CIDR">
            <Input
              v-model="item.ipv6Address"
              :aria-label="`${item.name} IPv6 地址`"
              placeholder="2001:db8::10/64"
              :disabled="!stopped || busy"
              required
            />
          </FormField>
        </div>
      </div>
    </div>

    <div v-if="ubuntuQemu" class="grid gap-3">
      <div>
        <h3>登录信息</h3>
        <p class="text-xs text-muted-foreground">
          来自 cloud-init seed，只读显示。
        </p>
      </div>
      <p
        v-if="credentialsLoading"
        role="status"
        class="text-xs text-muted-foreground"
      >
        正在读取登录信息…
      </p>
      <p
        v-else-if="credentialsMessage"
        role="status"
        class="text-xs text-amber-300"
      >
        {{ credentialsMessage }}
      </p>
      <template v-else-if="credentials">
        <FormField label="用户名">
          <div class="flex gap-2">
            <Input
              :model-value="credentials.username"
              aria-label="初始用户名"
              readonly
            />
            <Button
              type="button"
              variant="secondary"
              :aria-label="
                copied === 'username' ? '用户名已复制' : '复制用户名'
              "
              @click="copyCredential('username', credentials.username)"
            >
              <CheckCircle2 v-if="copied === 'username'" :size="14" />
              <Copy v-else :size="14" />
              {{ copied === "username" ? "已复制" : "复制" }}
            </Button>
          </div>
        </FormField>
        <FormField label="密码">
          <div class="flex gap-2">
            <Input
              :model-value="credentials.password"
              aria-label="初始密码"
              readonly
            />
            <Button
              type="button"
              variant="secondary"
              :aria-label="copied === 'password' ? '密码已复制' : '复制密码'"
              @click="copyCredential('password', credentials.password)"
            >
              <CheckCircle2 v-if="copied === 'password'" :size="14" />
              <Copy v-else :size="14" />
              {{ copied === "password" ? "已复制" : "复制" }}
            </Button>
          </div>
        </FormField>
      </template>
    </div>

    <div class="flex items-center gap-2">
      <Button size="sm" :disabled="!stopped || busy || !dirty" @click="save">
        {{ busy ? "保存中…" : "保存配置" }}
      </Button>
      <Button
        v-if="dirty"
        size="sm"
        variant="ghost"
        :disabled="busy"
        @click="reset"
      >
        放弃修改
      </Button>
    </div>
    <p role="status" class="text-xs text-muted-foreground">{{ status }}</p>
    <StructuredProblem v-if="problem" :problem="problem" />
  </section>
</template>
