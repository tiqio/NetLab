import type { EnvironmentSnapshot, VersionCoverage } from "./acceptanceTypes";

export function requiredVersionPlan(environment: EnvironmentSnapshot) {
  return environment.templates.flatMap((template) => {
    const available = template.versions.filter((version) => version.available);
    return available.map((version, index) => ({
      runtime: template.runtime,
      device_family: template.device_family,
      version_id: version.version_id,
      image_id: version.image_id,
      coverage_level:
        index === 0
          ? ("full-journey" as const)
          : ("lifecycle-connectivity" as const),
    }));
  });
}

export function assertCompleteVersionCoverage(
  environment: EnvironmentSnapshot,
  results: VersionCoverage[],
) {
  const required = requiredVersionPlan(environment);
  for (const item of required) {
    const result = results.find(
      (candidate) =>
        candidate.runtime === item.runtime &&
        candidate.device_family === item.device_family &&
        candidate.version_id === item.version_id,
    );
    if (
      !result ||
      result.result !== "passed" ||
      result.coverage_level !== item.coverage_level
    )
      throw new Error(
        `Incomplete version coverage: ${item.device_family}/${item.version_id}`,
      );
  }
}
