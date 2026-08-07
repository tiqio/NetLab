import { describe, expect, it } from "vitest";
import {
  cpuQuotaCoresToMicros,
  cpuQuotaMicrosToCores,
} from "./cpuQuota";

describe("CPU quota unit conversion", () => {
  it("converts between host-core units and cgroup period microseconds", () => {
    expect(cpuQuotaMicrosToCores(50_000)).toBe(0.5);
    expect(cpuQuotaMicrosToCores(100_000)).toBe(1);
    expect(cpuQuotaCoresToMicros(0.75)).toBe(75_000);
    expect(cpuQuotaCoresToMicros(2)).toBe(200_000);
  });

  it("keeps zero as unlimited", () => {
    expect(cpuQuotaMicrosToCores(0)).toBe(0);
    expect(cpuQuotaCoresToMicros(0)).toBe(0);
  });
});
