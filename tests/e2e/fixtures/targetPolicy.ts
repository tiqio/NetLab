import type { EnvironmentSnapshot } from "./acceptanceTypes";

export type TargetBaselineMode = "clean" | "preserve";

export function validateTargetAcceptance(
  environment: EnvironmentSnapshot,
  options: {
    profile: string | undefined;
    baselineMode: TargetBaselineMode;
    expectedHost?: string;
  },
) {
  if (options.profile !== "target-host")
    throw new Error("Target acceptance must use the target-host profile");
  if (environment.target_kind !== "remote-privileged")
    throw new Error("Target acceptance must be classified as remote-privileged");
  const host = new URL(environment.base_url).hostname;
  if (options.expectedHost && host !== options.expectedHost)
    throw new Error(
      `Target acceptance host ${host} does not match ${options.expectedHost}`,
    );
  if (!environment.baseline_clean && options.baselineMode !== "preserve")
    throw new Error(
      "Target-host acceptance has an existing baseline; set NETLAB_ACCEPTANCE_BASELINE_MODE=preserve",
    );
  const release = environment.release;
  const digest = /^sha256:[0-9a-f]{64}$/;
  if (
    !release ||
    [release.version, release.candidate_id].some((value) =>
      /^(?:dev|unknown|operator-supplied)/i.test(value || ""),
    ) ||
    !digest.test(release.binary_digest || "") ||
    !digest.test(release.contract_digest || "") ||
    !release.built_at
  ) {
    throw new Error("Target acceptance requires an immutable release identity");
  }
}
