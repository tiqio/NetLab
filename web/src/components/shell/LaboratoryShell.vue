<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import ResizableHandle from "@/components/ui/resizable/ResizableHandle.vue";
import Sheet from "@/components/ui/sheet/Sheet.vue";
import type { WorkspacePreferences } from "@/types/workspace";
import { clampBottomDrawerSize } from "./laboratoryShellSizing";

const props = defineProps<{ preferences: WorkspacePreferences }>();
const emit = defineEmits<{
  panel: [
    keyof WorkspacePreferences["panels"],
    { collapsed?: boolean; size?: number },
  ];
}>();
const compact = ref(false);
const paletteSheet = ref(false);
const inspectorSheet = ref(false);
const bottomSheet = ref(false);
const workspaceColumn = ref<HTMLElement>();
const workspaceHeight = ref(0);
let workspaceObserver: ResizeObserver | undefined;
const paletteStyle = computed(() => ({
  width: props.preferences.panels.devicePalette.collapsed
    ? "0px"
    : `${props.preferences.panels.devicePalette.size}px`,
}));
const inspectorStyle = computed(() => ({
  width: props.preferences.panels.inspector.collapsed
    ? "0px"
    : `${props.preferences.panels.inspector.size}px`,
}));
const bottomStyle = computed(() => ({
  height: props.preferences.panels.bottomDrawer.collapsed
    ? "0px"
    : `${effectiveBottomSize.value}px`,
}));
const effectiveBottomSize = computed(() =>
  workspaceHeight.value > 0
    ? clampBottomDrawerSize(
        props.preferences.panels.bottomDrawer.size,
        workspaceHeight.value,
      )
    : props.preferences.panels.bottomDrawer.size,
);

function measureWorkspace() {
  workspaceHeight.value = workspaceColumn.value?.clientHeight || 0;
}

function beginResize(
  event: PointerEvent,
  panel: keyof WorkspacePreferences["panels"],
  axis: "x" | "y",
  direction = 1,
) {
  event.preventDefault();
  measureWorkspace();
  const start = axis === "x" ? event.clientX : event.clientY;
  const initial =
    panel === "bottomDrawer"
      ? effectiveBottomSize.value
      : props.preferences.panels[panel].size;
  const move = (value: PointerEvent) => {
    const requested =
      initial +
      ((axis === "x" ? value.clientX : value.clientY) - start) * direction;
    emit("panel", panel, {
      size:
        panel === "bottomDrawer" && workspaceHeight.value > 0
          ? clampBottomDrawerSize(requested, workspaceHeight.value)
          : requested,
    });
  };
  const stop = () => {
    window.removeEventListener("pointermove", move);
    window.removeEventListener("pointerup", stop);
  };
  window.addEventListener("pointermove", move);
  window.addEventListener("pointerup", stop);
}
onMounted(() => {
  measureWorkspace();
  if (typeof ResizeObserver === "undefined") return;
  workspaceObserver = new ResizeObserver(measureWorkspace);
  if (workspaceColumn.value) workspaceObserver.observe(workspaceColumn.value);
});
onBeforeUnmount(() => workspaceObserver?.disconnect());
function openPalette() {
  if (window.innerWidth < 1024) paletteSheet.value = true;
  else
    emit("panel", "devicePalette", {
      collapsed: !props.preferences.panels.devicePalette.collapsed,
    });
}
function openInspector() {
  if (window.innerWidth < 1024) inspectorSheet.value = true;
  else emit("panel", "inspector", { collapsed: false });
}
defineExpose({
  openPalette,
  openInspector,
  openBottom: () =>
    window.innerWidth < 1024
      ? (bottomSheet.value = true)
      : emit("panel", "bottomDrawer", { collapsed: false }),
});
</script>
<template>
  <div class="flex h-full min-h-0 flex-col" :class="compact && 'text-xs'">
    <slot name="toolbar" :open-palette="openPalette" />
    <div class="flex min-h-0 flex-1">
      <aside
        v-show="!preferences.panels.devicePalette.collapsed"
        :style="paletteStyle"
        class="netlab-panel hidden min-w-0 overflow-hidden border-r lg:block"
      >
        <div class="h-full overflow-auto netlab-scrollbar">
          <slot name="palette" />
        </div>
      </aside>
      <ResizableHandle
        v-if="!preferences.panels.devicePalette.collapsed"
        class="hidden lg:block"
        @pointerdown="beginResize($event, 'devicePalette', 'x')"
      />
      <section ref="workspaceColumn" class="flex min-w-0 flex-1 flex-col">
        <div class="relative min-h-0 flex-1 overflow-hidden">
          <slot name="canvas" :open-inspector="openInspector" />
          <div class="absolute right-2 top-2 z-20 flex gap-1">
            <button
              class="shell-toggle"
              aria-label="展开或收起检查器"
              :aria-expanded="!preferences.panels.inspector.collapsed"
              @click="
                preferences.panels.inspector.collapsed
                  ? openInspector()
                  : $emit('panel', 'inspector', { collapsed: true })
              "
            >
              {{
                preferences.panels.inspector.collapsed
                  ? "◀ 检查器"
                  : "检查器 ▶"
              }}
            </button>
          </div>
          <button
            class="shell-toggle absolute bottom-2 right-2 z-20"
            aria-label="Toggle operations drawer"
            :aria-expanded="!preferences.panels.bottomDrawer.collapsed"
            @click="
              preferences.panels.bottomDrawer.collapsed
                ? $emit('panel', 'bottomDrawer', { collapsed: false })
                : $emit('panel', 'bottomDrawer', { collapsed: true })
            "
          >
            {{
              preferences.panels.bottomDrawer.collapsed ? "▲ Operations" : "▼"
            }}
          </button>
        </div>
        <ResizableHandle
          v-if="!preferences.panels.bottomDrawer.collapsed"
          orientation="horizontal"
          class="relative z-30"
          @pointerdown="beginResize($event, 'bottomDrawer', 'y', -1)"
        />
        <section
          v-show="!preferences.panels.bottomDrawer.collapsed"
          :style="bottomStyle"
          class="netlab-panel min-h-0 overflow-hidden border-t"
        >
          <slot name="bottom" />
        </section>
      </section>
      <ResizableHandle
        v-if="!preferences.panels.inspector.collapsed"
        class="hidden lg:block"
        @pointerdown="beginResize($event, 'inspector', 'x', -1)"
      />
      <aside
        v-show="!preferences.panels.inspector.collapsed"
        :style="inspectorStyle"
        class="netlab-panel hidden min-w-0 overflow-auto border-l lg:block netlab-scrollbar"
      >
        <slot name="inspector" />
      </aside>
    </div>
    <Sheet v-model="paletteSheet" side="left" title="Devices">
      <slot name="palette" /> </Sheet
    ><Sheet v-model="inspectorSheet" title="检查器">
      <slot name="inspector" /> </Sheet
    ><Sheet v-model="bottomSheet" side="bottom" title="Operations">
      <slot name="bottom" />
    </Sheet>
  </div>
</template>
<style scoped>
.shell-toggle {
  border: 1px solid var(--border);
  border-radius: 0.3rem;
  background: color-mix(in srgb, var(--card) 92%, transparent);
  padding: 0.3rem 0.5rem;
  color: var(--muted-foreground);
  font-size: 0.68rem;
  box-shadow: 0 4px 14px #0008;
}
.shell-toggle:hover {
  color: var(--foreground);
  border-color: var(--primary);
}
</style>
