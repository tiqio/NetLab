import type { Point } from "@/types/workspace";

export type TopologyResourceType =
  | "node"
  | "network_object"
  | "link"
  | "network_attachment"
  | "network_object_link";
export type TopologyInteractionMode =
  | "idle"
  | "pressing"
  | "panning"
  | "box_selecting"
  | "dragging_resources"
  | "connecting"
  | "choosing_target_port"
  | "cancelling";

export interface PointerSample extends Point {
  pointerId: number;
  button: number;
  time: number;
}

export interface ViewportState {
  centerX: number;
  centerY: number;
  zoom: number;
}

export interface ResourceSelection {
  ids: string[];
  anchorId?: string;
}

export interface ConnectionDraft {
  sourceInterfaceId: string;
  pointer: Point;
  targetResourceId?: string;
  targetInterfaceId?: string;
  valid: boolean;
  reason?: string;
}

export interface TopologyInteractionState {
  mode: TopologyInteractionMode;
  origin?: PointerSample;
  current?: PointerSample;
  resourceId?: string;
  resourceType?: TopologyResourceType;
  boxSelection?: boolean;
  selection: ResourceSelection;
  connection?: ConnectionDraft;
}
