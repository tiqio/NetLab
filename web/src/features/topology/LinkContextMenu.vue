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
        <Eye :size="13" /> Inspect
      </Button>
      <Button
        v-if="!objectLink"
        variant="ghost"
        class="justify-start"
        @click="emit('reconnect')"
      >
        <Cable :size="13" /> Reconnect endpoint
      </Button>
      <Button
        v-if="!objectLink"
        variant="ghost"
        class="justify-start"
        @click="emit('route')"
      >
        <Route :size="13" /> Edit local route
      </Button>
      <Button
        v-if="!objectLink"
        variant="destructive"
        class="justify-start"
        @click="emit('disconnect')"
      >
        <Unplug :size="13" /> Disconnect
      </Button>
      <Button
        v-else
        variant="destructive"
        class="justify-start"
        :disabled="pending"
        @click="emit('delete')"
      >
        <Trash2 :size="13" /> {{ pending ? "Deleting…" : "Delete link" }}
      </Button>
    </div>
  </DropdownMenu>
</template>
