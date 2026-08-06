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
    data: [props.node?.name || "未选择节点"],
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
  <div class="resource-charts grid min-h-[240px] gap-3 p-3">
    <section
      class="flex min-h-[220px] min-w-0 flex-col rounded-md border border-border/70 bg-background/40 p-2"
      aria-label="节点资源"
      data-layout-region="resource-chart"
    >
      <dl class="resource-metrics mb-2 grid shrink-0 gap-2">
        <div
          v-for="metric in resourceMetrics"
          :key="metric.label"
          class="min-w-0 rounded border border-border/70 bg-muted/30 px-2 py-1.5 text-center"
        >
          <dt
            class="flex min-w-0 items-center justify-center gap-1 text-[10px] leading-4 text-muted-foreground"
            :title="metric.label"
          >
            <span
              class="inline-block h-1.5 w-1.5 shrink-0 rounded-full"
              :style="{ backgroundColor: metric.color }"
            /><span class="min-w-0 break-words">{{ metric.label }}</span>
          </dt>
          <dd class="text-xs font-semibold leading-4 text-foreground">
            {{ Number(metric.value.toFixed(2)) }}
          </dd>
        </div>
      </dl>
      <EChart
        class="min-h-[150px] flex-1"
        :option="resourceOption"
        aria-label="所选节点资源分配图"
      />
    </section>
    <EChart
      v-if="hasTaskData"
      class="min-h-[220px] rounded-md border border-border/70 bg-background/40"
      :option="taskOption"
      aria-label="任务状态分布图"
    />
  </div>
</template>
<style scoped>
.resource-charts {
  container-type: inline-size;
  grid-template-columns: minmax(0, 1fr);
}
.resource-metrics {
  grid-template-columns: repeat(auto-fit, minmax(6.5rem, 1fr));
}
@container (min-width: 34rem) {
  .resource-charts:has(> [aria-label="任务状态分布图"]) {
    grid-template-columns: minmax(0, 1.2fr) minmax(15rem, 0.8fr);
  }
}
</style>
