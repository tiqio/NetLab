<script setup lang="ts">
import { computed, provide, useAttrs, useId } from "vue";
import { formFieldControlIdKey } from "./formFieldContext";

defineOptions({ inheritAttrs: false });
const props = defineProps<{
  label: string;
  error?: string;
  hint?: string;
  field?: string;
  controlId?: string;
}>();
const attrs = useAttrs();
const generatedId = useId();
const id = computed(() => props.controlId || `netlab-field-${generatedId}`);
const field = computed(
  () => props.field || String(attrs["data-field"] || "") || undefined,
);
provide(formFieldControlIdKey, id.value);
</script>
<template>
  <label
    v-bind="$attrs"
    :for="id"
    :data-field="field"
    class="netlab-region grid min-w-0 gap-1 text-xs font-medium text-muted-foreground"
  >
    <span class="netlab-copy">{{ label }}</span>
    <slot />
    <span v-if="error" class="netlab-copy text-destructive">{{ error }}</span>
    <span
      v-else-if="hint"
      class="netlab-copy font-normal text-muted-foreground"
    >
      {{ hint }}
    </span>
  </label>
</template>
