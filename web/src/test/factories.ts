import type {
  Laboratory,
  Link,
  NetworkAttachment,
  NetworkObject,
  Node,
  NodeInterface,
  OperationTask,
  TopologySnapshot,
} from "@/api/generated";
export const laboratoryFactory = (
  value: Partial<Laboratory> = {},
): Laboratory => ({
  id: "lab-1",
  name: "Demo lab",
  description: "",
  revision: 1,
  recovery_policy: "auto_restore",
  lifecycle_state: "active",
  ...value,
});
export const nodeFactory = (value: Partial<Node> = {}): Node => ({
  id: "node-1",
  laboratory_id: "lab-1",
  name: "Ubuntu",
  kind: "qemu",
  revision: 1,
  desired_state: "stopped",
  observed_state: "stopped",
  cpu_count: 2,
  cpu_quota_micros: 100000,
  memory_mib: 1024,
  storage_gib: 8,
  interface_limit: 8,
  process_limit: 128,
  ...value,
});
export const interfaceFactory = (
  value: Partial<NodeInterface> = {},
): NodeInterface => ({
  id: "interface-1",
  node_id: "node-1",
  slot: 0,
  name: "eth0",
  driver: "virtio-net-pci",
  mac_address: "02:00:00:00:00:01",
  operational_state: "down",
  revision: 1,
  ...value,
});
export const linkFactory = (value: Partial<Link> = {}): Link => ({
  id: "link-1",
  laboratory_id: "lab-1",
  endpoint_a_id: "interface-1",
  endpoint_b_id: "interface-2",
  revision: 1,
  desired_state: "connected",
  observed_state: "connected",
  ...value,
});
export const networkObjectFactory = (
  value: Partial<NetworkObject> = {},
): NetworkObject => ({
  id: "network-1",
  laboratory_id: "lab-1",
  name: "Bridge",
  kind: "bridge",
  revision: 1,
  desired_state: "ready",
  observed_state: "ready",
  config: {},
  ...value,
});
export const networkAttachmentFactory = (
  value: Partial<NetworkAttachment> = {},
): NetworkAttachment => ({
  id: "attachment-1",
  network_object_id: "network-1",
  interface_id: "interface-1",
  port_name: "eth0",
  revision: 1,
  observed_state: "active",
  ...value,
});
export const multiInterfaceNodeFactory = (count = 4) => ({
  node: nodeFactory({ interface_limit: Math.max(count, 1) }),
  interfaces: Array.from({ length: count }, (_, index) =>
    interfaceFactory({
      id: `interface-${index + 1}`,
      slot: index,
      name: `eth${index}`,
      mac_address: `02:00:00:00:00:${String(index + 1).padStart(2, "0")}`,
    }),
  ),
});
export const denseTopologyFactory = (nodeCount = 100, linkCount = 200) => {
  const nodes = Array.from({ length: nodeCount }, (_, index) =>
    nodeFactory({ id: `node-${index + 1}`, name: `Node ${index + 1}` }),
  );
  const interfaces = nodes.flatMap((node, nodeIndex) =>
    [0, 1, 2, 3].map((slot) =>
      interfaceFactory({
        id: `interface-${nodeIndex + 1}-${slot}`,
        node_id: node.id,
        slot,
        name: `eth${slot}`,
      }),
    ),
  );
  const links = Array.from({ length: linkCount }, (_, index) =>
    linkFactory({
      id: `link-${index + 1}`,
      endpoint_a_id: interfaces[(index * 2) % interfaces.length].id,
      endpoint_b_id: interfaces[(index * 2 + 1) % interfaces.length].id,
    }),
  );
  return snapshotFactory({ nodes, interfaces, links });
};
export const taskFactory = (
  value: Partial<OperationTask> = {},
): OperationTask => ({
  id: "task-1",
  kind: "node.start",
  resource_type: "node",
  resource_id: "node-1",
  state: "running",
  progress_current: 1,
  progress_total: 2,
  created_at: "2026-07-27T00:00:00Z",
  ...value,
});
export const snapshotFactory = (
  value: Partial<TopologySnapshot> = {},
): TopologySnapshot => ({
  laboratory: laboratoryFactory(),
  nodes: [nodeFactory()],
  interfaces: [],
  links: [],
  network_objects: [],
  placements: [],
  event_sequence: 1,
  ...value,
});
