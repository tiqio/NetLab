import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import { api, type DeviceTemplate, type ImageVersion } from "@/api";
import CreateTopologyResourceDialog from "./CreateTopologyResourceDialog.vue";

vi.mock("@/api", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/api")>()),
  api: {
    listTemplates: vi.fn().mockResolvedValue([]),
    listImages: vi.fn().mockResolvedValue([]),
    createNode: vi.fn(),
    createNetworkObject: vi.fn(),
  },
}));

describe("CreateTopologyResourceDialog", () => {
  it("submits user-configured L2 ports and VLANs", async () => {
    vi.mocked(api.createNetworkObject).mockClear();
    vi.mocked(api.createNetworkObject).mockResolvedValue({
      network_object: { kind: "switch_l2" },
    } as never);
    const wrapper = mount(CreateTopologyResourceDialog, {
      attachTo: document.body,
      props: {
        modelValue: true,
        laboratoryId: "lab-1",
        selection: {
          kind: "switch_l2",
          name: "Configured L2",
          networkObjectKind: "switch_l2",
        },
      },
    });
    const editor = document.body.querySelector(
      '[data-testid="lightweight-switch-config"]',
    )!;
    const inputs = editor.querySelectorAll("input");
    for (const [input, value] of [
      [inputs[1], "lan0"],
      [inputs[2], "10"],
      [inputs[3], "20,30"],
    ] as const) {
      input.value = value;
      input.dispatchEvent(new Event("input", { bubbles: true }));
    }
    await flushPromises();
    document.body
      .querySelector("form")!
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(api.createNetworkObject).toHaveBeenCalledWith("lab-1", {
      name: "Configured L2",
      kind: "switch_l2",
      config: {
        vlan_filtering: true,
        ports: [{ name: "lan0", pvid: 10, tagged: [20, 30] }],
      },
    });
    wrapper.unmount();
  });

  it.each([
    {
      name: "Lightweight L2 Switch",
      kind: "switch_l2" as const,
      expectedConfig: {
        vlan_filtering: true,
        ports: [{ name: "eth0", pvid: 1, tagged: [] }],
      },
    },
    {
      name: "Lightweight L3 Switch",
      kind: "switch_l3" as const,
      expectedConfig: {
        interfaces: [{ name: "eth0", addresses: [] }],
        routes: [],
        forward_ipv4: true,
        forward_ipv6: true,
      },
    },
  ])("creates $name as an independent netns object", async (scenario) => {
    vi.mocked(api.createNetworkObject).mockResolvedValue({
      network_object: {
        id: `${scenario.kind}-1`,
        laboratory_id: "lab-1",
        name: scenario.name,
        kind: scenario.kind,
        config: scenario.expectedConfig,
        revision: 1,
        desired_state: "running",
        observed_state: "running",
        created_at: "2026-07-31T00:00:00Z",
        updated_at: "2026-07-31T00:00:00Z",
      },
    } as never);
    const wrapper = mount(CreateTopologyResourceDialog, {
      attachTo: document.body,
      props: {
        modelValue: true,
        laboratoryId: "lab-1",
        selection: {
          kind: scenario.kind,
          name: scenario.name,
          networkObjectKind: scenario.kind,
        },
      },
    });

    document.body
      .querySelector("form")!
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(api.createNetworkObject).toHaveBeenCalledWith("lab-1", {
      name: scenario.name,
      kind: scenario.kind,
      config: scenario.expectedConfig,
    });
    expect(wrapper.emitted("created")?.[0]?.[0]).toMatchObject({
      networkObject: { kind: scenario.kind },
    });
    wrapper.unmount();
  });

  it("describes confirmed placement as shared state", async () => {
    const wrapper = mount(CreateTopologyResourceDialog, {
      attachTo: document.body,
      props: {
        modelValue: true,
        laboratoryId: "lab-1",
        selection: { kind: "pc", name: "PC", networkObjectKind: "pc" },
      },
    });
    expect(document.body.textContent).toContain(
      "resource and confirmed placement are shared with every client",
    );
    expect(document.body.textContent).toContain(
      "viewport and manual link routes remain local to this browser",
    );
    expect(document.body.textContent).not.toContain("placement remains local");
    wrapper.unmount();
  });

  it("blocks device creation when no compatible image exists", async () => {
    const wrapper = mount(CreateTopologyResourceDialog, {
      attachTo: document.body,
      props: {
        modelValue: true,
        laboratoryId: "lab-1",
        selection: {
          kind: "docker",
          name: "BusyBox",
          template: {
            id: "template-1",
            template_key: "busybox-container",
            display_name: "BusyBox",
            runtime_kind: "docker",
            versions: [],
            created_at: "2026-07-28T00:00:00Z",
          },
          version: {
            id: "version-1",
            template_id: "template-1",
            version: "latest",
            manifest_version: 1,
            defaults: {
              cpu_count: 1,
              memory_mib: 128,
              interfaces: 1,
              interface_name_format: "eth%d",
            },
            capabilities: [],
            supported_nic_drivers: ["veth"],
            console_modes: [],
            runtime_options: {},
            enabled: true,
            created_at: "2026-07-28T00:00:00Z",
          },
        },
      },
    });
    await flushPromises();
    expect(document.body.textContent).toContain(
      "No compatible image is available",
    );
    expect(
      document.body.querySelector<HTMLButtonElement>('button[type="submit"]')
        ?.disabled,
    ).toBe(true);
    wrapper.unmount();
  });

  it("submits Docker static and SLAAC configuration for eth0", async () => {
    const template: DeviceTemplate = {
      id: "template-ubuntu",
      template_key: "ubuntu-container",
      display_name: "Ubuntu",
      runtime_kind: "docker",
      versions: [
        {
          id: "version-ubuntu",
          template_id: "template-ubuntu",
          version: "24.04",
          manifest_version: 1,
          defaults: {
            cpu_count: 1,
            memory_mib: 512,
            interfaces: 1,
            interface_name_format: "eth%d",
          },
          capabilities: [],
          supported_nic_drivers: ["veth"],
          console_modes: [],
          runtime_options: {},
          enabled: true,
          created_at: "2026-07-28T00:00:00Z",
        },
      ],
      created_at: "2026-07-28T00:00:00Z",
    };
    const image: ImageVersion = {
      id: "image-nettools",
      runtime_kind: "docker",
      name: "ubuntu-network-tools",
      version: "24.04",
      digest: "sha256:abc",
      source_type: "oci_local",
      source_reference: "sha256:abc",
      format: "oci",
      size_bytes: 1,
      availability: "available",
      license_status: "reviewed",
      license_notes: "Ubuntu packages",
      validation_result: {},
      created_at: "2026-07-28T00:00:00Z",
    };
    vi.mocked(api.listTemplates).mockResolvedValue([template]);
    vi.mocked(api.listImages).mockResolvedValue([image]);
    vi.mocked(api.createNode).mockResolvedValue({
      node: {} as never,
      interfaces: [],
    });
    const wrapper = mount(CreateTopologyResourceDialog, {
      attachTo: document.body,
      props: {
        modelValue: false,
        laboratoryId: "lab-1",
        selection: {
          kind: "docker",
          name: "Ubuntu",
          template,
          version: template.versions[0],
        },
      },
    });
    await wrapper.setProps({ modelValue: true });
    await flushPromises();
    const ipv4Mode = document.body.querySelector<HTMLSelectElement>(
      '[data-testid="docker-ipv4-mode"]',
    )!;
    ipv4Mode.value = "static";
    ipv4Mode.dispatchEvent(new Event("change", { bubbles: true }));
    await wrapper.vm.$nextTick();
    const ipv4Address = document.body.querySelector<HTMLInputElement>(
      '[data-testid="docker-ipv4-address"]',
    )!;
    ipv4Address.value = "192.0.2.10/24";
    ipv4Address.dispatchEvent(new Event("input", { bubbles: true }));
    const ipv6Mode = document.body.querySelector<HTMLSelectElement>(
      '[data-testid="docker-ipv6-mode"]',
    )!;
    ipv6Mode.value = "slaac";
    ipv6Mode.dispatchEvent(new Event("change", { bubbles: true }));
    const addRoute = Array.from(document.body.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("Add IPv4 route"),
    )!;
    addRoute.click();
    await wrapper.vm.$nextTick();
    const routeGateway = document.body.querySelector<HTMLInputElement>(
      '[data-testid="docker-route-0-gateway"]',
    )!;
    routeGateway.value = "2001:db8::1";
    routeGateway.dispatchEvent(new Event("input", { bubbles: true }));
    const routeMetric = document.body.querySelector<HTMLInputElement>(
      '[data-testid="docker-route-0-metric"]',
    )!;
    routeMetric.value = "10";
    routeMetric.dispatchEvent(new Event("input", { bubbles: true }));
    vi.mocked(api.createNode).mockClear();
    document.body
      .querySelector("form")!
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    await flushPromises();
    expect(api.createNode).not.toHaveBeenCalled();
    expect(document.body.textContent).toContain(
      "Gateway and destination must use the same address family.",
    );

    routeGateway.value = "192.0.2.1";
    routeGateway.dispatchEvent(new Event("input", { bubbles: true }));
    document.body
      .querySelector("form")!
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(api.createNode).toHaveBeenCalledWith(
      "lab-1",
      expect.objectContaining({
        image_version_id: "image-nettools",
        config: {
          network_interfaces: [
            {
              name: "eth0",
              modes: ["static", "slaac"],
              addresses: ["192.0.2.10/24"],
              routes: [
                {
                  destination: "0.0.0.0/0",
                  gateway: "192.0.2.1",
                  metric: 10,
                },
              ],
            },
          ],
        },
      }),
    );
    wrapper.unmount();
  });

  it("submits a generated Ubuntu password through node-scoped cloud-init", async () => {
    const template: DeviceTemplate = {
      id: "template-ubuntu-qemu",
      template_key: "ubuntu-qemu",
      display_name: "Ubuntu",
      runtime_kind: "qemu",
      versions: [
        {
          id: "version-ubuntu-qemu",
          template_id: "template-ubuntu-qemu",
          version: "24.04",
          manifest_version: 1,
          defaults: {
            cpu_count: 2,
            memory_mib: 2048,
            interfaces: 1,
            interface_name_format: "ens%d",
          },
          capabilities: ["cloud_init"],
          supported_nic_drivers: ["virtio-net-pci"],
          console_modes: ["telnet", "vnc"],
          runtime_options: {},
          enabled: true,
          created_at: "2026-07-28T00:00:00Z",
        },
      ],
      created_at: "2026-07-28T00:00:00Z",
    };
    const image: ImageVersion = {
      id: "image-ubuntu-qemu",
      runtime_kind: "qemu",
      name: "ubuntu",
      version: "24.04",
      digest: "sha256:def",
      source_type: "local_import",
      source_reference: "ubuntu-24.04.qcow2",
      format: "qcow2",
      size_bytes: 1,
      availability: "available",
      license_status: "reviewed",
      license_notes: "Ubuntu cloud image",
      validation_result: {},
      created_at: "2026-07-28T00:00:00Z",
    };
    vi.mocked(api.listTemplates).mockResolvedValue([template]);
    vi.mocked(api.listImages).mockResolvedValue([image]);
    vi.mocked(api.createNode).mockResolvedValue({
      node: {} as never,
      interfaces: [],
    });
    const wrapper = mount(CreateTopologyResourceDialog, {
      attachTo: document.body,
      props: {
        modelValue: false,
        laboratoryId: "lab-1",
        selection: {
          kind: "qemu",
          name: "Ubuntu",
          template,
          version: template.versions[0],
        },
      },
    });
    await wrapper.setProps({ modelValue: true });
    await flushPromises();
    const name = document.body.querySelector<HTMLInputElement>(
      'input[required][maxlength="120"]',
    )!;
    name.value = "ubuntu-node";
    name.dispatchEvent(new Event("input", { bubbles: true }));
    const password = document.body.querySelector<HTMLInputElement>(
      'input[autocomplete="new-password"]',
    )!;
    password.value = "temporary-password-24";
    password.dispatchEvent(new Event("input", { bubbles: true }));
    const ipv4Mode = document.body.querySelector<HTMLSelectElement>(
      '[data-testid="docker-ipv4-mode"]',
    )!;
    ipv4Mode.value = "static";
    ipv4Mode.dispatchEvent(new Event("change", { bubbles: true }));
    await wrapper.vm.$nextTick();
    const ipv4Address = document.body.querySelector<HTMLInputElement>(
      '[data-testid="docker-ipv4-address"]',
    )!;
    ipv4Address.value = "192.0.2.10/24";
    ipv4Address.dispatchEvent(new Event("input", { bubbles: true }));
    document.body
      .querySelector("form")!
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(api.createNode).toHaveBeenCalledWith(
      "lab-1",
      expect.objectContaining({
        config: {
          network_interfaces: [
            {
              name: "ens0",
              modes: ["static"],
              addresses: ["192.0.2.10/24"],
              routes: [],
            },
          ],
        },
        bootstrap: {
          user_data: expect.stringContaining(
            '"password": "temporary-password-24"',
          ),
        },
      }),
    );
    wrapper.unmount();
  });
});
