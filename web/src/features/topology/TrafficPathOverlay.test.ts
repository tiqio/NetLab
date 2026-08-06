import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import TrafficPathOverlay from "./TrafficPathOverlay.vue";

describe("TrafficPathOverlay", () => {
  it("renders exact object-link identity, direction, and ambiguity", () => {
    const wrapper = mount(TrafficPathOverlay, {
      props: {
        ambiguous: true,
        observations: [
          {
            fingerprint: "udp",
            resource_type: "network_object_link",
            resource_id: "object-link-a",
            interface_id: "",
            network_object_link_id: "object-link-a",
            direction: "a_to_b",
            first_seen: "2026-08-03T00:00:00Z",
            last_seen: "2026-08-03T00:00:00.100Z",
            count: 2,
            bytes: 128,
          },
        ],
      },
    });
    expect(wrapper.text()).toContain("object-link-a · A 到 B");
    expect(wrapper.text()).toContain("2 个数据包");
    expect(wrapper.text()).toContain("无法唯一确定数据包路径");
  });

  it("keeps the observation summary static for reduced-motion users", () => {
    const wrapper = mount(TrafficPathOverlay, {
      props: { observations: [] },
    });
    expect(wrapper.find("aside").attributes("aria-live")).toBe("polite");
    expect(wrapper.find("[style*='animation']").exists()).toBe(false);
  });
});
