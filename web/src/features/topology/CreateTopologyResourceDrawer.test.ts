import { flushPromises, mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import { api, ApiError, type DeviceTemplate, type ImageVersion } from "@/api";
import CreateTopologyResourceDrawer from "./CreateTopologyResourceDrawer.vue";

vi.mock("@/api", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/api")>()),
  api: {
    listTemplates: vi.fn().mockResolvedValue([]),
    listImages: vi.fn().mockResolvedValue([]),
    createNode: vi.fn(),
    createNetworkObject: vi.fn(),
  },
}));

function qemuCatalog(id = "ubuntu-qemu") {
  const image: ImageVersion = {
    id: `${id}-image`,
    runtime_kind: "qemu",
    name: id,
    version: "24.04",
    digest: `sha256:${id}`,
    source_type: "local_import",
    source_reference: `${id}.qcow2`,
    format: "qcow2",
    size_bytes: 1,
    availability: "available",
    license_status: "reviewed",
    license_notes: "operator supplied",
    validation_result: {},
    created_at: "2026-08-06T00:00:00Z",
  };
  const template: DeviceTemplate = {
    id,
    template_key: id,
    display_name: id,
    runtime_kind: "qemu",
    versions: [
      {
        id: `${id}-version`,
        template_id: id,
        version: "24.04",
        manifest_version: 1,
        compatible_image_version_ids: [image.id],
        defaults: {
          cpu_count: 2,
          cpu_quota_micros: 100000,
          memory_mib: 2048,
          disk_gib: 16,
          interfaces: 2,
          interface_name_format: "ens%d",
        },
        capabilities: [],
        supported_nic_drivers: ["virtio-net-pci", "e1000"],
        console_modes: ["telnet"],
        runtime_options: { interface_limit: 16, process_limit: 1024 },
        enabled: true,
        created_at: "2026-08-06T00:00:00Z",
      },
    ],
    created_at: "2026-08-06T00:00:00Z",
  };
  return { template, version: template.versions[0], image };
}

describe("CreateTopologyResourceDrawer", () => {
  it.each(["vyos", "fancywan"])(
    "hides unsupported bootstrap controls for %s",
    async (templateKey) => {
      const { template, version, image } = qemuCatalog(templateKey);
      version.capabilities = ["guest_exec", "nic_hotplug"];
      vi.mocked(api.listTemplates).mockResolvedValue([template]);
      vi.mocked(api.listImages).mockResolvedValue([image]);
      const wrapper = mount(CreateTopologyResourceDrawer, {
        attachTo: document.body,
        props: {
          modelValue: true,
          laboratoryId: "lab-1",
          selection: {
            kind: "qemu",
            name: templateKey,
            template,
            version,
          },
        },
      });
      await flushPromises();

      expect(document.body.textContent).not.toContain("自动配置与初始登录信息");
      expect(document.body.textContent).not.toContain("高级 user-data（可选）");
      expect(
        document.body.querySelector('[data-testid="docker-ipv4-mode"]'),
      ).toBeNull();
      wrapper.unmount();
    },
  );

  it("suggests the next available name for repeated template additions", async () => {
    const { template, version } = qemuCatalog();
    const wrapper = mount(CreateTopologyResourceDrawer, {
      attachTo: document.body,
      props: {
        modelValue: true,
        laboratoryId: "lab-1",
        nodeNames: ["Ubuntu", "Ubuntu 2"],
        selection: {
          kind: "qemu",
          name: "Ubuntu",
          template,
          version,
        },
      },
    });
    await flushPromises();
    expect(
      document.body.querySelector('[data-testid="create-resource-name"]'),
    ).toHaveProperty("value", "Ubuntu 3");
    wrapper.unmount();
  });

  it("starts with the shared catalog and emits the selected resource", async () => {
    const wrapper = mount(CreateTopologyResourceDrawer, {
      attachTo: document.body,
      props: { modelValue: true, laboratoryId: "lab-1" },
    });
    await flushPromises();
    const pc = Array.from(document.body.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("Linux netns 主机"),
    ) as HTMLButtonElement;
    pc.click();
    await flushPromises();
    expect(wrapper.emitted("selectionChanged")?.[0]?.[0]).toMatchObject({
      networkObjectKind: "pc",
    });
    wrapper.unmount();
  });

  it("submits user-configured L2 ports and VLANs", async () => {
    vi.mocked(api.createNetworkObject).mockClear();
    vi.mocked(api.createNetworkObject).mockResolvedValue({
      network_object: { kind: "switch_l2" },
    } as never);
    const wrapper = mount(CreateTopologyResourceDrawer, {
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

    expect(api.createNetworkObject).toHaveBeenCalledWith("lab-1", 1, {
      name: "Configured L2",
      kind: "switch_l2",
      config: {
        vlan_filtering: true,
        ports: [{ name: "lan0", pvid: 10, tagged: [20, 30] }],
      },
      placement_intent: undefined,
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
    const wrapper = mount(CreateTopologyResourceDrawer, {
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

    expect(api.createNetworkObject).toHaveBeenCalledWith("lab-1", 1, {
      name: scenario.name,
      kind: scenario.kind,
      config: scenario.expectedConfig,
      placement_intent: undefined,
    });
    expect(wrapper.emitted("created")?.[0]?.[0]).toMatchObject({
      networkObject: { kind: scenario.kind },
    });
    wrapper.unmount();
  });

  it("describes confirmed placement as shared state", async () => {
    const wrapper = mount(CreateTopologyResourceDrawer, {
      attachTo: document.body,
      props: {
        modelValue: true,
        laboratoryId: "lab-1",
        selection: { kind: "pc", name: "PC", networkObjectKind: "pc" },
      },
    });
    expect(document.body.textContent).toContain(
      "资源及确认位置会共享给所有客户端",
    );
    expect(document.body.textContent).toContain(
      "画布视口、手工链路路径和当前抽屉草稿仅保存在当前浏览器",
    );
    wrapper.unmount();
  });

  it("blocks device creation when no compatible image exists", async () => {
    const wrapper = mount(CreateTopologyResourceDrawer, {
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
    expect(document.body.textContent).toContain("No reviewed compatible image");
    expect(
      document.body.querySelector<HTMLButtonElement>('button[type="submit"]')
        ?.disabled,
    ).toBe(true);
    wrapper.unmount();
  });

  it("shows only image versions assigned to the selected QEMU device family", async () => {
    const template: DeviceTemplate = {
      id: "template-vyos",
      template_key: "vyos",
      display_name: "VyOS",
      runtime_kind: "qemu",
      versions: [
        {
          id: "version-vyos",
          template_id: "template-vyos",
          version: "rolling",
          manifest_version: 1,
          compatible_image_version_ids: ["image-vyos"],
          defaults: {
            cpu_count: 1,
            memory_mib: 1024,
            interfaces: 4,
            interface_name_format: "eth%d",
          },
          capabilities: [],
          supported_nic_drivers: ["virtio-net-pci"],
          console_modes: ["telnet"],
          runtime_options: {},
          enabled: true,
          created_at: "2026-08-05T00:00:00Z",
        },
      ],
      created_at: "2026-08-05T00:00:00Z",
    };
    const image = (id: string, name: string): ImageVersion => ({
      id,
      runtime_kind: "qemu",
      name,
      version: "test",
      digest: `sha256:${id}`,
      source_type: "local_import",
      source_reference: `${name}.qcow2`,
      format: "qcow2",
      size_bytes: 1,
      availability: "available",
      license_status: "reviewed",
      license_notes: "operator supplied",
      validation_result: {},
      created_at: "2026-08-05T00:00:00Z",
    });
    vi.mocked(api.listTemplates).mockResolvedValue([template]);
    vi.mocked(api.listImages).mockResolvedValue([
      image("image-vyos", "VyOS"),
      image("image-ubuntu", "Ubuntu"),
      image("image-fortigate", "FortiGate"),
    ]);
    const wrapper = mount(CreateTopologyResourceDrawer, {
      attachTo: document.body,
      props: {
        modelValue: false,
        laboratoryId: "lab-1",
        selection: {
          kind: "qemu",
          name: "VyOS",
          template,
          version: template.versions[0],
        },
      },
    });
    await wrapper.setProps({ modelValue: true });
    await flushPromises();

    const options = Array.from(
      document.body.querySelectorAll<HTMLSelectElement>("select")[2].options,
    ).map((option) => option.textContent);
    expect(options).toContain("VyOS test");
    expect(options).not.toContain("Ubuntu test");
    expect(options).not.toContain("FortiGate test");
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
          compatible_image_version_ids: ["image-nettools"],
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
    } as never);
    const wrapper = mount(CreateTopologyResourceDrawer, {
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
      (button) => button.textContent?.includes("添加 IPv4 路由"),
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
      1,
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
          compatible_image_version_ids: ["image-ubuntu-qemu"],
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
    } as never);
    const wrapper = mount(CreateTopologyResourceDrawer, {
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
      1,
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

  it("preserves a dirty draft until the user explicitly discards it", async () => {
    const wrapper = mount(CreateTopologyResourceDrawer, {
      attachTo: document.body,
      props: {
        modelValue: true,
        laboratoryId: "lab-1",
        selection: { kind: "pc", name: "PC", networkObjectKind: "pc" },
      },
    });
    const name = document.body.querySelector<HTMLInputElement>(
      '[data-testid="create-resource-name"]',
    )!;
    name.value = "PC edited";
    name.dispatchEvent(new Event("input", { bubbles: true }));
    await flushPromises();

    Array.from(document.body.querySelectorAll("button"))
      .find((button) => button.textContent?.trim() === "取消")!
      .click();
    await flushPromises();
    expect(document.body.querySelector('[role="alertdialog"]')).not.toBeNull();
    expect(wrapper.emitted("update:modelValue")).toBeUndefined();

    document.body
      .querySelector<HTMLButtonElement>("[data-keep-editing]")!
      .click();
    await flushPromises();
    expect(name.value).toBe("PC edited");

    Array.from(document.body.querySelectorAll("button"))
      .find((button) => button.textContent?.trim() === "更换资源")!
      .click();
    await flushPromises();
    document.body
      .querySelector<HTMLButtonElement>("[data-discard-changes]")!
      .click();
    await flushPromises();
    expect(wrapper.emitted("selectionChanged")?.at(-1)).toEqual([undefined]);
    wrapper.unmount();
  });

  it("focuses and scrolls the first invalid long-form field", async () => {
    const scrollIntoView = vi.fn();
    Object.defineProperty(HTMLElement.prototype, "scrollIntoView", {
      configurable: true,
      value: scrollIntoView,
    });
    const template: DeviceTemplate = {
      id: "docker-template",
      template_key: "ubuntu-container",
      display_name: "Ubuntu",
      runtime_kind: "docker",
      versions: [
        {
          id: "docker-version",
          template_id: "docker-template",
          version: "24.04",
          manifest_version: 1,
          compatible_image_version_ids: ["docker-image"],
          defaults: {
            cpu_count: 1,
            memory_mib: 256,
            interfaces: 1,
            interface_name_format: "eth%d",
          },
          capabilities: [],
          supported_nic_drivers: ["veth"],
          console_modes: [],
          runtime_options: {},
          enabled: true,
          created_at: "2026-08-05T00:00:00Z",
        },
      ],
      created_at: "2026-08-05T00:00:00Z",
    };
    const image: ImageVersion = {
      id: "docker-image",
      runtime_kind: "docker",
      name: "ubuntu-network-tools",
      version: "24.04",
      digest: "sha256:docker",
      source_type: "registry",
      source_reference: "ubuntu:24.04",
      format: "oci",
      size_bytes: 1,
      availability: "available",
      license_status: "reviewed",
      license_notes: "",
      validation_result: {},
      created_at: "2026-08-05T00:00:00Z",
    };
    vi.mocked(api.listTemplates).mockResolvedValue([template]);
    vi.mocked(api.listImages).mockResolvedValue([image]);
    const wrapper = mount(CreateTopologyResourceDrawer, {
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
    Array.from(document.body.querySelectorAll("button"))
      .find((button) => button.textContent?.includes("添加 IPv4 路由"))!
      .click();
    await flushPromises();
    const gateway = document.body.querySelector<HTMLInputElement>(
      '[data-testid="docker-route-0-gateway"]',
    )!;
    gateway.value = "2001:db8::1";
    gateway.dispatchEvent(new Event("input", { bubbles: true }));
    document.body
      .querySelector("form")!
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    await flushPromises();
    expect(scrollIntoView).toHaveBeenCalled();
    expect(document.activeElement).toBe(
      document.body.querySelector('[data-testid="docker-route-0-destination"]'),
    );
    wrapper.unmount();
  });

  it("locks duplicate submissions while the first request is pending", async () => {
    let resolveRequest!: (value: never) => void;
    vi.mocked(api.createNetworkObject).mockReturnValue(
      new Promise((resolve) => {
        resolveRequest = resolve;
      }),
    );
    const wrapper = mount(CreateTopologyResourceDrawer, {
      attachTo: document.body,
      props: {
        modelValue: true,
        laboratoryId: "lab-1",
        selection: { kind: "pc", name: "PC", networkObjectKind: "pc" },
      },
    });
    const form = document.body.querySelector("form")!;
    form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    );
    form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    );
    await flushPromises();
    expect(api.createNetworkObject).toHaveBeenCalledTimes(1);
    expect(
      document.body.querySelector("fieldset")?.hasAttribute("disabled"),
    ).toBe(true);
    resolveRequest({ network_object: { kind: "pc" } } as never);
    await flushPromises();
    wrapper.unmount();
  });

  it("keeps an asynchronously selected default image clean", async () => {
    const catalog = qemuCatalog();
    vi.mocked(api.listTemplates).mockResolvedValue([catalog.template]);
    vi.mocked(api.listImages).mockResolvedValue([catalog.image]);
    const wrapper = mount(CreateTopologyResourceDrawer, {
      attachTo: document.body,
      props: {
        modelValue: true,
        laboratoryId: "lab-1",
        selection: {
          kind: "qemu",
          name: "Ubuntu",
          template: catalog.template,
          version: catalog.version,
        },
      },
    });
    await flushPromises();
    expect(
      (wrapper.vm as unknown as { isDirty: () => boolean }).isDirty(),
    ).toBe(false);
    document.body
      .querySelector<HTMLButtonElement>('[aria-label="关闭抽屉"]')!
      .click();
    await flushPromises();
    expect(document.body.querySelector('[role="alertdialog"]')).toBeNull();
    wrapper.unmount();
  });

  it("filters cross-runtime templates and confirms destructive template changes", async () => {
    const first = qemuCatalog("ubuntu-one");
    const second = qemuCatalog("ubuntu-two");
    const docker = {
      ...qemuCatalog("busybox").template,
      runtime_kind: "docker" as const,
    };
    vi.mocked(api.listTemplates).mockResolvedValue([
      first.template,
      second.template,
      docker,
    ]);
    vi.mocked(api.listImages).mockResolvedValue([first.image, second.image]);
    const wrapper = mount(CreateTopologyResourceDrawer, {
      attachTo: document.body,
      props: {
        modelValue: true,
        laboratoryId: "lab-1",
        selection: {
          kind: "qemu",
          name: "Ubuntu",
          template: first.template,
          version: first.version,
        },
      },
    });
    await flushPromises();
    expect(document.body.textContent).not.toContain("busybox · DOCKER");
    const name = document.body.querySelector<HTMLInputElement>(
      '[data-testid="create-resource-name"]',
    )!;
    name.value = "edited";
    name.dispatchEvent(new Event("input", { bubbles: true }));
    await flushPromises();
    const templateSelect = Array.from(
      document.body.querySelectorAll<HTMLSelectElement>("select"),
    ).find((select) =>
      Array.from(select.options).some(
        (option) => option.value === second.template.id,
      ),
    )!;
    expect(templateSelect).toBeDefined();
    templateSelect.value = second.template.id;
    templateSelect.dispatchEvent(new Event("change", { bubbles: true }));
    await flushPromises();
    expect(document.body.querySelector('[role="alertdialog"]')).not.toBeNull();
    document.body
      .querySelector<HTMLButtonElement>("[data-keep-editing]")!
      .click();
    await flushPromises();
    expect(
      Array.from(
        document.body.querySelectorAll<HTMLSelectElement>("select"),
      ).find((select) =>
        Array.from(select.options).some(
          (option) => option.value === second.template.id,
        ),
      )?.value,
    ).toBe(first.template.id);
    expect(name.value).toBe("edited");
    wrapper.unmount();
  });

  it("maps structured server fields and focuses the related control", async () => {
    const catalog = qemuCatalog();
    vi.mocked(api.listTemplates).mockResolvedValue([catalog.template]);
    vi.mocked(api.listImages).mockResolvedValue([catalog.image]);
    vi.mocked(api.createNode).mockRejectedValue(
      new ApiError(409, {
        code: "quota_exceeded",
        message: "资源配额不足，请调整后重试。",
        details: { fields: { cpu_count: "vCPU 超过当前配额。" } },
      } as never),
    );
    const wrapper = mount(CreateTopologyResourceDrawer, {
      attachTo: document.body,
      props: {
        modelValue: true,
        laboratoryId: "lab-1",
        selection: {
          kind: "qemu",
          name: "Ubuntu",
          template: catalog.template,
          version: catalog.version,
        },
      },
    });
    await flushPromises();
    document.body
      .querySelector("form")!
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    await flushPromises();
    expect(document.body.textContent).toContain("vCPU 超过当前配额");
    expect(document.activeElement).toBe(
      document.body.querySelector('[data-field="cpuCount"] input'),
    );
    wrapper.unmount();
  });

  it("preserves draft values while configuration groups collapse", async () => {
    const catalog = qemuCatalog();
    vi.mocked(api.listTemplates).mockResolvedValue([catalog.template]);
    vi.mocked(api.listImages).mockResolvedValue([catalog.image]);
    const wrapper = mount(CreateTopologyResourceDrawer, {
      attachTo: document.body,
      props: {
        modelValue: true,
        laboratoryId: "lab-1",
        selection: {
          kind: "qemu",
          name: "Ubuntu",
          template: catalog.template,
          version: catalog.version,
        },
      },
    });
    await flushPromises();
    const name = document.body.querySelector<HTMLInputElement>(
      '[data-testid="create-resource-name"]',
    )!;
    name.value = "kept-name";
    name.dispatchEvent(new Event("input", { bubbles: true }));
    const details = document.body.querySelector("details")!;
    details.open = false;
    details.dispatchEvent(new Event("toggle"));
    await flushPromises();
    expect(name.value).toBe("kept-name");
    wrapper.unmount();
  });

  it("submits viewport placement intent and returns the authoritative assignment", async () => {
    vi.mocked(api.createNetworkObject).mockResolvedValue({
      network_object: {
        id: "pc-authoritative",
        laboratory_id: "lab-1",
        name: "PC",
        kind: "pc",
        revision: 1,
        desired_state: "active",
        observed_state: "provisioning",
        config: {},
      },
      placement_assignment: {
        placement: {
          laboratory_id: "lab-1",
          resource_id: "pc-authoritative",
          resource_type: "network_object",
          x: 260,
          y: 140,
          revision: 1,
        },
        requested_center: { x: 200, y: 100 },
        assigned_center: { x: 260, y: 140 },
        adjusted: true,
        reason: "collision_avoided",
        footprint_class: "network-object-standard",
        algorithm_version: 1,
      },
      laboratory_revision: 8,
      task: {} as never,
    });
    const wrapper = mount(CreateTopologyResourceDrawer, {
      attachTo: document.body,
      props: {
        modelValue: true,
        laboratoryId: "lab-1",
        laboratoryRevision: 7,
        placementIntent: {
          preferred_x: 200,
          preferred_y: 100,
          footprint_class: "network-object-standard",
        },
        selection: { kind: "pc", name: "PC", networkObjectKind: "pc" },
      },
    });
    document.body
      .querySelector("form")!
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    await flushPromises();
    expect(api.createNetworkObject).toHaveBeenCalledWith(
      "lab-1",
      7,
      expect.objectContaining({
        placement_intent: {
          preferred_x: 200,
          preferred_y: 100,
          footprint_class: "network-object-standard",
        },
      }),
    );
    expect(wrapper.emitted("created")?.[0]?.[0]).toMatchObject({
      laboratory_revision: 8,
      placement_assignment: {
        adjusted: true,
        assigned_center: { x: 260, y: 140 },
      },
    });
    wrapper.unmount();
  });
});
