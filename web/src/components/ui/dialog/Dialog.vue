<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";
import { X } from "lucide-vue-next";
import Button from "../button/Button.vue";
defineOptions({ name: "UiDialog" });
const open = defineModel<boolean>({ required: true });
const props = defineProps<{
  title: string;
  description?: string;
  preventClose?: boolean;
}>();
const discardOpen = ref(false);
function requestClose() {
  if (props.preventClose) discardOpen.value = true;
  else open.value = false;
}
function handleEscape(event: KeyboardEvent) {
  if (open.value && event.key === "Escape") {
    event.preventDefault();
    requestClose();
  }
}
onMounted(() => window.addEventListener("keydown", handleEscape));
onBeforeUnmount(() => window.removeEventListener("keydown", handleEscape));
</script>
<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="netlab-overlay fixed inset-0 z-50 grid place-items-center p-4"
      @click.self="requestClose"
    >
      <section
        role="dialog"
        aria-modal="true"
        :aria-label="title"
        class="netlab-surface-elevated flex max-h-[90vh] min-h-0 w-full max-w-lg flex-col overflow-hidden rounded-lg"
      >
        <header
          class="flex shrink-0 items-start justify-between gap-3 border-b border-border px-4 py-3"
        >
          <div class="min-w-0 flex-1">
            <h2 class="netlab-copy font-semibold">
              {{ title }}
            </h2>
            <p
              v-if="description"
              class="netlab-copy mt-1 text-xs text-muted-foreground"
            >
              {{ description }}
            </p>
          </div>
          <Button
            variant="ghost"
            size="icon"
            aria-label="关闭对话框"
            @click="requestClose"
          >
            <X :size="16" />
          </Button>
        </header>
        <div class="netlab-region-scroll p-4">
          <slot />
        </div>
        <footer
          v-if="$slots.footer"
          class="netlab-actions shrink-0 justify-end border-t border-border p-3"
        >
          <slot name="footer" />
        </footer>
      </section>
      <section
        v-if="discardOpen"
        role="alertdialog"
        aria-modal="true"
        aria-label="放弃未保存的更改"
        class="absolute w-full max-w-sm rounded-lg border border-border bg-card p-4 shadow-2xl"
      >
        <h3 class="font-semibold">要放弃未保存的更改吗？</h3>
        <p class="mt-1 text-xs text-muted-foreground">
          关闭对话框后，已输入的内容将丢失。
        </p>
        <div class="mt-4 flex justify-end gap-2">
          <Button variant="secondary" @click="discardOpen = false"
            >继续编辑</Button
          >
          <Button
            variant="destructive"
            @click="
              discardOpen = false;
              open = false;
            "
            >放弃</Button
          >
        </div>
      </section>
    </div>
  </Teleport>
</template>
