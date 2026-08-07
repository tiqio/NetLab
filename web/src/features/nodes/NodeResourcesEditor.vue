<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { api, type Node } from "@/api";
import { Button, FormField, Input } from "@/components/ui";
import { cpuQuotaCoresToMicros, cpuQuotaMicrosToCores } from "@/lib/cpuQuota";

const props = defineProps<{ node: Node }>();
const emit = defineEmits<{ changed: [] }>();
const cpu = ref<number | string>(props.node.cpu_count);
const quotaCores = ref<number | string>(
  cpuQuotaMicrosToCores(props.node.cpu_quota_micros),
);
const memory = ref<number | string>(props.node.memory_mib);
const status = ref("");
const busy = ref(false);
const error = computed(() => {
  const value = Number(quotaCores.value);
  if (!Number.isFinite(value) || value < 0) return "CPU 配额必须是非负数。";
  return "";
});

watch(
  () => props.node,
  (value) => {
    cpu.value = value.cpu_count;
    quotaCores.value = cpuQuotaMicrosToCores(value.cpu_quota_micros);
    memory.value = value.memory_mib;
  },
);

async function save() {
  if (error.value) return;
  busy.value = true;
  const cpuCount = Number(cpu.value);
  const quota = Number(quotaCores.value);
  const memoryMiB = Number(memory.value);
  try {
    await api.updateNodeResources(props.node, {
      cpu_count: cpuCount,
      cpu_quota_micros: cpuQuotaCoresToMicros(quota),
      memory_mib: memoryMiB,
    });
    status.value = `${cpuCount} 个 vCPU，CPU 配额${quota > 0 ? `为 ${quota} 核` : "不限制"}`;
    emit("changed");
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <form class="panel-section" @submit.prevent="save">
    <h3>资源限制</h3>
    <div class="grid grid-cols-3 gap-2">
      <FormField label="vCPU 数量">
        <Input v-model="cpu" type="number" min="1" step="1" />
      </FormField>
      <FormField
        label="CPU 配额（核心）"
        :error="error"
        hint="1 表示最多使用一个宿主机核心的 CPU 时间；0 表示不限制。"
      >
        <Input v-model="quotaCores" type="number" min="0" step="0.1" />
      </FormField>
      <FormField label="内存（MiB）">
        <Input v-model="memory" type="number" min="64" step="64" />
      </FormField>
    </div>
    <Button
      class="mt-2"
      size="sm"
      type="submit"
      :disabled="busy || Boolean(error)"
    >
      应用限制
    </Button>
    <p role="status" class="mt-1 text-xs text-muted-foreground">{{ status }}</p>
  </form>
</template>
