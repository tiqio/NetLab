<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import {
  api,
  ApiError,
  type Problem,
  type RuntimeCapabilityObservation,
} from "@/api";
import StructuredProblem from "@/components/common/StructuredProblem.vue";
import { Button } from "@/components/ui";
import { useLaboratoryStore } from "@/stores/laboratory";

const props = defineProps<{ nodeId: string }>();
const store = useLaboratoryStore();
const observations = computed<RuntimeCapabilityObservation[]>(
  () => store.nodeCapabilities[props.nodeId] || [],
);
const problem = ref<Problem>();
const loading = ref(false);

async function refresh() {
  loading.value = true;
  problem.value = undefined;
  try {
    store.setNodeCapabilities(
      props.nodeId,
      (await api.getNodeCapabilities(props.nodeId)).observations,
    );
  } catch (error) {
    problem.value =
      error instanceof ApiError
        ? error.problem
        : {
            code: "capability_query_failed",
            message: error instanceof Error ? error.message : String(error),
            retryable: true,
          };
  } finally {
    loading.value = false;
  }
}

onMounted(refresh);
watch(() => props.nodeId, refresh);
</script>

<template>
  <section class="panel-section" aria-label="Runtime capabilities">
    <div class="flex items-center justify-between gap-2">
      <h3>Runtime capabilities</h3>
      <Button size="sm" variant="outline" :disabled="loading" @click="refresh"
        >Refresh</Button
      >
    </div>
    <p
      v-if="!loading && observations.length === 0"
      class="text-xs text-muted-foreground"
    >
      No capability observations yet.
    </p>
    <ul v-else class="grid gap-2">
      <li
        v-for="item in observations"
        :key="item.capability"
        class="rounded-md border p-2 text-xs"
      >
        <div class="flex justify-between gap-2">
          <strong>{{ item.capability }}</strong
          ><span>{{ item.state }}</span>
        </div>
        <p v-if="item.required" class="text-muted-foreground">
          Required by this template
        </p>
        <StructuredProblem
          v-if="item.problem"
          class="mt-2"
          :problem="item.problem"
        />
      </li>
    </ul>
    <StructuredProblem v-if="problem" class="mt-2" :problem="problem" />
  </section>
</template>

<style scoped>
.panel-section {
  border-bottom: 1px solid var(--border);
  padding: 1rem;
}
.panel-section h3 {
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--muted-foreground);
}
</style>
