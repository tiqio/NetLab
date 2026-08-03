<script setup lang="ts">
import { Cable, Eye, Route, Unplug } from "lucide-vue-next";
import { Button, DropdownMenu } from "@/components/ui";

defineProps<{ disabled?: boolean }>();
const emit = defineEmits<{
  inspect: [];
  reconnect: [];
  disconnect: [];
  route: [];
}>();
</script>

<template>
  <DropdownMenu>
    <template #trigger>
      <Button size="sm" variant="secondary" :disabled="disabled">
        <Cable :size="13" /> Link actions
      </Button>
    </template>
    <div class="grid gap-1" role="menu" aria-label="Link actions">
      <Button variant="ghost" class="justify-start" @click="emit('inspect')">
        <Eye :size="13" /> Inspect
      </Button>
      <Button variant="ghost" class="justify-start" @click="emit('reconnect')">
        <Cable :size="13" /> Reconnect endpoint
      </Button>
      <Button variant="ghost" class="justify-start" @click="emit('route')">
        <Route :size="13" /> Edit local route
      </Button>
      <Button
        variant="destructive"
        class="justify-start"
        @click="emit('disconnect')"
      >
        <Unplug :size="13" /> Disconnect
      </Button>
    </div>
  </DropdownMenu>
</template>
