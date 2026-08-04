import type { APIRequestContext } from "@playwright/test";
import type {
  CapabilityDecision,
  EnvironmentSnapshot,
  TemplateObservation,
} from "./acceptanceTypes";

const requiredProductCapabilities = [
  "qemu",
  "docker",
  "namespace",
  "telnet",
  "vnc",
  "live_capture",
  "traffic_filter",
  "qmp_hotplug",
  "qga_exec",
  "port_mapping",
  "cpu_quota",
  "mcp",
];

function imageMatchesTemplate(imageName: string, templateKey: string) {
  const normalize = (value: string) =>
    value.toLowerCase().replace(/[^a-z0-9]+/g, "");
  const family = templateKey.replace(/-(?:container|qemu)$/, "");
  return [templateKey, family].map(normalize).includes(normalize(imageName));
}

export async function discoverEnvironment(
  request: APIRequestContext,
  baseUrl: string,
  targetKind: EnvironmentSnapshot["target_kind"],
): Promise<EnvironmentSnapshot> {
  const responses = await Promise.all([
    request.get("/healthz"),
    request.get("/api/v1/capabilities"),
    request.get("/api/v1/templates"),
    request.get("/api/v1/images"),
    request.get("/api/v1/labs"),
    request.get("/api/v1/runtime-ownership"),
  ]);
  const [
    healthResponse,
    capabilityResponse,
    templateResponse,
    imageResponse,
    labsResponse,
    ownershipResponse,
  ] = responses;
  if (!healthResponse.ok()) {
    throw new Error(`Health preflight failed: ${healthResponse.status()}`);
  }
  for (const response of [
    capabilityResponse,
    templateResponse,
    imageResponse,
    labsResponse,
    ownershipResponse,
  ]) {
    if (!response.ok()) {
      throw new Error(
        `Acceptance preflight failed: ${response.url()} returned ${response.status()}`,
      );
    }
  }
  const rawCapabilities = (await capabilityResponse.json()) as {
    runtimes?: string[];
    console_modes?: string[];
    features?: string[];
    api_version?: string;
    release?: EnvironmentSnapshot["release"];
  };
  const available = new Set([
    ...(rawCapabilities.runtimes || []),
    ...(rawCapabilities.console_modes || []),
    ...(rawCapabilities.features || []),
  ]);
  const decisions: CapabilityDecision[] = requiredProductCapabilities.map(
    (name) => ({
      name,
      class: "product-supported",
      available: available.has(name),
      decision: available.has(name) ? "run" : "fail",
      reason: available.has(name)
        ? undefined
        : `Service did not declare ${name}`,
      evidence: "/api/v1/capabilities",
    }),
  );
  const failed = decisions.filter((decision) => decision.decision === "fail");
  if (targetKind === "remote-privileged" && failed.length) {
    throw new Error(
      `Missing supported capabilities: ${failed.map((item) => item.name).join(", ")}`,
    );
  }
  const templates = (await templateResponse.json()) as Array<{
    id: string;
    template_key: string;
    runtime_kind: "qemu" | "docker";
    versions: Array<{
      id: string;
      image_version_id?: string;
      enabled: boolean;
    }>;
  }>;
  const images = ((await imageResponse.json()) || []) as Array<{
    id: string;
    name: string;
    runtime_kind: "qemu" | "docker";
    availability: string;
  }>;
  const templateObservations: TemplateObservation[] = templates.map(
    (template) => {
      const matchingImage = images.find(
        (image) =>
          image.runtime_kind === template.runtime_kind &&
          image.availability.toLowerCase() === "available" &&
          imageMatchesTemplate(image.name, template.template_key),
      );
      return {
        template_id: template.id,
        device_family: template.template_key,
        runtime: template.runtime_kind,
        versions: template.versions.map((version) => ({
          version_id: version.id,
          image_id: version.image_version_id || matchingImage?.id,
          available:
            version.enabled &&
            Boolean(version.image_version_id || matchingImage?.id),
        })),
      };
    },
  );
  const rawLabs = await labsResponse.json();
  const labs = (Array.isArray(rawLabs) ? rawLabs : []) as Array<{ id: string }>;
  const rawOwnership = await ownershipResponse.json();
  const runtimeOwnership: EnvironmentSnapshot["baseline_runtime_ownership"] =
    (Array.isArray(rawOwnership) ? rawOwnership : []).map(
      (record: Record<string, unknown>) => ({
        resource_type: String(record.resource_type || ""),
        resource_id: String(record.resource_id || ""),
        object_kind: String(record.object_kind || ""),
        object_name: String(record.object_name || ""),
        cleanup_state: String(record.cleanup_state || ""),
        ownership_class: String(record.ownership_class || "foreign_observed") as
          | "managed"
          | "acceptance_owned"
          | "foreign_observed",
      }),
    );
  const capabilities = Object.fromEntries(
    decisions.map((decision) => [decision.name, decision.available]),
  );
  return {
    base_url: baseUrl,
    target_kind: targetKind,
    service_version: rawCapabilities.api_version,
    release: rawCapabilities.release,
    capabilities,
    capability_decisions: decisions,
    templates: templateObservations,
    baseline_clean:
      labs.length === 0 &&
      runtimeOwnership.every((record) => record.cleanup_state !== "active"),
    baseline_laboratory_ids: labs.map((laboratory) => laboratory.id),
    baseline_runtime_ownership: runtimeOwnership,
  };
}

export function optionalEnvironmentDecision(
  name: string,
  available: boolean,
  reason: string,
): CapabilityDecision {
  return {
    name,
    class: "environment-optional",
    available,
    decision: available ? "run" : "skip",
    reason: available ? undefined : reason,
    evidence: "acceptance runner environment",
  };
}
