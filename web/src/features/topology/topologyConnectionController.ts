import type { Point } from "@/types/workspace";

export interface ConnectionPort {
  id: string;
  ownerId: string;
  name: string;
  available: boolean;
}

export type ConnectionDecision =
  | { type: "none" }
  | { type: "preview"; sourceInterfaceId: string; pointer: Point }
  | {
      type: "choose_port";
      sourceInterfaceId: string;
      targetResourceId: string;
      candidates: ConnectionPort[];
    }
  | {
      type: "ready";
      sourceInterfaceId: string;
      targetInterfaceId: string;
    }
  | { type: "invalid"; reason: string }
  | { type: "cancelled" };

export class TopologyConnectionController {
  sourceInterfaceId = "";
  targetResourceId = "";
  pointer: Point = { x: 0, y: 0 };

  begin(sourceInterfaceId: string, pointer: Point): ConnectionDecision {
    if (!sourceInterfaceId)
      return { type: "invalid", reason: "source interface is required" };
    this.sourceInterfaceId = sourceInterfaceId;
    this.targetResourceId = "";
    this.pointer = pointer;
    return { type: "preview", sourceInterfaceId, pointer };
  }

  move(pointer: Point): ConnectionDecision {
    if (!this.sourceInterfaceId) return { type: "none" };
    this.pointer = pointer;
    return {
      type: "preview",
      sourceInterfaceId: this.sourceInterfaceId,
      pointer,
    };
  }

  dropOnPort(port: ConnectionPort): ConnectionDecision {
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

  dropOnResource(
    targetResourceId: string,
    ports: ConnectionPort[],
  ): ConnectionDecision {
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

  choose(port: ConnectionPort): ConnectionDecision {
    return this.dropOnPort(port);
  }

  cancel(): ConnectionDecision {
    this.sourceInterfaceId = "";
    this.targetResourceId = "";
    return { type: "cancelled" };
  }
}
