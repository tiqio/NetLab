<script setup lang="ts">
import type { TrafficObservation } from "./trafficPathTypes";
defineProps<{ observations: TrafficObservation[]; ambiguous?: boolean }>();
</script>

<template>
  <aside aria-live="polite">
    <h2>Observed traffic path</h2>
    <p v-if="ambiguous">
      The packet path is ambiguous because a loop or bidirectional observation
      was detected.
    </p>
    <ol>
      <li
        v-for="item in observations"
        :key="`${item.fingerprint}-${item.interface_id}-${item.direction}`"
      >
        {{ item.link_id || item.interface_id }} · {{ item.direction }} ·
        {{ item.count }} packets · {{ item.bytes }} bytes ·
        {{ item.first_seen }} — {{ item.last_seen }}
      </li>
    </ol>
  </aside>
</template>
