import type { APIRequestContext, Locator, Page } from "@playwright/test";
import { expect } from "@playwright/test";
import { scaledTimeout } from "./timeoutScale";

export async function waitForVisibleFeedback(locator: Locator, timeout = 500) {
  const started = Date.now();
  await expect(locator).toBeVisible({ timeout: scaledTimeout(timeout) });
  return Date.now() - started;
}

export async function waitForTask(
  request: APIRequestContext,
  taskId: string,
  timeout = 120_000,
) {
  const states: string[] = [];
  const started = Date.now();
  const boundedTimeout = scaledTimeout(timeout);
  while (Date.now() - started < boundedTimeout) {
    const response = await request.get(`/api/v1/tasks/${taskId}`);
    if (!response.ok()) {
      throw new Error(`Task ${taskId} returned ${response.status()}`);
    }
    const task = (await response.json()) as {
      id: string;
      kind: string;
      state: string;
      error?: { code?: string };
    };
    if (states.at(-1) !== task.state) states.push(task.state);
    if (["succeeded", "failed", "cancelled"].includes(task.state)) {
      return {
        task_id: task.id,
        kind: task.kind,
        states,
        terminal_state: task.state as "succeeded" | "failed" | "cancelled",
        duration_ms: Date.now() - started,
        cleanup_state: task.error?.code || "terminal",
        problem_code: task.error?.code,
      };
    }
    await new Promise((resolve) => setTimeout(resolve, 200));
  }
  throw new Error(
    `Task ${taskId} did not reach terminal state within ${boundedTimeout}ms`,
  );
}

export async function waitForCondition<T>(
  read: () => Promise<T>,
  accept: (value: T) => boolean,
  description: string,
  timeout = 10_000,
  interval = 200,
): Promise<T> {
  const started = Date.now();
  let last: T | undefined;
  const boundedTimeout = scaledTimeout(timeout);
  while (Date.now() - started < boundedTimeout) {
    last = await read();
    if (accept(last)) return last;
    await new Promise((resolve) => setTimeout(resolve, interval));
  }
  throw new Error(
    `${description} did not converge within ${boundedTimeout}ms; last=${JSON.stringify(last)}`,
  );
}

export async function waitForPageRefresh(page: Page) {
  await Promise.all([page.waitForLoadState("domcontentloaded"), page.reload()]);
}
