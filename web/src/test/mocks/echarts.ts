import { vi } from "vitest";
export const chartMock = {
  setOption: vi.fn(),
  resize: vi.fn(),
  dispose: vi.fn(),
  on: vi.fn(),
  off: vi.fn(),
  dispatchAction: vi.fn(),
  getOption: vi.fn(() => ({})),
};
