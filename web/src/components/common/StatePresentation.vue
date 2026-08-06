<script setup lang="ts">
import {
  AlertTriangle,
  CloudOff,
  LockKeyhole,
  RefreshCw,
} from "lucide-vue-next";
import { Button } from "@/components/ui";
import { localizeState } from "@/locales";

withDefaults(
  defineProps<{
    state:
      | "empty"
      | "loading"
      | "stale"
      | "reconnecting"
      | "unsupported"
      | "permission"
      | "quota"
      | "conflict"
      | "partial-failure"
      | "cleanup"
      | "terminal-error";
    title?: string;
    description?: string;
    actionLabel?: string;
    actionAvailable?: boolean;
  }>(),
  {
    title: "",
    description: "",
    actionLabel: "重试",
    actionAvailable: false,
  },
);
defineEmits<{ action: [] }>();
</script>
<template>
  <section
    role="status"
    class="grid min-h-28 place-items-center rounded border border-border bg-muted/20 p-4 text-center"
  >
    <div>
      <RefreshCw
        v-if="state === 'loading' || state === 'reconnecting'"
        class="mx-auto animate-spin text-primary"
        :size="20"
      />
      <LockKeyhole
        v-else-if="state === 'permission'"
        class="mx-auto text-[color:var(--warning)]"
        :size="20"
      />
      <CloudOff
        v-else-if="state === 'stale' || state === 'unsupported'"
        class="mx-auto text-[color:var(--warning)]"
        :size="20"
      />
      <AlertTriangle v-else class="mx-auto text-destructive" :size="20" />
      <h3 class="mt-2 text-sm font-medium">
        {{ title || localizeState(state) }}
      </h3>
      <p v-if="description" class="mt-1 text-xs text-muted-foreground">
        {{ description }}
      </p>
      <Button
        v-if="actionAvailable"
        size="sm"
        variant="secondary"
        class="mt-3"
        @click="$emit('action')"
        >{{ actionLabel }}</Button
      >
    </div>
  </section>
</template>
