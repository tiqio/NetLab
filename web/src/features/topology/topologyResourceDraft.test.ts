import { describe, expect, it } from "vitest";
import type { DeviceTemplate, ImageVersion, TemplateVersion } from "@/api";
import type { PaletteSelection } from "./TopologyResourceCatalog.vue";
import {
  buildResourceCreateRequest,
  createResourceDraft,
  draftSignature,
  validateResourceDraft,
} from "./topologyResourceDraft";

const version: TemplateVersion = {
  id: "version-1",
  template_id: "template-1",
  version: "24.04",
  manifest_version: 1,
  compatible_image_version_ids: ["image-1"],
  defaults: {
    cpu_count: 1,
    memory_mib: 512,
    interfaces: 2,
    interface_name_format: "ens%d",
  },
  capabilities: ["cloud_init"],
  supported_nic_drivers: ["virtio-net-pci"],
  console_modes: ["telnet"],
  runtime_options: {},
  enabled: true,
  created_at: "2026-08-05T00:00:00Z",
};
const template: DeviceTemplate = {
  id: "template-1",
  template_key: "ubuntu-qemu",
  display_name: "Ubuntu",
  runtime_kind: "qemu",
  versions: [version],
  created_at: "2026-08-05T00:00:00Z",
};
const image: ImageVersion = {
  id: "image-1",
  runtime_kind: "qemu",
  name: "Ubuntu",
  version: "24.04",
  digest: "sha256:test",
  source_type: "local_import",
  source_reference: "ubuntu.qcow2",
  format: "qcow2",
  size_bytes: 1,
  availability: "available",
  license_status: "reviewed",
  license_notes: "operator supplied",
  validation_result: {},
  created_at: "2026-08-05T00:00:00Z",
};
const qemuSelection: PaletteSelection = {
  kind: "qemu",
  name: "Ubuntu",
  template,
  version,
};

describe("topology resource draft", () => {
  it("initializes node defaults and produces a stable normalized signature", () => {
    const draft = createResourceDraft(qemuSelection, () => "Secret-123456");
    expect(draft).toMatchObject({
      name: "Ubuntu",
      templateId: "template-1",
      templateVersionId: "version-1",
      imageVersionId: "",
      interfaceCount: 2,
      cloudUsername: "ubuntu",
      cloudPassword: "Secret-123456",
    });
    const first = draftSignature(draft);
    draft.routes.push({
      id: "route-random-a",
      family: "ipv4",
      destination: "0.0.0.0/0",
      gateway: "192.0.2.1",
      metric: "10",
    });
    const second = draftSignature({
      ...draft,
      routes: [{ ...draft.routes[0], id: "route-random-b", metric: 10 }],
    });
    expect(draftSignature(draft)).toBe(second);
    expect(first).not.toBe(second);
  });

  it.each([
    ["pc", { hostname: "PC", interfaces: [{ name: "eth0" }] }],
    ["bridge", { mtu: 1500, stp: false }],
    ["nat_bridge", { ipv4_prefix: "10.10.0.0/24", uplink: "auto" }],
    ["switch_l2", { vlan_filtering: true }],
    ["switch_l3", { forward_ipv4: true, forward_ipv6: true }],
  ] as const)("builds %s network object defaults", (kind, expected) => {
    const selection: PaletteSelection = {
      kind:
        kind === "bridge"
          ? "switch_l2"
          : kind === "nat_bridge"
            ? "switch_l3"
            : kind,
      name: kind === "pc" ? "PC" : kind,
      networkObjectKind: kind,
    };
    const draft = createResourceDraft(selection, () => "unused");
    const result = buildResourceCreateRequest(selection, draft, {
      template: undefined,
      version: undefined,
    });
    expect(result.kind).toBe("network-object");
    if (result.kind === "network-object")
      expect(result.request.config).toMatchObject(expected);
  });

  it("validates long node fields and builds Docker/QEMU-compatible network payloads", () => {
    const draft = createResourceDraft(qemuSelection, () => "short");
    draft.name = " ";
    draft.ipv4Mode = "static";
    draft.ipv4Address = "192.0.2.10";
    draft.routes.push({
      id: "route-1",
      family: "ipv6",
      destination: "0.0.0.0/0",
      gateway: "192.0.2.1",
      metric: -1,
    });
    expect(
      validateResourceDraft(qemuSelection, draft, { template, version, image }),
    ).toMatchObject({
      name: expect.any(String),
      ipv4Address: expect.any(String),
      "route.route-1": expect.any(String),
      cloudPassword: expect.any(String),
    });

    draft.name = "Ubuntu WAN";
    draft.ipv4Address = "192.0.2.10/24";
    draft.routes = [
      {
        id: "route-2",
        family: "ipv4",
        destination: "0.0.0.0/0",
        gateway: "192.0.2.1",
        metric: "20",
      },
    ];
    draft.cloudPassword = "Secret-123456";
    const result = buildResourceCreateRequest(qemuSelection, draft, {
      template,
      version,
      image,
    });
    expect(result.kind).toBe("node");
    if (result.kind === "node") {
      expect(result.request.interface_count).toBe(2);
      expect(result.request.config).toMatchObject({
        network_interfaces: [
          {
            name: "ens0",
            modes: ["static"],
            addresses: ["192.0.2.10/24"],
            routes: [
              { destination: "0.0.0.0/0", gateway: "192.0.2.1", metric: 20 },
            ],
          },
        ],
      });
      expect(result.request.bootstrap?.user_data).toContain("Secret-123456");
    }
  });
});
