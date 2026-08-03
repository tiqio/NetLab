import type { Point } from "@/types/workspace";
import type { ViewportState } from "./interactionTypes";

export const MIN_ZOOM = 0.1;
export const MAX_ZOOM = 8;
export const DEFAULT_DRAG_THRESHOLD = 5;

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
