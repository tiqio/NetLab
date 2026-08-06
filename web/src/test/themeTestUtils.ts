import { vi } from "vitest";

export function mockColorScheme(light: boolean) {
  const listeners = new Set<(event: MediaQueryListEvent) => void>();
  const media = {
    matches: light,
    media: "(prefers-color-scheme: light)",
    onchange: null,
    addEventListener: vi.fn((_name: string, listener: (event: MediaQueryListEvent) => void) => listeners.add(listener)),
    removeEventListener: vi.fn((_name: string, listener: (event: MediaQueryListEvent) => void) => listeners.delete(listener)),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  } as unknown as MediaQueryList;
  vi.stubGlobal("matchMedia", vi.fn(() => media));
  return { media, emit: (matches: boolean) => listeners.forEach((listener) => listener({ matches } as MediaQueryListEvent)) };
}

