import { describe, expect, it } from "vitest";
import { buildPlacementBatch } from "./topologyPlacementBatch";

describe("buildPlacementBatch", () => {
  it("moves a selected group by one shared delta", () => {
    expect(
      buildPlacementBatch(
        "node-1",
        { x: 30, y: 40 },
        ["node-1", "node-2"],
        {
          "node-1": { x: 10, y: 10 },
          "node-2": { x: 50, y: 20 },
        },
        { "node-1": "node", "node-2": "node" },
        [
          {
            laboratory_id: "lab",
            resource_id: "node-2",
            resource_type: "node",
            x: 50,
            y: 20,
            revision: 3,
          },
        ],
      ),
    ).toEqual([
      {
        resource_id: "node-1",
        resource_type: "node",
        x: 30,
        y: 40,
        revision: undefined,
      },
      {
        resource_id: "node-2",
        resource_type: "node",
        x: 70,
        y: 50,
        revision: 3,
      },
    ]);
  });
});
