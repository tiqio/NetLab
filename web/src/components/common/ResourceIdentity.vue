<script setup lang="ts">
defineProps<{ type: string; id: string; name?: string }>();
</script>
<template>
  <div class="netlab-region min-w-0">
    <strong class="block truncate text-sm" :title="name || id">{{
      name || id
    }}</strong
    ><code
      class="block truncate text-[10px] text-muted-foreground"
      :title="`${type} · ${id}`"
      >{{ type }} · {{ id }}</code
    >
  </div>
</template>
