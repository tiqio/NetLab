<script setup lang="ts">
import { onMounted, ref } from "vue";
import { RouterLink } from "vue-router";
import { ArrowLeft, Bot, RefreshCw, Upload } from "lucide-vue-next";
import { api, type AuditEvent, type OperationTask } from "@/api";
import { Button, Textarea } from "@/components/ui";
import { ThemeSwitcher } from "@/components/appearance";
import TaskCenter from "@/features/tasks/TaskCenter.vue";
const tasks = ref<OperationTask[]>([]);
const audit = ref<AuditEvent[]>([]);
const importText = ref("");
const status = ref("");
async function refresh() {
  [tasks.value, audit.value] = await Promise.all([
    api.listTasks(),
    api.listAuditEvents(),
  ]);
}
async function importBundle() {
  try {
    const value = await api.importLab(JSON.parse(importText.value));
    status.value = `导入任务已进入队列：${value.task.id}`;
    await refresh();
  } catch (error) {
    status.value = error instanceof Error ? error.message : String(error);
  }
}
onMounted(refresh);
</script>
<template>
  <main class="flex h-full min-h-0 flex-col bg-background">
    <header
      class="flex min-h-12 flex-wrap items-center gap-3 border-b border-border bg-card px-3 py-2"
    >
      <RouterLink
        to="/"
        class="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft :size="15" /> 工作区 </RouterLink
      ><Bot :size="17" class="text-primary" />
      <h1 class="font-semibold">自动化与共享控制</h1>
      <div
        class="ml-auto flex min-w-0 flex-wrap items-center justify-end gap-2"
      >
        <ThemeSwitcher />
        <Button variant="ghost" size="sm" @click="refresh">
          <RefreshCw :size="14" /> 刷新
        </Button>
      </div>
    </header>
    <div class="grid min-h-0 flex-1 grid-cols-1 lg:grid-cols-[1.4fr_1fr]">
      <section class="min-h-0 border-r border-border">
        <TaskCenter :tasks="tasks" @refresh="refresh" />
      </section>
      <aside class="min-w-0 overflow-auto p-4 netlab-scrollbar">
        <section class="rounded border border-border bg-card p-3">
          <h2 class="font-semibold">REST 与 MCP 能力一致性</h2>
          <p class="mt-2 text-sm text-muted-foreground">
            SPA、REST API 与 MCP
            工具共用相同的应用命令、持久化任务、修订版本和有序资源事件。布局偏好仅保存在当前浏览器中。
          </p>
          <ul class="mt-3 grid gap-1 text-xs text-muted-foreground">
            <li>• 节点状态不按账户隔离</li>
            <li>• 不存在仅界面可用的变更操作</li>
            <li>• 抓包响应使用元数据与产物句柄</li>
            <li>• 事件缺口会触发权威快照刷新</li>
          </ul>
        </section>
        <section class="mt-3 rounded border border-border bg-card p-3">
          <h2 class="font-semibold">导入脱敏数据包</h2>
          <Textarea
            v-model="importText"
            rows="8"
            aria-label="实验室导出 JSON"
            class="mt-2 w-full rounded border border-input bg-background p-2 font-mono text-xs"
          /><Button class="mt-2" size="sm" @click="importBundle">
            <Upload :size="14" /> 导入
          </Button>
          <p role="status" class="mt-2 text-xs text-muted-foreground">
            {{ status }}
          </p>
        </section>
        <section class="mt-3 rounded border border-border bg-card p-3">
          <h2 class="font-semibold">审计</h2>
          <ul class="mt-2 grid gap-2">
            <li
              v-for="event in audit"
              :key="event.id"
              class="border-b border-border pb-2 text-xs"
            >
              <div>{{ event.action }} · {{ event.outcome }}</div>
              <div class="text-muted-foreground">
                {{ event.occurred_at }} · {{ event.resource_type }}/{{
                  event.resource_id
                }}
              </div>
            </li>
          </ul>
        </section>
      </aside>
    </div>
  </main>
</template>
