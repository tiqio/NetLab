<script setup lang="ts">
import { ThemeSwitcher } from "@/components/appearance";
import { useUiLocalization } from "@/composables/useUiLocalization";

useUiLocalization();
</script>

<template>
  <RouterView />
  <div class="global-appearance-control">
    <ThemeSwitcher />
  </div>
</template>
