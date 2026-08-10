import type { OperationTask } from "@/api";

const terminalStates = new Set(["succeeded", "failed", "cancelled"]);

export async function waitForTopologyTaskFinal(
  getTask: (taskId: string) => Promise<OperationTask>,
  taskId: string,
  options: {
    attempts?: number;
    intervalMs?: number;
    delay?: (milliseconds: number) => Promise<void>;
  } = {},
): Promise<OperationTask> {
  const attempts = options.attempts ?? 40;
  const intervalMs = options.intervalMs ?? 250;
  const delay =
    options.delay ??
    ((milliseconds: number) =>
      new Promise<void>((resolve) => window.setTimeout(resolve, milliseconds)));
  let task = await getTask(taskId);
  for (
    let attempt = 1;
    attempt < attempts && !terminalStates.has(task.state);
    attempt += 1
  ) {
    await delay(intervalMs);
    task = await getTask(taskId);
  }
  return task;
}
