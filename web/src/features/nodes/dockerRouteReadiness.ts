import type { DockerStaticRoute, Node } from "@/api";

export type DockerRouteDeclaration = DockerStaticRoute & {
  interfaceName: string;
};

export type DockerRouteReadiness = {
  state: "none" | "pending" | "applying" | "applied" | "failed";
  label: string;
  routes: DockerRouteDeclaration[];
};

export function dockerRouteReadiness(node: Node): DockerRouteReadiness {
  const interfaces = Array.isArray(node.config?.network_interfaces)
    ? (node.config.network_interfaces as Array<Record<string, unknown>>)
    : [];
  const routes = interfaces.flatMap((interfaceValue) => {
    const interfaceName = String(interfaceValue.name || "interface");
    const values = Array.isArray(interfaceValue.routes)
      ? (interfaceValue.routes as Array<Record<string, unknown>>)
      : [];
    return values.map((route) => ({
      interfaceName,
      destination: String(route.destination || ""),
      gateway: route.gateway ? String(route.gateway) : undefined,
      metric:
        route.metric === undefined || route.metric === null
          ? undefined
          : Number(route.metric),
    }));
  });
  if (node.kind !== "docker" || routes.length === 0)
    return { state: "none", label: "未声明自定义路由", routes };

  const errorText = [
    node.last_error?.code,
    node.last_error?.phase,
    node.last_error?.message,
  ]
    .filter(Boolean)
    .join(" ")
    .toLowerCase();
  if (
    node.last_error &&
    (errorText.includes("route") ||
      errorText.includes("gateway") ||
      errorText.includes("runtime_configuration"))
  )
    return { state: "failed", label: "路由应用失败", routes };
  if (node.observed_state === "running")
    return { state: "applied", label: "路由已应用", routes };
  if (
    node.desired_state === "running" &&
    ["provisioning", "starting", "pending"].includes(node.observed_state)
  )
    return { state: "applying", label: "正在应用路由", routes };
  return { state: "pending", label: "将在下次启动时应用", routes };
}
