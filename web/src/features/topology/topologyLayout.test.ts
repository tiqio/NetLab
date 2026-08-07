import { describe, expect, it } from "vitest";
import { deterministicPlacement, resolvePlacements } from "./topologyLayout";
describe("topology layout", () => {
  it("is deterministic and collision aware", () => {
    const first = deterministicPlacement("node-1");
    expect(deterministicPlacement("node-1")).toEqual(first);
    expect(deterministicPlacement("node-2", [first])).not.toEqual(first);
  });

  it("keeps existing fallback coordinates when another resource becomes authoritative", () => {
    const resources = [
      { id: "node-a", name: "A", kind: "docker" },
      { id: "node-b", name: "B", kind: "docker" },
    ] as never[];
    const first = resolvePlacements(resources, {});
    const second = resolvePlacements(
      resources,
      {
        "node-a": {
          x: 900,
          y: 700,
          pinned: true,
          updatedAt: "",
        },
      },
      first,
    );
    expect(second["node-a"]).toEqual({
      x: 900,
      y: 700,
      pinned: true,
      updatedAt: "",
    });
    expect(second["node-b"]).toEqual(first["node-b"]);
  });

  it("does not retain fallback coordinates for deleted resources", () => {
    const resources = [
      { id: "node-a", name: "A", kind: "docker" },
    ] as never[];
    const resolved = resolvePlacements(resources, {}, {
      "node-a": { x: 10, y: 20 },
      deleted: { x: 30, y: 40 },
    });
    expect(resolved).toEqual({ "node-a": { x: 10, y: 20 } });
  });
});
