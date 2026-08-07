import type {
  NetworkObject,
  Node,
  NodeInterface,
  PlacementAssignment,
} from "@/api";

export interface AuthoritativeCreation {
  node?: Node;
  interfaces?: NodeInterface[];
  networkObject?: NetworkObject;
  placement_assignment?: PlacementAssignment;
  laboratory_revision?: number;
}

export function applyAuthoritativeCreation(
  value: AuthoritativeCreation,
  actions: {
    merge: (value: AuthoritativeCreation) => void;
    select: (id: string, type: "node" | "network_object") => void;
    focus: (id: string) => void;
    announce: (message: string) => void;
  },
) {
  actions.merge(value);
  const resource = value.node || value.networkObject;
  if (!resource) return;
  actions.select(resource.id, value.node ? "node" : "network_object");
  actions.focus(resource.id);
  actions.announce(
    value.placement_assignment?.adjusted
      ? `已创建 ${resource.name}，为避免重叠已自动放置到附近空白位置。`
      : `已创建 ${resource.name}，可使用“定位所选”查看。`,
  );
}
