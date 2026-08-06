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
        'h-8 w-full min-w-0 rounded-md border border-input bg-background px-2.5 text-sm text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[color:var(--focus-outline)] disabled:opacity-55',
        props.class,
      )
    "
  >
    <slot />
  </select>
</template>
