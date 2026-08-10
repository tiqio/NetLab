import type { Point } from "@/types/workspace";
import type { ViewportState } from "./interactionTypes";

export const MIN_ZOOM = 0.1;
export const MAX_ZOOM = 8;
export const DEFAULT_DRAG_THRESHOLD = 5;

export type PortTrackSide = "top" | "right" | "bottom" | "left";

export interface PortTrackPosition extends Point {
  side: PortTrackSide;
  labelX: number;
  labelY: number;
  textAnchor: "start" | "middle" | "end";
}

const PORT_SIDE_ORDER: PortTrackSide[] = ["right", "bottom", "left", "top"];

export function clamp(value: number, minimum: number, maximum: number) {
  return Math.min(maximum, Math.max(minimum, value));
}

export function distance(left: Point, right: Point) {
  return Math.hypot(right.x - left.x, right.y - left.y);
}

export function exceedsDragThreshold(
  origin: Point,
  current: Point,
  threshold = DEFAULT_DRAG_THRESHOLD,
) {
  return distance(origin, current) >= Math.max(0, threshold);
}

export function screenToWorld(point: Point, viewport: ViewportState): Point {
  return {
    x: (point.x - viewport.centerX) / viewport.zoom,
    y: (point.y - viewport.centerY) / viewport.zoom,
  };
}

export function worldToScreen(point: Point, viewport: ViewportState): Point {
  return {
    x: point.x * viewport.zoom + viewport.centerX,
    y: point.y * viewport.zoom + viewport.centerY,
  };
}

export function zoomAroundPoint(
  viewport: ViewportState,
  screenPoint: Point,
  zoomFactor: number,
): ViewportState {
  const worldPoint = screenToWorld(screenPoint, viewport);
  const zoom = clamp(viewport.zoom * zoomFactor, MIN_ZOOM, MAX_ZOOM);
  return {
    zoom,
    centerX: screenPoint.x - worldPoint.x * zoom,
    centerY: screenPoint.y - worldPoint.y * zoom,
  };
}

export function fitViewport(
  points: Point[],
  width: number,
  height: number,
  padding = 80,
): ViewportState {
  if (!points.length || width <= 0 || height <= 0)
    return { centerX: 0, centerY: 0, zoom: 1 };
  const left = Math.min(...points.map((point) => point.x));
  const right = Math.max(...points.map((point) => point.x));
  const top = Math.min(...points.map((point) => point.y));
  const bottom = Math.max(...points.map((point) => point.y));
  const availableWidth = Math.max(1, width - padding * 2);
  const availableHeight = Math.max(1, height - padding * 2);
  const zoom = clamp(
    Math.min(
      availableWidth / Math.max(right - left, 1),
      availableHeight / Math.max(bottom - top, 1),
    ),
    MIN_ZOOM,
    MAX_ZOOM,
  );
  const centerWorldX = (left + right) / 2;
  const centerWorldY = (top + bottom) / 2;
  return {
    zoom,
    centerX: centerWorldX,
    centerY: centerWorldY,
  };
}

export function pointInCircle(point: Point, center: Point, radius: number) {
  return distance(point, center) <= Math.max(0, radius);
}

export function nearestPoint(
  point: Point,
  candidates: Array<Point & { id: string }>,
  radius: number,
) {
  return candidates
    .map((candidate) => ({ candidate, distance: distance(point, candidate) }))
    .filter((value) => value.distance <= radius)
    .sort((left, right) => left.distance - right.distance)[0]?.candidate;
}

export function nearestPortHit<T extends Point & { id: string }>(
  point: Point,
  candidates: T[],
  radius = 14,
) {
  return nearestPoint(point, candidates, radius) as T | undefined;
}

export interface ResourceBodyFootprint {
  id: string;
  center: Point;
  halfWidth: number;
  halfHeight: number;
}

export function resolveResourceBodyHit<T extends ResourceBodyFootprint>(
  point: Point,
  resources: T[],
) {
  return resources
    .filter(
      (resource) =>
        Math.abs(point.x - resource.center.x) <= resource.halfWidth &&
        Math.abs(point.y - resource.center.y) <= resource.halfHeight,
    )
    .sort((left, right) => {
      const leftDistance = distance(point, left.center);
      const rightDistance = distance(point, right.center);
      return leftDistance - rightDistance;
    })[0];
}

export function quadraticRoute(
  source: Point,
  target: Point,
  offset = 0,
): [Point, Point, Point] {
  const midpoint = {
    x: (source.x + target.x) / 2,
    y: (source.y + target.y) / 2,
  };
  const length = Math.max(distance(source, target), 1);
  const normal = {
    x: -(target.y - source.y) / length,
    y: (target.x - source.x) / length,
  };
  return [
    source,
    { x: midpoint.x + normal.x * offset, y: midpoint.y + normal.y * offset },
    target,
  ];
}

export function deterministicPortTrack(
  count: number,
  center: Point = { x: 0, y: 0 },
  horizontalRadius = 48,
  verticalRadius = 40,
): PortTrackPosition[] {
  const portCount = Math.max(0, Math.floor(count));
  if (!portCount) return [];

  const sideCounts = [0, 0, 0, 0];
  for (let index = 0; index < portCount; index += 1)
    sideCounts[index % PORT_SIDE_ORDER.length] += 1;

  const sideIndexes = [0, 0, 0, 0];
  return Array.from({ length: portCount }, (_, index) => {
    const sideIndex = index % PORT_SIDE_ORDER.length;
    const side = PORT_SIDE_ORDER[sideIndex];
    const slot = sideIndexes[sideIndex]++;
    const slots = sideCounts[sideIndex];
    const ratio = slots === 1 ? 0 : (slot + 1) / (slots + 1) - 0.5;
    const xOffset = ratio * horizontalRadius * 1.35;
    const yOffset = ratio * verticalRadius * 1.35;

    if (side === "right")
      return {
        x: center.x + horizontalRadius,
        y: center.y + yOffset,
        side,
        labelX: center.x + horizontalRadius + 11,
        labelY: center.y + yOffset + 3,
        textAnchor: "start",
      };
    if (side === "left")
      return {
        x: center.x - horizontalRadius,
        y: center.y + yOffset,
        side,
        labelX: center.x - horizontalRadius - 11,
        labelY: center.y + yOffset + 3,
        textAnchor: "end",
      };
    if (side === "bottom")
      return {
        x: center.x + xOffset,
        y: center.y + verticalRadius,
        side,
        labelX: center.x + xOffset,
        labelY: center.y + verticalRadius + 15,
        textAnchor: "middle",
      };
    return {
      x: center.x + xOffset,
      y: center.y - verticalRadius,
      side,
      labelX: center.x + xOffset,
      labelY: center.y - verticalRadius - 10,
      textAnchor: "middle",
    };
  });
}

export function topologyLabelPriority(
  zoom: number,
  density: "comfortable" | "compact" | "minimal",
) {
  if (density === "minimal" || zoom < 0.55) return "identity";
  if (density === "compact" || zoom < 0.9) return "identity-state";
  return "full";
}
