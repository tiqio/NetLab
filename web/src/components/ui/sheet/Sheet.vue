<script setup lang="ts">
import Button from "../button/Button.vue";
defineOptions({ name: "UiSheet" });
const open = defineModel<boolean>({ required: true });
withDefaults(
  defineProps<{ side?: "left" | "right" | "bottom"; title: string }>(),
  { side: "right" },
);
</script>
<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-40 bg-black/50"
      @click.self="open = false"
    >
      <section
        role="dialog"
        :aria-label="title"
        class="absolute border-border bg-card shadow-2xl"
        :class="
          side === 'left'
            ? 'inset-y-0 left-0 w-[min(90vw,360px)] border-r'
            : side === 'bottom'
              ? 'inset-x-0 bottom-0 max-h-[75vh] border-t'
              : 'inset-y-0 right-0 w-[min(90vw,420px)] border-l'
        "
      >
        <header
          class="flex items-center justify-between border-b border-border p-3"
        >
          <h2 class="font-semibold">
            {{ title }}
          </h2>
          <Button
            variant="ghost"
            size="icon"
            aria-label="Close"
            @click="open = false"
            >×</Button
          >
        </header>
        <div class="max-h-[calc(75vh-48px)] overflow-auto p-3">
          <slot />
        </div>
      </section>
    </div>
  </Teleport>
</template>
