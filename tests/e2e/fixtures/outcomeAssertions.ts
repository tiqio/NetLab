import {
  expect,
  type APIRequestContext,
  type Locator,
  type Page,
} from "@playwright/test";
import type { InteractionDefinition } from "./acceptanceTypes";

export async function assertObservableOutcome(options: {
  definition: InteractionDefinition;
  page: Page;
  beforeUrl: string;
  beforeText: string;
  feedback?: Locator;
  request?: APIRequestContext;
}) {
  const { definition, page, beforeUrl, beforeText, feedback } = options;
  if (definition.outcome_class === "navigation") {
    await expect.poll(() => page.url()).not.toBe(beforeUrl);
    return;
  }
  if (feedback) {
    await expect(feedback).toBeVisible();
    return;
  }
  await expect
    .poll(async () => (await page.locator("body").innerText()).trim())
    .not.toBe(beforeText.trim());
}

export function assertSkipEligibility(definition: InteractionDefinition) {
  if (!definition.optional_environment_capabilities?.length) {
    throw new Error(`${definition.id} cannot be skipped`);
  }
}
