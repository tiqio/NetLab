import { vi } from "vitest";
export const terminalMock = {
  open: vi.fn(),
  write: vi.fn(),
  dispose: vi.fn(),
  fit: vi.fn(),
};
export const vncMock = {
  disconnect: vi.fn(),
  scaleViewport: true,
  resizeSession: true,
};
