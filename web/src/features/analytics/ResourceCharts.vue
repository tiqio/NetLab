<script setup lang="ts">
import { computed } from "vue";
import EChart from "@/components/charts/EChart.vue";
import type { Node, OperationTask } from "@/api";
const props = defineProps<{ node?: Node; tasks?: OperationTask[] }>();
const resourceMetrics = computed(() => [
  {
    label: "vCPU",
    value: props.node?.cpu_count || 0,
    color: "#2dd4bf",
  },
  {
    label: "CPU quota cores",
    value: (props.node?.cpu_quota_micros || 0) / 100000,
    color: "#38bdf8",
  },
  {
    label: "Memory GiB",
    value: (props.node?.memory_mib || 0) / 1024,
    color: "#a78bfa",
  },
]);
const hasTaskData = computed(() => Boolean(props.tasks?.length));
const resourceOption = computed(() => ({
  tooltip: { trigger: "axis" },
  grid: { left: 34, right: 12, top: 8, bottom: 25 },
  xAxis: {
    type: "category",
    data: [props.node?.name || "No node"],
    axisLabel: { color: "#91a4b5" },
  },
  yAxis: {
    type: "value",
    axisLabel: { color: "#91a4b5" },
    splitLine: { lineStyle: { color: "#203445" } },
  },
  series: [
    {
      name: "vCPU",
      type: "bar",
      data: [props.node?.cpu_count || 0],
      itemStyle: { color: "#2dd4bf" },
    },
    {
      name: "CPU quota cores",
      type: "bar",
      data: [(props.node?.cpu_quota_micros || 0) / 100000],
      itemStyle: { color: "#38bdf8" },
    },
    {
      name: "Memory GiB",
      type: "bar",
      data: [(props.node?.memory_mib || 0) / 1024],
      itemStyle: { color: "#a78bfa" },
    },
  ],
}));
const taskOption = computed(() => ({
  tooltip: { trigger: "item" },
  legend: { bottom: 0, textStyle: { color: "#91a4b5", fontSize: 10 } },
  series: [
    {
      type: "pie",
      radius: ["45%", "68%"],
      label: { color: "#d9e5ef", fontSize: 10 },
      data: Object.entries(
        (props.tasks || []).reduce<Record<string, number>>(
          (result, task) => ({
            ...result,
            [task.state]: (result[task.state] || 0) + 1,
          }),
          {},
        ),
      ).map(([name, value]) => ({ name, value })),
    },
  ],
}));
</script>
<template>
  <div
    class="grid h-full min-h-[220px] gap-2 p-2"
    :class="hasTaskData ? 'grid-cols-2' : 'grid-cols-1'"
  >
    <section class="flex min-h-0 min-w-0 flex-col" aria-label="Node resources">
      <dl class="mb-1 grid shrink-0 grid-cols-3 gap-1">
        <div
          v-for="metric in resourceMetrics"
          :key="metric.label"
          class="min-w-0 rounded border border-border/70 bg-muted/30 px-1.5 py-1 text-center"
        >
          <dt
            class="truncate text-[9px] leading-3 text-muted-foreground"
            :title="metric.label"
          >
            <span
              class="mr-1 inline-block h-1.5 w-1.5 rounded-full"
              :style="{ backgroundColor: metric.color }"
            />{{ metric.label }}
          </dt>
          <dd class="text-xs font-semibold leading-4 text-foreground">
            {{ Number(metric.value.toFixed(2)) }}
          </dd>
        </div>
      </dl>
      <EChart
        class="min-h-0 flex-1"
        :option="resourceOption"
        aria-label="Selected node resource allocation chart"
      />
    </section>
    <EChart
      v-if="hasTaskData"
      :option="taskOption"
      aria-label="Task state distribution chart"
    />
  </div>
</template>
