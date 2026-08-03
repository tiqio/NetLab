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
