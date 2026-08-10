import type { APIRequestContext, Page } from "@playwright/test";
import type { InteractionResult } from "../fixtures/acceptanceTypes";
import type { ResourceLedger } from "../fixtures/resourceLedger";
import { LaboratoryPage } from "../pages/LaboratoryPage";
import { TemplatePage } from "../pages/TemplatePage";

export function result(
  interactionId: string,
  viewport: { width: number; height: number },
  actual: string,
  resourceIds: string[] = [],
  activation: InteractionResult["activation"] = "pointer",
  durationMs = 1,
  extra: Partial<InteractionResult> = {},
): InteractionResult {
  return {
    interaction_id: interactionId,
    status: "passed",
    viewport,
    activation,
    precondition: "declared journey precondition satisfied",
    action: interactionId,
    expected: "visible and authoritative outcome",
    actual,
    duration_ms: Math.max(1, durationMs),
    cleanup_status: "owned by acceptance ledger",
    resource_ids: resourceIds,
    ...extra,
  };
}

function imageMatchesTemplate(imageName: string, templateKey: string) {
  const normalize = (value: string) =>
    value.toLowerCase().replace(/[^a-z0-9]+/g, "");
  const family = templateKey.replace(/-(?:container|qemu)$/, "");
  return [templateKey, family].map(normalize).includes(normalize(imageName));
}

export async function createOwnedLaboratory(
  page: Page,
  request: APIRequestContext,
  ledger: ResourceLedger,
  runId: string,
) {
  const laboratories = new LaboratoryPage(page, request);
  await laboratories.open();
  const prefix = process.env.NETLAB_ACCEPTANCE_LAB_PREFIX || "accept";
  const laboratory = await laboratories.create(
    `${prefix}-${runId.slice(0, 8)}-${crypto.randomUUID().slice(0, 6)}`,
  );
  await ledger.add({
    resource_type: "laboratory",
    resource_id: laboratory.id,
    revision: laboratory.revision,
    cleanup_method: "frontend-delete-with-api-fallback",
  });
  return { laboratories, laboratory };
}

export async function createOwnedLightweightPair(
  page: Page,
  request: APIRequestContext,
  ledger: ResourceLedger,
  laboratoryId: string,
  kind: "PC" | "Layer-2 switch" = "PC",
) {
  const templates = new TemplatePage(page, request);
  const first = await templates.createLightweight(laboratoryId, kind, "pc-a");
  const second = await templates.createLightweight(laboratoryId, kind, "pc-b");
  for (const resource of [first, second]) {
    await ledger.add({
      resource_type: "network_object",
      resource_id: resource.id,
      laboratory_id: laboratoryId,
      revision: resource.revision,
      cleanup_method: "laboratory-cascade",
    });
  }
  return { templates, first, second };
}

export async function resolveTemplateSelection(
  request: APIRequestContext,
  templateKey: string,
  versionId?: string,
) {
  const [templatesResponse, imagesResponse] = await Promise.all([
    request.get("/api/v1/templates"),
    request.get("/api/v1/images"),
  ]);
  const templates = (await templatesResponse.json()) as Array<{
    id: string;
    template_key: string;
    display_name: string;
    runtime_kind: "qemu" | "docker";
    versions: Array<{
      id: string;
      enabled: boolean;
      image_version_id?: string;
    }>;
  }>;
  const images = ((await imagesResponse.json()) || []) as Array<{
    id: string;
    name: string;
    runtime_kind: "qemu" | "docker";
    availability: string;
  }>;
  const template = templates.find((item) => item.template_key === templateKey);
  if (!template) throw new Error(`Missing template ${templateKey}`);
  const version = versionId
    ? template.versions.find((item) => item.id === versionId && item.enabled)
    : template.versions.find((item) => item.enabled);
  if (!version) throw new Error(`No enabled version for ${templateKey}`);
  const imageId =
    version.image_version_id ||
    images.find(
      (image) =>
        image.runtime_kind === template.runtime_kind &&
        imageMatchesTemplate(image.name, templateKey) &&
        image.availability.toLowerCase() === "available",
    )?.id;
  if (!imageId) throw new Error(`No available image for ${templateKey}`);
  return {
    templateId: template.id,
    templateKey,
    displayName: template.display_name,
    runtime:
      template.runtime_kind === "qemu"
        ? ("QEMU" as const)
        : ("DOCKER" as const),
    versionId: version.id,
    imageId,
  };
}
