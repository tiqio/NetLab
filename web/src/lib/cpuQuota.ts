export const CPU_QUOTA_PERIOD_MICROS = 100_000;

export function cpuQuotaMicrosToCores(value: number | string | undefined) {
  const micros = Number(value || 0);
  if (!Number.isFinite(micros) || micros <= 0) return 0;
  return Number((micros / CPU_QUOTA_PERIOD_MICROS).toFixed(3));
}

export function cpuQuotaCoresToMicros(value: number | string | undefined) {
  const cores = Number(value || 0);
  if (!Number.isFinite(cores) || cores <= 0) return 0;
  return Math.round(cores * CPU_QUOTA_PERIOD_MICROS);
}
