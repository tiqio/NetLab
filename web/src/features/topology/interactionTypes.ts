import type { Point } from "@/types/workspace";

export type TopologyResourceType =
  | "node"
  | "network_object"
  | "link"
  | "network_attachment"
  | "network_object_link";

export type TopologyConnectionKind =
  "node_link" | "network_attachment" | "network_object_link";

export type TopologyConnectionState =
  "pending" | "connected" | "failed" | "disconnecting" | "unknown";

export type TopologyConnectionSemanticMarker =
  "managed-nat-uplink" | "shared-broadcast-domain";

export interface ConnectionEndpointPresentation {
  resourceId: string;
  resourceType: "node" | "network_object";
  resourceKind: string;
  resourceName: string;
  portId: string;
  portName: string;
  endpointKey: string;
  symbolRole?: "nat" | "shared-domain" | "router";
}

export interface ConnectionStatusVisual {
  state: TopologyConnectionState;
  label: string;
  colorToken: string;
  lineType: "solid" | "dashed" | "dotted";
  width: number;
  cue: "normal" | "transition" | "warning" | "removing" | "unknown";
}

export interface ConnectionPresentationCapabilities {
  selectable: boolean;
  deletable: boolean;
  capturable: boolean;
  trafficFilterable: boolean;
}

export interface ConnectionPresentation {
  id: string;
  persistedKind: TopologyConnectionKind;
  source: ConnectionEndpointPresentation;
  target: ConnectionEndpointPresentation;
  desiredState: string;
  actualState: string;
  statusVisual: ConnectionStatusVisual;
  semanticMarkers: TopologyConnectionSemanticMarker[];
  routeGroupKey: string;
  routeIndex: number;
  routeCount: number;
  label: string;
  capabilities: ConnectionPresentationCapabilities;
  accessibilityLabel: string;
}
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
