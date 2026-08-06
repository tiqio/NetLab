<script setup lang="ts">
import { inject } from "vue";
import { cn } from "@/lib/utils";
import { formFieldControlIdKey } from "../form/formFieldContext";
defineOptions({ name: "UiSelect", inheritAttrs: false });
const model = defineModel<string | number>();
const props = defineProps<{ class?: string }>();
const formFieldControlId = inject(formFieldControlIdKey, undefined);
</script>
<template>
  <select
    v-model="model"
    v-bind="$attrs"
    :id="String($attrs.id || formFieldControlId || '') || undefined"
    :class="
      cn(
        'h-8 w-full rounded-md border border-input bg-background px-2.5 text-sm text-foreground disabled:opacity-50',
        props.class,
      )
    "
  >
    <slot />
  </select>
</template>
