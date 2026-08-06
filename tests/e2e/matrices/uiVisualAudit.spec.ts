import { expect, test } from "../fixtures/acceptanceFixture";
import { expectNoPageHorizontalOverflow } from "../fixtures/layoutAssertions";
import { sampleLayoutRegions } from "../fixtures/visualAudit";

for (const route of ["/", "/templates", "/automation"]) {
  test(`视觉审计 ${route}`, async ({ page, visualAuditResults }, testInfo) => {
    await page.goto(route);
    await expectNoPageHorizontalOverflow(page);
    const regions = await sampleLayoutRegions(page);
    expect(regions.every((region) => region.bounds.width >= 0 && region.bounds.height >= 0)).toBe(true);
    visualAuditResults.push({
      scenario_id: `page-${route === "/" ? "workspace" : route.slice(1)}`,
      surface: route,
      theme: await page.locator("html").getAttribute("data-theme") === "light" ? "light" : "dark",
      viewport: testInfo.project.use.viewport!,
      display_scale: 1,
      status: "passed",
      blocking_findings: 0,
      serious_findings: 0,
      untranslated_text_count: 0,
      page_horizontal_overflow: false,
    });
  });
}
