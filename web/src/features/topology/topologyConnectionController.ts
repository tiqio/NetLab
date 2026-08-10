import type { Point } from "@/types/workspace";
import {
  endpointKey,
  endpointsCompatible,
  type UnifiedConnectionBackingKind,
  type UnifiedConnectionEndpoint,
} from "./topologyEndpointCompatibility";

export interface ConnectionPort {
  id: string;
  ownerId: string;
  name: string;
  available: boolean;
  endpoint?: UnifiedConnectionEndpoint;
}

export type ConnectionDraftPhase =
  "idle" | "dragging" | "choosing" | "configuring" | "submitting";

export type ConnectionDecision =
  | { type: "none" }
  | {
      type: "preview";
      sourceInterfaceId: string;
      pointer: Point;
      source?: UnifiedConnectionEndpoint;
      candidate?: UnifiedConnectionEndpoint;
      valid?: boolean;
      reason?: string;
      backingKind?: UnifiedConnectionBackingKind;
      phase?: ConnectionDraftPhase;
    }
  | {
      type: "choose_port";
      sourceInterfaceId: string;
      targetResourceId: string;
      candidates: ConnectionPort[];
    }
  | {
      type: "choose_endpoint";
      source: UnifiedConnectionEndpoint;
      targetResourceId: string;
      candidates: UnifiedConnectionEndpoint[];
      phase: "choosing";
    }
  | {
      type: "ready";
      sourceInterfaceId: string;
      targetInterfaceId: string;
    }
  | {
      type: "ready_endpoint";
      source: UnifiedConnectionEndpoint;
      target: UnifiedConnectionEndpoint;
      backingKind: UnifiedConnectionBackingKind;
      phase: "configuring";
    }
  | {
      type: "submitting";
      source: UnifiedConnectionEndpoint;
      target: UnifiedConnectionEndpoint;
      backingKind: UnifiedConnectionBackingKind;
      phase: "submitting";
    }
  | { type: "invalid"; reason: string }
  | { type: "cancelled"; reason?: string };

export class TopologyConnectionController {
  sourceInterfaceId = "";
  targetResourceId = "";
  pointer: Point = { x: 0, y: 0 };
  source?: UnifiedConnectionEndpoint;
  target?: UnifiedConnectionEndpoint;
  candidate?: UnifiedConnectionEndpoint;
  backingKind?: UnifiedConnectionBackingKind;
  phase: ConnectionDraftPhase = "idle";

  begin(sourceInterfaceId: string, pointer: Point): ConnectionDecision {
    if (!sourceInterfaceId)
      return { type: "invalid", reason: "source interface is required" };
    this.reset();
    this.sourceInterfaceId = sourceInterfaceId;
    this.pointer = pointer;
    this.phase = "dragging";
    return { type: "preview", sourceInterfaceId, pointer };
  }

  beginEndpoint(
    source: UnifiedConnectionEndpoint,
    pointer: Point,
  ): ConnectionDecision {
    if (!endpointUsable(source))
      return {
        type: "invalid",
        reason: source.unavailableReason || "endpoint_unavailable",
      };
    this.reset();
    this.source = source;
    this.sourceInterfaceId = source.portId || endpointKey(source);
    this.pointer = pointer;
    this.phase = "dragging";
    return this.previewDecision();
  }

  move(pointer: Point): ConnectionDecision {
    if (!this.sourceInterfaceId && !this.source) return { type: "none" };
    this.pointer = pointer;
    return this.previewDecision();
  }

  setCandidate(candidate?: UnifiedConnectionEndpoint): ConnectionDecision {
    if (!this.source)
      return { type: "invalid", reason: "connection source is missing" };
    this.candidate = candidate;
    if (!candidate) return this.previewDecision();
    const compatibility = endpointsCompatible(this.source, candidate);
    return this.previewDecision(
      compatibility.compatible,
      compatibility.reason,
      compatibility.backingKind,
    );
  }

  dropOnPort(port: ConnectionPort): ConnectionDecision {
    if (port.endpoint) return this.dropOnEndpoint(port.endpoint);
    if (!this.sourceInterfaceId)
      return { type: "invalid", reason: "connection source is missing" };
    if (!port.available || port.id === this.sourceInterfaceId)
      return { type: "invalid", reason: "target interface is unavailable" };
    return {
      type: "ready",
      sourceInterfaceId: this.sourceInterfaceId,
      targetInterfaceId: port.id,
    };
  }

  dropOnEndpoint(target: UnifiedConnectionEndpoint): ConnectionDecision {
    if (!this.source)
      return { type: "invalid", reason: "connection source is missing" };
    if (endpointKey(this.source) === endpointKey(target)) {
      this.reset();
      return { type: "cancelled", reason: "same_endpoint" };
    }
    const compatibility = endpointsCompatible(this.source, target);
    if (!compatibility.compatible || !compatibility.backingKind) {
      this.candidate = target;
      return {
        type: "invalid",
        reason: compatibility.reason || "endpoint_incompatible",
      };
    }
    this.target = target;
    this.candidate = target;
    this.backingKind = compatibility.backingKind;
    this.phase = "configuring";
    return {
      type: "ready_endpoint",
      source: this.source,
      target,
      backingKind: compatibility.backingKind,
      phase: "configuring",
    };
  }

  dropOnResource(
    targetResourceId: string,
    ports: ConnectionPort[],
  ): ConnectionDecision {
    if (this.source && ports.some((port) => port.endpoint))
      return this.dropOnResourceEndpoints(
        targetResourceId,
        ports.flatMap((port) => (port.endpoint ? [port.endpoint] : [])),
      );
    if (!this.sourceInterfaceId)
      return { type: "invalid", reason: "connection source is missing" };
    const candidates = ports.filter(
      (port) => port.available && port.id !== this.sourceInterfaceId,
    );
    if (!candidates.length)
      return { type: "invalid", reason: "target has no available interfaces" };
    this.targetResourceId = targetResourceId;
    if (candidates.length === 1) return this.dropOnPort(candidates[0]);
    return {
      type: "choose_port",
      sourceInterfaceId: this.sourceInterfaceId,
      targetResourceId,
      candidates,
    };
  }

  dropOnResourceEndpoints(
    targetResourceId: string,
    endpoints: UnifiedConnectionEndpoint[],
  ): ConnectionDecision {
    if (!this.source)
      return { type: "invalid", reason: "connection source is missing" };
    const candidates = endpoints.filter(
      (endpoint) => endpointsCompatible(this.source!, endpoint).compatible,
    );
    if (!candidates.length)
      return { type: "invalid", reason: "target has no compatible endpoints" };
    this.targetResourceId = targetResourceId;
    if (candidates.length === 1) return this.dropOnEndpoint(candidates[0]);
    this.phase = "choosing";
    return {
      type: "choose_endpoint",
      source: this.source,
      targetResourceId,
      candidates,
      phase: "choosing",
    };
  }

  choose(port: ConnectionPort): ConnectionDecision {
    return this.dropOnPort(port);
  }

  chooseEndpoint(endpoint: UnifiedConnectionEndpoint): ConnectionDecision {
    return this.dropOnEndpoint(endpoint);
  }

  markSubmitting(): ConnectionDecision {
    if (!this.source || !this.target || !this.backingKind)
      return { type: "invalid", reason: "connection target is missing" };
    this.phase = "submitting";
    return {
      type: "submitting",
      source: this.source,
      target: this.target,
      backingKind: this.backingKind,
      phase: "submitting",
    };
  }

  cancel(): ConnectionDecision {
    this.reset();
    return { type: "cancelled" };
  }

  private previewDecision(
    valid?: boolean,
    reason?: string,
    backingKind?: UnifiedConnectionBackingKind,
  ): ConnectionDecision {
    return {
      type: "preview",
      sourceInterfaceId: this.sourceInterfaceId,
      pointer: this.pointer,
      source: this.source,
      candidate: this.candidate,
      valid,
      reason,
      backingKind,
      phase: this.phase,
    };
  }

  private reset() {
    this.sourceInterfaceId = "";
    this.targetResourceId = "";
    this.source = undefined;
    this.target = undefined;
    this.candidate = undefined;
    this.backingKind = undefined;
    this.phase = "idle";
  }
}

function endpointUsable(endpoint: UnifiedConnectionEndpoint) {
  return !endpoint.availability || endpoint.availability === "free";
}
