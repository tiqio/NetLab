<script setup lang="ts">
defineOptions({ name: "UiTooltip" });
defineProps<{ content: string }>();
</script>
<template>
  <span class="group relative inline-flex"
    ><slot /><span
      role="tooltip"
      class="pointer-events-none absolute bottom-full left-1/2 z-50 mb-1 hidden max-w-72 -translate-x-1/2 whitespace-normal break-words rounded border border-border bg-popover px-2 py-1 text-xs text-popover-foreground shadow group-hover:block group-focus-within:block"
      >{{ content }}</span
    ></span
  >
</template>
