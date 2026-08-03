<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from "vue";
import { Play, Square, Trash2 } from "lucide-vue-next";
import {
  api,
  ApiError,
  type Node,
  type NodeInterface,
  type Problem,
} from "@/api";
import { Button } from "@/components/ui";
import StructuredProblem from "@/components/common/StructuredProblem.vue";
import InterfaceOperations from "./InterfaceOperations.vue";
import NodeResourcesEditor from "./NodeResourcesEditor.vue";
import NodeConfigurationPanel from "./NodeConfigurationPanel.vue";
import GuestCommandPanel from "./GuestCommandPanel.vue";
import PortMappingsPanel from "./PortMappingsPanel.vue";
import NodeCapabilityPanel from "./NodeCapabilityPanel.vue";
import ConfirmationDialog from "@/components/common/ConfirmationDialog.vue";
import { dockerRouteReadiness } from "./dockerRouteReadiness";
const props = defineProps<{ node: Node; interfaces: NodeInterface[] }>();
const emit = defineEmits<{ changed: []; deleted: [] }>();
const busy = ref(false);
const status = ref("");
const problem = ref<Problem>();
const deleteOpen = ref(false);
let lifecycleGeneration = 0;
const desired = computed(() =>
  props.node.desired_state === "running" ? "stopped" : "running",
);
const routeReadiness = computed(() => dockerRouteReadiness(props.node));
async function run(action: () => Promise<unknown>, message: string) {
  if (busy.value) return;
  busy.value = true;
  problem.value = undefined;
  try {
    const value = await action();
    const task =
      value && typeof value === "object" && "task" in value
        ? (value as { task: { id: string } }).task
        : undefined;
    status.value = `${message}${task ? ` · task ${task.id}` : ""}`;
    emit("changed");
  } catch (error) {
    problem.value =
      error instanceof ApiError
        ? error.problem
        : {
            code: "operation_failed",
            message: error instanceof Error ? error.message : String(error),
          };
  } finally {
    busy.value = false;
  }
}
function wait(milliseconds: number) {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}
async function runLifecycle() {
  if (busy.value) return;
  const target = desired.value;
  const generation = ++lifecycleGeneration;
  busy.value = true;
  problem.value = undefined;
  try {
    const value = await api.setNodeState(props.node, target);
    let task = value.task;
    const action = target === "running" ? "正在启动" : "正在停止";
    status.value = `${action} · task ${task.id} · ${task.state}`;
    emit("changed");
    while (
      generation === lifecycleGeneration &&
      ["queued", "running", "cancelling"].includes(task.state)
    ) {
      await wait(500);
      if (generation !== lifecycleGeneration) return;
      task = await api.getTask(task.id);
      status.value = `${action} · task ${task.id} · ${task.state} · ${task.progress_current}/${task.progress_total}`;
      emit("changed");
    }
    if (generation !== lifecycleGeneration) return;
    if (task.state === "succeeded") {
      status.value = target === "running" ? "运行中" : "已停止";
      emit("changed");
      return;
    }
    problem.value =
      task.error ||
      ({
        code: "node_lifecycle_failed",
        message: `运行任务结束，状态为 ${task.state}`,
        task_id: task.id,
      } as Problem);
  } catch (error) {
    problem.value =
      error instanceof ApiError
        ? error.problem
        : {
            code: "operation_failed",
            message: error instanceof Error ? error.message : String(error),
          };
  } finally {
    if (generation === lifecycleGeneration) busy.value = false;
  }
}
onBeforeUnmount(() => {
  lifecycleGeneration += 1;
});
</script>
<template>
  <div class="grid gap-3">
    <section class="panel-section">
      <h3>运行控制</h3>
      <div class="flex gap-2">
        <Button size="sm" :disabled="busy" @click="runLifecycle">
          <Play v-if="desired === 'running'" :size="14" /><Square
            v-else
            :size="14"
          />{{ desired === "running" ? "启动" : "停止" }} </Button
        ><Button
          size="sm"
          variant="destructive"
          :disabled="busy"
          @click="deleteOpen = true"
        >
          <Trash2 :size="14" /> 删除
        </Button>
      </div>
      <p role="status" class="mt-2 text-xs text-muted-foreground">
        {{ status }}
      </p>
      <StructuredProblem v-if="problem" class="mt-2" :problem="problem" />
    </section>
    <section
      v-if="routeReadiness.state !== 'none'"
      class="panel-section"
      data-testid="docker-route-readiness"
    >
      <div class="flex items-center justify-between gap-2">
        <h3>Docker 路由</h3>
        <span
          class="rounded-full border px-2 py-0.5 text-[11px]"
          :class="
            routeReadiness.state === 'failed'
              ? 'border-destructive/50 text-destructive'
              : routeReadiness.state === 'applied'
                ? 'border-emerald-500/40 text-emerald-300'
                : 'border-amber-500/40 text-amber-300'
          "
        >
          {{ routeReadiness.label }}
        </span>
      </div>
      <p class="mt-1 text-xs text-muted-foreground">
        {{ routeReadiness.routes.length }}
        条声明；启动和恢复时会在容器网络命名空间内精确收敛。
      </p>
      <ul class="mt-2 grid gap-1 text-xs">
        <li
          v-for="route in routeReadiness.routes"
          :key="`${route.interfaceName}:${route.destination}`"
        >
          <code>{{ route.interfaceName }}</code> · {{ route.destination }}
          <span v-if="route.gateway"> via {{ route.gateway }}</span>
          <span v-if="route.metric !== undefined">
            · metric {{ route.metric }}</span
          >
        </li>
      </ul>
    </section>
    <ConfirmationDialog
      v-model="deleteOpen"
      title="删除节点"
      :resource="`${node.name} · ${node.id}`"
      description="删除节点及其拥有的运行资源。"
      :impact="
        node.observed_state === 'running'
          ? '正在运行的虚拟机或容器会先停止；相连线路、终端和抓包会同时清理。'
          : '相连线路、接口、端口映射和运行资源会同时清理。'
      "
      confirm-label="确认删除"
      @confirm="
        deleteOpen = false;
        run(() => api.deleteNode(node), '已提交删除').then(() =>
          $emit('deleted'),
        );
      "
    />
    <PortMappingsPanel :node-id="node.id" />
    <NodeResourcesEditor :node="node" @changed="$emit('changed')" />
    <NodeConfigurationPanel
      :node="node"
      :interfaces="interfaces"
      @changed="$emit('changed')"
    />
    <InterfaceOperations
      :node="node"
      :interfaces="interfaces"
      @changed="$emit('changed')"
    />
    <GuestCommandPanel :node-id="node.id" />
    <NodeCapabilityPanel :node-id="node.id" />
  </div>
</template>
<style scoped>
.panel-section {
  border-bottom: 1px solid var(--border);
  padding: 1rem;
}
.panel-section h3 {
  margin-bottom: 0.65rem;
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--muted-foreground);
}
</style>
