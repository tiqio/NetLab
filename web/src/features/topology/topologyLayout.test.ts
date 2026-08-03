import { describe, expect, it } from "vitest";
import { deterministicPlacement } from "./topologyLayout";
describe("topology layout", () => {
  it("is deterministic and collision aware", () => {
    const first = deterministicPlacement("node-1");
    expect(deterministicPlacement("node-1")).toEqual(first);
    expect(deterministicPlacement("node-2", [first])).not.toEqual(first);
  });
});
