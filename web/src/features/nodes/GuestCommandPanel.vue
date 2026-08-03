<script setup lang="ts">
import { ref } from "vue";
import { api } from "@/api";
import { Button, FormField, Input } from "@/components/ui";
const props = defineProps<{ nodeId: string }>();
const command = ref("uname -a");
const status = ref("");
const busy = ref(false);
async function execute() {
  const argv =
    command.value
      .match(/(?:[^\s"]+|"[^"]*")+/g)
      ?.map((value) => value.replace(/^"|"$/g, "")) || [];
  if (!argv.length) return;
  busy.value = true;
  try {
    const task = await api.executeGuestCommand(props.nodeId, {
      argv,
      timeout_seconds: 30,
      output_limit: 1 << 20,
    });
    status.value = `Guest command queued: ${task.id}`;
  } finally {
    busy.value = false;
  }
}
</script>
<template>
  <form class="panel-section" @submit.prevent="execute">
    <h3>Guest command</h3>
    <FormField
      label="Bounded command"
      hint="Output is never persisted in browser preferences."
    >
      <Input v-model="command" autocomplete="off" /> </FormField
    ><Button class="mt-2" size="sm" :disabled="busy">
      Run through QEMU guest agent
    </Button>
    <p role="status" class="mt-1 text-xs text-muted-foreground">
      {{ status }}
    </p>
  </form>
</template>
