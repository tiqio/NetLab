<script setup lang="ts">
import { onMounted, ref } from "vue";
import { RouterLink } from "vue-router";
import { ArrowLeft, Bot, RefreshCw, Upload } from "lucide-vue-next";
import { api, type AuditEvent, type OperationTask } from "@/api";
import { Button, Textarea } from "@/components/ui";
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
    status.value = `Import queued: ${value.task.id}`;
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
      class="flex h-12 items-center gap-3 border-b border-border bg-card px-3"
    >
      <RouterLink
        to="/"
        class="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft :size="15" /> Workspace </RouterLink
      ><Bot :size="17" class="text-primary" />
      <h1 class="font-semibold">Automation and shared control</h1>
      <Button class="ml-auto" variant="ghost" size="sm" @click="refresh">
        <RefreshCw :size="14" /> Refresh
      </Button>
    </header>
    <div class="grid min-h-0 flex-1 grid-cols-1 lg:grid-cols-[1.4fr_1fr]">
      <section class="min-h-0 border-r border-border">
        <TaskCenter :tasks="tasks" @refresh="refresh" />
      </section>
      <aside class="overflow-auto p-4 netlab-scrollbar">
        <section class="rounded border border-border bg-card p-3">
          <h2 class="font-semibold">REST and MCP parity</h2>
          <p class="mt-2 text-sm text-muted-foreground">
            The SPA, REST API, and MCP tools use the same application commands,
            durable tasks, revisions, and ordered resource events. Layout
            preferences are intentionally browser-local.
          </p>
          <ul class="mt-3 grid gap-1 text-xs text-muted-foreground">
            <li>• No account-scoped node state</li>
            <li>• No UI-only mutations</li>
            <li>• Capture responses use metadata and artifact handles</li>
            <li>• Event gaps trigger authoritative snapshot refresh</li>
          </ul>
        </section>
        <section class="mt-3 rounded border border-border bg-card p-3">
          <h2 class="font-semibold">Import redacted bundle</h2>
          <Textarea
            v-model="importText"
            rows="8"
            aria-label="Laboratory export JSON"
            class="mt-2 w-full rounded border border-input bg-background p-2 font-mono text-xs"
          /><Button class="mt-2" size="sm" @click="importBundle">
            <Upload :size="14" /> Import
          </Button>
          <p role="status" class="mt-2 text-xs text-muted-foreground">
            {{ status }}
          </p>
        </section>
        <section class="mt-3 rounded border border-border bg-card p-3">
          <h2 class="font-semibold">Audit</h2>
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
