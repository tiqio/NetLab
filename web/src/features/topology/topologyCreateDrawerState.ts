import type { PaletteSelection } from "./TopologyResourceCatalog.vue";

export interface TopologyCreateDrawerState {
  open: boolean;
  selection?: PaletteSelection;
}

export function openTopologyCreateDrawer(
  selection?: PaletteSelection,
): TopologyCreateDrawerState {
  return { open: true, selection };
}
