<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { KeyRound, ShieldCheck, Trash2 } from "lucide-vue-next";
import {
  api,
  ApiError,
  type Node,
  type NodeCredentialMetadata,
  type OperationTask,
  type Problem,
} from "@/api";
import StructuredProblem from "@/components/common/StructuredProblem.vue";
import { Button, FormField, Input } from "@/components/ui";

const props = defineProps<{ node: Node }>();
const emit = defineEmits<{ changed: [] }>();
const metadata = ref<NodeCredentialMetadata>();
const username = ref("admin");
const currentPassword = ref("");
const newPassword = ref("");
const busy = ref(false);
const status = ref("");
const problem = ref<Problem>();

const stateLabel = computed(() => {
  const labels: Record<string, string> = {
    credential_missing: "未配置",
    credential_store_locked: "凭证库已锁定",
    pending_verification: "待验证",
    verification_failed: "验证失败",
    authenticated: "已验证",
  };
  return (
    labels[metadata.value?.state || "credential_missing"] ||
    metadata.value?.state
  );
});

async function load() {
  problem.value = undefined;
  try {
    metadata.value = await api.getNodeCredentials(props.node.id);
  } catch (error) {
    problem.value = asProblem(error, "credential_status_failed");
  }
}

async function save() {
  if (busy.value || !username.value.trim()) return;
  busy.value = true;
  problem.value = undefined;
  try {
    metadata.value = await api.setFortiGateCredential(props.node.id, {
      username: username.value.trim(),
      current_password: currentPassword.value,
      new_password: newPassword.value || undefined,
    });
    status.value = "凭证已加密保存，明文输入已清空。";
    emit("changed");
  } catch (error) {
    problem.value = asProblem(error, "credential_save_failed");
  } finally {
    currentPassword.value = "";
    newPassword.value = "";
    busy.value = false;
  }
}

async function submitTask(
  operation: () => Promise<{ task: OperationTask }>,
  label: string,
) {
  if (busy.value) return;
  busy.value = true;
  problem.value = undefined;
  try {
    let task = (await operation()).task;
    status.value = `${label} · ${task.state}`;
    while (["queued", "running", "cancelling"].includes(task.state)) {
      await new Promise((resolve) => window.setTimeout(resolve, 500));
      task = await api.getTask(task.id);
      status.value = `${label} · ${task.state} · ${task.progress_current}/${task.progress_total}`;
    }
    if (task.state !== "succeeded") {
      problem.value = task.error || {
        code: "credential_task_failed",
        message: `任务结束，状态为 ${task.state}`,
      };
    } else {
      status.value = `${label}完成。`;
      await load();
      emit("changed");
    }
  } catch (error) {
    problem.value = asProblem(error, "credential_task_failed");
  } finally {
    busy.value = false;
  }
}

async function remove() {
  if (busy.value) return;
  busy.value = true;
  problem.value = undefined;
  try {
    await api.deleteFortiGateCredential(props.node.id);
    currentPassword.value = "";
    newPassword.value = "";
    status.value = "FortiGate 凭证已删除。";
    await load();
    emit("changed");
  } catch (error) {
    problem.value = asProblem(error, "credential_delete_failed");
  } finally {
    busy.value = false;
  }
}

function asProblem(error: unknown, code: string): Problem {
  return error instanceof ApiError
    ? error.problem
    : { code, message: error instanceof Error ? error.message : String(error) };
}

watch(() => props.node.id, load, { immediate: true });
</script>

<template>
  <section class="panel-section grid gap-3" aria-label="FortiGate 凭证">
    <div class="flex items-center justify-between gap-2">
      <div>
        <h3>FortiGate 凭证</h3>
        <p class="text-xs text-muted-foreground">
          密码仅发送到加密凭证库，不会在检查器、任务或日志中回显。
        </p>
      </div>
      <span class="rounded-full border border-border px-2 py-0.5 text-[11px]">
        {{ stateLabel }}
      </span>
    </div>
    <FormField label="管理员用户名">
      <Input
        v-model="username"
        aria-label="FortiGate 管理员用户名"
        autocomplete="username"
        :disabled="busy"
      />
    </FormField>
    <FormField label="当前或出厂密码">
      <Input
        v-model="currentPassword"
        aria-label="FortiGate 当前密码"
        type="password"
        autocomplete="new-password"
        :disabled="busy"
      />
    </FormField>
    <FormField label="首次登录新密码（可选）">
      <Input
        v-model="newPassword"
        aria-label="FortiGate 新密码"
        type="password"
        autocomplete="new-password"
        :disabled="busy"
      />
    </FormField>
    <p v-if="metadata?.staged" class="text-xs text-[color:var(--warning)]">
      已保存待验证的新密码；首次改密成功前不会覆盖当前凭证。
    </p>
    <div class="flex flex-wrap gap-2">
      <Button size="sm" :disabled="busy || !username.trim()" @click="save">
        <KeyRound :size="14" /> 保存凭证
      </Button>
      <Button
        size="sm"
        variant="secondary"
        :disabled="busy || !metadata?.configured"
        @click="
          submitTask(() => api.verifyFortiGateCredential(node.id), '验证登录')
        "
      >
        <ShieldCheck :size="14" /> 验证登录
      </Button>
      <Button
        size="sm"
        variant="secondary"
        :disabled="busy || !metadata?.staged"
        @click="submitTask(() => api.bootstrapFortiGate(node.id), '首次改密')"
      >
        首次改密
      </Button>
      <Button
        size="sm"
        variant="destructive"
        :disabled="busy || !metadata?.configured"
        @click="remove"
      >
        <Trash2 :size="14" /> 删除凭证
      </Button>
    </div>
    <p role="status" class="text-xs text-muted-foreground">{{ status }}</p>
    <StructuredProblem v-if="problem" :problem="problem" />
  </section>
</template>

<style scoped>
.panel-section {
  border-bottom: 1px solid var(--border);
  padding: 1rem;
}
.panel-section h3 {
  margin-bottom: 0.35rem;
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--muted-foreground);
}
</style>
