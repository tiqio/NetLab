export function requiresCompleteAcceptanceCoverage(
  targetKind: string,
  scope: string | undefined,
) {
  return targetKind === "remote-privileged" && (scope ?? "full") === "full";
}
