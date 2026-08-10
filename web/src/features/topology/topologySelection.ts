export interface SelectionRectangle {
  left: number;
  top: number;
  right: number;
  bottom: number;
}

export interface SelectableBounds extends SelectionRectangle {
  id: string;
}

export interface SelectionSnapshot<T extends string = string> {
  ids: string[];
  type?: T;
  focusedResourceId: string;
}

export function captureSelection<T extends string>(
  ids: readonly string[],
  type: T | undefined,
  focusedResourceId: string,
): SelectionSnapshot<T> {
  return { ids: [...ids], type, focusedResourceId };
}

export function restoreSelection<T extends string>(
  snapshot: SelectionSnapshot<T>,
) {
  return {
    ids: [...snapshot.ids],
    type: snapshot.type,
    focusedResourceId: snapshot.focusedResourceId,
  };
}

export function selectOne(id: string): string[] {
  return id ? [id] : [];
}

export function selectAll(ids: readonly string[]): string[] {
  return [...new Set(ids.filter(Boolean))];
}

export function toggleSelected(
  selected: readonly string[],
  id: string,
): string[] {
  return selected.includes(id)
    ? selected.filter((item) => item !== id)
    : [...selected, id];
}

export function rangeSelect(
  order: readonly string[],
  anchorId: string,
  targetId: string,
): string[] {
  const anchor = order.indexOf(anchorId);
  const target = order.indexOf(targetId);
  if (anchor < 0 || target < 0) return targetId ? [targetId] : [];
  const start = Math.min(anchor, target);
  const end = Math.max(anchor, target);
  return order.slice(start, end + 1);
}

export function boxSelect(
  rectangle: SelectionRectangle,
  resources: readonly SelectableBounds[],
  selected: readonly string[] = [],
  additive = false,
): string[] {
  const normalized = normalizeRectangle(rectangle);
  const hits = resources
    .filter((resource) => intersects(normalized, resource))
    .map((resource) => resource.id);
  return additive ? [...new Set([...selected, ...hits])] : hits;
}

export function cleanSelection(
  selected: readonly string[],
  available: ReadonlySet<string>,
): string[] {
  return selected.filter((id) => available.has(id));
}

export function normalizeRectangle(
  rectangle: SelectionRectangle,
): SelectionRectangle {
  return {
    left: Math.min(rectangle.left, rectangle.right),
    top: Math.min(rectangle.top, rectangle.bottom),
    right: Math.max(rectangle.left, rectangle.right),
    bottom: Math.max(rectangle.top, rectangle.bottom),
  };
}

function intersects(left: SelectionRectangle, right: SelectionRectangle) {
  return !(
    left.right < right.left ||
    left.left > right.right ||
    left.bottom < right.top ||
    left.top > right.bottom
  );
}
