<script setup lang="ts">
import { computed } from "vue";
import EChart from "./EChart.vue";

const props = defineProps<{
  bytes: number;
  maximum: number;
  packets: number;
  truncated?: boolean;
}>();
const percent = computed(() =>
  Math.min(
    100,
    Math.max(0, props.maximum ? (props.bytes / props.maximum) * 100 : 0),
  ),
);
const option = computed(() => ({
  grid: { left: 0, right: 0, top: 0, bottom: 0 },
  xAxis: { type: "value", min: 0, max: 100, show: false },
  yAxis: { type: "category", data: ["capture"], show: false },
  tooltip: {
    formatter: `${props.bytes} / ${props.maximum} bytes · ${props.packets} packets`,
  },
  series: [
    {
      type: "bar",
      data: [percent.value],
      barWidth: 10,
      showBackground: true,
      backgroundStyle: { color: "var(--chart-track)", borderRadius: 5 },
      itemStyle: {
        color: props.truncated
          ? "var(--chart-warning)"
          : "var(--chart-series-secondary)",
        borderRadius: 5,
      },
    },
  ],
}));
</script>
<template>
  <div class="h-4 w-full">
    <EChart
      :option="option"
      :aria-label="`抓包配额已使用 ${Math.round(percent)}%`"
    />
  </div>
</template>
