import type { OperationTask, Problem } from "@/api/generated";

export type SyncState =
  "initializing" | "live" | "reconnecting" | "refreshing" | "degraded";
export type BottomTab =
  "tasks" | "console" | "captures" | "traffic-filter" | "traffic-workloads";
export type TopologyLabelDensity = "comfortable" | "compact" | "minimal";

export interface PanelPreference {
  collapsed: boolean;
  size: number;
}
export interface Point {
  x: number;
  y: number;
}
export interface Placement extends Point {
  pinned: boolean;
  updatedAt: string;
}
export interface LocalVisualGroup {
  id: string;
  label: string;
  memberResourceIds: string[];
  collapsed: boolean;
  bounds?: Point & { width: number; height: number };
}

export interface WorkspacePreferences {
  schemaVersion: 1;
  laboratoryId: string;
  updatedAt: string;
  panels: {
    devicePalette: PanelPreference;
    inspector: PanelPreference;
    bottomDrawer: PanelPreference;
  };
  viewport: { centerX: number; centerY: number; zoom: number };
  groups: LocalVisualGroup[];
  linkRoutes: Record<string, Point[]>;
  labelDensity: TopologyLabelDensity;
  reducedMotion: boolean;
  activeBottomTab: BottomTab;
}

export interface PendingSubmission {
  operationKey: string;
  idempotencyKey: string;
  submittedAt: string;
  taskId?: string;
  state: "submitting" | "accepted" | "reconciling" | "terminal" | "unknown";
}

export interface ProblemPresentation {
  code: string;
  message: string;
  resource?: string;
  phase?: string;
  retryable: boolean;
  retryAfter?: number;
  cleanup?: string;
  operatorAction?: string;
  details?: Record<string, unknown>;
}

export interface TaskPresentation extends OperationTask {
  progressPercent: number;
  terminal: boolean;
  problem?: ProblemPresentation;
}

export interface DiagnosticSessionPresentation {
  sessionKey: string;
  kind: "telnet" | "vnc" | "capture" | "traffic-filter";
  resourceId: string;
  state:
    "idle" | "connecting" | "connected" | "reconnecting" | "closed" | "failed";
  reconnectCount: number;
  problem?: Problem;
}
