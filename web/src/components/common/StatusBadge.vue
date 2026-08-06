<script setup lang="ts">
import { computed } from "vue";
import { localizeState } from "@/locales";
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
const label = computed(() => localizeState(props.state));
</script>
<template>
  <span
    class="inline-flex max-w-full items-center gap-1 rounded-full border px-2 py-0.5 text-[11px]"
    :title="label"
    :class="
      tone === 'success'
        ? 'border-[color:var(--success)]/40 bg-[color:var(--success)]/10 text-[color:var(--success)]'
        : tone === 'danger'
          ? 'border-destructive/40 bg-destructive/10 text-destructive'
          : tone === 'warning'
            ? 'border-[color:var(--warning)]/40 bg-[color:var(--warning)]/10 text-[color:var(--warning)]'
            : 'border-border bg-muted text-muted-foreground'
    "
    ><span
      class="h-1.5 w-1.5 shrink-0 rounded-full bg-current"
      aria-hidden="true"
    /><span class="truncate">{{ label }}</span></span
  >
</template>
