<script setup lang="ts">
import TopologyResourceCatalog, {
  type PaletteSelection,
} from "./TopologyResourceCatalog.vue";

export type { PaletteSelection } from "./TopologyResourceCatalog.vue";

const emit = defineEmits<{ choose: [PaletteSelection] }>();
</script>

<template>
  <TopologyResourceCatalog @choose="emit('choose', $event)" />
</template>
