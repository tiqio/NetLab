import { describe, expect, it } from "vitest";
import { resolve } from "node:path";
import { loadVisualAuditInventory, validateVisualAuditInventory } from "../fixtures/visualAudit";

describe("UI overlap scenario inventory", () => {
  it("has unique scenarios and the required visual matrix", async () => {
    const inventory = await loadVisualAuditInventory(resolve(process.cwd(), "../tests/e2e/matrices/ui-overlap-scenarios.json"));
    expect(() => validateVisualAuditInventory(inventory)).not.toThrow();
    expect(inventory.scenarios.some((scenario) => scenario.surface === "topology")).toBe(true);
    expect(inventory.scenarios.some((scenario) => scenario.surface === "inspector")).toBe(true);
  });

  it("rejects duplicate scenario IDs and missing matrix coverage", async () => {
    const inventory = await loadVisualAuditInventory(resolve(process.cwd(), "../tests/e2e/matrices/ui-overlap-scenarios.json"));
    expect(() => validateVisualAuditInventory({
      ...inventory,
      scenarios: [...inventory.scenarios, inventory.scenarios[0]],
    })).toThrow("unique");
    expect(() => validateVisualAuditInventory({
      ...inventory,
      display_scales: [1],
    })).toThrow("125%");
  });
});
