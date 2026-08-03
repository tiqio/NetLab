import { defineStore } from "pinia";
import {
  api,
  ApiError,
  type Laboratory,
  type Link,
  type NetworkObject,
  type Node,
  type NodeInterface,
  type OperationTask,
  type RuntimeCapabilityObservation,
  type TopologySnapshot,
} from "../api";

export interface StateEvent {
  sequence: number;
  type: string;
  laboratory_id?: string;
  resource_type: string;
  resource_id: string;
  revision: number;
  task_id?: string;
  data: Record<string, unknown>;
}

export function shouldReloadEventStream(sequence: number, event: StateEvent) {
  return (
    event.type === "stream.reset_required" ||
    (sequence > 0 && event.sequence > sequence + 1)
  );
}

let eventSocket: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectAttempt = 0;
let eventsStopped = true;

function normalizeSnapshot(snapshot: TopologySnapshot): TopologySnapshot {
  return {
    ...snapshot,
    nodes: snapshot.nodes || [],
    interfaces: snapshot.interfaces || [],
    links: snapshot.links || [],
    network_objects: snapshot.network_objects || [],
    network_attachments: snapshot.network_attachments || [],
    network_object_links: snapshot.network_object_links || [],
  };
}

export const useLaboratoryStore = defineStore("laboratory", {
  state: () => ({
    labs: [] as Laboratory[],
    active: null as TopologySnapshot | null,
    tasks: [] as OperationTask[],
    nodeCapabilities: {} as Record<string, RuntimeCapabilityObservation[]>,
    hiddenLaboratoryIds: [] as string[],
    sequence: 0,
    eventStatus: "disconnected" as
      "disconnected" | "connecting" | "connected" | "reconnecting",
    syncState: "initializing" as
      "initializing" | "live" | "reconnecting" | "refreshing" | "degraded",
    loading: false,
    error: "",
  }),
  actions: {
    async loadLabs() {
      this.loading = true;
      try {
        this.labs = ((await api.listLabs()) || []).filter(
          (laboratory) =>
            laboratory.lifecycle_state !== "deleting" &&
            !this.hiddenLaboratoryIds.includes(laboratory.id),
        );
        this.error = "";
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error);
      } finally {
        this.loading = false;
      }
    },
    async open(id: string) {
      this.syncState = this.active ? "refreshing" : "initializing";
      try {
        this.active = normalizeSnapshot(await api.getLab(id));
        this.sequence = this.active.event_sequence;
        this.syncState = "live";
        this.connectEvents();
      } catch (error) {
        this.syncState = "degraded";
        throw error;
      }
    },
    async loadTasks() {
      this.tasks = (await api.listTasks()) || [];
    },
    setNodeCapabilities(
      nodeId: string,
      values: RuntimeCapabilityObservation[],
    ) {
      this.nodeCapabilities[nodeId] = values;
    },
    hideLaboratory(id: string) {
      if (!this.hiddenLaboratoryIds.includes(id))
        this.hiddenLaboratoryIds.push(id);
      this.labs = this.labs.filter((laboratory) => laboratory.id !== id);
      if (this.active?.laboratory.id === id) this.active = null;
    },
    applyEvent(event: StateEvent) {
      if (event.sequence <= this.sequence) return;
      this.sequence = event.sequence;
      if (event.resource_type === "operation_task") {
        this.upsert(
          this.tasks,
          event.resource_id,
          event.data as unknown as OperationTask,
        );
        return;
      }
      if (event.type === "node.capability_changed") {
        const observation =
          event.data as unknown as RuntimeCapabilityObservation;
        const values = this.nodeCapabilities[event.resource_id] || [];
        const index = values.findIndex(
          (item) => item.capability === observation.capability,
        );
        if (index >= 0) values[index] = observation;
        else values.push(observation);
        this.nodeCapabilities[event.resource_id] = [...values];
        return;
      }
      if (
        event.resource_type === "laboratory" &&
        (event.type.endsWith(".deleting") || event.type.endsWith(".deleted"))
      ) {
        this.hideLaboratory(event.resource_id);
        return;
      }
      if (
        event.resource_type === "laboratory" &&
        event.type.endsWith(".delete_failed")
      ) {
        this.hiddenLaboratoryIds = this.hiddenLaboratoryIds.filter(
          (id) => id !== event.resource_id,
        );
        void this.loadLabs();
        return;
      }
      if (!this.active || event.laboratory_id !== this.active.laboratory.id)
        return;
      if (event.resource_type === "laboratory") {
        if (event.revision < this.active.laboratory.revision) return;
        Object.assign(this.active.laboratory, event.data, {
          revision: event.revision,
        });
        const lab = this.labs.find((item) => item.id === event.resource_id);
        if (lab) Object.assign(lab, this.active.laboratory);
        return;
      }
      if (event.resource_type === "node") {
        if (event.type.endsWith(".deleted")) {
          const interfaceIds = new Set(
            this.active.interfaces
              .filter((item) => item.node_id === event.resource_id)
              .map((item) => item.id),
          );
          this.active.nodes = this.active.nodes.filter(
            (item) => item.id !== event.resource_id,
          );
          this.active.interfaces = this.active.interfaces.filter(
            (item) => item.node_id !== event.resource_id,
          );
          this.active.links = this.active.links.filter(
            (item) =>
              !interfaceIds.has(item.endpoint_a_id) &&
              !interfaceIds.has(item.endpoint_b_id),
          );
          this.active.placements = this.active.placements.filter(
            (item) => item.resource_id !== event.resource_id,
          );
          delete this.nodeCapabilities[event.resource_id];
          return;
        }
        const data = { ...event.data } as unknown as Node & {
          interfaces?: NodeInterface[];
        };
        const interfaces = data.interfaces;
        delete data.interfaces;
        this.upsert(this.active.nodes, event.resource_id, data, event.revision);
        for (const item of interfaces || [])
          this.upsert(this.active.interfaces, item.id, item, item.revision);
        return;
      }
      if (event.resource_type === "interface") {
        if (event.type.endsWith(".deleted")) {
          this.active.interfaces = this.active.interfaces.filter(
            (item) => item.id !== event.resource_id,
          );
          return;
        }
        this.upsert(
          this.active.interfaces,
          event.resource_id,
          event.data as unknown as NodeInterface,
          event.revision,
        );
        return;
      }
      if (event.resource_type === "link") {
        const previous = this.active.links.find(
          (item) => item.id === event.resource_id,
        );
        if (event.type.endsWith(".deleted")) {
          this.active.links = this.active.links.filter(
            (item) => item.id !== event.resource_id,
          );
          for (const item of this.active.interfaces)
            if (item.desired_link_id === event.resource_id)
              item.desired_link_id = undefined;
          return;
        }
        this.upsert(
          this.active.links,
          event.resource_id,
          event.data as unknown as Link,
          event.revision,
        );
        const link =
          this.active.links.find((item) => item.id === event.resource_id) ||
          previous;
        if (link)
          for (const interfaceId of [link.endpoint_a_id, link.endpoint_b_id]) {
            const item = this.active.interfaces.find(
              (value) => value.id === interfaceId,
            );
            if (item) item.desired_link_id = link.id;
          }
        return;
      }
      if (event.resource_type === "network_object") {
        if (event.type.endsWith(".deleted")) {
          this.active.network_objects = (
            this.active.network_objects || []
          ).filter((item) => item.id !== event.resource_id);
          this.active.network_attachments = (
            this.active.network_attachments || []
          ).filter((item) => item.network_object_id !== event.resource_id);
          this.active.network_object_links = (
            this.active.network_object_links || []
          ).filter(
            (item) =>
              item.object_a_id !== event.resource_id &&
              item.object_b_id !== event.resource_id,
          );
          this.active.placements = this.active.placements.filter(
            (item) => item.resource_id !== event.resource_id,
          );
          return;
        }
        this.active.network_objects ||= [];
        this.upsert(
          this.active.network_objects,
          event.resource_id,
          event.data as unknown as NetworkObject,
          event.revision,
        );
      }
    },
    upsert<T extends { id: string; revision?: number }>(
      items: T[],
      id: string,
      data: Partial<T>,
      revision?: number,
    ) {
      const current = items.find((item) => item.id === id);
      if (
        current &&
        revision !== undefined &&
        current.revision !== undefined &&
        revision < current.revision
      )
        return;
      if (current) Object.assign(current, data, { revision });
      else items.push({ id, ...data, revision } as T);
    },
    connectEvents() {
      eventsStopped = false;
      if (eventSocket) eventSocket.close();
      if (reconnectTimer) clearTimeout(reconnectTimer);
      this.eventStatus = reconnectAttempt ? "reconnecting" : "connecting";
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      eventSocket = new WebSocket(
        `${protocol}//${window.location.host}/api/v1/events?after=${this.sequence}`,
      );
      eventSocket.onopen = () => {
        reconnectAttempt = 0;
        this.eventStatus = "connected";
        this.syncState = "live";
      };
      eventSocket.onmessage = async (message) => {
        const event = JSON.parse(String(message.data)) as StateEvent;
        if (shouldReloadEventStream(this.sequence, event)) {
          this.syncState = "refreshing";
          if (this.active) await this.open(this.active.laboratory.id);
          return;
        }
        this.applyEvent(event);
      };
      eventSocket.onclose = () => {
        eventSocket = null;
        if (eventsStopped) return;
        this.eventStatus = "reconnecting";
        this.syncState = "reconnecting";
        const delay = Math.min(1000 * 2 ** reconnectAttempt++, 15000);
        reconnectTimer = setTimeout(() => this.connectEvents(), delay);
      };
      eventSocket.onerror = () => eventSocket?.close();
    },
    stopEvents() {
      eventsStopped = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      reconnectTimer = null;
      reconnectAttempt = 0;
      eventSocket?.close();
      eventSocket = null;
      this.eventStatus = "disconnected";
    },
    async saveLab(lab: Laboratory) {
      try {
        const updated = await api.updateLab(lab);
        if (this.active) this.active.laboratory = updated;
      } catch (error) {
        if (error instanceof ApiError && error.status === 412)
          await this.open(lab.id);
        throw error;
      }
    },
    async mutate(action: () => Promise<unknown>) {
      if (!this.active) return;
      const id = this.active.laboratory.id;
      this.error = "";
      try {
        await action();
      } catch (error) {
        if (error instanceof ApiError && error.status === 412) {
          this.error =
            "Topology changed elsewhere; refreshed to the latest revision.";
        } else {
          this.error = error instanceof Error ? error.message : String(error);
        }
        throw error;
      }
    },
  },
});
