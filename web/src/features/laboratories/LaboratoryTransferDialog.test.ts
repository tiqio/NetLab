import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { api, type Laboratory, type OperationTask } from "@/api";
import LaboratoryTransferDialog from "./LaboratoryTransferDialog.vue";

const laboratory: Laboratory = {
  id: "lab-1",
  name: "Lab",
  description: "",
  recovery_policy: "remain_stopped",
  revision: 1,
  lifecycle_state: "active",
};

const succeededTask: OperationTask = {
  id: "task-1",
  kind: "laboratory.export",
  resource_type: "laboratory",
  resource_id: "lab-1",
  state: "succeeded",
  progress_current: 2,
  progress_total: 2,
  created_at: "2026-07-27T00:00:00Z",
  result: {
    artifact: {
      id: "artifact-1",
      kind: "laboratory_export",
      media_type: "application/json",
      size_bytes: 42,
      sha256: `sha256:${"a".repeat(64)}`,
      owner_type: "laboratory",
      owner_id: "lab-1",
      created_at: "2026-07-27T00:00:00Z",
    },
  },
};

describe("LaboratoryTransferDialog", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("shows durable export artifact metadata", async () => {
    vi.spyOn(api, "exportLab").mockResolvedValue({ task: succeededTask });
    vi.spyOn(api, "getTask").mockResolvedValue(succeededTask);
    const wrapper = mount(LaboratoryTransferDialog, {
      props: { modelValue: true, mode: "export", laboratory },
      global: { stubs: { teleport: true } },
    });
    const createExport = wrapper
      .findAll("button")
      .find((button) => button.text().includes("创建导出"));
    expect(createExport).toBeDefined();
    await createExport!.trigger("click");
    await flushPromises();
    expect(wrapper.text()).toContain("导出产物已就绪");
    expect(wrapper.text()).toContain("42 字节");
    expect(wrapper.get("a").attributes("href")).toBe(
      "/api/v1/artifacts/artifact-1",
    );
  });

  it("reports missing image digests and redaction metadata", async () => {
    vi.spyOn(api, "listImages").mockResolvedValue([]);
    const wrapper = mount(LaboratoryTransferDialog, {
      props: { modelValue: false, mode: "import", laboratory },
      global: { stubs: { teleport: true } },
    });
    await wrapper.setProps({ modelValue: true });
    await flushPromises();
    await wrapper.get("textarea").setValue(
      JSON.stringify({
        schema_version: 1,
        nodes: [{ image_digest: `sha256:${"b".repeat(64)}` }],
        redaction: {
          images_excluded: true,
          credentials_excluded: true,
        },
      }),
    );
    expect(wrapper.text()).toContain("缺少镜像");
    expect(wrapper.text()).toContain("凭据已排除");
    const importButton = wrapper
      .findAll("button")
      .find((button) => button.text().trim() === "导入");
    expect(importButton?.attributes("disabled")).toBeDefined();
  });
});
