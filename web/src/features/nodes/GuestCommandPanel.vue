<script setup lang="ts">
import { ref, watch } from "vue";
import { api, ApiError, type OperationTask, type Problem } from "@/api";
import { Button, FormField, Textarea } from "@/components/ui";
import StatusBadge from "@/components/common/StatusBadge.vue";
import StructuredProblem from "@/components/common/StructuredProblem.vue";

const props = defineProps<{ nodeId: string }>();
const command = ref("uname -a");
const status = ref("");
const busy = ref(false);
const problem = ref<Problem>();
const completedTask = ref<OperationTask>();
const stdout = ref("");
const stderr = ref("");

function decodeBase64(value: unknown) {
  const encoded = String(value || "");
  if (!encoded) return "";
  const binary = atob(encoded);
  const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0));
  return new TextDecoder().decode(bytes);
}

function errorProblem(error: unknown): Problem {
  return error instanceof ApiError
    ? error.problem
    : {
        code: "guest_exec_failed",
        message: error instanceof Error ? error.message : String(error),
      };
}

async function waitForTask(task: OperationTask) {
  let current = task;
  for (let attempt = 0; attempt < 240; attempt += 1) {
    if (["succeeded", "failed", "cancelled"].includes(current.state))
      return current;
    await new Promise((resolve) => setTimeout(resolve, 250));
    current = await api.getTask(current.id);
    status.value = `正在执行 · 任务 ${current.id} · ${current.progress_current}/${current.progress_total}`;
  }
  throw new Error(`任务 ${task.id} 在 60 秒内未完成`);
}

async function execute() {
  const shellCommand = command.value.trim();
  if (!shellCommand || busy.value) return;
  busy.value = true;
  problem.value = undefined;
  completedTask.value = undefined;
  stdout.value = "";
  stderr.value = "";
  try {
    const queued = await api.executeGuestCommand(props.nodeId, {
      argv: ["/bin/sh", "-lc", shellCommand],
      timeout_seconds: 30,
      output_limit: 1 << 20,
    });
    status.value = `正在执行 · 任务 ${queued.id}`;
    const task = await waitForTask(queued);
    completedTask.value = task;
    stdout.value = decodeBase64(task.result?.stdout_base64);
    stderr.value = decodeBase64(task.result?.stderr_base64);
    if (task.state !== "succeeded")
      throw task.error || new Error(`任务状态：${task.state}`);
    status.value = `执行完成 · 退出码 ${Number(task.result?.exit_code ?? 0)}`;
  } catch (error) {
    problem.value = errorProblem(error);
    status.value = "执行失败";
  } finally {
    busy.value = false;
  }
}

watch(
  () => props.nodeId,
  () => {
    status.value = "";
    problem.value = undefined;
    completedTask.value = undefined;
    stdout.value = "";
    stderr.value = "";
  },
);
</script>

<template>
  <form class="panel-section" @submit.prevent="execute">
    <div class="flex items-center justify-between gap-2">
      <h3>客户机命令</h3>
      <StatusBadge v-if="completedTask" :state="completedTask.state" />
    </div>
    <FormField
      label="Shell 命令"
      hint="通过 QEMU Guest Agent 在客户机内使用 /bin/sh -lc 执行，支持管道、重定向和命令组合。"
    >
      <Textarea
        v-model="command"
        rows="3"
        autocomplete="off"
        spellcheck="false"
        class="font-mono"
      />
    </FormField>
    <Button class="mt-2" size="sm" type="submit" :disabled="busy">
      {{ busy ? "正在执行…" : "通过 QEMU Guest Agent 执行" }}
    </Button>
    <p role="status" class="mt-2 text-xs text-muted-foreground">
      {{ status }}
    </p>
    <StructuredProblem v-if="problem" class="mt-2" :problem="problem" />
    <section
      v-if="completedTask"
      class="mt-3 grid gap-2 rounded-md border border-border bg-muted/30 p-2"
      data-testid="guest-command-result"
    >
      <div class="flex flex-wrap gap-x-4 gap-y-1 text-xs">
        <span>退出码：{{ Number(completedTask.result?.exit_code ?? 0) }}</span>
        <span
          >输出截断：{{ completedTask.result?.truncated ? "是" : "否" }}</span
        >
      </div>
      <div v-if="stdout">
        <p class="mb-1 text-xs font-medium text-muted-foreground">标准输出</p>
        <pre
          class="max-h-64 overflow-auto whitespace-pre-wrap break-all rounded bg-background p-2 text-xs text-foreground"
          >{{ stdout }}</pre>
      </div>
      <div v-if="stderr">
        <p class="mb-1 text-xs font-medium text-destructive">标准错误</p>
        <pre
          class="max-h-64 overflow-auto whitespace-pre-wrap break-all rounded bg-background p-2 text-xs text-destructive"
          >{{ stderr }}</pre>
      </div>
      <p v-if="!stdout && !stderr" class="text-xs text-muted-foreground">
        命令没有产生输出。
      </p>
    </section>
  </form>
</template>
