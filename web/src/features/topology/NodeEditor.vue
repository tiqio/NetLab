<script setup lang="ts">
import TemplatePicker from "../templates/TemplatePicker.vue";
defineProps({
  nodeName: { type: String, default: "" },
  interfaceCount: { type: Number, default: 0 },
});
</script>
<template>
  <aside>
    <h3>{{ nodeName || "节点" }}</h3>
    <p>{{ interfaceCount || 0 }} 个接口</p>
    <TemplatePicker />
  </aside>
</template>
