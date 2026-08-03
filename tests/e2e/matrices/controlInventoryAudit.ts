import type { Locator, Page } from "@playwright/test";
import type { InteractionDefinition } from "../fixtures/acceptanceTypes";

export interface DiscoveredControl {
  role: string;
  name: string;
  enabled: boolean;
  locator: Locator;
}

function globMatches(pattern: string | undefined, value: string) {
  if (!pattern) return true;
  const expression = new RegExp(
    `^${pattern
      .replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
      .replace(/\\\*/g, ".*")}$`,
    "i",
  );
  return expression.test(value.trim());
}

export async function discoverControls(
  page: Page,
): Promise<DiscoveredControl[]> {
  const locator = page.locator(
    'button, a[href], input, select, textarea, [role="tab"], [tabindex="0"]',
  );
  const controls: DiscoveredControl[] = [];
  for (let index = 0; index < (await locator.count()); index += 1) {
    const item = locator.nth(index);
    if (!(await item.isVisible().catch(() => false))) continue;
    const role =
      (await item.getAttribute("role")) ||
      (await item.evaluate((element) => element.tagName.toLowerCase()));
    const name = await item.evaluate((element) => {
      const aria = element.getAttribute("aria-label");
      if (aria) return aria;
      const labelledBy = element.getAttribute("aria-labelledby");
      if (labelledBy) {
        return labelledBy
          .split(/\s+/)
          .map((id) => document.getElementById(id)?.textContent || "")
          .join(" ")
          .trim();
      }
      const label = element.closest("label")?.textContent;
      return (label || element.textContent || "").replace(/\s+/g, " ").trim();
    });
    controls.push({
      role,
      name,
      enabled: await item.isEnabled().catch(() => true),
      locator: item,
    });
  }
  return controls;
}

export function matchInventory(
  control: DiscoveredControl,
  inventory: InteractionDefinition[],
) {
  return inventory.find((item) => globMatches(item.locator.name, control.name));
}

export function auditControls(
  controls: DiscoveredControl[],
  inventory: InteractionDefinition[],
) {
  return controls.filter((control) => !matchInventory(control, inventory));
}
