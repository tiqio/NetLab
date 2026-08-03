<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { Cable, CheckCircle2, TerminalSquare } from "lucide-vue-next";
import {
  api,
  ApiError,
  type Node,
  type NodeInterface,
  type RuijieConfigRequest,
} from "@/api";
import { Button, FormField, Input, Select } from "@/components/ui";
import StatusBadge from "@/components/common/StatusBadge.vue";

const props = defineProps<{ node: Node; interfaces: NodeInterface[] }>();
const emit = defineEmits<{ terminal: []; changed: [] }>();
const templateKey = computed(() =>
  String(props.node.config?.template_key || ""),
);
const isSwitch = computed(() => templateKey.value === "ruijie-switch");
const dataInterfaces = computed(() =>
  props.interfaces.filter((item) => !item.name.startsWith("internal")),
);
const internalInterfaces = computed(() =>
  props.interfaces.filter((item) => item.name.startsWith("internal")),
);
const operation = ref<RuijieConfigRequest["operation"]>("l2_access");
const selectedInterface = ref("");
const vlanId = ref(10);
const vlanName = ref("");
const allowedVlans = ref("10,20");
const addressCidr = ref("192.0.2.1/24");
const adminState = ref<"up" | "down">("up");
const save = ref(true);
const busy = ref(false);
const error = ref("");
const success = ref("");
const commands = ref<string[]>([]);

const operations = computed(() =>
  isSwitch.value
    ? [
        { value: "l2_access", label: "配置 Access VLAN" },
        { value: "l2_trunk", label: "配置 Trunk" },
        { value: "create_vlan", label: "创建或修改 VLAN" },
        { value: "admin_state", label: "启用或禁用接口" },
      ]
    : [
        { value: "l3_address", label: "配置三层 IPv4 地址" },
        { value: "admin_state", label: "启用或禁用接口" },
      ],
);
const requiresInterface = computed(() => operation.value !== "create_vlan");
const canApply = computed(
  () =>
    props.node.observed_state === "running" &&
    (!requiresInterface.value || selectedInterface.value) &&
    !busy.value,
);

watch(
  () => [props.node.id, props.interfaces],
  () => {
    if (
      !dataInterfaces.value.some(
        (item) => item.name === selectedInterface.value,
      )
    )
      selectedInterface.value = dataInterfaces.value[0]?.name || "";
  },
  { immediate: true, deep: true },
);
watch(isSwitch, (value) => {
  operation.value = value ? "l2_access" : "l3_address";
});

async function applyConfiguration() {
  if (!canApply.value) return;
  busy.value = true;
  error.value = "";
  success.value = "";
  try {
    const result = await api.configureRuijie(props.node.id, {
      operation: operation.value,
      interface: selectedInterface.value,
      vlan_id: Number(vlanId.value),
      vlan_name: vlanName.value.trim(),
      allowed_vlans: allowedVlans.value.trim(),
      address_cidr: addressCidr.value.trim(),
      admin_up: adminState.value === "up",
      save: save.value,
    });
    commands.value = result.commands;
    success.value = result.verified
      ? "配置已执行，并通过交换机 CLI 提示符确认。"
      : "配置命令已发送到交换机 Console。";
    emit("changed");
  } catch (cause) {
    error.value =
      cause instanceof ApiError ? cause.problem.message : String(cause);
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <section class="border-b border-border p-3" data-testid="ruijie-config-panel">
    <div class="flex items-center justify-between gap-2">
      <div>
        <h3 class="text-xs font-semibold">交换机接口与配置</h3>
        <p class="text-[11px] text-muted-foreground">
          {{ isSwitch ? "Ruijie 二层交换机" : "Ruijie 三层交换机" }} ·
          配置通过同一个串口会话下发
        </p>
      </div>
      <Button size="sm" variant="secondary" @click="$emit('terminal')">
        <TerminalSquare :size="13" /> 打开终端
      </Button>
    </div>

    <div class="mt-3 grid gap-1.5">
      <div
        v-for="item in dataInterfaces"
        :key="item.id"
        class="grid grid-cols-[1fr_auto] gap-2 rounded border border-border bg-background/40 px-2 py-1.5 text-[11px]"
      >
        <div class="min-w-0">
          <div class="flex items-center gap-1.5">
            <Cable :size="12" class="text-primary" />
            <strong>{{ item.name }}</strong>
            <span class="text-muted-foreground">{{ item.driver }}</span>
          </div>
          <p class="truncate text-muted-foreground" :title="item.mac_address">
            {{ item.mac_address }} ·
            {{ item.desired_link_id ? "已接线" : "未接线" }}
          </p>
        </div>
        <StatusBadge :state="item.operational_state" />
      </div>
    </div>

    <p
      v-if="internalInterfaces.length"
      class="mt-2 text-[10px] text-muted-foreground"
    >
      {{ internalInterfaces.length }} 个内部 TIPC
      控制口已隐藏，不能用于拓扑接线。
    </p>

    <div
      class="mt-3 grid gap-2 rounded-md border border-border bg-muted/10 p-2.5"
    >
      <FormField label="常用操作">
        <Select v-model="operation" data-testid="ruijie-operation">
          <option
            v-for="item in operations"
            :key="item.value"
            :value="item.value"
          >
            {{ item.label }}
          </option>
        </Select>
      </FormField>
      <FormField v-if="requiresInterface" label="目标接口">
        <Select v-model="selectedInterface" data-testid="ruijie-interface">
          <option
            v-for="item in dataInterfaces"
            :key="item.id"
            :value="item.name"
          >
            {{ item.name }} · {{ item.mac_address }}
          </option>
        </Select>
      </FormField>
      <FormField
        v-if="['l2_access', 'create_vlan'].includes(operation)"
        label="VLAN ID"
        hint="1–4094"
      >
        <Input v-model="vlanId" type="number" min="1" max="4094" />
      </FormField>
      <FormField v-if="operation === 'create_vlan'" label="VLAN 名称">
        <Input v-model="vlanName" placeholder="OFFICE" maxlength="32" />
      </FormField>
      <FormField
        v-if="operation === 'l2_trunk'"
        label="允许的 VLAN"
        hint="例如 10,20-30"
      >
        <Input v-model="allowedVlans" placeholder="10,20-30" />
      </FormField>
      <FormField v-if="operation === 'l3_address'" label="IPv4 地址/CIDR">
        <Input v-model="addressCidr" placeholder="192.0.2.1/24" />
      </FormField>
      <FormField v-if="operation !== 'create_vlan'" label="接口状态">
        <Select v-model="adminState">
          <option value="up">启用（no shutdown）</option>
          <option value="down">禁用（shutdown）</option>
        </Select>
      </FormField>
      <label class="flex items-center gap-2 text-xs">
        <input v-model="save" type="checkbox" /> 保存到设备配置（write）
      </label>
      <Button size="sm" :disabled="!canApply" @click="applyConfiguration">
        {{ busy ? "正在下发…" : "应用配置" }}
      </Button>
      <p
        v-if="node.observed_state !== 'running'"
        class="text-xs text-amber-300"
      >
        请先启动交换机，再应用 CLI 配置。
      </p>
      <p v-if="error" role="alert" class="text-xs text-destructive">
        {{ error }}
      </p>
      <p
        v-if="success"
        role="status"
        class="flex items-center gap-1 text-xs text-emerald-300"
      >
        <CheckCircle2 :size="13" /> {{ success }}
      </p>
      <details v-if="commands.length" class="text-[11px] text-muted-foreground">
        <summary class="cursor-pointer">查看已下发命令</summary>
        <pre class="mt-1 overflow-auto rounded bg-background p-2">{{
          commands.join("\n")
        }}</pre>
      </details>
    </div>
  </section>
</template>
