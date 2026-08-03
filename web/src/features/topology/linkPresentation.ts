import type { Link, Node, NodeInterface } from "@/api";

export function linkEndpointName(
  interfaceId: string,
  interfaces: NodeInterface[],
  nodes: Node[],
) {
  const interfaceValue = interfaces.find((item) => item.id === interfaceId);
  if (!interfaceValue) return interfaceId;
  const node = nodes.find((item) => item.id === interfaceValue.node_id);
  return `${node?.name || interfaceValue.node_id}:${interfaceValue.name}`;
}

export function linkDisplayName(
  link: Link,
  interfaces: NodeInterface[],
  nodes: Node[],
) {
  return `${linkEndpointName(link.endpoint_a_id, interfaces, nodes)} ↔ ${linkEndpointName(link.endpoint_b_id, interfaces, nodes)}`;
}

export function parallelLinkCurveness(
  link: Link,
  links: Link[],
  owners: Record<string, string>,
) {
  const ownerA = owners[link.endpoint_a_id];
  const ownerB = owners[link.endpoint_b_id];
  if (!ownerA || !ownerB || ownerA === ownerB) return 0;
  const pair = [ownerA, ownerB].sort().join(":");
  const siblings = links
    .filter((item) => {
      const left = owners[item.endpoint_a_id];
      const right = owners[item.endpoint_b_id];
      return left && right && [left, right].sort().join(":") === pair;
    })
    .sort((left, right) => left.id.localeCompare(right.id));
  if (siblings.length < 2) return 0;
  const index = siblings.findIndex((item) => item.id === link.id);
  if (index < 0) return 0;
  const spacing = Math.min(0.24, 0.84 / (siblings.length - 1));
  const offset = (index - (siblings.length - 1) / 2) * spacing;
  return ownerA <= ownerB ? offset : -offset;
}
