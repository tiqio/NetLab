<script setup lang="ts">
import { cn } from "@/lib/utils";
defineOptions({ name: "UiInput", inheritAttrs: false });
const model = defineModel<string | number>();
const props = defineProps<{ class?: string }>();
</script>
<template>
  <input
    v-model="model"
    v-bind="$attrs"
    :class="
      cn(
        'h-8 w-full rounded-md border border-input bg-background px-2.5 text-sm text-foreground placeholder:text-muted-foreground disabled:opacity-50',
        props.class,
      )
    "
  />
</template>
