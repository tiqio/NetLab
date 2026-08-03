<script setup lang="ts">
defineOptions({ name: "UiTabs" });
const model = defineModel<string>({ required: true });
defineProps<{ tabs: Array<{ value: string; label: string }> }>();
</script>
<template>
  <div>
    <div role="tablist" class="flex gap-1 border-b border-border px-1">
      <button
        v-for="tab in tabs"
        :key="tab.value"
        role="tab"
        class="border-b-2 px-3 py-2 text-xs"
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
