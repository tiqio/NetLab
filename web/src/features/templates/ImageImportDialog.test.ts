import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import ImageImportDialog from "./ImageImportDialog.vue";
describe("ImageImportDialog", () => {
  it("warns against credentials and proprietary browser uploads", () => {
    mount(ImageImportDialog, {
      attachTo: document.body,
      props: { modelValue: true },
    });
    expect(document.body.textContent).toContain("引用中不得包含凭据");
    expect(document.body.textContent).toContain("嵌入专有镜像数据");
  });
});
