import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import StructuredProblem from "./StructuredProblem.vue";
describe("StructuredProblem", () => {
  it("shows actionable fields", () => {
    const wrapper = mount(StructuredProblem, {
      props: {
        problem: {
          code: "revision_conflict",
          message: "stale",
          retryable: true,
          phase: "validate",
          cleanup: "complete",
          operator_hint: "refresh",
        },
      },
    });
    expect(wrapper.text()).toContain("stale");
    expect(wrapper.text()).toContain("refresh");
    expect(wrapper.get('[role="alert"]')).toBeTruthy();
  });
});
