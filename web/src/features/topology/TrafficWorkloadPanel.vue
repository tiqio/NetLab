<script setup lang="ts">
import { computed, onMounted, reactive, watch } from "vue";
import { Button, Input, Select } from "@/components/ui";
import {
  api,
  type Node,
  type NetworkObject,
  type TrafficObservation,
  type TrafficWorkload,
} from "@/api";
import { useTrafficWorkloadsStore } from "@/stores/trafficWorkloads";

const props = defineProps<{
  laboratoryId?: string;
  nodes?: Node[];
  networkObjects?: NetworkObject[];
}>();
const emit = defineEmits<{
  overlay: [TrafficObservation[], boolean, string];
}>();
const store = useTrafficWorkloadsStore();
const form = reactive({
  name: "持续探测",
  source: "",
  protocol: "icmp",
  family: "ipv4",
  destination: "",
  interval: 5,
  timeout: 2,
});
const sources = computed(() => [
  ...(props.nodes || []).map((value) => ({
    id: value.id,
    kind: "node",
    label: value.name,
  })),
  ...(props.networkObjects || [])
    .filter((value) => ["pc", "switch_l2", "switch_l3"].includes(value.kind))
    .map((value) => ({
      id: value.id,
      kind: "network_object",
      label: value.name,
    })),
]);
const degraded = (value: TrafficWorkload) =>
  value.desired_state === "running" && value.observed_state !== "running";
async function refresh() {
  if (props.laboratoryId) await store.load(props.laboratoryId);
}
async function create() {
  if (!props.laboratoryId || !form.source || !form.destination) return;
  const source = sources.value.find((value) => value.id === form.source)!;
  await store.create({
    laboratory_id: props.laboratoryId,
    name: form.name,
    source: { kind: source.kind, resource_id: source.id },
    protocol: form.protocol as "icmp" | "http" | "dns",
    address_family: form.family as "auto" | "ipv4" | "ipv6",
    destination:
      form.protocol === "icmp"
        ? { address: form.destination }
        : form.protocol === "http"
          ? { url: form.destination }
          : { name: form.destination },
    interval_seconds: Number(form.interval),
    timeout_seconds: Number(form.timeout),
  });
}
async function highlight(value: TrafficWorkload) {
  if (!props.laboratoryId) return;
  const filters = await api.listTrafficFilters(props.laboratoryId);
  const successTime = value.last_success_at
    ? Date.parse(value.last_success_at)
    : undefined;
  const correlationWindowMs = Math.max(value.timeout_seconds * 1000, 2000);
  const matched = filters.filter(
    (item) =>
      item.traffic_filter.expression.toLowerCase().includes(value.protocol) &&
      item.traffic_filter.last_match_at &&
      (successTime === undefined ||
        Date.parse(item.traffic_filter.last_match_at) >=
          successTime - correlationWindowMs),
  );
  emit(
    "overlay",
    matched.flatMap((item) => item.traffic_filter.observations),
    matched.length > 0,
    matched[0]?.traffic_filter.color || "#f59e0b",
  );
}
watch(() => props.laboratoryId, refresh, { immediate: true });
onMounted(refresh);
</script>
<template>
  <section class="space-y-3 p-3" aria-label="稳定流量任务">
    <div class="grid gap-2 md:grid-cols-6">
      <Input v-model="form.name" aria-label="任务名称" />
      <Select v-model="form.source" aria-label="流量源"
        ><option value="">选择流量源</option>
        <option v-for="source in sources" :key="source.id" :value="source.id">
          {{ source.label }}
        </option></Select
      >
      <Select v-model="form.protocol" aria-label="协议"
        ><option value="icmp">ICMP</option>
        <option value="http">HTTP</option>
        <option value="dns">DNS</option></Select
      >
      <Select v-model="form.family" aria-label="地址族"
        ><option value="ipv4">IPv4</option>
        <option value="ipv6">IPv6</option>
        <option value="auto">自动</option></Select
      >
      <Input
        v-model="form.destination"
        aria-label="目标"
        :placeholder="form.protocol === 'http' ? 'http://…' : '地址或域名'"
      />
      <Button :disabled="!form.source || !form.destination" @click="create"
        >创建任务</Button
      >
    </div>
    <p v-if="store.error" role="alert" class="text-sm text-destructive">
      {{ store.error }}
    </p>
    <div
      v-for="value in store.values"
      :key="value.id"
      class="rounded-lg border border-border bg-card p-3 text-sm"
    >
      <div class="flex flex-wrap items-center justify-between gap-2">
        <div>
          <strong>{{ value.name }}</strong
          ><span class="ml-2 text-xs text-muted-foreground"
            >{{ value.protocol.toUpperCase() }} ·
            {{ value.address_family }}</span
          >
        </div>
        <span
          :class="
            degraded(value) ? 'text-destructive' : 'text-muted-foreground'
          "
          >{{ degraded(value) ? "运行降级" : value.observed_state }}</span
        >
      </div>
      <dl class="mt-2 grid grid-cols-2 gap-1 text-xs md:grid-cols-5">
        <div>尝试 {{ value.attempts }}</div>
        <div>成功 {{ value.successes }}</div>
        <div>失败 {{ value.failures }}</div>
        <div>匹配字节 {{ value.matched_bytes }}</div>
        <div>最近成功 {{ value.last_success_at || "—" }}</div>
      </dl>
      <p
        v-if="value.last_error"
        role="alert"
        class="mt-2 text-xs text-destructive"
      >
        {{ value.last_error.message }}
      </p>
      <div class="mt-2 flex gap-2">
        <Button
          size="sm"
          v-if="value.desired_state !== 'running'"
          @click="store.start(value)"
          >启动</Button
        ><Button size="sm" variant="secondary" v-else @click="store.stop(value)"
          >停止</Button
        ><Button size="sm" variant="ghost" @click="highlight(value)"
          >高亮匹配路径</Button
        ><Button size="sm" variant="destructive" @click="store.remove(value)"
          >删除</Button
        >
      </div>
    </div>
  </section>
</template>
