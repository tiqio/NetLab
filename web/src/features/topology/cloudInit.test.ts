import { describe, expect, it, vi } from "vitest";
import {
  buildTemplateCloudInit,
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

  it("builds official VyOS config commands from structured fields", () => {
    const document = buildTemplateCloudInit({
      templateKey: "vyos",
      hostname: "Branch Router",
      username: "vyos",
      password: "Secret-123456",
      interfaceName: "eth0",
      ipv4Mode: "static",
      ipv4Address: "192.0.2.1/24",
      routes: [
        {
          family: "ipv4",
          destination: "0.0.0.0/0",
          gateway: "192.0.2.254",
        },
      ],
    });
    expect(document).toContain('"vyos_config_commands"');
    expect(document).toContain("set system host-name 'branch-router'");
    expect(document).toContain("set interfaces ethernet eth0 address '192.0.2.1/24'");
    expect(document).toContain("set protocols static route '0.0.0.0/0'");
  });

  it("builds standard Linux cloud-config for FancyWAN", () => {
    const document = buildTemplateCloudInit({
      templateKey: "fancywan",
      hostname: "Fancy WAN 1",
      username: "ubuntu",
      password: "Secret-123456",
      interfaceName: "eth0",
      ipv4Mode: "dhcpv4",
      ipv4Address: "",
      routes: [],
    });
    expect(document).toContain('"hostname": "fancy-wan-1"');
    expect(document).toContain('"name": "ubuntu"');
    expect(document).toContain('"ssh_pwauth": true');
  });
});
