<script setup lang="ts">
import {
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  shallowRef,
  watch,
} from "vue";
import { init, use, type ECharts, type EChartsCoreOption } from "echarts/core";
import { GraphChart, LineChart, BarChart, PieChart } from "echarts/charts";
import { CanvasRenderer } from "echarts/renderers";
import {
  GridComponent,
  LegendComponent,
  TooltipComponent,
  DataZoomComponent,
  GraphicComponent,
  TitleComponent,
} from "echarts/components";

use([
  GraphChart,
  LineChart,
  BarChart,
  PieChart,
  CanvasRenderer,
  GridComponent,
  LegendComponent,
  TooltipComponent,
  DataZoomComponent,
  GraphicComponent,
  TitleComponent,
]);
const props = withDefaults(
  defineProps<{
    option: EChartsCoreOption;
    ariaLabel?: string;
    notMerge?: boolean;
  }>(),
  { ariaLabel: "Interactive chart", notMerge: false },
);
const emit = defineEmits<{
  ready: [ECharts];
  rendered: [];
  resized: [];
  chartClick: [unknown];
  graphRoam: [unknown];
  nodeDragStart: [unknown];
  nodeDragMove: [unknown];
  nodeDrag: [unknown];
  canvasPointer: [MouseEvent];
  chartOver: [unknown];
  chartOut: [unknown];
  chartContext: [unknown];
}>();
const root = ref<HTMLDivElement>();
const chart = shallowRef<ECharts>();
let observer: ResizeObserver | undefined;
let activeGraphDrag:
  { id: string; offsetX: number; offsetY: number } | undefined;
let pendingApply = false;

function graphPointAt(point: [number, number], seriesIndex = 0) {
  const value = chart.value?.convertFromPixel({ seriesIndex }, point) as
    number[] | undefined;
  if (!value || !Number.isFinite(value[0]) || !Number.isFinite(value[1]))
    return undefined;
  return { x: value[0], y: value[1] };
}
function canvasPointAt(point: { x: number; y: number }, seriesIndex = 0) {
  const value = chart.value?.convertToPixel({ seriesIndex }, [
    point.x,
    point.y,
  ]) as number[] | undefined;
  if (!value || !Number.isFinite(value[0]) || !Number.isFinite(value[1]))
    return undefined;
  return { x: value[0], y: value[1] };
}
function graphItemCanvasPoint(id: string, seriesIndex = 0) {
  const series = (
    chart.value as unknown as {
      getModel: () => {
        getSeriesByIndex: (index: number) => {
          getData: () => {
            count: () => number;
            getId: (index: number) => string;
            getItemGraphicEl: (index: number) => {
              transformCoordToGlobal?: (x: number, y: number) => number[];
            };
          };
        };
      };
    }
  )
    .getModel?.()
    .getSeriesByIndex(seriesIndex);
  const data = series?.getData();
  if (!data) return undefined;
  for (let index = 0; index < data.count(); index += 1) {
    if (data.getId(index) !== id) continue;
    const point = data.getItemGraphicEl(index)?.transformCoordToGlobal?.(0, 0);
    if (!point || !Number.isFinite(point[0]) || !Number.isFinite(point[1]))
      return undefined;
    return { x: point[0], y: point[1] };
  }
  return undefined;
}

function graphDragEvent(event: unknown, phase: "start" | "move" | "end") {
  const value = event as {
    target?: { parent?: unknown };
    offsetX?: number;
    offsetY?: number;
  };
  const series = (
    chart.value as unknown as {
      getModel: () => {
        getSeriesByIndex: (index: number) => {
          getData: () => {
            count: () => number;
            getId: (index: number) => string;
            getRawDataItem: (index: number) => Record<string, unknown>;
            getItemGraphicEl: (index: number) => unknown;
            getItemLayout: (index: number) => number[];
          };
        };
      };
    }
  )
    .getModel?.()
    .getSeriesByIndex(0);
  const data = series?.getData();
  if (!data || !value.target) return undefined;
  for (let index = 0; index < data.count(); index += 1) {
    const element = data.getItemGraphicEl(index);
    let target: unknown = value.target;
    while (target) {
      if (target === element) {
        const layout = data.getItemLayout(index);
        const position = element as { x?: number; y?: number };
        const x = Number(position.x);
        const y = Number(position.y);
        const id = data.getId(index);
        const pointerPoint = graphPointAt([
          Number(value.offsetX || 0),
          Number(value.offsetY || 0),
        ]);
        if (
          phase === "start" &&
          pointerPoint &&
          layout &&
          Number.isFinite(layout[0]) &&
          Number.isFinite(layout[1])
        )
          activeGraphDrag = {
            id,
            offsetX: layout[0] - pointerPoint.x,
            offsetY: layout[1] - pointerPoint.y,
          };
        const pointerDragPoint =
          pointerPoint && activeGraphDrag?.id === id
            ? {
                x: pointerPoint.x + activeGraphDrag.offsetX,
                y: pointerPoint.y + activeGraphDrag.offsetY,
              }
            : undefined;
        const graphPoint =
          pointerDragPoint ||
          (Number.isFinite(x) && Number.isFinite(y)
            ? { x, y }
            : layout && Number.isFinite(layout[0]) && Number.isFinite(layout[1])
              ? { x: layout[0], y: layout[1] }
              : pointerPoint);
        if (phase === "end") activeGraphDrag = undefined;
        return {
          data: {
            ...data.getRawDataItem(index),
            id,
          },
          event: value,
          graphPoint,
        };
      }
      target = (target as { parent?: unknown }).parent;
    }
  }
  return undefined;
}

function apply() {
  if (activeGraphDrag) {
    pendingApply = true;
    return;
  }
  chart.value?.setOption(props.option, {
    notMerge: props.notMerge,
    lazyUpdate: true,
  });
}
function graphViewport(event: unknown) {
  const option = chart.value?.getOption() as
    { series?: Array<{ center?: unknown; zoom?: unknown }> } | undefined;
  const series = option?.series?.[0];
  const center = Array.isArray(series?.center) ? series.center : undefined;
  const centerX = Number(center?.[0]);
  const centerY = Number(center?.[1]);
  const zoom = Number(series?.zoom);
  if (
    Number.isFinite(centerX) &&
    Number.isFinite(centerY) &&
    Number.isFinite(zoom)
  )
    return { centerX, centerY, zoom };
  return event;
}
onMounted(() => {
  if (!root.value) return;
  chart.value = init(root.value, undefined, { renderer: "canvas" });
  chart.value.on("click", (event) => emit("chartClick", event));
  chart.value.on("mouseover", (event) => emit("chartOver", event));
  chart.value.on("mouseout", (event) => emit("chartOut", event));
  chart.value.on("contextmenu", (event) => emit("chartContext", event));
  chart.value.on("finished", () => emit("rendered"));
  chart.value.on("graphroam", (event) =>
    emit("graphRoam", graphViewport(event)),
  );
  const zr = chart.value.getZr();
  zr.on("dragstart", (event) => {
    chart.value?.dispatchAction({ type: "hideTip" });
    const normalized = graphDragEvent(event, "start");
    if (normalized) emit("nodeDragStart", normalized);
  });
  zr.on("drag", (event) => {
    const normalized = graphDragEvent(event, "move");
    if (normalized) emit("nodeDragMove", normalized);
  });
  zr.on("dragend", (event) => {
    const normalized = graphDragEvent(event, "end");
    if (normalized) emit("nodeDrag", normalized);
    activeGraphDrag = undefined;
    if (pendingApply)
      void nextTick(() => {
        pendingApply = false;
        apply();
      });
  });
  observer = new ResizeObserver(() => {
    chart.value?.resize();
    emit("resized");
  });
  observer.observe(root.value);
  apply();
  emit("ready", chart.value);
});
watch(() => props.option, apply, { deep: true });
onBeforeUnmount(() => {
  observer?.disconnect();
  activeGraphDrag = undefined;
  pendingApply = false;
  chart.value?.getZr().off("dragstart");
  chart.value?.getZr().off("drag");
  chart.value?.getZr().off("dragend");
  chart.value?.off();
  chart.value?.dispose();
  chart.value = undefined;
});
defineExpose({
  resize: () => chart.value?.resize(),
  getInstance: () => chart.value,
  dataPointToPixel: (point: { x: number; y: number }) => canvasPointAt(point),
  graphItemPixel: (id: string) => graphItemCanvasPoint(id),
  canvasSize: () => ({
    width: root.value?.clientWidth || 0,
    height: root.value?.clientHeight || 0,
  }),
  dataPointAtCanvasCenter: () =>
    root.value
      ? graphPointAt([root.value.clientWidth / 2, root.value.clientHeight / 2])
      : undefined,
});
</script>
<template>
  <div
    ref="root"
    role="img"
    :aria-label="ariaLabel"
    class="h-full min-h-0 w-full"
    @mousemove="$emit('canvasPointer', $event)"
  />
</template>
