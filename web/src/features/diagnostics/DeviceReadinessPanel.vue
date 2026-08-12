<script setup lang="ts">
import { computed, ref, watch } from "vue";
import {
  api,
  type DeviceInterfaceRole,
  type DeviceReadiness,
  type Node,
  type NodeInterface,
} from "@/api";

const props = defineProps<{ node?: Node; interfaces?: NodeInterface[] }>();
const emit = defineEmits<{ updated: [Node] }>();
const readiness = ref<DeviceReadiness>();
const loading = ref(false);
const error = ref("");
const saving = ref(false);
const roles = ref<Record<string, DeviceInterfaceRole["role"] | "">>({});
const nodeInterfaces = computed(() =>
  (props.interfaces || []).filter((item) => item.node_id === props.node?.id),
);
const labels: Record<string, string> = {
  cable: "线缆",
  guest: "设备系统",
  management: "管理可达",
  data_path: "数据路径",
};
const readinessKeys = ["cable", "guest", "management", "data_path"] as const;

async function load() {
  readiness.value = undefined;
  error.value = "";
  if (!props.node?.id) return;
  loading.value = true;
  try {
    readiness.value = await api.getDeviceReadiness(props.node.id);
    roles.value = Object.fromEntries(
      nodeInterfaces.value.map((iface) => [
        iface.id,
        readiness.value?.roles.find((role) => role.interface_id === iface.id)
          ?.role || "",
      ]),
    );
  } catch (value) {
    error.value = value instanceof Error ? value.message : String(value);
  } finally {
    loading.value = false;
  }
}

async function saveRoles() {
  if (!props.node) return;
  saving.value = true;
  error.value = "";
  try {
    const updated = await api.updateNodeSettings(props.node, {
      name: props.node.name,
      cpu_count: props.node.cpu_count,
      cpu_quota_micros: props.node.cpu_quota_micros,
      memory_mib: props.node.memory_mib,
      interface_limit: props.node.interface_limit,
      process_limit: props.node.process_limit,
      device_roles: nodeInterfaces.value.flatMap((iface) =>
        roles.value[iface.id]
          ? [
              {
                interface_id: iface.id,
                role: roles.value[iface.id] as DeviceInterfaceRole["role"],
              },
            ]
          : [],
      ),
    });
    emit("updated", updated);
    await load();
  } catch (value) {
    error.value = value instanceof Error ? value.message : String(value);
  } finally {
    saving.value = false;
  }
}

watch(() => props.node?.id, load, { immediate: true });
</script>

<template>
  <section
    v-if="node"
    class="m-3 rounded-lg border border-border bg-card p-3 text-sm"
    data-testid="device-readiness"
  >
    <strong>设备就绪度</strong>
    <p v-if="loading" class="text-xs text-muted-foreground">正在读取…</p>
    <p v-else-if="error" role="alert" class="text-xs text-destructive">
      {{ error }}
    </p>
    <div v-else-if="readiness" class="mt-2 grid gap-2 sm:grid-cols-4">
      <div
        v-for="key in readinessKeys"
        :key="key"
        class="rounded border border-border p-2"
      >
        <div class="text-xs text-muted-foreground">{{ labels[key] }}</div>
        <div class="font-medium">{{ readiness[key].state }}</div>
      </div>
    </div>
    <div v-if="readiness" class="mt-3 space-y-2 border-t border-border pt-3">
      <div
        v-for="iface in nodeInterfaces"
        :key="iface.id"
        class="grid grid-cols-[1fr_10rem] items-center gap-2 text-xs"
      >
        <span>{{ iface.name }}</span>
        <select
          v-model="roles[iface.id]"
          class="rounded border border-border bg-background px-2 py-1"
          :aria-label="`${iface.name} 接口角色`"
        >
          <option value="">未声明</option>
          <option value="management">管理</option>
          <option value="lan">LAN</option>
          <option value="wan">WAN</option>
          <option value="trunk">Trunk</option>
          <option value="client-facing">客户端</option>
        </select>
      </div>
      <button
        class="rounded border border-border px-3 py-1"
        :disabled="saving || node.observed_state !== 'stopped'"
        @click="saveRoles"
      >
        保存角色
      </button>
      <p
        v-if="node.observed_state !== 'stopped'"
        class="text-xs text-muted-foreground"
      >
        停止节点后可修改角色元数据。
      </p>
    </div>
  </section>
</template>
