import { ref } from "vue";

export function useTemporaryPanShortcut() {
  const temporaryPanHeld = ref(false);

  function handleTemporaryPanKeyDown(event: KeyboardEvent) {
    if (event.key === "Control") temporaryPanHeld.value = true;
  }

  function handleTemporaryPanKeyUp(event: KeyboardEvent) {
    if (event.key === "Control") temporaryPanHeld.value = event.ctrlKey;
  }

  function releaseTemporaryPan() {
    temporaryPanHeld.value = false;
  }

  return {
    temporaryPanHeld,
    handleTemporaryPanKeyDown,
    handleTemporaryPanKeyUp,
    releaseTemporaryPan,
  };
}
