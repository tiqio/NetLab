<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { Command, Search } from "lucide-vue-next";
import Dialog from "@/components/ui/dialog/Dialog.vue";
import Input from "@/components/ui/input/Input.vue";
const props = defineProps<{
  modelValue: boolean;
  actions: Array<{
    id: string;
    label: string;
    keywords?: string;
    disabled?: boolean;
  }>;
}>();
const emit = defineEmits<{ "update:modelValue": [boolean]; run: [string] }>();
const query = ref("");
const open = computed({
  get: () => props.modelValue,
  set: (value) => emit("update:modelValue", value),
});
const filtered = computed(() =>
  props.actions.filter((item) =>
    `${item.label} ${item.keywords || ""}`
      .toLowerCase()
      .includes(query.value.toLowerCase()),
  ),
);
watch(open, (value) => {
  if (value) query.value = "";
});
function run(id: string) {
  emit("run", id);
  open.value = false;
}
</script>
<template>
  <Dialog
    v-model="open"
    title="命令面板"
    description="所有命令都与自动化客户端使用相同的 API 操作。"
  >
    <div class="relative">
      <Search
        :size="15"
        class="absolute left-2 top-2 text-muted-foreground"
      /><Input v-model="query" class="pl-8" autofocus placeholder="输入命令" />
    </div>
    <div class="mt-2 max-h-80 overflow-auto">
      <button
        v-for="action in filtered"
        :key="action.id"
        class="flex w-full items-center gap-2 rounded px-2 py-2 text-left text-sm hover:bg-accent disabled:opacity-40"
        :disabled="action.disabled"
        @click="run(action.id)"
      >
        <Command :size="14" class="text-primary" />{{ action.label }}
      </button>
      <p
        v-if="!filtered.length"
        class="p-4 text-center text-xs text-muted-foreground"
      >
        没有匹配的命令。
      </p>
    </div>
  </Dialog>
</template>
