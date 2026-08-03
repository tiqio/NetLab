export function injectControlledFailure(point: string) {
  if (process.env.NETLAB_ACCEPTANCE_FAILURE_INJECTION === point)
    throw new Error(`Controlled acceptance failure: ${point}`);
}
