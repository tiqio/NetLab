import { flushPromises, mount } from "@vue/test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "@/api";
import { interfaceFactory, nodeFactory } from "@/test/factories";
import NodeConfigurationPanel from "./NodeConfigurationPanel.vue";

describe("NodeConfigurationPanel", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("locks restart-required fields while the node is running", () => {
    const wrapper = mount(NodeConfigurationPanel, {
      props: {
        node: nodeFactory({
          desired_state: "running",
          observed_state: "running",
          config: { template_key: "vyos" },
        }),
        interfaces: [interfaceFactory({ name: "ens0" })],
      },
    });

    expect(wrapper.text()).toContain("名称、网卡驱动和 IP 预配置已锁定");
    expect(
      wrapper.get('input[aria-label="节点名称"]').attributes("disabled"),
    ).toBeDefined();
    expect(
      wrapper.get('select[aria-label="ens0 网卡驱动"]').attributes("disabled"),
    ).toBeDefined();
    expect(wrapper.get("button").attributes("disabled")).toBeDefined();
  });

  it("saves stopped-node name and all QEMU network interfaces", async () => {
    const node = nodeFactory({
      name: "旧名称",
      config: {
        template_key: "vyos",
        network_interfaces: [
          { name: "ens0", modes: [], addresses: [] },
          { name: "ens1", modes: ["slaac"], addresses: [] },
        ],
      },
    });
    const interfaces = [
      interfaceFactory({ id: "if-0", name: "ens0" }),
      interfaceFactory({
        id: "if-1",
        slot: 1,
        name: "ens1",
        driver: "e1000",
      }),
    ];
    const update = vi.spyOn(api, "updateNodeSettings").mockResolvedValue({
      ...node,
      name: "新名称",
      revision: 2,
    });
    const wrapper = mount(NodeConfigurationPanel, {
      props: { node, interfaces },
    });

    await wrapper.get('input[aria-label="节点名称"]').setValue("新名称");
    await wrapper.get('select[aria-label="ens0 IPv4 配置"]').setValue("dhcp");
    await wrapper.get("button:not([type='button'])").trigger("click");
    await flushPromises();

    expect(update).toHaveBeenCalledWith(node, {
      name: "新名称",
      cpu_count: node.cpu_count,
      cpu_quota_micros: node.cpu_quota_micros,
      memory_mib: node.memory_mib,
      interface_limit: node.interface_limit,
      process_limit: node.process_limit,
      network_interfaces: [
        {
          id: "if-0",
          name: "ens0",
          driver: "virtio-net-pci",
          modes: ["dhcpv4"],
          addresses: [],
          routes: [],
        },
        {
          id: "if-1",
          name: "ens1",
          driver: "e1000",
          modes: ["slaac"],
          addresses: [],
          routes: [],
        },
      ],
    });
    expect(wrapper.emitted("changed")).toHaveLength(1);
  });

  it("reads and saves Docker static routes while stopped", async () => {
    const node = nodeFactory({
      kind: "docker",
      config: {
        template_key: "ubuntu-docker",
        interfaces: [
          {
            id: "if-0",
            name: "eth0",
            driver: "",
            mac_address: "02:00:00:00:00:01",
          },
        ],
        network_interfaces: [
          {
            name: "eth0",
            modes: ["static"],
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
    });
    const interfaces = [
      interfaceFactory({ id: "if-0", name: "eth0", driver: "" }),
    ];
    const update = vi.spyOn(api, "updateNodeSettings").mockResolvedValue({
      ...node,
      revision: 2,
    });
    const wrapper = mount(NodeConfigurationPanel, {
      props: { node, interfaces },
    });

    expect(
      wrapper.get<HTMLInputElement>('input[aria-label="eth0 路由 1 目标"]')
        .element.value,
    ).toBe("0.0.0.0/0");
    await wrapper.get('input[aria-label="eth0 路由 1 Metric"]').setValue("20");
    await wrapper.get("button:not([type='button'])").trigger("click");
    await flushPromises();
    await wrapper.setProps({ node: { ...node, revision: 2 } });
    await flushPromises();

    expect(update).toHaveBeenCalledWith(
      node,
      expect.objectContaining({
        network_interfaces: [
          expect.objectContaining({
            id: "if-0",
            name: "eth0",
            driver: "",
            addresses: ["192.0.2.10/24"],
            routes: [
              {
                destination: "0.0.0.0/0",
                gateway: "192.0.2.1",
                metric: 20,
              },
            ],
          }),
        ],
      }),
    );
    expect(wrapper.get('[role="status"]').text()).toContain("节点配置已保存");
  });

  it("shows read-only Ubuntu credentials and copies them", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", { clipboard: { writeText } });
    vi.spyOn(api, "getNodeBootstrapCredentials").mockResolvedValue({
      username: "ubuntu",
      password: "secret",
      source: "cloud-init-seed",
    });
    const wrapper = mount(NodeConfigurationPanel, {
      props: {
        node: nodeFactory({ config: { template_key: "ubuntu-qemu" } }),
        interfaces: [],
      },
    });
    await flushPromises();

    const username = wrapper.get<HTMLInputElement>(
      'input[aria-label="初始用户名"]',
    );
    const password = wrapper.get<HTMLInputElement>(
      'input[aria-label="初始密码"]',
    );
    expect(username.element.value).toBe("ubuntu");
    expect(password.element.value).toBe("secret");
    expect(username.attributes("readonly")).toBeDefined();
    expect(password.attributes("readonly")).toBeDefined();

    await wrapper.get('button[aria-label="复制用户名"]').trigger("click");
    await flushPromises();
    expect(writeText).toHaveBeenCalledWith("ubuntu");
    expect(wrapper.text()).toContain("用户名已复制");
  });

  it("does not expose low-frequency admission limits", () => {
    const wrapper = mount(NodeConfigurationPanel, {
      props: { node: nodeFactory(), interfaces: [] },
    });
    expect(wrapper.text()).not.toContain("Interface limit");
    expect(wrapper.text()).not.toContain("Process limit");
    expect(wrapper.text()).not.toContain("接口上限");
    expect(wrapper.text()).not.toContain("进程上限");
  });
});
