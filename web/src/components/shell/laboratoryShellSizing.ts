export const BOTTOM_DRAWER_MIN_HEIGHT = 160;
export const CANVAS_MIN_HEIGHT = 180;
export const RESIZE_HANDLE_HEIGHT = 8;

export function clampBottomDrawerSize(size: number, availableHeight: number) {
  const maximum = Math.max(
    BOTTOM_DRAWER_MIN_HEIGHT,
    availableHeight - CANVAS_MIN_HEIGHT - RESIZE_HANDLE_HEIGHT,
  );
  return Math.min(maximum, Math.max(BOTTOM_DRAWER_MIN_HEIGHT, size));
}
