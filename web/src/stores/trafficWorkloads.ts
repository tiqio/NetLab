import { defineStore } from "pinia";
import {
  api,
  type CreateTrafficWorkloadRequest,
  type TrafficWorkload,
} from "@/api";

export const useTrafficWorkloadsStore = defineStore("traffic-workloads", {
  state: () => ({ values: [] as TrafficWorkload[], loading: false, error: "" }),
  actions: {
    async load(laboratoryId: string) {
      this.loading = true;
      try {
        this.values = await api.listTrafficWorkloads(laboratoryId);
        this.error = "";
      } catch (error) {
        this.error = error instanceof Error ? error.message : String(error);
      } finally {
        this.loading = false;
      }
    },
    async create(value: CreateTrafficWorkloadRequest) {
      await api.createTrafficWorkload(value);
      await this.load(value.laboratory_id);
    },
    async start(value: TrafficWorkload) {
      await api.startTrafficWorkload(value.id, value.revision);
      await this.load(value.laboratory_id);
    },
    async stop(value: TrafficWorkload) {
      await api.stopTrafficWorkload(value.id, value.revision);
      await this.load(value.laboratory_id);
    },
    async remove(value: TrafficWorkload) {
      await api.deleteTrafficWorkload(value.id, value.revision);
      await this.load(value.laboratory_id);
    },
  },
});
