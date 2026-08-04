import { describe, expect, it } from "vitest";
import { operationRegistry } from "../../../web/src/api/operationRegistry";
import { validateInventory } from "../fixtures/schemaValidation";
import inventoryDocument from "./interaction-inventory.json";

describe("interaction inventory", () => {
  it("is schema-valid, unique, and maps mutations to real operations", async () => {
    const interactions = await validateInventory(inventoryDocument);
    expect(new Set(interactions.map((item) => item.id)).size).toBe(
      interactions.length,
    );
    for (const interaction of interactions.filter((item) => item.operation)) {
      for (const operation of interaction.operation!.split(","))
        expect(operationRegistry).toHaveProperty(operation);
    }
  });
});
