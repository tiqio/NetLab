import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import StatePresentation from "./StatePresentation.vue";
import { localizeState } from "@/locales";

describe("StatePresentation", () => {
  for (const state of [
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
  ] as const) {
    it(`renders ${state} distinctly`, () => {
      const wrapper = mount(StatePresentation, { props: { state } });
      expect(wrapper.text()).toContain(localizeState(state));
    });
  }
});
