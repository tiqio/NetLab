import type { TopologyResourceType } from "./interactionTypes";
import type { ConnectionPort } from "./topologyConnectionController";

export interface KeyboardResource {
  id: string;
  type: TopologyResourceType;
  x: number;
  y: number;
}

export interface KeyboardInput {
  key: string;
  shiftKey?: boolean;
  altKey?: boolean;
  ctrlKey?: boolean;
  metaKey?: boolean;
}

export interface KeyboardTransientState {
  connection?: boolean;
  boxSelection?: boolean;
}

export type TopologyKeyboardAction =
  | {
      type: "focus_resource";
      resourceId: string;
      resourceType: TopologyResourceType;
      extend: boolean;
      announcement: string;
    }
  | { type: "focus_port"; interfaceId: string; announcement: string }
  | { type: "choose_port"; interfaceId: string }
  | { type: "toggle_resource"; resourceId: string }
  | { type: "open_inspector"; resourceId: string }
  | { type: "open_terminal"; resourceId: string }
  | { type: "begin_connection"; resourceId: string }
  | { type: "choose_connection_target"; resourceId: string }
  | { type: "disconnect_link"; resourceId: string }
  | { type: "move_selection"; dx: number; dy: number }
  | { type: "zoom_view"; factor: number; announcement: string }
  | { type: "select_all"; resourceIds: string[]; announcement: string }
  | { type: "cancel_connection" }
  | { type: "cancel_box_selection" }
  | { type: "clear_selection" }
  | { type: "none" };

export class TopologyKeyboardController {
  private resources: KeyboardResource[];
  private ports: ConnectionPort[];
  private focusedResourceId = "";
  private focusedPortId = "";

  constructor(
    resources: KeyboardResource[] = [],
    ports: ConnectionPort[] = [],
  ) {
    this.resources = [...resources];
    this.ports = [...ports];
  }

  update(resources: KeyboardResource[], ports: ConnectionPort[]) {
    this.resources = [...resources];
    this.ports = [...ports];
    if (!this.resources.some((item) => item.id === this.focusedResourceId))
      this.focusedResourceId = "";
    if (!this.ports.some((item) => item.id === this.focusedPortId))
      this.focusedPortId = "";
  }

  focusResource(id: string) {
    if (this.focusedResourceId === id) return;
    this.focusedResourceId = id;
    this.focusedPortId = "";
  }

  handle(
    input: KeyboardInput,
    selectedIds: readonly string[],
    transient: KeyboardTransientState = {},
  ): TopologyKeyboardAction {
    if (input.key === "Escape") {
      if (transient.connection) return { type: "cancel_connection" };
      if (transient.boxSelection) return { type: "cancel_box_selection" };
      return { type: "clear_selection" };
    }
    if ((input.ctrlKey || input.metaKey) && input.key.toLowerCase() === "a")
      return {
        type: "select_all",
        resourceIds: this.resources.map((item) => item.id),
        announcement: `已选择全部 ${this.resources.length} 个拓扑资源`,
      };
    if (["+", "=", "Add"].includes(input.key))
      return { type: "zoom_view", factor: 1.2, announcement: "已放大" };
    if (["-", "_", "Subtract"].includes(input.key))
      return { type: "zoom_view", factor: 1 / 1.2, announcement: "已缩小" };
    if (input.altKey && input.key.startsWith("Arrow"))
      return movement(input.key);
    if (this.focusedPortId) return this.handlePort(input.key);
    if (["ArrowRight", "ArrowDown", "ArrowLeft", "ArrowUp"].includes(input.key))
      return this.navigateResource(input.key, Boolean(input.shiftKey));
    if (input.key === " " && this.focusedResourceId)
      return { type: "toggle_resource", resourceId: this.focusedResourceId };
    if (input.key.toLowerCase() === "c" && this.focusedResourceId) {
      const resource = this.resources.find(
        (item) => item.id === this.focusedResourceId,
      );
      if (resource?.type === "node" || resource?.type === "network_object")
        return { type: "begin_connection", resourceId: resource.id };
    }
    if (input.key.toLowerCase() === "t" && this.focusedResourceId) {
      const resource = this.resources.find(
        (item) => item.id === this.focusedResourceId,
      );
      if (resource?.type === "node")
        return { type: "open_terminal", resourceId: resource.id };
    }
    if (input.key === "Enter" && transient.connection && this.focusedResourceId)
      return {
        type: "choose_connection_target",
        resourceId: this.focusedResourceId,
      };
    if (input.key === "Delete" && this.focusedResourceId) {
      const resource = this.resources.find(
        (item) => item.id === this.focusedResourceId,
      );
      if (resource?.type === "link")
        return { type: "disconnect_link", resourceId: resource.id };
    }
    if (input.key.toLowerCase() === "p" && this.focusedResourceId) {
      const port = this.ports.find(
        (item) => item.ownerId === this.focusedResourceId && item.available,
      );
      if (port) {
        this.focusedPortId = port.id;
        return {
          type: "focus_port",
          interfaceId: port.id,
          announcement: `接口 ${port.name}`,
        };
      }
    }
    if (input.key === "Enter" && this.focusedResourceId)
      return { type: "open_inspector", resourceId: this.focusedResourceId };
    if (!selectedIds.length) return { type: "none" };
    return { type: "none" };
  }

  private navigateResource(
    key: string,
    extend: boolean,
  ): TopologyKeyboardAction {
    if (!this.resources.length) return { type: "none" };
    const current = this.resources.findIndex(
      (item) => item.id === this.focusedResourceId,
    );
    const delta = key === "ArrowRight" || key === "ArrowDown" ? 1 : -1;
    const next =
      this.resources[
        current < 0
          ? delta > 0
            ? 0
            : this.resources.length - 1
          : (current + delta + this.resources.length) % this.resources.length
      ];
    this.focusedResourceId = next.id;
    return {
      type: "focus_resource",
      resourceId: next.id,
      resourceType: next.type,
      extend,
      announcement: `${next.type === "node" ? "节点" : next.type === "link" ? "链路" : "网络对象"} ${next.id}`,
    };
  }

  private handlePort(key: string): TopologyKeyboardAction {
    const owner = this.ports.find(
      (item) => item.id === this.focusedPortId,
    )?.ownerId;
    const ports = this.ports.filter((item) => item.ownerId === owner);
    const current = ports.findIndex((item) => item.id === this.focusedPortId);
    if (["ArrowDown", "ArrowRight", "ArrowUp", "ArrowLeft"].includes(key)) {
      const delta = key === "ArrowDown" || key === "ArrowRight" ? 1 : -1;
      const next = ports[(current + delta + ports.length) % ports.length];
      this.focusedPortId = next.id;
      return {
        type: "focus_port",
        interfaceId: next.id,
        announcement: `接口 ${next.name}`,
      };
    }
    if (key === "Enter") {
      const interfaceId = this.focusedPortId;
      this.focusedPortId = "";
      return { type: "choose_port", interfaceId };
    }
    if (key === "Escape") {
      this.focusedPortId = "";
      return { type: "clear_selection" };
    }
    return { type: "none" };
  }
}

function movement(key: string): TopologyKeyboardAction {
  if (key === "ArrowLeft") return { type: "move_selection", dx: -10, dy: 0 };
  if (key === "ArrowRight") return { type: "move_selection", dx: 10, dy: 0 };
  if (key === "ArrowUp") return { type: "move_selection", dx: 0, dy: -10 };
  return { type: "move_selection", dx: 0, dy: 10 };
}
