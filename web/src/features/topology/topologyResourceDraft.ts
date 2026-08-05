import type {
  CreateNodeRequest,
  DeviceTemplate,
  ImageVersion,
  NetworkObject,
  TemplateVersion,
} from "@/api";
import {
  defaultLightweightSwitchConfig,
  validateLightweightSwitchConfig,
} from "@/features/nodes/lightweightSwitchConfig";
import {
  buildUbuntuPasswordCloudInit,
  supportsUbuntuPasswordBootstrap,
} from "./cloudInit";
import type { PaletteSelection } from "./TopologyResourceCatalog.vue";

export type AddressFamily = "ipv4" | "ipv6";
export type IPv4Mode = "none" | "static" | "dhcpv4";
export type IPv6Mode = "none" | "static" | "slaac" | "dhcpv6";

export interface RouteDraft {
  id: string;
  family: AddressFamily;
  destination: string;
  gateway: string;
  metric: string | number;
}

export interface ResourceCreateDraft {
  name: string;
  templateId: string;
  templateVersionId: string;
  imageVersionId: string;
  interfaceCount: number;
  ipv4Mode: IPv4Mode;
  ipv4Address: string;
  ipv6Mode: IPv6Mode;
  ipv6Address: string;
  routes: RouteDraft[];
  cloudUsername: string;
  cloudPassword: string;
  networkObjectConfig: Record<string, unknown>;
}

export interface DraftCatalogContext {
  template?: DeviceTemplate;
  version?: TemplateVersion;
  image?: ImageVersion;
}

export type ResourceCreateRequest =
  | { kind: "node"; request: CreateNodeRequest }
  | {
      kind: "network-object";
      request: Pick<NetworkObject, "name" | "kind"> & {
        config?: Record<string, unknown>;
      };
    };

function networkObjectDefaults(
  selection: PaletteSelection,
): Record<string, unknown> {
  switch (selection.networkObjectKind) {
    case "pc":
      return {
        hostname: selection.name,
        interfaces: [{ name: "eth0", modes: ["dhcpv4", "dhcpv6", "slaac"] }],
      };
    case "nat_bridge":
      return {
        ipv4_prefix: "10.10.0.0/24",
        ipv6_prefix: "2001:db8:10::/64",
        uplink: "auto",
        dhcpv4: {
          start: "10.10.0.100",
          end: "10.10.0.200",
          lease_time: "1h",
        },
        dns_servers: ["1.1.1.1", "8.8.8.8"],
      };
    case "switch_l2":
    case "switch_l3":
      return defaultLightweightSwitchConfig(selection.networkObjectKind);
    default:
      return { mtu: 1500, stp: false };
  }
}

export function createResourceDraft(
  selection: PaletteSelection,
  generatePassword: () => string,
): ResourceCreateDraft {
  const supportsBootstrap = supportsUbuntuPasswordBootstrap(
    selection.template?.template_key,
    selection.version?.capabilities,
  );
  return {
    name: selection.name,
    templateId: selection.template?.id || "",
    templateVersionId: selection.version?.id || "",
    imageVersionId: selection.version?.image_version_id || "",
    interfaceCount: selection.version?.defaults.interfaces || 2,
    ipv4Mode: "none",
    ipv4Address: "",
    ipv6Mode: "none",
    ipv6Address: "",
    routes: [],
    cloudUsername: "ubuntu",
    cloudPassword: supportsBootstrap ? generatePassword() : "",
    networkObjectConfig: networkObjectDefaults(selection),
  };
}

function normalize(value: unknown, key = ""): unknown {
  if (Array.isArray(value)) return value.map((item) => normalize(item));
  if (value && typeof value === "object")
    return Object.fromEntries(
      Object.entries(value)
        .filter(([childKey]) => !(key === "routes" && childKey === "id"))
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([childKey, childValue]) => [
          childKey,
          normalize(
            key === "" && childKey === "routes"
              ? (childValue as RouteDraft[]).map((route) =>
                  Object.fromEntries(
                    Object.entries(route).filter(
                      ([routeKey]) => routeKey !== "id",
                    ),
                  ),
                )
              : childValue,
            childKey,
          ),
        ]),
    );
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (key === "metric" && trimmed !== "" && Number.isFinite(Number(trimmed)))
      return Number(trimmed);
    return trimmed;
  }
  return value;
}

export function draftSignature(draft: ResourceCreateDraft) {
  return JSON.stringify(normalize(draft));
}

export function validateResourceDraft(
  selection: PaletteSelection,
  draft: ResourceCreateDraft,
  context: DraftCatalogContext,
) {
  const errors: Record<string, string> = {};
  const name = draft.name.trim();
  if (!name) errors.name = "请输入资源名称。";
  else if (name.length > 120) errors.name = "资源名称不能超过 120 个字符。";

  if (selection.networkObjectKind) {
    if (
      selection.networkObjectKind === "switch_l2" ||
      selection.networkObjectKind === "switch_l3"
    ) {
      const messages = validateLightweightSwitchConfig(
        selection.networkObjectKind,
        draft.networkObjectConfig,
      );
      if (messages.length) errors.switchConfig = messages.join(" ");
    }
    return errors;
  }

  if (!context.version?.enabled)
    errors.version = "请选择已启用的设备模板版本。";
  if (!draft.imageVersionId) errors.image = "请选择可用镜像版本。";
  else if (
    !context.image ||
    context.image.runtime_kind !== context.template?.runtime_kind ||
    context.image.availability.toLowerCase() !== "available" ||
    context.image.license_status.toLowerCase() !== "reviewed" ||
    !context.version?.compatible_image_version_ids?.includes(context.image.id)
  )
    errors.image = "所选镜像已不可用或与当前模板不兼容，请重新选择。";
  if (
    !Number.isInteger(Number(draft.interfaceCount)) ||
    Number(draft.interfaceCount) < 1 ||
    Number(draft.interfaceCount) > 64
  )
    errors.interfaces = "接口数量必须是 1 到 64 的整数。";
  if (draft.ipv4Mode === "static" && !draft.ipv4Address.includes("/"))
    errors.ipv4Address = "请输入 IPv4 CIDR，例如 192.0.2.10/24。";
  if (draft.ipv6Mode === "static" && !draft.ipv6Address.includes("/"))
    errors.ipv6Address = "请输入 IPv6 CIDR，例如 2001:db8::10/64。";
  for (const route of draft.routes) {
    let message = "";
    if (!route.destination.includes("/")) message = "请输入目标 CIDR。";
    else if ((route.family === "ipv6") !== route.destination.includes(":"))
      message = `请输入 ${route.family === "ipv6" ? "IPv6" : "IPv4"} 目标。`;
    else if (
      route.gateway &&
      (route.family === "ipv6") !== route.gateway.includes(":")
    )
      message =
        "网关和目标必须使用相同地址族。 Gateway and destination must use the same address family.";
    else if (
      String(route.metric).trim() &&
      (!Number.isInteger(Number(route.metric)) || Number(route.metric) < 0)
    )
      message = "Metric 必须是非负整数。";
    if (message) errors[`route.${route.id}`] = message;
  }
  if (
    supportsUbuntuPasswordBootstrap(
      context.template?.template_key,
      context.version?.capabilities,
    )
  ) {
    if (!/^[a-z_][a-z0-9_-]{0,31}$/.test(draft.cloudUsername))
      errors.cloudUsername = "请输入最多 32 位的小写 Linux 用户名。";
    if (draft.cloudPassword.length < 12)
      errors.cloudPassword = "初始密码至少需要 12 个字符。";
  }
  return errors;
}

export function buildResourceCreateRequest(
  selection: PaletteSelection,
  draft: ResourceCreateDraft,
  context: DraftCatalogContext,
): ResourceCreateRequest {
  if (selection.networkObjectKind)
    return {
      kind: "network-object",
      request: {
        name: draft.name.trim(),
        kind: selection.networkObjectKind,
        config:
          selection.networkObjectKind === "pc"
            ? { ...draft.networkObjectConfig, hostname: draft.name.trim() }
            : draft.networkObjectConfig,
      },
    };

  const networkConfigurable =
    context.template?.runtime_kind === "docker" ||
    supportsUbuntuPasswordBootstrap(
      context.template?.template_key,
      context.version?.capabilities,
    );
  const format = context.version?.defaults.interface_name_format;
  const interfaceName = format?.includes("%d")
    ? format.replace("%d", "0")
    : "eth0";
  const supportsBootstrap = supportsUbuntuPasswordBootstrap(
    context.template?.template_key,
    context.version?.capabilities,
  );
  return {
    kind: "node",
    request: {
      name: draft.name.trim(),
      kind: context.template?.runtime_kind,
      template_version_id: draft.templateVersionId,
      image_version_id: draft.imageVersionId || undefined,
      interface_count: Number(draft.interfaceCount),
      config: networkConfigurable
        ? {
            network_interfaces: [
              {
                name: interfaceName,
                modes: [draft.ipv4Mode, draft.ipv6Mode].filter(
                  (mode) => mode !== "none",
                ),
                addresses: [
                  draft.ipv4Mode === "static" ? draft.ipv4Address.trim() : "",
                  draft.ipv6Mode === "static" ? draft.ipv6Address.trim() : "",
                ].filter(Boolean),
                routes: draft.routes.map((route) => ({
                  destination: route.destination.trim(),
                  gateway: route.gateway.trim() || undefined,
                  metric: String(route.metric).trim()
                    ? Number(route.metric)
                    : undefined,
                })),
              },
            ],
          }
        : undefined,
      bootstrap: supportsBootstrap
        ? {
            user_data: buildUbuntuPasswordCloudInit(
              draft.cloudUsername,
              draft.cloudPassword,
            ),
          }
        : undefined,
    },
  };
}
