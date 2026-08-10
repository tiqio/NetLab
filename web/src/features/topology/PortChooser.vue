<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import type { NodeInterface } from "@/api";
import type { UnifiedConnectionEndpoint } from "./topologyEndpointCompatibility";
import { Button, Dialog } from "@/components/ui";

const open = defineModel<boolean>({ required: true });
const props = defineProps<{
  title?: string;
  description?: string;
  mode?: "source" | "target" | "capture" | "reconnect";
  interfaces?: NodeInterface[];
  endpoints?: UnifiedConnectionEndpoint[];
}>();
const emit = defineEmits<{
  choose: [NodeInterface | UnifiedConnectionEndpoint];
  cancel: [];
}>();
const listbox = ref<HTMLDivElement>();
let returnFocus: HTMLElement | null = null;

watch(
  open,
  async (value) => {
    if (!value) {
      await nextTick();
      returnFocus?.focus();
      returnFocus = null;
      return;
    }
    returnFocus = document.activeElement as HTMLElement | null;
    await nextTick();
    listbox.value?.querySelector<HTMLButtonElement>('[role="option"]')?.focus();
  },
  { immediate: true },
);

const choices = computed(() => props.endpoints || props.interfaces || []);
const dialogTitle = computed(() => {
  if (props.title) return props.title;
  if (props.mode === "source") return "选择源端点";
  if (props.mode === "target") return "选择目标端点";
  return "选择接口";
});
const dialogDescription = computed(() => {
  if (props.description) return props.description;
  if (props.mode === "source")
    return "选择一个空闲源端点；后续目标、配置和提交与拖拽连接完全一致。";
  if (props.mode === "target") return "仅显示与当前源兼容且可用的目标端点。";
  return "请选择一个可用接口，然后确认连接。";
});

function choose(value: NodeInterface | UnifiedConnectionEndpoint) {
  emit("choose", value);
  open.value = false;
}

function cancel() {
  emit("cancel");
  open.value = false;
}

function choiceName(value: NodeInterface | UnifiedConnectionEndpoint) {
  return "displayName" in value ? value.displayName : value.name;
}

function choiceDetail(value: NodeInterface | UnifiedConnectionEndpoint) {
  if ("displayName" in value)
    return value.kind === "network_object_access"
      ? "逻辑接入"
      : value.portName || value.kind;
  return value.driver;
}
</script>

<template>
  <Dialog v-model="open" :title="dialogTitle" :description="dialogDescription">
    <div ref="listbox" class="grid gap-2" role="listbox" aria-label="可用接口">
      <Button
        v-for="item in choices"
        :key="
          'displayName' in item
            ? `${item.kind}:${item.resourceId}:${item.portId || item.portName || ''}`
            : item.id
        "
        variant="secondary"
        class="justify-between"
        role="option"
        :aria-label="`使用接口 ${choiceName(item)}`"
        @click="choose(item)"
      >
        <span>{{ choiceName(item) }}</span>
        <span class="text-xs text-muted-foreground">{{
          choiceDetail(item)
        }}</span>
      </Button>
    </div>
    <template #footer>
      <Button variant="ghost" @click="cancel">取消</Button>
    </template>
  </Dialog>
</template>
