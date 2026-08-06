<script setup lang="ts">
import { computed, useAttrs } from "vue";
import { cn } from "@/lib/utils";

defineOptions({ name: "UiButton", inheritAttrs: false });
const props = withDefaults(
  defineProps<{
    variant?: "default" | "secondary" | "ghost" | "destructive" | "outline";
    size?: "sm" | "default" | "icon";
    class?: string;
  }>(),
  { variant: "default", size: "default", class: "" },
);
const attrs = useAttrs();
const classes = computed(() =>
  cn(
    "netlab-hit-target inline-flex shrink-0 items-center justify-center gap-2 rounded-md border border-transparent font-medium transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[color:var(--focus-outline)] disabled:pointer-events-none disabled:opacity-55",
    props.size === "sm" && "h-7 px-2.5 text-xs",
    props.size === "default" && "h-8 px-3 text-sm",
    props.size === "icon" && "h-8 w-8",
    props.variant === "default" &&
      "bg-primary text-primary-foreground hover:brightness-110",
    props.variant === "secondary" &&
      "bg-secondary text-secondary-foreground hover:bg-accent",
    props.variant === "ghost" &&
      "bg-transparent text-foreground hover:bg-accent",
    props.variant === "outline" &&
      "border-border bg-transparent text-foreground hover:bg-accent",
    props.variant === "destructive" &&
      "bg-destructive text-white hover:brightness-110",
    props.class,
  ),
);
</script>
<template>
  <button v-bind="attrs" :class="classes">
    <slot />
  </button>
</template>
