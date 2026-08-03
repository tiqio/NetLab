<script setup lang="ts">
import { cn } from "@/lib/utils";
defineOptions({ name: "UiTextarea", inheritAttrs: false });
const model = defineModel<string>();
const props = defineProps<{ class?: string }>();
</script>
<template>
  <textarea
    v-model="model"
    v-bind="$attrs"
    :class="
      cn(
        'min-h-24 w-full rounded-md border border-input bg-background px-2.5 py-2 text-sm text-foreground disabled:opacity-50',
        props.class,
      )
    "
  />
</template>
