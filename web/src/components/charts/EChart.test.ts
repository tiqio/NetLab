import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
const setOption = vi.fn();
const resize = vi.fn();
const dispose = vi.fn();
const dispatchAction = vi.fn();
const on = vi.fn();
const off = vi.fn();
const zrOn = vi.fn();
const zrOff = vi.fn();
const convertFromPixel = vi.fn();
const graphicElement = { x: 0, y: 0 };
const graphData = {
  count: () => 1,
  getId: () => "node-1",
  getRawDataItem: () => ({ resourceType: "node" }),
  getItemGraphicEl: () => graphicElement,
  getItemLayout: () => [0, 0],
};
vi.mock("echarts/core", () => ({
  init: () => ({
    setOption,
    resize,
    dispose,
    dispatchAction,
    on,
    off,
    convertFromPixel,
    getModel: () => ({
      getSeriesByIndex: () => ({ getData: () => graphData }),
    }),
    getZr: () => ({ on: zrOn, off: zrOff }),
  }),
  use: vi.fn(),
}));
vi.mock("echarts/charts", () => ({
  GraphChart: {},
  LineChart: {},
  BarChart: {},
  PieChart: {},
}));
vi.mock("echarts/renderers", () => ({ CanvasRenderer: {} }));
vi.mock("echarts/components", () => ({
  GridComponent: {},
  LegendComponent: {},
  TooltipComponent: {},
  DataZoomComponent: {},
  GraphicComponent: {},
  TitleComponent: {},
}));
import EChart from "./EChart.vue";

describe("EChart", () => {
  beforeEach(() => vi.clearAllMocks());
  it("applies options and disposes resources", async () => {
    const wrapper = mount(EChart, { props: { option: { series: [] } } });
    expect(setOption).toHaveBeenCalled();
    expect(on).toHaveBeenCalled();
    wrapper.unmount();
    expect(off).toHaveBeenCalled();
    expect(dispose).toHaveBeenCalled();
  });
  it("preserves caller chart configuration while layering theme styles", async () => {
    const formatter = vi.fn();
    const wrapper = mount(EChart, {
      props: {
        option: {
          title: { text: "资源趋势", left: "center" },
          legend: { bottom: 8, selected: { CPU: true } },
          tooltip: { trigger: "axis", formatter },
          dataZoom: [{ type: "inside", start: 10, end: 90 }],
          series: [{ type: "line", data: [1, 2, 3] }],
        },
      },
    });
    const option = setOption.mock.calls.at(-1)?.[0];
    expect(option.title).toMatchObject({ text: "资源趋势", left: "center" });
    expect(option.legend).toMatchObject({
      bottom: 8,
      selected: { CPU: true },
    });
    expect(option.tooltip).toMatchObject({ trigger: "axis", formatter });
    expect(option.dataZoom).toEqual([{ type: "inside", start: 10, end: 90 }]);

    document.documentElement.dataset.theme = "light";
    await new Promise((resolve) => setTimeout(resolve, 0));
    const themedOption = setOption.mock.calls.at(-1)?.[0];
    expect(themedOption.tooltip.formatter).toBe(formatter);
    expect(themedOption.dataZoom).toEqual([
      { type: "inside", start: 10, end: 90 },
    ]);
    wrapper.unmount();
  });
  it("resolves semantic CSS variables before passing options to ECharts", () => {
    document.documentElement.style.setProperty(
      "--chart-series-primary",
      "#123456",
    );
    const wrapper = mount(EChart, {
      props: {
        option: {
          series: [
            {
              type: "bar",
              itemStyle: { color: "var(--chart-series-primary)" },
            },
          ],
        },
      },
    });
    const option = setOption.mock.calls.at(-1)?.[0];
    expect(option.series[0].itemStyle.color).toBe("#123456");
    wrapper.unmount();
    document.documentElement.style.removeProperty("--chart-series-primary");
  });
  it("keeps the pointer grab offset when committing a graph drag", () => {
    convertFromPixel.mockImplementation(
      (_finder: unknown, point: [number, number]) => [
        point[0] / 2,
        point[1] / 2,
      ],
    );
    const wrapper = mount(EChart, { props: { option: { series: [] } } });
    const handler = (name: string) =>
      zrOn.mock.calls.find(([event]) => event === name)?.[1] as (
        event: unknown,
      ) => void;
    handler("dragstart")({ target: graphicElement, offsetX: 100, offsetY: 80 });
    handler("dragend")({ target: graphicElement, offsetX: 300, offsetY: 180 });
    expect(wrapper.emitted("nodeDrag")?.[0]?.[0]).toMatchObject({
      data: { id: "node-1" },
      graphPoint: { x: 100, y: 50 },
    });
  });

  it("exposes chart readiness and reapplies options after resize", async () => {
    const wrapper = mount(EChart, { props: { option: { series: [] } } });
    await wrapper.vm.$nextTick();
    expect(wrapper.attributes("data-chart-ready")).toBe("true");
    expect(setOption).toHaveBeenCalled();
    wrapper.unmount();
  });
});
