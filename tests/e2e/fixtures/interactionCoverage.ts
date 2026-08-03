import type {
  AcceptanceEvidence,
  InteractionDefinition,
  Viewport,
} from "./acceptanceTypes";

function viewportKey(viewport: Viewport) {
  return `${viewport.width}x${viewport.height}`;
}

export function assertCompleteInteractionCoverage(
  inventory: InteractionDefinition[],
  evidence: Pick<AcceptanceEvidence, "interaction_results" | "viewports">,
  requireComplete = true,
) {
  const inventoryIds = new Set(inventory.map((item) => item.id));
  const unknown = evidence.interaction_results
    .map((item) => item.interaction_id)
    .filter((id) => !inventoryIds.has(id) && !id.startsWith("test."));
  if (unknown.length) {
    throw new Error(
      `Interaction evidence uses unknown ids: ${[...new Set(unknown)].join(", ")}`,
    );
  }

  if (!requireComplete) return;

  const missing: string[] = [];
  for (const definition of inventory) {
    for (const viewport of evidence.viewports) {
      for (const activation of definition.activation) {
        const present = evidence.interaction_results.some(
          (result) =>
            result.interaction_id === definition.id &&
            result.activation === activation &&
            viewportKey(result.viewport) === viewportKey(viewport),
        );
        if (!present) {
          missing.push(
            `${definition.id}/${activation}/${viewportKey(viewport)}`,
          );
        }
      }
    }
  }
  if (missing.length) {
    throw new Error(`Missing interaction coverage: ${missing.join(", ")}`);
  }
}
