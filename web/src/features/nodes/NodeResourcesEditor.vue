<script setup lang="ts">
import { ref, watch } from "vue";
import { api, type Node } from "@/api";
import { Button, FormField, Input } from "@/components/ui";
const props = defineProps<{ node: Node }>();
const emit = defineEmits<{ changed: [] }>();
const cpu = ref(props.node.cpu_count);
const quota = ref(props.node.cpu_quota_micros);
const memory = ref(props.node.memory_mib);
const status = ref("");
const busy = ref(false);
watch(
  () => props.node,
  (value) => {
    cpu.value = value.cpu_count;
    quota.value = value.cpu_quota_micros;
    memory.value = value.memory_mib;
  },
);
async function save() {
  busy.value = true;
  try {
    await api.updateNodeResources(props.node, {
      cpu_count: cpu.value,
      cpu_quota_micros: quota.value,
      memory_mib: memory.value,
    });
    status.value = `${cpu.value} 个 vCPU，CPU 时间限制为宿主机单核周期的 ${Math.round(quota.value / 1000)}%`;
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
      <FormField label="vCPUs"
        ><Input v-model="cpu" type="number" min="1" /></FormField
      ><FormField label="CPU 配额（微秒）"
        ><Input v-model="quota" type="number" min="1000" /></FormField
      ><FormField label="内存（MiB）"
        ><Input v-model="memory" type="number" min="64"
      /></FormField>
    </div>
    <Button class="mt-2" size="sm" :disabled="busy">应用限制</Button>
    <p role="status" class="mt-1 text-xs text-muted-foreground">{{ status }}</p>
  </form>
</template>
