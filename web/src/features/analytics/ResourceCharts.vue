<script setup lang="ts">
import { computed } from "vue";
import EChart from "@/components/charts/EChart.vue";
import type { Node, OperationTask } from "@/api";
const props = defineProps<{ node?: Node; tasks?: OperationTask[] }>();
const resourceMetrics = computed(() => [
  {
    label: "vCPU",
    value: props.node?.cpu_count || 0,
    color: "var(--chart-series-primary)",
  },
  {
    label: "CPU 配额（核心）",
    value: (props.node?.cpu_quota_micros || 0) / 100000,
    color: "var(--chart-series-secondary)",
  },
  {
    label: "内存（GiB）",
    value: (props.node?.memory_mib || 0) / 1024,
    color: "var(--chart-series-tertiary)",
  },
]);
const hasTaskData = computed(() => Boolean(props.tasks?.length));
const resourceOption = computed(() => ({
  tooltip: { trigger: "axis" },
  grid: { left: 34, right: 12, top: 8, bottom: 25 },
  xAxis: {
    type: "category",
    data: [props.node?.name || "No node"],
    axisLabel: { color: "var(--muted-foreground)" },
  },
  yAxis: {
    type: "value",
    axisLabel: { color: "var(--muted-foreground)" },
    splitLine: { lineStyle: { color: "var(--chart-grid)" } },
  },
  series: [
    {
      name: "vCPU",
      type: "bar",
      data: [props.node?.cpu_count || 0],
      itemStyle: { color: "var(--chart-series-primary)" },
    },
    {
      name: "CPU 配额（核心）",
      type: "bar",
      data: [(props.node?.cpu_quota_micros || 0) / 100000],
      itemStyle: { color: "var(--chart-series-secondary)" },
    },
    {
      name: "内存（GiB）",
      type: "bar",
      data: [(props.node?.memory_mib || 0) / 1024],
      itemStyle: { color: "var(--chart-series-tertiary)" },
    },
  ],
}));
const taskOption = computed(() => ({
  tooltip: { trigger: "item" },
  legend: {
    bottom: 0,
    textStyle: { color: "var(--muted-foreground)", fontSize: 10 },
  },
  series: [
    {
      type: "pie",
      radius: ["45%", "68%"],
      label: { color: "var(--chart-label)", fontSize: 10 },
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
    <section class="flex min-h-0 min-w-0 flex-col" aria-label="节点资源">
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
        aria-label="所选节点资源分配图"
      />
    </section>
    <EChart
      v-if="hasTaskData"
      :option="taskOption"
      aria-label="任务状态分布图"
    />
  </div>
</template>
