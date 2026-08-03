<script setup lang="ts">
defineProps<{ title: string; description?: string }>();
</script>
<template>
  <div class="grid min-h-32 place-items-center p-6 text-center">
    <div>
      <h3 class="font-medium">
        {{ title }}
      </h3>
      <p v-if="description" class="mt-1 max-w-sm text-xs text-muted-foreground">
        {{ description }}
      </p>
      <div v-if="$slots.default" class="mt-3">
        <slot />
      </div>
    </div>
  </div>
</template>
