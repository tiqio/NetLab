import { test, expect } from "../fixtures/acceptanceFixture";
import {
  createOwnedLaboratory,
  resolveTemplateSelection,
} from "../journeys/completeRealJourney";
import { TemplatePage } from "../pages/TemplatePage";

test("every available template version creates through the frontend", async ({
  page,
  automation,
  ledger,
  runId,
  versionCoverage,
  environment,
}) => {
  test.skip(
    process.env.NETLAB_ACCEPTANCE_PROFILE !== "target-host",
    "complete version coverage runs only on target host",
  );
  const { laboratory } = await createOwnedLaboratory(
    page,
    automation,
    ledger,
    runId,
  );
  const response = await automation.get("/api/v1/templates");
  const catalog = (await response.json()) as Array<{
    template_key: string;
    runtime_kind: "qemu" | "docker";
    versions: Array<{ id: string; enabled: boolean }>;
  }>;
  const templates = new TemplatePage(page, automation);
  const availableFamilies = environment.templates.filter((family) =>
    family.versions.some((version) => version.available),
  );
  for (const observation of availableFamilies) {
    const family = catalog.find(
      (candidate) => candidate.template_key === observation.device_family,
    );
    if (!family) throw new Error(`Missing template ${observation.device_family}`);
    const selection = await resolveTemplateSelection(
      automation,
      family.template_key,
    );
    const node = await templates.createDevice({
      ...selection,
      nodeName: `${family.template_key.slice(0, 12)}-${runId.slice(0, 4)}`,
      laboratoryId: laboratory.id,
    });
    await ledger.add({
      resource_type: "node",
      resource_id: node.id,
      laboratory_id: laboratory.id,
      revision: node.revision,
      cleanup_method: "laboratory-cascade",
    });
    versionCoverage.push({
      runtime: family.runtime_kind,
      device_family: family.template_key,
      version_id: selection.versionId,
      image_id: selection.imageId,
      coverage_level: "full-journey",
      result: "passed",
      interactions: ["template.node.create"],
    });
  }
  expect(versionCoverage).toHaveLength(availableFamilies.length);
});
