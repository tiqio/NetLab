<script setup lang="ts">
withDefaults(defineProps<{ orientation?: "horizontal" | "vertical" }>(), {
  orientation: "vertical",
});
defineEmits<{ pointerdown: [PointerEvent] }>();
</script>
<template>
  <div
    role="separator"
    :aria-orientation="orientation"
    aria-label="调整面板大小"
    aria-valuemin="0"
    aria-valuemax="100"
    aria-valuenow="50"
    tabindex="0"
    class="shrink-0 bg-border transition-colors hover:bg-primary"
    :class="
      orientation === 'vertical'
        ? 'w-1 cursor-col-resize'
        : 'h-2 cursor-row-resize bg-border/80 hover:bg-primary'
    "
    @pointerdown="$emit('pointerdown', $event)"
  />
</template>
