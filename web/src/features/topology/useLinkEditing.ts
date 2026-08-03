import { computed, ref, type Ref } from "vue";
import { api, type NodeInterface } from "@/api";

export function useLinkEditing(
  laboratoryId: Ref<string | undefined>,
  interfaces: Ref<NodeInterface[]>,
  onChanged: () => Promise<void> | void,
) {
  const endpointA = ref("");
  const endpointB = ref("");
  const busy = ref(false);
  const error = ref("");
  const available = computed(() =>
    interfaces.value.filter((item) => !item.desired_link_id),
  );
  const valid = computed(
    () =>
      endpointA.value &&
      endpointB.value &&
      endpointA.value !== endpointB.value &&
      available.value.some((item) => item.id === endpointA.value) &&
      available.value.some((item) => item.id === endpointB.value),
  );
  async function connect() {
    if (!laboratoryId.value || !valid.value) return;
    busy.value = true;
    error.value = "";
    try {
      await api.connectLink(
        laboratoryId.value,
        endpointA.value,
        endpointB.value,
      );
      endpointA.value = "";
      endpointB.value = "";
      await onChanged();
    } catch (value) {
      error.value = value instanceof Error ? value.message : String(value);
    } finally {
      busy.value = false;
    }
  }
  async function disconnect(linkId: string) {
    busy.value = true;
    error.value = "";
    try {
      await api.disconnectLink(linkId);
      await onChanged();
    } catch (value) {
      error.value = value instanceof Error ? value.message : String(value);
    } finally {
      busy.value = false;
    }
  }
  return {
    endpointA,
    endpointB,
    available,
    valid,
    busy,
    error,
    connect,
    disconnect,
  };
}
