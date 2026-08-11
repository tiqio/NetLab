import { mount } from "@vue/test-utils";
import { nextTick } from "vue";
import { describe, expect, it } from "vitest";
import LightweightSwitchConfigEditor from "@/features/nodes/LightweightSwitchConfigEditor.vue";

describe("network object VLAN editor", () => {
  it("preserves invalid VLAN drafts and publishes after correction", async () => {
    const wrapper = mount(LightweightSwitchConfigEditor, {
      props: {
        kind: "switch_l2",
        modelValue: {
          vlan_filtering: true,
          ports: [{ name: "eth0", pvid: 10, tagged: [20] }],
        },
      },
    });
    const inputs = wrapper.findAll("input");
    await inputs[3].setValue("10,20");
    await nextTick();
    expect(wrapper.text()).toContain("PVID 不能同时出现在 Tagged VLAN 中");
    expect((inputs[3].element as HTMLInputElement).value).toBe("10,20");
    expect(wrapper.emitted("update:modelValue")).toBeUndefined();

    await inputs[3].setValue("20,30");
    await nextTick();
    const updates = wrapper.emitted("update:modelValue") || [];
    expect(updates.at(-1)?.[0]).toEqual({
      vlan_filtering: true,
      ports: [{ name: "eth0", pvid: 10, tagged: [20, 30] }],
    });
  });

  it("reports duplicate and out-of-range VLAN drafts", async () => {
    const wrapper = mount(LightweightSwitchConfigEditor, {
      props: {
        kind: "switch_l2",
        modelValue: {
          vlan_filtering: true,
          ports: [
            { name: "eth0", pvid: 10, tagged: [] },
            { name: "eth1", pvid: 20, tagged: [] },
          ],
        },
      },
    });
    const inputs = wrapper.findAll("input");
    await inputs[4].setValue("eth0");
    await inputs[6].setValue("20,4095,20");
    await nextTick();
    expect(wrapper.text()).toContain("端口名称不能重复");
    expect(wrapper.text()).toContain("Tagged VLAN 必须是 1-4094 的整数列表");
    expect(wrapper.text()).toContain("Tagged VLAN 不能重复");
  });
});
