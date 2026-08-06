import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import { taskFactory } from "@/test/factories";
import TaskCenter from "./TaskCenter.vue";
describe("TaskCenter", () => {
  it("renders large task histories incrementally", async () => {
    const wrapper = mount(TaskCenter, {
      props: {
        tasks: Array.from({ length: 100 }, (_, index) =>
          taskFactory({ id: `task-${index}`, kind: `task.${index}` }),
        ),
      },
    });
    expect(wrapper.findAll("article")).toHaveLength(30);
    await wrapper.get('button[aria-label="显示更多任务"]').trigger("click");
    expect(wrapper.findAll("article")).toHaveLength(60);
  });

  it("filters tasks and keeps resource navigation context", async () => {
    const wrapper = mount(TaskCenter, {
      props: {
        tasks: [
          taskFactory({ id: "start", kind: "node.start" }),
          taskFactory({
            id: "capture",
            kind: "capture.start",
            state: "failed",
          }),
        ],
      },
    });
    expect(wrapper.text()).toContain("node.start");
    await wrapper.get('input[aria-label="筛选任务"]').setValue("capture");
    const cards = wrapper
      .findAll("article")
      .map((item) => item.text())
      .join(" ");
    expect(cards).not.toContain("node.start");
    expect(cards).toContain("capture.start");
  });
});
