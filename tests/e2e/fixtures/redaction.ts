const sensitiveKeys =
  /^(authorization|cookie|set-cookie|password|passwd|secret|token|bootstrap|console_output|guest_output|packet_payload|capture_bytes|image_bytes)$/i;
const credentialValue =
  /(authorization:\s*\S+|password\s*[=:]\s*\S+|passwd\s*[=:]\s*\S+|bootstrap(?:_secret)?\s*[=:]\s*\S+|-----BEGIN [A-Z ]+PRIVATE KEY-----)/i;
const prohibitedArtifact = /\.(pcap|pcapng|qcow2|raw|iso)$/i;

export function sanitizeEvidence<T>(value: T): T {
  if (Array.isArray(value)) {
    return value.map((item) => sanitizeEvidence(item)) as T;
  }
  if (value && typeof value === "object") {
    const output: Record<string, unknown> = {};
    for (const [key, item] of Object.entries(
      value as Record<string, unknown>,
    )) {
      if (sensitiveKeys.test(key)) continue;
      output[key] = sanitizeEvidence(item);
    }
    return output as T;
  }
  if (typeof value === "string") {
    if (credentialValue.test(value)) return "[REDACTED]" as T;
    if (prohibitedArtifact.test(value)) return "[PROHIBITED_ARTIFACT]" as T;
  }
  return value;
}

export function assertNoSensitiveEvidence(value: unknown): void {
  const violations: string[] = [];
  const visit = (item: unknown, path: string) => {
    if (Array.isArray(item)) {
      item.forEach((entry, index) => visit(entry, `${path}[${index}]`));
      return;
    }
    if (item && typeof item === "object") {
      for (const [key, entry] of Object.entries(
        item as Record<string, unknown>,
      )) {
        if (sensitiveKeys.test(key)) violations.push(`${path}.${key}`);
        visit(entry, `${path}.${key}`);
      }
      return;
    }
    if (
      typeof item === "string" &&
      (credentialValue.test(item) || prohibitedArtifact.test(item))
    ) {
      violations.push(path);
    }
  };
  visit(value, "$");
  if (violations.length) {
    throw new Error(
      `Sensitive acceptance evidence at ${violations.join(", ")}`,
    );
  }
}
