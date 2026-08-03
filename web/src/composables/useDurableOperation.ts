import { computed, ref } from "vue";
import type { OperationTask, Problem, TaskEnvelope } from "@/api/generated";
import { ApiError } from "@/api";
import { randomUUID } from "@/lib/uuid";

export function useDurableOperation() {
  const pending = ref(
    new Map<string, { idempotencyKey: string; task?: OperationTask }>(),
  );
  const problem = ref<Problem>();
  const busy = computed(() => pending.value.size > 0);

  async function run<T extends TaskEnvelope | void>(
    key: string,
    operation: (idempotencyKey: string) => Promise<T>,
  ) {
    if (pending.value.has(key)) return undefined;
    const idempotencyKey = randomUUID();
    pending.value.set(key, { idempotencyKey });
    problem.value = undefined;
    try {
      const result = await operation(idempotencyKey);
      if (result && "task" in result)
        pending.value.set(key, { idempotencyKey, task: result.task });
      return result;
    } catch (error) {
      problem.value =
        error instanceof ApiError
          ? error.problem
          : {
              code: "client_error",
              message: error instanceof Error ? error.message : String(error),
            };
      throw error;
    } finally {
      pending.value.delete(key);
    }
  }
  return {
    pending,
    problem,
    busy,
    run,
    isPending: (key: string) => pending.value.has(key),
  };
}
