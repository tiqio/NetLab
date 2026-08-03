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
});
