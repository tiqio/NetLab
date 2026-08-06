<script setup lang="ts">
import { Cable, Eye, Route, Trash2, Unplug } from "lucide-vue-next";
import { Button, DropdownMenu } from "@/components/ui";

defineProps<{ disabled?: boolean; objectLink?: boolean; pending?: boolean }>();
const emit = defineEmits<{
  inspect: [];
  reconnect: [];
  disconnect: [];
  route: [];
  delete: [];
}>();
</script>

<template>
  <DropdownMenu>
    <template #trigger>
      <Button size="sm" variant="secondary" :disabled="disabled">
        <Cable :size="13" /> 链路操作
      </Button>
    </template>
    <div class="grid gap-1" role="menu" aria-label="链路操作">
      <Button
        variant="ghost"
        class="w-full justify-start"
        @click="emit('inspect')"
      >
        <Eye :size="13" class="shrink-0" /> <span>检查</span>
      </Button>
      <Button
        v-if="!objectLink"
        variant="ghost"
        class="w-full justify-start"
        @click="emit('reconnect')"
      >
        <Cable :size="13" class="shrink-0" /> <span>重新连接端点</span>
      </Button>
      <Button
        v-if="!objectLink"
        variant="ghost"
        class="w-full justify-start"
        @click="emit('route')"
      >
        <Route :size="13" class="shrink-0" /> <span>编辑本地路由</span>
      </Button>
      <Button
        v-if="!objectLink"
        variant="destructive"
        class="w-full justify-start"
        @click="emit('disconnect')"
      >
        <Unplug :size="13" class="shrink-0" /> <span>断开连接</span>
      </Button>
      <Button
        v-else
        variant="destructive"
        class="w-full justify-start"
        :disabled="pending"
        @click="emit('delete')"
      >
        <Trash2 :size="13" class="shrink-0" />
        <span>{{ pending ? "正在删除…" : "删除链路" }}</span>
      </Button>
    </div>
  </DropdownMenu>
</template>
