export interface TrafficObservation {
  fingerprint: string;
  resource_type?: string;
  resource_id?: string;
  interface_id: string;
  link_id?: string;
  network_object_link_id?: string;
  direction: string;
  first_seen: string;
  last_seen: string;
  count: number;
  bytes: number;
}

export type TrafficDirectionMode = "single" | "bidirectional" | "unknown";

export interface TrafficPathOverlay {
  id: string;
  connectionId: string;
  x1: number;
  y1: number;
  x2: number;
  y2: number;
  pathData: string;
  mode: TrafficDirectionMode;
  guideMode: "single" | "initiator" | "none";
  particleMode: TrafficDirectionMode;
  particlesActive: boolean;
  sourceId: string;
  targetId: string;
  count: number;
  bytes: number;
  expiresAt: number;
}
