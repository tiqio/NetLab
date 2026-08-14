const GENERATED_ACCESS_PORT = /^access-[0-9a-f]{8}$/i;

export function networkAttachmentPortLabel(portName?: string) {
  const normalized = portName?.trim() || "";
  return !normalized || GENERATED_ACCESS_PORT.test(normalized)
    ? "接入口"
    : normalized;
}
