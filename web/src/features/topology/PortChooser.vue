<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import type { NodeInterface } from "@/api";
import type { UnifiedConnectionEndpoint } from "./topologyEndpointCompatibility";
import { Button, Dialog } from "@/components/ui";

const open = defineModel<boolean>({ required: true });
const props = defineProps<{
  title?: string;
  description?: string;
  interfaces?: NodeInterface[];
  endpoints?: UnifiedConnectionEndpoint[];
}>();
const emit = defineEmits<{
  choose: [NodeInterface | UnifiedConnectionEndpoint];
  cancel: [];
}>();
const listbox = ref<HTMLDivElement>();

watch(
  open,
  async (value) => {
    if (!value) return;
    await nextTick();
    listbox.value?.querySelector<HTMLButtonElement>('[role="option"]')?.focus();
  },
  { immediate: true },
);

const choices = computed(() => props.endpoints || props.interfaces || []);

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
  <Dialog
    v-model="open"
    :title="title || '选择接口'"
    :description="description || '请选择一个可用接口，然后确认连接。'"
  >
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
