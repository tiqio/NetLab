import { readFile, writeFile } from "node:fs/promises";
import type { Page } from "@playwright/test";
import type { VisualAuditResult } from "./acceptanceTypes";
import type {
  LayoutRegionObservation,
  VisualAuditScenarioInventory,
  VisualFinding,
  VisualSeverity,
} from "./visualAuditTypes";

export async function loadVisualAuditInventory(path: string) {
  return JSON.parse(await readFile(path, "utf8")) as VisualAuditScenarioInventory;
}

export function validateVisualAuditInventory(inventory: VisualAuditScenarioInventory) {
  const ids = inventory.scenarios.map((scenario) => scenario.id);
  if (new Set(ids).size !== ids.length) throw new Error("visual audit scenario IDs must be unique");
  if (!inventory.themes.includes("light") || !inventory.themes.includes("dark"))
    throw new Error("visual audit inventory must cover both themes");
  for (const width of [1024, 1366, 1920])
    if (!inventory.viewports.some((viewport) => viewport.width === width))
      throw new Error(`visual audit inventory is missing viewport ${width}`);
  if (!inventory.display_scales.includes(1.25))
    throw new Error("visual audit inventory must cover 125% display scale");
}

export async function sampleLayoutRegions(page: Page) {
  return page.locator("[data-layout-region]").evaluateAll((elements) =>
    elements.map((element, index) => {
      const bounds = element.getBoundingClientRect();
      return {
        id: element.getAttribute("data-layout-region") || `region-${index}`,
        role: element.getAttribute("role") || element.tagName.toLowerCase(),
        bounds: { x: bounds.x, y: bounds.y, width: bounds.width, height: bounds.height },
        interactive: Boolean(element.matches("button,a,input,select,textarea,[tabindex]")),
        may_overlap: (element.getAttribute("data-may-overlap") || "").split(",").filter(Boolean),
      } satisfies LayoutRegionObservation;
    }),
  );
}

export function severityFor(kind: VisualFinding["kind"]): VisualSeverity {
  if (["hidden-focus", "hit-target-conflict"].includes(kind)) return "blocking";
  if (["overlap", "clipping", "overflow", "untranslated-text"].includes(kind)) return "serious";
  if (kind === "low-contrast" || kind === "layout-shift") return "moderate";
  return "cosmetic";
}

export function summarizeVisualAudit(results: VisualAuditResult[]) {
  return {
    total: results.length,
    passed: results.filter((result) => result.status === "passed").length,
    failed: results.filter((result) => result.status === "failed").length,
    waived: results.filter((result) => result.status === "waived").length,
    blocking_findings: results.reduce((total, result) => total + result.blocking_findings, 0),
    serious_findings: results.reduce((total, result) => total + result.serious_findings, 0),
  };
}

export async function writeVisualAudit(path: string, results: VisualAuditResult[]) {
  await writeFile(path, `${JSON.stringify({ schema_version: "1.0", summary: summarizeVisualAudit(results), results }, null, 2)}\n`, { mode: 0o600 });
}
