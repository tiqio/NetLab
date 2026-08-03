import type { Point } from "@/types/workspace";
import type {
  ConnectionDraft,
  PointerSample,
  TopologyInteractionState,
  TopologyResourceType,
  ViewportState,
} from "./interactionTypes";
import type { ConnectionPort } from "./topologyConnectionController";
import {
  DEFAULT_DRAG_THRESHOLD,
  exceedsDragThreshold,
  screenToWorld,
  zoomAroundPoint,
} from "./topologyGeometry";

export type InteractionTarget =
  | { kind: "background" }
  | { kind: "resource"; id: string; resourceType: TopologyResourceType };

export type InteractionAction =
  | { type: "capture_pointer"; pointerId: number }
  | { type: "release_pointer"; pointerId: number }
  | { type: "select"; id: string; resourceType: TopologyResourceType }
  | { type: "pan_preview"; dx: number; dy: number }
  | { type: "pan_commit"; dx: number; dy: number }
  | { type: "drag_preview"; ids: string[]; dx: number; dy: number }
  | { type: "drag_commit"; ids: string[]; dx: number; dy: number }
  | {
      type: "box_preview";
      left: number;
      top: number;
      right: number;
      bottom: number;
    }
  | {
      type: "box_commit";
      left: number;
      top: number;
      right: number;
      bottom: number;
    }
  | { type: "cancel" }
  | { type: "connection_preview"; connection: ConnectionDraft }
  | {
      type: "connection_choose_port";
      connection: ConnectionDraft;
      candidates: ConnectionPort[];
    }
  | {
      type: "connection_ready";
      sourceInterfaceId: string;
      targetInterfaceId: string;
    }
  | { type: "connection_invalid"; connection: ConnectionDraft }
  | { type: "viewport"; viewport: ViewportState };

export class TopologyInteractionController {
  state: TopologyInteractionState = { mode: "idle", selection: { ids: [] } };
  private threshold: number;

  constructor(threshold = DEFAULT_DRAG_THRESHOLD) {
    this.threshold = threshold;
  }

  pointerDown(
    sample: PointerSample,
    target: InteractionTarget,
    selectedIds: string[] = [],
    boxSelection = false,
  ): InteractionAction[] {
    if (sample.button !== 0) return [];
    this.state = {
      mode: "pressing",
      origin: sample,
      current: sample,
      resourceId: target.kind === "resource" ? target.id : undefined,
      resourceType:
        target.kind === "resource" ? target.resourceType : undefined,
      boxSelection: boxSelection && target.kind === "background",
      selection: { ids: [...selectedIds] },
    };
    return [{ type: "capture_pointer", pointerId: sample.pointerId }];
  }

  pointerMove(sample: PointerSample): InteractionAction[] {
    if (!this.state.origin || sample.pointerId !== this.state.origin.pointerId)
      return [];
    this.state.current = sample;
    const dx = sample.x - this.state.origin.x;
    const dy = sample.y - this.state.origin.y;
    if (
      this.state.mode === "pressing" &&
      exceedsDragThreshold(this.state.origin, sample, this.threshold)
    ) {
      this.state.mode = this.state.boxSelection
        ? "box_selecting"
        : this.state.resourceId
          ? "dragging_resources"
          : "panning";
    }
    if (this.state.mode === "panning") return [{ type: "pan_preview", dx, dy }];
    if (this.state.mode === "dragging_resources")
      return [{ type: "drag_preview", ids: this.dragIds(), dx, dy }];
    if (this.state.mode === "box_selecting")
      return [{ type: "box_preview", ...rectangle(this.state.origin, sample) }];
    return [];
  }

  pointerUp(sample: PointerSample): InteractionAction[] {
    if (!this.state.origin || sample.pointerId !== this.state.origin.pointerId)
      return [];
    const previous = this.state;
    const dx = sample.x - previous.origin!.x;
    const dy = sample.y - previous.origin!.y;
    this.state = { mode: "idle", selection: previous.selection };
    const actions: InteractionAction[] = [];
    if (previous.mode === "pressing" && previous.resourceId)
      actions.push({
        type: "select",
        id: previous.resourceId,
        resourceType: previous.resourceType!,
      });
    if (previous.mode === "panning")
      actions.push({ type: "pan_commit", dx, dy });
    if (previous.mode === "dragging_resources")
      actions.push({
        type: "drag_commit",
        ids: this.dragIds(previous),
        dx,
        dy,
      });
    if (previous.mode === "box_selecting")
      actions.push({
        type: "box_commit",
        ...rectangle(previous.origin!, sample),
      });
    actions.push({ type: "release_pointer", pointerId: sample.pointerId });
    return actions;
  }

  cancel(): InteractionAction[] {
    if (this.state.mode === "idle") return [];
    const pointerId = this.state.origin?.pointerId;
    const selection = this.state.selection;
    this.state = { mode: "idle", selection };
    return [
      { type: "cancel" },
      ...(pointerId === undefined
        ? []
        : ([{ type: "release_pointer", pointerId }] as InteractionAction[])),
    ];
  }

  beginConnection(
    sourceInterfaceId: string,
    pointer: Point,
  ): InteractionAction[] {
    if (!sourceInterfaceId) return [];
    const connection: ConnectionDraft = {
      sourceInterfaceId,
      pointer,
      valid: true,
    };
    this.state = {
      mode: "connecting",
      selection: this.state.selection,
      connection,
    };
    return [{ type: "connection_preview", connection }];
  }

  moveConnection(pointer: Point): InteractionAction[] {
    if (
      !this.state.connection ||
      !["connecting", "choosing_target_port"].includes(this.state.mode)
    )
      return [];
    this.state.connection = { ...this.state.connection, pointer };
    return [{ type: "connection_preview", connection: this.state.connection }];
  }

  chooseConnectionTarget(
    targetResourceId: string,
    ports: ConnectionPort[],
  ): InteractionAction[] {
    if (!this.state.connection) return [];
    const candidates = ports.filter(
      (port) =>
        port.available && port.id !== this.state.connection?.sourceInterfaceId,
    );
    if (!candidates.length) {
      this.state.connection = {
        ...this.state.connection,
        targetResourceId,
        valid: false,
        reason: "target has no available interfaces",
      };
      return [
        { type: "connection_invalid", connection: this.state.connection },
      ];
    }
    if (candidates.length === 1)
      return this.chooseConnectionPort(candidates[0]);
    this.state.mode = "choosing_target_port";
    this.state.connection = {
      ...this.state.connection,
      targetResourceId,
      valid: true,
      reason: undefined,
    };
    return [
      {
        type: "connection_choose_port",
        connection: this.state.connection,
        candidates,
      },
    ];
  }

  chooseConnectionPort(port: ConnectionPort): InteractionAction[] {
    if (
      !this.state.connection ||
      !port.available ||
      port.id === this.state.connection.sourceInterfaceId
    )
      return [];
    const sourceInterfaceId = this.state.connection.sourceInterfaceId;
    const selection = this.state.selection;
    this.state = { mode: "idle", selection };
    return [
      {
        type: "connection_ready",
        sourceInterfaceId,
        targetInterfaceId: port.id,
      },
    ];
  }

  wheel(
    viewport: ViewportState,
    point: Point,
    deltaY: number,
    ctrlKey = false,
  ): InteractionAction[] {
    if (this.state.mode !== "idle" && this.state.mode !== "pressing") return [];
    const factor = Math.exp(-deltaY * (ctrlKey ? 0.006 : 0.002));
    return [
      { type: "viewport", viewport: zoomAroundPoint(viewport, point, factor) },
    ];
  }

  roam(
    viewport: ViewportState,
    value: { zoom?: number; dx?: number; dy?: number },
  ): InteractionAction[] {
    if (this.state.mode !== "idle") return [];
    return [
      {
        type: "viewport",
        viewport: {
          centerX: viewport.centerX + (value.dx || 0),
          centerY: viewport.centerY + (value.dy || 0),
          zoom: Math.min(8, Math.max(0.1, viewport.zoom * (value.zoom || 1))),
        },
      },
    ];
  }

  worldDelta(viewport: ViewportState, dx: number, dy: number) {
    const origin = screenToWorld({ x: 0, y: 0 }, viewport);
    const target = screenToWorld({ x: dx, y: dy }, viewport);
    return { x: target.x - origin.x, y: target.y - origin.y };
  }

  private dragIds(state = this.state) {
    if (!state.resourceId) return [];
    return state.selection.ids.includes(state.resourceId)
      ? [...state.selection.ids]
      : [state.resourceId];
  }
}

function rectangle(origin: Point, current: Point) {
  return {
    left: Math.min(origin.x, current.x),
    top: Math.min(origin.y, current.y),
    right: Math.max(origin.x, current.x),
    bottom: Math.max(origin.y, current.y),
  };
}
