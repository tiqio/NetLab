import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import Input from "../input/Input.vue";
import FormField from "./FormField.vue";

describe("FormField", () => {
  it("associates a clean accessible label with its control", () => {
    const wrapper = mount(FormField, {
      props: { label: "名称" },
      slots: { default: Input },
    });

    const label = wrapper.get("label");
    const input = wrapper.get("input");
    expect(label.text()).toBe("名称");
    expect(label.attributes("for")).toBe(input.attributes("id"));
  });
});
