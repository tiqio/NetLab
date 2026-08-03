<script setup lang="ts">
import {
  AlertTriangle,
  Clock,
  RotateCcw,
  ShieldCheck,
  Wrench,
} from "lucide-vue-next";
import type { Problem } from "@/api/generated";
import { Button } from "@/components/ui";
defineProps<{ problem: Problem; title?: string }>();
defineEmits<{ retry: []; refresh: [] }>();
</script>
<template>
  <section
    role="alert"
    class="rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm"
  >
    <div class="flex gap-2">
      <AlertTriangle class="mt-0.5 shrink-0 text-destructive" :size="17" />
      <div class="min-w-0 flex-1">
        <h3 class="font-semibold">
          {{ title || problem.code }}
        </h3>
        <p class="mt-1 text-muted-foreground">
          {{ problem.message }}
        </p>
        <dl class="mt-2 grid grid-cols-[auto_1fr] gap-x-2 gap-y-1 text-xs">
          <dt v-if="problem.phase">Phase</dt>
          <dd v-if="problem.phase">
            {{ problem.phase }}
          </dd>
          <dt v-if="problem.cleanup">
            <Wrench :size="13" class="inline" /> Cleanup
          </dt>
          <dd v-if="problem.cleanup">
            {{ problem.cleanup }}
          </dd>
          <dt v-if="problem.retry_after_seconds">
            <Clock :size="13" class="inline" /> Retry
          </dt>
          <dd v-if="problem.retry_after_seconds">
            after {{ problem.retry_after_seconds }}s
          </dd>
          <dt v-if="problem.operator_hint">
            <ShieldCheck :size="13" class="inline" /> Action
          </dt>
          <dd v-if="problem.operator_hint">
            {{ problem.operator_hint }}
          </dd>
        </dl>
        <details v-if="problem.details" class="mt-2">
          <summary>Technical details</summary>
          <pre class="mt-1 overflow-auto text-xs">{{
            JSON.stringify(problem.details, null, 2)
          }}</pre>
        </details>
        <div class="mt-3 flex gap-2">
          <Button
            v-if="problem.retryable"
            size="sm"
            variant="secondary"
            @click="$emit('retry')"
          >
            <RotateCcw :size="13" class="inline" /> Retry</Button
          ><Button
            v-if="problem.code.includes('revision')"
            size="sm"
            variant="secondary"
            @click="$emit('refresh')"
          >
            Refresh state
          </Button>
        </div>
      </div>
    </div>
  </section>
</template>
