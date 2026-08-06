<script setup lang="ts">
import { computed, ref, watch } from "vue";
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
const visibleLimit = ref(30);
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
const visibleTasks = computed(() =>
  filtered.value.slice(0, visibleLimit.value),
);
watch([state, kind, resourceType, scope, timeRange, query], () => {
  visibleLimit.value = 30;
});

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
const taskStateLabel = (value: string) =>
  ({
    queued: "排队中",
    running: "运行中",
    cancelling: "正在取消",
    succeeded: "已成功",
    failed: "失败",
    cancelled: "已取消",
  })[value] || value;
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <header
      class="netlab-scrollbar flex max-h-[45%] shrink-0 flex-wrap items-center gap-2 overflow-y-auto border-b border-border p-2"
      data-layout-region="task-filters"
    >
      <Filter :size="14" class="text-muted-foreground" />
      <Input
        v-model="query"
        class="max-w-64"
        placeholder="筛选操作或资源"
        aria-label="筛选任务"
      />
      <Select v-model="state" class="max-w-36" aria-label="任务状态">
        <option value="all">全部状态</option>
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
        >
          {{ taskStateLabel(value) }}
        </option>
      </Select>
      <Select v-model="kind" class="max-w-44" aria-label="操作类型">
        <option value="all">全部操作</option>
        <option v-for="value in kinds" :key="value" :value="value">
          {{ value }}
        </option>
      </Select>
      <Select v-model="resourceType" class="max-w-40" aria-label="资源类型">
        <option value="all">全部资源</option>
        <option v-for="value in resourceTypes" :key="value" :value="value">
          {{ value }}
        </option>
      </Select>
      <Select v-model="scope" class="max-w-40" aria-label="实验室范围">
        <option value="laboratory">当前实验室</option>
        <option value="all">全部实验室</option>
      </Select>
      <Select v-model="timeRange" class="max-w-32" aria-label="任务时间范围">
        <option value="all">任意时间</option>
        <option value="15">15 分钟</option>
        <option value="60">1 小时</option>
        <option value="1440">24 小时</option>
      </Select>
      <Button
        class="ml-auto"
        variant="ghost"
        size="icon"
        aria-label="刷新任务"
        @click="$emit('refresh')"
      >
        <RefreshCw :size="14" />
      </Button>
    </header>
    <div
      class="min-h-0 flex-1 overflow-auto p-2 netlab-scrollbar"
      data-layout-region="task-results"
    >
      <div
        v-if="!filtered.length"
        class="grid h-full place-items-center text-xs text-muted-foreground"
      >
        没有符合当前筛选条件的任务。
      </div>
      <article
        v-for="task in visibleTasks"
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
                  ? '打开资源'
                  : '资源已删除或不在当前实验室中'
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
          <dt>创建时间</dt>
          <dd>{{ task.created_at }}</dd>
          <dt>开始时间</dt>
          <dd>{{ task.started_at || "尚未开始" }}</dd>
          <dt>完成时间</dt>
          <dd>{{ task.finished_at || "尚未完成" }}</dd>
          <dt>取消</dt>
          <dd>
            {{
              ["queued", "running"].includes(task.state)
                ? "可用"
                : "终态任务不可取消"
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
            <RotateCcw :size="12" /> 重新读取任务
          </Button>
          <Button
            v-if="['queued', 'running'].includes(task.state)"
            variant="ghost"
            size="sm"
            :disabled="cancelling === task.id"
            @click="cancel(task)"
          >
            <Ban :size="12" /> 取消
          </Button>
        </div>
        <details v-if="task.result" class="text-xs">
          <summary>终端执行结果</summary>
          <pre class="mt-1 max-h-40 overflow-auto rounded bg-muted/40 p-2">{{
            JSON.stringify(task.result, null, 2)
          }}</pre>
        </details>
        <StructuredProblem v-if="task.error" :problem="task.error" />
      </article>
      <Button
        v-if="visibleTasks.length < filtered.length"
        variant="secondary"
        class="mb-2 w-full"
        aria-label="显示更多任务"
        @click="visibleLimit += 30"
      >
        再显示 {{ Math.min(30, filtered.length - visibleTasks.length) }} 条
      </Button>
    </div>
  </div>
</template>
