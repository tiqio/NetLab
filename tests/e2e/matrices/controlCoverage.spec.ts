import { test, expect } from "../fixtures/acceptanceFixture";
import { validateInventory } from "../fixtures/schemaValidation";
import { auditControls, discoverControls } from "./controlInventoryAudit";
import inventoryDocument from "./interaction-inventory.json";

test("visible empty-workspace controls are inventoried and explain unavailability", async ({
  page,
}) => {
  const inventory = await validateInventory(inventoryDocument);
  await page.goto("/");
  const controls = await discoverControls(page);
  const expectedNames = [
    "Toggle device palette",
    "Laboratory",
    "Templates",
    "Automation",
    "Refresh",
    "⌘K",
  ];
  const missing = auditControls(
    controls.filter((control) => expectedNames.includes(control.name)),
    inventory,
  );
  expect(missing.map((item) => item.name)).toEqual([]);
  const palette = page.getByRole("button", {
    name: "Toggle device palette",
  });
  if (await page.getByText(/暂无实验室/).isVisible()) {
    await expect(palette).toBeDisabled();
    await expect(palette).toHaveAttribute("title", /Create or select/);
    await page.getByTestId("laboratory-switcher").click();
    await expect(page.getByTestId("new-laboratory")).toBeVisible();
    await expect(
      page.getByRole("button", { name: /^Actions for / }),
    ).toHaveCount(0);
  }
});
