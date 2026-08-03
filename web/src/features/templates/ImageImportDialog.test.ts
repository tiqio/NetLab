import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import ImageImportDialog from "./ImageImportDialog.vue";
describe("ImageImportDialog", () => {
  it("warns against credentials and proprietary browser uploads", () => {
    mount(ImageImportDialog, {
      attachTo: document.body,
      props: { modelValue: true },
    });
    expect(document.body.textContent).toContain("Do not include credentials");
    expect(document.body.textContent).toContain("embedded proprietary bytes");
  });
});
