import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import StatePresentation from "./StatePresentation.vue";
import EmptyState from "./EmptyState.vue";
import ResourceIdentity from "./ResourceIdentity.vue";

const states = [
  "empty",
  "loading",
  "stale",
  "reconnecting",
  "unsupported",
  "permission",
  "quota",
  "conflict",
  "partial-failure",
  "cleanup",
  "terminal-error",
] as const;

describe("interaction state matrix", () => {
  for (const state of states) {
    it(`presents ${state} with an observable status`, () => {
      const wrapper = mount(StatePresentation, {
        props: {
          state,
          title: `${state} title`,
          description: `${state} description`,
          actionAvailable: true,
          onAction: () => undefined,
        },
      });
      expect(wrapper.get('[role="status"]').text()).toContain(`${state} title`);
      expect(wrapper.get("button").text()).toContain("重试");
    });
  }

  it("keeps long empty-state copy inside a bounded region", () => {
    const wrapper = mount(EmptyState, {
      props: {
        title: "这是一个非常长的空状态标题，用于验证中文换行不会覆盖操作区域",
        description: "x".repeat(256),
      },
    });
    expect(wrapper.classes()).toContain("netlab-region");
    expect(wrapper.get("p").classes()).toContain("netlab-copy");
  });

  it("exposes the full resource identity without expanding the layout", () => {
    const id = "019fcff54c18-c97a2af8e396c1b95e96";
    const wrapper = mount(ResourceIdentity, {
      props: { type: "network_object", id, name: "x".repeat(128) },
    });
    expect(wrapper.get("strong").attributes("title")).toBe("x".repeat(128));
    expect(wrapper.get("code").attributes("title")).toContain(id);
  });
});
