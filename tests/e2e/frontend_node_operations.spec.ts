import { expect, test } from "./fixtures/acceptanceFixture";
import { waitForCondition, waitForTask } from "./fixtures/waiters";

test("durable automation task is visible in the task center", async ({
  page,
  automation,
  ledger,
}) => {
  const created = await automation.post("/api/v1/labs", {
    data: {
      name: `task-source-${Date.now()}`,
      recovery_policy: "remain_stopped",
    },
    headers: { "Idempotency-Key": crypto.randomUUID() },
  });
  expect(created.ok()).toBeTruthy();
  const laboratory = await created.json();
  await ledger.add({
    resource_type: "laboratory",
    resource_id: laboratory.id,
    revision: laboratory.revision,
    cleanup_method: "laboratory-delete",
  });
  const duplicateName = `task-copy-${Date.now()}`;
  const duplicated = await automation.post(
    `/api/v1/labs/${laboratory.id}/duplicate`,
    {
      data: { name: duplicateName },
      headers: { "Idempotency-Key": crypto.randomUUID() },
    },
  );
  expect(duplicated.ok()).toBeTruthy();
  const envelope = await duplicated.json();
  const outcome = await waitForTask(automation, envelope.task.id);
  expect(outcome.terminal_state).toBe("succeeded");
  const duplicate = await waitForCondition(
    async () => {
      const response = await automation.get("/api/v1/labs");
      expect(response.ok()).toBeTruthy();
      const laboratories = (await response.json()) as Array<{
        id: string;
        name: string;
        revision: number;
      }>;
      return laboratories.find((value) => value.name === duplicateName);
    },
    Boolean,
    "duplicated laboratory",
  );
  await ledger.add({
    resource_type: "laboratory",
    resource_id: duplicate!.id,
    revision: duplicate!.revision,
    cleanup_method: "laboratory-delete",
  });

  await page.goto("/");
  await page.getByRole("combobox", { name: "实验室范围" }).selectOption("all");
  const taskCard = page
    .locator("article")
    .filter({ hasText: envelope.task.id });
  await expect(taskCard).toBeVisible();
  await expect(taskCard).toContainText(envelope.task.kind);
});
