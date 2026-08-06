import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import StatePresentation from "./StatePresentation.vue";

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
});
