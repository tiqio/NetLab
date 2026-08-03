<script setup lang="ts">
defineProps<{ type: string; id: string; name?: string }>();
</script>
<template>
  <div class="min-w-0">
    <strong class="block truncate text-sm">{{ name || id }}</strong
    ><code class="block truncate text-[10px] text-muted-foreground"
      >{{ type }} · {{ id }}</code
    >
  </div>
</template>
