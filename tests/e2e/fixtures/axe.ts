import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import type { Page } from "@playwright/test";
import { expect } from "./acceptanceFixture";

const axeSource = readFileSync(
  resolve(process.cwd(), "node_modules/axe-core/axe.min.js"),
  "utf8",
);

export async function expectNoSeriousAxeViolations(page: Page) {
  await page.addScriptTag({ content: axeSource });
  const violations = await page.evaluate(async () => {
    const axe = (
      window as typeof window & {
        axe: {
          run: (
            context: Document,
            options: object,
          ) => Promise<{
            violations: Array<{
              id: string;
              impact: string | null;
              nodes: Array<{ target: string[]; failureSummary?: string }>;
            }>;
          }>;
        };
      }
    ).axe;
    const result = await axe.run(document, {
      resultTypes: ["violations"],
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21aa"] },
    });
    return result.violations.filter((item) =>
      ["serious", "critical"].includes(item.impact || ""),
    );
  });
  expect(violations, JSON.stringify(violations, null, 2)).toEqual([]);
}
