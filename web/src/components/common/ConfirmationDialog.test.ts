import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import ConfirmationDialog from "./ConfirmationDialog.vue";

describe("ConfirmationDialog", () => {
  it("identifies the resource, cleanup impact, and explicit action", async () => {
    const wrapper = mount(ConfirmationDialog, {
      attachTo: document.body,
      props: {
        modelValue: true,
        title: "Delete node",
        resource: "router-1 · node-1",
        description: "Owned resources are removed.",
        impact: "Streams are interrupted.",
      },
    });
    expect(document.body.textContent).toContain("router-1");
    expect(document.body.textContent).toContain("Streams are interrupted");
    wrapper.unmount();
  });
});
