import { describe, expect, it } from "vitest";
import {
  defaultLightweightSwitchConfig,
  validateLightweightSwitchConfig,
} from "./lightweightSwitchConfig";

describe("lightweight switch defaults", () => {
  it("creates four editable L2 ports", () => {
    const config = defaultLightweightSwitchConfig("switch_l2");
    expect(config.ports!.map((item) => item.name)).toEqual([
      "eth0",
      "eth1",
      "eth2",
      "eth3",
    ]);
  });

  it("creates four editable L3 interfaces", () => {
    const config = defaultLightweightSwitchConfig("switch_l3");
    expect(config.interfaces!.map((item) => item.name)).toEqual([
      "eth0",
      "eth1",
      "eth2",
      "eth3",
    ]);
    expect(validateLightweightSwitchConfig("switch_l3", config)).toEqual([]);
  });

  it("rejects duplicate edited names", () => {
    const config = defaultLightweightSwitchConfig("switch_l2");
    config.ports![1].name = "eth0";
    expect(validateLightweightSwitchConfig("switch_l2", config)).toContain(
      "端口名称 eth0 重复。",
    );
  });
});
