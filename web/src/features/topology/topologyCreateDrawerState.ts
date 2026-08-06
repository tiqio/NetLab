import type { PaletteSelection } from "./TopologyResourceCatalog.vue";

export interface TopologyCreateDrawerState {
  open: boolean;
  selection?: PaletteSelection;
}

export interface TopologyCreateWorkspaceSnapshot<TSelection> {
  inspector: { collapsed: boolean; size: number };
  selectedIds: string[];
  selectedType: TSelection;
  focusedResourceId: string;
  activeElement: HTMLElement | null;
}

export function captureTopologyCreateWorkspace<TSelection>(value: {
  inspector: { collapsed: boolean; size: number };
  selectedIds: string[];
  selectedType: TSelection;
  focusedResourceId: string;
  activeElement: HTMLElement | null;
}): TopologyCreateWorkspaceSnapshot<TSelection> {
  return {
    inspector: { ...value.inspector },
    selectedIds: [...value.selectedIds],
    selectedType: value.selectedType,
    focusedResourceId: value.focusedResourceId,
    activeElement: value.activeElement,
  };
}

export function openTopologyCreateDrawer(
  selection?: PaletteSelection,
): TopologyCreateDrawerState {
  return { open: true, selection };
}
