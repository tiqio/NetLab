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
      <Button variant="ghost" class="justify-start" @click="emit('inspect')">
        <Eye :size="13" /> 检查
      </Button>
      <Button
        v-if="!objectLink"
        variant="ghost"
        class="justify-start"
        @click="emit('reconnect')"
      >
        <Cable :size="13" /> 重新连接端点
      </Button>
      <Button
        v-if="!objectLink"
        variant="ghost"
        class="justify-start"
        @click="emit('route')"
      >
        <Route :size="13" /> 编辑本地路由
      </Button>
      <Button
        v-if="!objectLink"
        variant="destructive"
        class="justify-start"
        @click="emit('disconnect')"
      >
        <Unplug :size="13" /> 断开连接
      </Button>
      <Button
        v-else
        variant="destructive"
        class="justify-start"
        :disabled="pending"
        @click="emit('delete')"
      >
        <Trash2 :size="13" /> {{ pending ? "正在删除…" : "删除链路" }}
      </Button>
    </div>
  </DropdownMenu>
</template>
