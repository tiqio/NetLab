<script setup lang="ts">
import { computed } from "vue";
const props = defineProps<{ state: string }>();
const tone = computed(() =>
  [
    "active",
    "attached",
    "running",
    "succeeded",
    "connected",
    "present",
    "up",
  ].includes(props.state)
    ? "success"
    : ["failed", "error"].includes(props.state)
      ? "danger"
      : ["queued", "starting", "stopping", "reconnecting"].includes(props.state)
        ? "warning"
        : "neutral",
);
</script>
<template>
  <span
    class="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px]"
    :class="
      tone === 'success'
        ? 'border-green-500/40 bg-green-500/10 text-green-300'
        : tone === 'danger'
          ? 'border-red-500/40 bg-red-500/10 text-red-300'
          : tone === 'warning'
            ? 'border-amber-500/40 bg-amber-500/10 text-amber-300'
            : 'border-border bg-muted text-muted-foreground'
    "
    ><span class="h-1.5 w-1.5 rounded-full bg-current" aria-hidden="true" />{{
      state
    }}</span
  >
</template>
