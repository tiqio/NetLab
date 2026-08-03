<script setup lang="ts">
import { ref } from "vue";
const open = ref(false);
</script>
<template>
  <div class="relative inline-flex">
    <span @click="open = !open"><slot name="trigger" /></span>
    <div
      v-if="open"
      class="absolute right-0 top-full z-30 mt-1 min-w-44 rounded-md border border-border bg-popover p-1 shadow-xl"
      @click="open = false"
    >
      <slot />
    </div>
  </div>
</template>
