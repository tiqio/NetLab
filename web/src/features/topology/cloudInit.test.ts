import { describe, expect, it, vi } from "vitest";
import {
  buildUbuntuPasswordCloudInit,
  generateInitialPassword,
  supportsUbuntuPasswordBootstrap,
} from "./cloudInit";

describe("Ubuntu cloud-init bootstrap", () => {
  it("detects Ubuntu cloud-init templates across capability spelling", () => {
    expect(supportsUbuntuPasswordBootstrap("ubuntu-qemu", ["cloud_init"])).toBe(
      true,
    );
    expect(supportsUbuntuPasswordBootstrap("ubuntu-qemu", ["cloud-init"])).toBe(
      true,
    );
    expect(supportsUbuntuPasswordBootstrap("vyos", ["cloud_init"])).toBe(false);
  });

  it("builds seed data without YAML interpolation", () => {
    const document = buildUbuntuPasswordCloudInit(
      "ubuntu",
      'safe:"quoted"-value',
    );
    expect(document.startsWith("#cloud-config\n{")).toBe(true);
    expect(document).toContain('"ssh_pwauth": true');
    expect(document).toContain('"name": "ubuntu"');
    expect(document).toContain('safe:\\"quoted\\"-value');
  });

  it("generates a non-ambiguous initial password", () => {
    vi.stubGlobal("crypto", {
      getRandomValues: (value: Uint8Array) => value.fill(7),
    });
    expect(generateInitialPassword()).toHaveLength(18);
    vi.unstubAllGlobals();
  });
});
