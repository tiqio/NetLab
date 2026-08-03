import { vi } from "vitest";
export const apiMock = {
  listLabs: vi.fn(),
  getLab: vi.fn(),
  createLab: vi.fn(),
  updateLab: vi.fn(),
  deleteLab: vi.fn(),
  createNode: vi.fn(),
  connectLink: vi.fn(),
  disconnectLink: vi.fn(),
  setNodeState: vi.fn(),
  listTasks: vi.fn(),
};
