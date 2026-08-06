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
import { problemContext } from "@/locales";
defineProps<{ problem: Problem; title?: string }>();
defineEmits<{ retry: []; refresh: [] }>();
</script>
<template>
  <section
    role="alert"
    class="netlab-region rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm"
  >
    <div class="flex gap-2">
      <AlertTriangle class="mt-0.5 shrink-0 text-destructive" :size="17" />
      <div class="min-w-0 flex-1">
        <h3 class="netlab-copy font-semibold">
          {{ title || problem.code }}
        </h3>
        <p class="netlab-copy mt-1 text-muted-foreground">
          {{ problemContext(problem.code) }}
        </p>
        <dl
          class="mt-2 grid min-w-0 grid-cols-[minmax(4rem,auto)_minmax(0,1fr)] gap-x-2 gap-y-1 text-xs"
        >
          <dt v-if="problem.phase">阶段</dt>
          <dd v-if="problem.phase">
            {{ problem.phase }}
          </dd>
          <dt v-if="problem.cleanup">
            <Wrench :size="13" class="inline" /> 清理
          </dt>
          <dd v-if="problem.cleanup">
            {{ problem.cleanup }}
          </dd>
          <dt v-if="problem.retry_after_seconds">
            <Clock :size="13" class="inline" /> 重试
          </dt>
          <dd v-if="problem.retry_after_seconds">
            {{ problem.retry_after_seconds }} 秒后
          </dd>
          <dt v-if="problem.operator_hint">
            <ShieldCheck :size="13" class="inline" /> 建议操作
          </dt>
          <dd v-if="problem.operator_hint">
            {{ problem.operator_hint }}
          </dd>
        </dl>
        <details class="mt-2">
          <summary class="cursor-pointer">原始诊断详情</summary>
          <p class="netlab-copy mt-1 text-xs text-muted-foreground">
            {{ problem.message }}
          </p>
          <pre
            v-if="problem.details"
            class="mt-1 max-h-48 overflow-auto text-xs"
            >{{ JSON.stringify(problem.details, null, 2) }}</pre>
        </details>
        <div class="netlab-actions mt-3">
          <Button
            v-if="problem.retryable"
            size="sm"
            variant="secondary"
            @click="$emit('retry')"
          >
            <RotateCcw :size="13" class="inline" /> 重试</Button
          ><Button
            v-if="problem.code.includes('revision')"
            size="sm"
            variant="secondary"
            @click="$emit('refresh')"
          >
            刷新状态
          </Button>
        </div>
      </div>
    </div>
  </section>
</template>
