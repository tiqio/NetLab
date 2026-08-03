export function timeoutScale() {
  const raw = Number(process.env.NETLAB_ACCEPTANCE_TIMEOUT_SCALE || "1");
  if (!Number.isFinite(raw) || raw <= 0) {
    throw new Error(
      "NETLAB_ACCEPTANCE_TIMEOUT_SCALE must be a positive number",
    );
  }
  return raw;
}

export function scaledTimeout(milliseconds: number) {
  return Math.ceil(milliseconds * timeoutScale());
}
