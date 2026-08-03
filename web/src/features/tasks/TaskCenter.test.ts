import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import { taskFactory } from "@/test/factories";
import TaskCenter from "./TaskCenter.vue";
describe("TaskCenter", () => {
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
    await wrapper.get('input[aria-label="Filter tasks"]').setValue("capture");
    const cards = wrapper
      .findAll("article")
      .map((item) => item.text())
      .join(" ");
    expect(cards).not.toContain("node.start");
    expect(cards).toContain("capture.start");
  });
});
