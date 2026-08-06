<script setup lang="ts">
import type { TrafficObservation } from "./trafficPathTypes";
defineProps<{ observations: TrafficObservation[]; ambiguous?: boolean }>();
</script>

<template>
  <aside aria-live="polite">
    <h2>观测到的流量路径</h2>
    <p v-if="ambiguous">检测到环路或双向观测，无法唯一确定数据包路径。</p>
    <ol>
      <li
        v-for="item in observations"
        :key="`${item.fingerprint}-${item.network_object_link_id || item.link_id || item.interface_id}-${item.direction}`"
      >
        {{
          item.network_object_link_id ||
          item.resource_id ||
          item.link_id ||
          item.interface_id
        }}
        · {{ item.direction }} · {{ item.count }} 个数据包 ·
        {{ item.bytes }} 字节 · {{ item.first_seen }} — {{ item.last_seen }}
      </li>
    </ol>
  </aside>
</template>
