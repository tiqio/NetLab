import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import StructuredProblem from "@/components/common/StructuredProblem.vue";

describe("node operation problem flows", () => {
  it("preserves actionable revision, quota, and cleanup fields", () => {
    const wrapper = mount(StructuredProblem, {
      props: {
        problem: {
          code: "revision_conflict",
          message: "Node changed in another client",
          retryable: true,
          phase: "resource_update",
          cleanup: "complete",
          operator_hint: "Refresh state, compare values, then retry",
          retry_after_seconds: 2,
          details: { limit: "cpu_quota" },
        },
      },
    });
    expect(wrapper.text()).toContain(
      "Refresh state, compare values, then retry",
    );
    expect(wrapper.text()).toContain("cpu_quota");
    expect(wrapper.text()).toContain("2 秒后");
  });
});
