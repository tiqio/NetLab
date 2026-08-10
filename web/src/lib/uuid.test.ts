import { afterEach, describe, expect, it, vi } from "vitest";
import { randomUUID } from "./uuid";

describe("randomUUID", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("generates a UUID when randomUUID is unavailable", () => {
    vi.stubGlobal("crypto", {
      getRandomValues(bytes: Uint8Array) {
        bytes.forEach((_, index) => {
          bytes[index] = index;
        });
        return bytes;
      },
    });

    expect(randomUUID()).toBe("00010203-0405-4607-8809-0a0b0c0d0e0f");
  });
});
