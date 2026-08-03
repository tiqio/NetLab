export type LightweightSwitchKind = "switch_l2" | "switch_l3";

export interface L2PortDraft {
  name: string;
  pvid: number;
  tagged: string;
}

export interface L3InterfaceDraft {
  name: string;
  addresses: string;
}

export interface L3RouteDraft {
  destination: string;
  gateway: string;
  metric: number;
}

export function splitList(value: string) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

export function defaultLightweightSwitchConfig(kind: LightweightSwitchKind) {
  if (kind === "switch_l2")
    return {
      vlan_filtering: true,
      ports: [{ name: "eth0", pvid: 1, tagged: [] as number[] }],
    };
  return {
    interfaces: [{ name: "eth0", addresses: [] as string[] }],
    routes: [] as Array<{
      destination: string;
      gateway?: string;
      metric?: number;
    }>,
    forward_ipv4: true,
    forward_ipv6: true,
  };
}

export function validateLightweightSwitchConfig(
  kind: LightweightSwitchKind,
  config: Record<string, unknown>,
) {
  const errors: string[] = [];
  const namePattern = /^[A-Za-z0-9_.-]{1,15}$/;
  if (kind === "switch_l2") {
    const ports = Array.isArray(config.ports) ? config.ports : [];
    if (!ports.length) errors.push("至少配置一个二层端口。");
    const names = new Set<string>();
    for (const raw of ports) {
      const port = raw as { name?: string; pvid?: number; tagged?: number[] };
      const name = String(port.name || "");
      if (!namePattern.test(name)) errors.push(`端口名称 ${name || "(空)"} 无效。`);
      if (names.has(name)) errors.push(`端口名称 ${name} 重复。`);
      names.add(name);
      const pvid = Number(port.pvid || 0);
      if (!Number.isInteger(pvid) || pvid < 0 || pvid > 4094)
        errors.push(`${name} 的 PVID 必须是 0–4094。`);
      const tagged = Array.isArray(port.tagged) ? port.tagged : [];
      for (const vlan of tagged) {
        if (!Number.isInteger(vlan) || vlan < 1 || vlan > 4094)
          errors.push(`${name} 包含无效的 Tagged VLAN。`);
        if (vlan === pvid)
          errors.push(`${name} 的 PVID 不能同时作为 Tagged VLAN。`);
      }
    }
  } else {
    const interfaces = Array.isArray(config.interfaces)
      ? config.interfaces
      : [];
    if (!interfaces.length) errors.push("至少配置一个三层接口。");
    const names = new Set<string>();
    for (const raw of interfaces) {
      const iface = raw as { name?: string; addresses?: string[] };
      const name = String(iface.name || "");
      if (!namePattern.test(name)) errors.push(`接口名称 ${name || "(空)"} 无效。`);
      if (names.has(name)) errors.push(`接口名称 ${name} 重复。`);
      names.add(name);
      for (const address of iface.addresses || [])
        if (!address.includes("/")) errors.push(`${address} 必须使用 CIDR 格式。`);
    }
    for (const raw of Array.isArray(config.routes) ? config.routes : []) {
      const route = raw as {
        destination?: string;
        gateway?: string;
        metric?: number;
      };
      if (!String(route.destination || "").includes("/"))
        errors.push("路由目标必须使用 CIDR 格式。");
      if (Number(route.metric || 0) < 0) errors.push("路由 Metric 不能为负数。");
    }
  }
  return errors;
}
