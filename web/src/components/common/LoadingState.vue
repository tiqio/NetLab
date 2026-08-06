<script setup lang="ts">
withDefaults(defineProps<{ label?: string }>(), { label: "正在加载" });
</script>
<template>
  <div
    role="status"
    class="flex items-center gap-2 p-4 text-sm text-muted-foreground"
  >
    <span
      class="h-4 w-4 animate-spin rounded-full border-2 border-muted-foreground border-t-primary"
    />{{ label }}
  </div>
</template>
