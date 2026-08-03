import { onBeforeUnmount, onMounted, type Ref } from "vue";

export interface WorkspaceKeyboardActions {
  openCommands: () => void;
  clearSelection: () => void;
  selectNext: (direction: 1 | -1) => void;
  openInspector: () => void;
  openTasks: () => void;
}

export function useWorkspaceKeyboard(
  enabled: Ref<boolean>,
  actions: WorkspaceKeyboardActions,
) {
  function handler(event: KeyboardEvent) {
    if (!enabled.value) return;
    const target = event.target;
    const editing =
      target instanceof Element &&
      target.matches("input, textarea, select, [contenteditable='true']");
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "k") {
      event.preventDefault();
      actions.openCommands();
      return;
    }
    if (editing) return;
    if (event.key === "Escape") actions.clearSelection();
    if (event.key === "ArrowRight" || event.key === "ArrowDown") {
      event.preventDefault();
      actions.selectNext(1);
    }
    if (event.key === "ArrowLeft" || event.key === "ArrowUp") {
      event.preventDefault();
      actions.selectNext(-1);
    }
    if (event.key.toLowerCase() === "i") actions.openInspector();
    if (event.key.toLowerCase() === "t") actions.openTasks();
  }
  onMounted(() => window.addEventListener("keydown", handler));
  onBeforeUnmount(() => window.removeEventListener("keydown", handler));
  return { handler };
}
