<script setup lang="ts">
import type { TopologyConnectionLegendItem } from "./topologyConnectionLegend";

defineProps<{ items: TopologyConnectionLegendItem[] }>();
const emit = defineEmits<{ highlight: [string[]]; clear: [] }>();
</script>

<template>
  <aside
    v-if="items.length"
    data-testid="topology-connection-legend"
    class="absolute bottom-16 left-3 z-20 max-h-40 w-64 overflow-auto rounded-lg border border-border bg-card/95 p-2 shadow-sm backdrop-blur"
    aria-label="连接语义图例"
  >
    <p class="px-1 pb-1 text-[11px] font-medium text-foreground">连接语义</p>
    <button
      v-for="item in items"
      :key="item.key"
      type="button"
      :data-semantic-marker="item.key"
      :data-connection-count="item.count"
      class="flex w-full items-start gap-2 rounded-md px-1.5 py-1.5 text-left hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      :aria-label="`${item.label}，${item.description}，${item.count} 条连接`"
      @mouseenter="emit('highlight', item.connectionIds)"
      @mouseleave="emit('clear')"
      @focus="emit('highlight', item.connectionIds)"
      @blur="emit('clear')"
    >
      <span
        class="mt-1 h-2.5 w-6 shrink-0 rounded-full border border-[color:var(--topology-connection-focus)] bg-[color:var(--topology-connection-success)]"
        aria-hidden="true"
      />
      <span class="min-w-0">
        <span class="block text-[11px] font-medium text-foreground">
          {{ item.label }} · {{ item.count }}
        </span>
        <span class="block text-[10px] leading-4 text-muted-foreground">
          {{ item.description }}
        </span>
      </span>
    </button>
  </aside>
</template>
