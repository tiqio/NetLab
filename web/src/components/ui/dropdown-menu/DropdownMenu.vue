<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref } from "vue";
const open = ref(false);
const root = ref<HTMLElement>();
const menu = ref<HTMLElement>();
const position = ref<Record<string, string>>({});

async function toggle() {
  open.value = !open.value;
  if (!open.value) return;
  await nextTick();
  place();
}

function place() {
  if (!root.value || !menu.value) return;
  const trigger = root.value.getBoundingClientRect();
  const bounds = menu.value.getBoundingClientRect();
  const gap = 6;
  const left = Math.min(
    Math.max(gap, trigger.right - bounds.width),
    window.innerWidth - bounds.width - gap,
  );
  const below = trigger.bottom + gap;
  const top =
    below + bounds.height <= window.innerHeight - gap
      ? below
      : Math.max(gap, trigger.top - bounds.height - gap);
  position.value = {
    left: `${left}px`,
    top: `${top}px`,
    maxHeight: `${Math.max(96, window.innerHeight - top - gap)}px`,
  };
}

function handleKeydown(event: KeyboardEvent) {
  if (open.value && event.key === "Escape") {
    event.preventDefault();
    open.value = false;
  }
}

window.addEventListener("resize", place);
window.addEventListener("keydown", handleKeydown);
onBeforeUnmount(() => {
  window.removeEventListener("resize", place);
  window.removeEventListener("keydown", handleKeydown);
});
</script>
<template>
  <div ref="root" class="relative inline-flex">
    <span @click="toggle"><slot name="trigger" /></span>
    <div
      v-if="open"
      ref="menu"
      class="netlab-surface-elevated netlab-scrollbar fixed z-50 min-w-44 max-w-[min(22rem,calc(100vw-0.75rem))] overflow-auto rounded-md p-1 text-popover-foreground"
      :style="position"
      @click="open = false"
    >
      <slot />
    </div>
  </div>
</template>
