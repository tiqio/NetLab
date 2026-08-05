<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue";
import { X } from "lucide-vue-next";
import Button from "../button/Button.vue";

defineOptions({ name: "UiSheet" });

export type SheetCloseReason = "button" | "overlay" | "escape";

const open = defineModel<boolean>({ required: true });
const props = withDefaults(
  defineProps<{
    side?: "left" | "right" | "bottom";
    title: string;
    description?: string;
    size?: string;
    preventClose?: boolean;
  }>(),
  {
    side: "right",
    description: undefined,
    size: undefined,
    preventClose: false,
  },
);
const emit = defineEmits<{ closeRequested: [SheetCloseReason] }>();
const panel = ref<HTMLElement>();
let trigger: HTMLElement | null = null;

const panelStyle = computed(() => {
  if (!props.size) return undefined;
  return props.side === "bottom"
    ? { maxHeight: props.size }
    : { width: props.size };
});
const panelClass = computed(() => {
  if (props.side === "left")
    return "inset-y-0 left-0 h-full w-[min(90vw,360px)] border-r";
  if (props.side === "bottom")
    return "inset-x-0 bottom-0 max-h-[75vh] w-full border-t";
  return "inset-y-0 right-0 h-full w-[min(90vw,420px)] border-l";
});

function focusableElements() {
  return Array.from(
    panel.value?.querySelectorAll<HTMLElement>(
      'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
    ) || [],
  ).filter((element) => !element.hasAttribute("hidden"));
}

function requestClose(reason: SheetCloseReason) {
  emit("closeRequested", reason);
  if (!props.preventClose) open.value = false;
}

function handleKeydown(event: KeyboardEvent) {
  if (!open.value) return;
  if (event.key === "Escape") {
    event.preventDefault();
    requestClose("escape");
    return;
  }
  if (event.key !== "Tab") return;
  const elements = focusableElements();
  if (!elements.length) {
    event.preventDefault();
    panel.value?.focus();
    return;
  }
  const first = elements[0];
  const last = elements[elements.length - 1];
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault();
    last.focus();
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault();
    first.focus();
  }
}

watch(
  open,
  async (value, previous) => {
    if (value) {
      trigger = document.activeElement as HTMLElement | null;
      await nextTick();
      const preferred = panel.value?.querySelector<HTMLElement>("[autofocus]");
      (preferred || focusableElements()[0] || panel.value)?.focus();
    } else if (previous) {
      await nextTick();
      trigger?.focus();
      trigger = null;
    }
  },
  { immediate: true },
);

window.addEventListener("keydown", handleKeydown);
onBeforeUnmount(() => {
  window.removeEventListener("keydown", handleKeydown);
  if (open.value) trigger?.focus();
});
</script>

<template>
  <Teleport to="body">
    <div
      v-if="open"
      data-sheet-overlay
      class="fixed inset-0 z-40 bg-black/50"
      @click.self="requestClose('overlay')"
    >
      <section
        ref="panel"
        role="dialog"
        aria-modal="true"
        :aria-label="title"
        :aria-description="description"
        tabindex="-1"
        class="absolute flex min-h-0 flex-col border-border bg-card shadow-2xl"
        :class="panelClass"
        :style="panelStyle"
      >
        <header
          data-sheet-header
          class="flex shrink-0 items-start justify-between border-b border-border p-3"
        >
          <div class="min-w-0">
            <h2 class="font-semibold">
              {{ title }}
            </h2>
            <p v-if="description" class="mt-1 text-xs text-muted-foreground">
              {{ description }}
            </p>
          </div>
          <Button
            variant="ghost"
            size="icon"
            aria-label="Close sheet"
            @click="requestClose('button')"
          >
            <X :size="16" />
          </Button>
        </header>
        <div
          data-sheet-body
          class="min-h-0 flex-1 overflow-y-auto p-3 netlab-scrollbar"
        >
          <slot />
        </div>
        <footer
          v-if="$slots.footer"
          data-sheet-footer
          class="flex shrink-0 justify-end gap-2 border-t border-border p-3"
        >
          <slot name="footer" />
        </footer>
      </section>
    </div>
  </Teleport>
</template>
