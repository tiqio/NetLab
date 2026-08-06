<script setup lang="ts">
defineOptions({ name: "UiTabs" });
const model = defineModel<string>({ required: true });
defineProps<{ tabs: Array<{ value: string; label: string }> }>();
</script>
<template>
  <div class="netlab-region min-w-0">
    <div
      role="tablist"
      class="netlab-scrollbar flex min-w-0 gap-1 overflow-x-auto border-b border-border px-1"
    >
      <button
        v-for="tab in tabs"
        :key="tab.value"
        role="tab"
        class="netlab-hit-target whitespace-nowrap border-b-2 px-3 py-2 text-xs focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-[color:var(--focus-outline)]"
        :class="
          model === tab.value
            ? 'border-primary text-foreground'
            : 'border-transparent text-muted-foreground hover:text-foreground'
        "
        :aria-selected="model === tab.value"
        @click="model = tab.value"
      >
        {{ tab.label }}
      </button>
    </div>
    <slot :value="model" />
  </div>
</template>
