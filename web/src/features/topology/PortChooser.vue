<script setup lang="ts">
import { nextTick, ref, watch } from "vue";
import type { NodeInterface } from "@/api";
import { Button, Dialog } from "@/components/ui";

const open = defineModel<boolean>({ required: true });
defineProps<{
  title?: string;
  description?: string;
  interfaces: NodeInterface[];
}>();
const emit = defineEmits<{
  choose: [NodeInterface];
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

function choose(value: NodeInterface) {
  emit("choose", value);
  open.value = false;
}

function cancel() {
  emit("cancel");
  open.value = false;
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
        v-for="item in interfaces"
        :key="item.id"
        variant="secondary"
        class="justify-between"
        role="option"
        :aria-label="`使用接口 ${item.name}`"
        @click="choose(item)"
      >
        <span>{{ item.name }}</span>
        <span class="text-xs text-muted-foreground">{{ item.driver }}</span>
      </Button>
    </div>
    <template #footer>
      <Button variant="ghost" @click="cancel">取消</Button>
    </template>
  </Dialog>
</template>
