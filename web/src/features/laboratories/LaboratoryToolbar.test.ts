import { mount } from "@vue/test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";
import { laboratoryFactory } from "@/test/factories";
import { api } from "@/api";
import LaboratoryToolbar from "./LaboratoryToolbar.vue";

const mounted: Array<ReturnType<typeof mount>> = [];
afterEach(() => {
  for (const wrapper of mounted.splice(0)) wrapper.unmount();
  vi.restoreAllMocks();
  document.body.innerHTML = "";
});

function render(active = laboratoryFactory()) {
  const labs = [
    active,
    laboratoryFactory({
      id: "lab-2",
      name: "Branch validation",
      revision: 4,
      recovery_policy: "remain_stopped",
    }),
  ];
  const wrapper = mount(LaboratoryToolbar, {
    attachTo: document.body,
    props: { labs, active, eventStatus: "connected" },
    global: {
      stubs: {
        RouterLink: { template: "<a><slot /></a>" },
      },
    },
  });
  mounted.push(wrapper);
  return wrapper;
}

describe("LaboratoryToolbar", () => {
  it("exposes a direct add-resource action without replacing the palette toggle", async () => {
    const wrapper = render();
    await wrapper.get('[aria-label="添加资源"]').trigger("click");
    expect(wrapper.emitted("openCreate")).toHaveLength(1);
    expect(wrapper.find('[aria-label="Toggle device palette"]').exists()).toBe(
      true,
    );
  });

  it("keeps the switcher DOM stable while live laboratory props change", async () => {
    const wrapper = render(laboratoryFactory({ id: "lab-1", name: "One" }));
    await wrapper.get('[data-testid="laboratory-switcher"]').trigger("click");
    const switcher = wrapper.get('[aria-label="Laboratory switcher"]').element;
    await wrapper.setProps({
      labs: [
        laboratoryFactory({ id: "lab-1", name: "One", revision: 2 }),
        laboratoryFactory({ id: "lab-2", name: "Two" }),
      ],
    });
    expect(wrapper.get('[aria-label="Laboratory switcher"]').element).toBe(
      switcher,
    );
    await wrapper.get('[data-testid="new-laboratory"]').trigger("click");
    expect(wrapper.get('[aria-label="Laboratory switcher"]').element).toBe(
      switcher,
    );
  });

  it("uses one searchable vertical laboratory switcher", async () => {
    const wrapper = render();
    expect(wrapper.find('[aria-label="Open laboratories"]').exists()).toBe(
      false,
    );
    await wrapper.get('[data-testid="laboratory-switcher"]').trigger("click");
    expect(wrapper.get('[role="listbox"]').isVisible()).toBe(true);
    await wrapper.get('[aria-label="Search laboratories"]').setValue("branch");
    const options = wrapper.findAll('[role="option"]');
    expect(options).toHaveLength(1);
    expect(options[0].text()).toContain("Branch validation");
    await options[0].trigger("click");
    expect(wrapper.emitted("select")?.[0]).toEqual(["lab-2"]);
    expect(wrapper.get('[role="listbox"]').isVisible()).toBe(false);
  });

  it("opens contextual management for a non-active laboratory", async () => {
    const wrapper = render();
    await wrapper.get('[data-testid="laboratory-switcher"]').trigger("click");
    const rows = wrapper.findAll('[data-laboratory-row="true"]');
    await rows[1].trigger("contextmenu", { clientX: 240, clientY: 160 });
    const menu = document.body.querySelector('[role="menu"]');
    expect(menu?.textContent).toContain("Branch validation");
    expect(menu?.textContent).toContain("Rename");
    expect(menu?.textContent).toContain("Duplicate");
    expect(menu?.textContent).toContain("Export");
    expect(menu?.textContent).toContain("Import");
    expect(menu?.textContent).toContain("Delete");
    expect(wrapper.emitted("select")).toBeUndefined();
  });

  it("keeps laboratory creation in the switcher", async () => {
    const wrapper = render();
    await wrapper.get('[data-testid="laboratory-switcher"]').trigger("click");
    await wrapper.get('[data-testid="new-laboratory"]').trigger("click");
    expect(document.body.textContent).toContain("Create laboratory");
    expect(wrapper.get('[role="listbox"]').isVisible()).toBe(false);
  });

  it("deletes the context target without selecting it first", async () => {
    const active = laboratoryFactory({ id: "lab-delete" });
    const target = laboratoryFactory({
      id: "lab-2",
      name: "Branch validation",
      revision: 4,
      recovery_policy: "remain_stopped",
    });
    vi.spyOn(api, "deleteLab").mockResolvedValueOnce({
      task: {
        id: "task-delete",
        kind: "laboratory.delete",
        resource_type: "laboratory",
        resource_id: target.id,
        state: "queued",
        progress_current: 0,
        progress_total: 1,
        created_at: "2026-07-28T00:00:00Z",
      },
    });
    const wrapper = render(active);
    await wrapper.get('[data-testid="laboratory-switcher"]').trigger("click");
    const rows = wrapper.findAll('[data-laboratory-row="true"]');
    await rows[1].trigger("contextmenu", { clientX: 240, clientY: 160 });
    const deleteAction = Array.from(
      document.body.querySelectorAll('[role="menu"] button'),
    ).find((button) => button.textContent?.trim() === "Delete") as
      HTMLButtonElement | undefined;
    expect(deleteAction).toBeDefined();
    deleteAction!.click();
    await wrapper.vm.$nextTick();
    const confirm = Array.from(document.body.querySelectorAll("button")).find(
      (button) => button.textContent?.trim() === "Delete",
    );
    expect(confirm).toBeDefined();
    confirm!.click();
    await wrapper.vm.$nextTick();

    expect(api.deleteLab).toHaveBeenCalledWith(target);
    expect(wrapper.emitted("deleteAccepted")?.[0]).toEqual([target.id]);
    expect(wrapper.emitted("select")).toBeUndefined();
  });
});
