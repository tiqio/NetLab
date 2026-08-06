<script setup lang="ts">
import { computed } from "vue";
import EChart from "./EChart.vue";

const props = defineProps<{ current: number; total: number; state: string }>();
const percent = computed(() =>
  Math.min(
    100,
    Math.max(0, props.total ? (props.current / props.total) * 100 : 0),
  ),
);
const option = computed(() => ({
  animationDurationUpdate: 180,
  grid: { left: 0, right: 0, top: 0, bottom: 0 },
  xAxis: { type: "value", min: 0, max: 100, show: false },
  yAxis: { type: "category", data: ["progress"], show: false },
  tooltip: {
    formatter: `${props.current}/${props.total || "?"} · ${Math.round(percent.value)}% · ${props.state}`,
  },
  series: [
    {
      type: "bar",
      data: [percent.value],
      barWidth: 7,
      showBackground: true,
      backgroundStyle: { color: "var(--chart-track)", borderRadius: 4 },
      itemStyle: {
        color:
          props.state === "failed"
            ? "var(--chart-danger)"
            : props.state === "cancelled"
              ? "var(--chart-warning)"
              : "var(--chart-series-primary)",
        borderRadius: 4,
      },
    },
  ],
}));
</script>
<template>
  <div class="h-3 w-full">
    <EChart :option="option" :aria-label="`任务进度 ${Math.round(percent)}%`" />
  </div>
</template>
