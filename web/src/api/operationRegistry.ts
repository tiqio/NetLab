import { generatedApi } from "./index";

export type OperationName =
  | "lab.create"
  | "lab.update"
  | "lab.duplicate"
  | "lab.export"
  | "lab.import"
  | "lab.delete"
  | "node.create"
  | "node.start"
  | "node.stop"
  | "node.delete"
  | "interface.add"
  | "interface.remove"
  | "link.connect"
  | "link.disconnect"
  | "link.reconnect"
  | "guest.exec"
  | "port_mapping.create"
  | "port_mapping.delete"
  | "resources.update"
  | "network_object.create"
  | "network_object.delete"
  | "network_object.attach"
  | "capture.start"
  | "capture.stop"
  | "traffic_filter.start"
  | "traffic_filter.stop"
  | "task.cancel"
  | "topology.positions.update";

export interface OperationDefinition {
  name: OperationName;
  asynchronous: boolean;
  revisionSensitive: boolean;
  idempotent: boolean;
  apiMethod: keyof typeof generatedApi;
}

export const operationRegistry: Record<OperationName, OperationDefinition> = {
  "lab.create": {
    name: "lab.create",
    asynchronous: false,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "createLab",
  },
  "lab.update": {
    name: "lab.update",
    asynchronous: false,
    revisionSensitive: true,
    idempotent: true,
    apiMethod: "updateLab",
  },
  "lab.duplicate": {
    name: "lab.duplicate",
    asynchronous: true,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "duplicateLab",
  },
  "lab.export": {
    name: "lab.export",
    asynchronous: true,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "exportLab",
  },
  "lab.import": {
    name: "lab.import",
    asynchronous: true,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "importLab",
  },
  "lab.delete": {
    name: "lab.delete",
    asynchronous: true,
    revisionSensitive: true,
    idempotent: true,
    apiMethod: "deleteLab",
  },
  "node.create": {
    name: "node.create",
    asynchronous: true,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "createNode",
  },
  "node.start": {
    name: "node.start",
    asynchronous: true,
    revisionSensitive: true,
    idempotent: true,
    apiMethod: "setNodeState",
  },
  "node.stop": {
    name: "node.stop",
    asynchronous: true,
    revisionSensitive: true,
    idempotent: true,
    apiMethod: "setNodeState",
  },
  "node.delete": {
    name: "node.delete",
    asynchronous: true,
    revisionSensitive: true,
    idempotent: true,
    apiMethod: "deleteNode",
  },
  "interface.add": {
    name: "interface.add",
    asynchronous: true,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "addInterface",
  },
  "interface.remove": {
    name: "interface.remove",
    asynchronous: true,
    revisionSensitive: true,
    idempotent: true,
    apiMethod: "removeInterface",
  },
  "link.connect": {
    name: "link.connect",
    asynchronous: true,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "connectLink",
  },
  "link.disconnect": {
    name: "link.disconnect",
    asynchronous: true,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "disconnectLink",
  },
  "link.reconnect": {
    name: "link.reconnect",
    asynchronous: true,
    revisionSensitive: true,
    idempotent: true,
    apiMethod: "reconnectLink",
  },
  "guest.exec": {
    name: "guest.exec",
    asynchronous: true,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "executeGuestCommand",
  },
  "port_mapping.create": {
    name: "port_mapping.create",
    asynchronous: true,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "createPortMapping",
  },
  "port_mapping.delete": {
    name: "port_mapping.delete",
    asynchronous: true,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "deletePortMapping",
  },
  "resources.update": {
    name: "resources.update",
    asynchronous: true,
    revisionSensitive: true,
    idempotent: true,
    apiMethod: "updateNodeResources",
  },
  "network_object.create": {
    name: "network_object.create",
    asynchronous: true,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "createNetworkObject",
  },
  "network_object.delete": {
    name: "network_object.delete",
    asynchronous: true,
    revisionSensitive: true,
    idempotent: true,
    apiMethod: "deleteNetworkObject",
  },
  "network_object.attach": {
    name: "network_object.attach",
    asynchronous: false,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "attachNetworkObject",
  },
  "capture.start": {
    name: "capture.start",
    asynchronous: true,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "startCapture",
  },
  "capture.stop": {
    name: "capture.stop",
    asynchronous: true,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "stopCapture",
  },
  "traffic_filter.start": {
    name: "traffic_filter.start",
    asynchronous: true,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "startTrafficFilter",
  },
  "traffic_filter.stop": {
    name: "traffic_filter.stop",
    asynchronous: true,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "stopTrafficFilter",
  },
  "task.cancel": {
    name: "task.cancel",
    asynchronous: true,
    revisionSensitive: false,
    idempotent: true,
    apiMethod: "cancelTask",
  },
  "topology.positions.update": {
    name: "topology.positions.update",
    asynchronous: false,
    revisionSensitive: true,
    idempotent: true,
    apiMethod: "updateTopologyPlacements",
  },
};
