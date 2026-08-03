<script setup lang="ts">
defineProps<{ label: string; error?: string; hint?: string }>();
</script>
<template>
  <label class="grid gap-1 text-xs font-medium text-muted-foreground"
    ><span>{{ label }}</span
    ><slot /><span v-if="error" class="text-destructive">{{ error }}</span
    ><span v-else-if="hint" class="font-normal text-muted-foreground">{{
      hint
    }}</span></label
  >
</template>
