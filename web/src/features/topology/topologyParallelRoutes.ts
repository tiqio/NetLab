import type { ConnectionPresentation } from "./interactionTypes";

export interface RoutedConnectionPresentation extends ConnectionPresentation {
  curveness: number;
}

function routeGroupKey(value: ConnectionPresentation) {
  return [value.source.resourceId, value.target.resourceId].sort().join(":");
}

export function assignParallelRoutes(
  values: ConnectionPresentation[],
): RoutedConnectionPresentation[] {
  const groups = new Map<string, ConnectionPresentation[]>();
  for (const value of values) {
    const key = routeGroupKey(value);
    const group = groups.get(key) || [];
    group.push(value);
    groups.set(key, group);
  }
  const routeById = new Map<
    string,
    Pick<
      RoutedConnectionPresentation,
      "routeGroupKey" | "routeIndex" | "routeCount" | "curveness"
    >
  >();
  for (const [key, group] of groups) {
    const sorted = [...group].sort((left, right) =>
      left.id.localeCompare(right.id),
    );
    const spacing =
      sorted.length < 2 ? 0 : Math.min(0.22, 0.66 / (sorted.length - 1));
    sorted.forEach((value, index) => {
      const offset = (index - (sorted.length - 1) / 2) * spacing;
      const direction =
        value.source.resourceId <= value.target.resourceId ? 1 : -1;
      routeById.set(value.id, {
        routeGroupKey: key,
        routeIndex: index,
        routeCount: sorted.length,
        curveness: offset * direction,
      });
    });
  }
  return values.map((value) => ({ ...value, ...routeById.get(value.id)! }));
}
