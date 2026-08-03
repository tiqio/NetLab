<script setup lang="ts">
import { computed, ref } from "vue";
import {
  Ban,
  ExternalLink,
  Filter,
  RefreshCw,
  RotateCcw,
} from "lucide-vue-next";
import { api, type OperationTask } from "@/api";
import { Button, Input, Select } from "@/components/ui";
import StatusBadge from "@/components/common/StatusBadge.vue";
import StructuredProblem from "@/components/common/StructuredProblem.vue";
import TaskProgressChart from "@/components/charts/TaskProgressChart.vue";

const props = defineProps<{
  tasks: OperationTask[];
  laboratoryId?: string;
  resourceIds?: string[];
}>();
const emit = defineEmits<{ refresh: []; navigate: [string, string] }>();
const state = ref("all");
const kind = ref("all");
const resourceType = ref("all");
const scope = ref("laboratory");
const timeRange = ref("all");
const query = ref("");
const cancelling = ref("");
const replaying = ref("");
const refreshed = ref<Record<string, OperationTask>>({});
const kinds = computed(() =>
  [...new Set(props.tasks.map((task) => task.kind))].sort(),
);
const resourceTypes = computed(() =>
  [...new Set(props.tasks.map((task) => task.resource_type))].sort(),
);
const inActiveLaboratory = (task: OperationTask) =>
  task.resource_type === "laboratory"
    ? task.resource_id === props.laboratoryId
    : !props.resourceIds?.length ||
      props.resourceIds.includes(task.resource_id);
const newerThan = (task: OperationTask) => {
  if (timeRange.value === "all") return true;
  const age = Date.now() - new Date(task.created_at).getTime();
  return age <= Number(timeRange.value) * 60_000;
};
const filtered = computed(() =>
  props.tasks
    .map((task) => refreshed.value[task.id] || task)
    .filter(
      (task) =>
        (state.value === "all" || task.state === state.value) &&
        (kind.value === "all" || task.kind === kind.value) &&
        (resourceType.value === "all" ||
          task.resource_type === resourceType.value) &&
        (scope.value === "all" || inActiveLaboratory(task)) &&
        newerThan(task) &&
        `${task.kind} ${task.resource_type} ${task.resource_id}`
          .toLowerCase()
          .includes(query.value.toLowerCase()),
    ),
);

async function cancel(task: OperationTask) {
  cancelling.value = task.id;
  try {
    await api.cancelTask(task.id);
    emit("refresh");
  } finally {
    cancelling.value = "";
  }
}

async function replay(task: OperationTask) {
  replaying.value = task.id;
  try {
    refreshed.value[task.id] = await api.getTask(task.id);
  } finally {
    replaying.value = "";
  }
}

const percent = (task: OperationTask) =>
  Math.round(
    task.progress_total
      ? (task.progress_current / task.progress_total) * 100
      : 0,
  );
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <header
      class="flex flex-wrap items-center gap-2 border-b border-border p-2"
    >
      <Filter :size="14" class="text-muted-foreground" />
      <Input
        v-model="query"
        class="max-w-64"
        placeholder="Filter operation or resource"
        aria-label="Filter tasks"
      />
      <Select v-model="state" class="max-w-36" aria-label="Task state">
        <option value="all">All states</option>
        <option
          v-for="value in [
            'queued',
            'running',
            'cancelling',
            'succeeded',
            'failed',
            'cancelled',
          ]"
          :key="value"
          :value="value"
          :label="value"
        />
      </Select>
      <Select v-model="kind" class="max-w-44" aria-label="Operation kind">
        <option value="all">All operations</option>
        <option v-for="value in kinds" :key="value" :value="value">
          {{ value }}
        </option>
      </Select>
      <Select
        v-model="resourceType"
        class="max-w-40"
        aria-label="Resource type"
      >
        <option value="all">All resources</option>
        <option v-for="value in resourceTypes" :key="value" :value="value">
          {{ value }}
        </option>
      </Select>
      <Select v-model="scope" class="max-w-40" aria-label="Laboratory scope">
        <option value="laboratory">Active laboratory</option>
        <option value="all">All laboratories</option>
      </Select>
      <Select v-model="timeRange" class="max-w-32" aria-label="Task time range">
        <option value="all">Any time</option>
        <option value="15">15 minutes</option>
        <option value="60">1 hour</option>
        <option value="1440">24 hours</option>
      </Select>
      <Button
        class="ml-auto"
        variant="ghost"
        size="icon"
        aria-label="Refresh tasks"
        @click="$emit('refresh')"
      >
        <RefreshCw :size="14" />
      </Button>
    </header>
    <div class="flex-1 overflow-auto p-2 netlab-scrollbar">
      <div
        v-if="!filtered.length"
        class="grid h-full place-items-center text-xs text-muted-foreground"
      >
        No tasks match the current filter.
      </div>
      <article
        v-for="task in filtered"
        :key="task.id"
        class="mb-2 grid gap-2 rounded-md border border-border bg-background/40 p-2"
      >
        <div class="flex items-start gap-2">
          <StatusBadge :state="task.state" />
          <div class="min-w-0 flex-1">
            <strong class="block truncate text-xs">{{ task.kind }}</strong>
            <Button
              variant="ghost"
              size="sm"
              class="h-auto max-w-full justify-start truncate px-0 text-[10px] text-muted-foreground hover:text-primary"
              :title="
                inActiveLaboratory(task)
                  ? 'Open resource'
                  : 'Resource is deleted or outside the active laboratory'
              "
              :disabled="!inActiveLaboratory(task)"
              @click="$emit('navigate', task.resource_type, task.resource_id)"
            >
              {{ task.resource_type }} · {{ task.resource_id }}
              <ExternalLink :size="10" class="inline" />
            </Button>
          </div>
          <span class="font-mono text-[10px] text-muted-foreground">
            {{ task.id }}
          </span>
        </div>
        <div>
          <div
            class="mb-1 flex justify-between text-[10px] text-muted-foreground"
          >
            <span>{{ task.progress_current }}/{{ task.progress_total }}</span>
            <span>{{ percent(task) }}%</span>
          </div>
          <TaskProgressChart
            :current="task.progress_current"
            :total="task.progress_total"
            :state="task.state"
          />
        </div>
        <dl
          class="grid grid-cols-[auto_1fr_auto_1fr] gap-x-2 text-[10px] text-muted-foreground"
        >
          <dt>Created</dt>
          <dd>{{ task.created_at }}</dd>
          <dt>Started</dt>
          <dd>{{ task.started_at || "not started" }}</dd>
          <dt>Finished</dt>
          <dd>{{ task.finished_at || "not finished" }}</dd>
          <dt>Cancel</dt>
          <dd>
            {{
              ["queued", "running"].includes(task.state)
                ? "available"
                : "not available in terminal state"
            }}
          </dd>
        </dl>
        <div class="flex justify-end gap-1 text-[10px] text-muted-foreground">
          <Button
            variant="ghost"
            size="sm"
            :disabled="replaying === task.id"
            @click="replay(task)"
          >
            <RotateCcw :size="12" /> Re-read task
          </Button>
          <Button
            v-if="['queued', 'running'].includes(task.state)"
            variant="ghost"
            size="sm"
            :disabled="cancelling === task.id"
            @click="cancel(task)"
          >
            <Ban :size="12" /> Cancel
          </Button>
        </div>
        <details v-if="task.result" class="text-xs">
          <summary>Terminal result</summary>
          <pre class="mt-1 max-h-40 overflow-auto rounded bg-muted/40 p-2">{{
            JSON.stringify(task.result, null, 2)
          }}</pre>
        </details>
        <StructuredProblem v-if="task.error" :problem="task.error" />
      </article>
    </div>
  </div>
</template>
