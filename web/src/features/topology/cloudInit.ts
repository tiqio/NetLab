export function supportsCloudInitBootstrap(
  templateKey: string | undefined,
  capabilities: string[] | undefined,
) {
  const normalized = new Set(
    (capabilities || []).map((value) =>
      value.toLowerCase().replaceAll("-", "_"),
    ),
  );
  return Boolean(templateKey) && normalized.has("cloud_init");
}

export function supportsUbuntuPasswordBootstrap(
  templateKey: string | undefined,
  capabilities: string[] | undefined,
) {
  return (
    Boolean(templateKey?.toLowerCase().includes("ubuntu")) &&
    supportsCloudInitBootstrap(templateKey, capabilities)
  );
}

export function generateInitialPassword(length = 18) {
  const alphabet =
    "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789-_.";
  const bytes = crypto.getRandomValues(new Uint8Array(length));
  return Array.from(bytes, (value) => alphabet[value % alphabet.length]).join(
    "",
  );
}

function standardLinuxCloudInit(
  hostname: string,
  username: string,
  password: string,
) {
  return `#cloud-config\n${JSON.stringify(
    {
      hostname,
      preserve_hostname: false,
      users: [
        "default",
        {
          name: username,
          groups: "sudo",
          shell: "/bin/bash",
          sudo: "ALL=(ALL) NOPASSWD:ALL",
        },
      ],
      ssh_pwauth: true,
      chpasswd: {
        expire: false,
        users: [{ name: username, password, type: "text" }],
      },
    },
    null,
    2,
  )}\n`;
}

export function buildTemplateCloudInit(input: {
  templateKey: string;
  hostname: string;
  username: string;
  password: string;
  interfaceName: string;
  ipv4Mode: string;
  ipv4Address: string;
  routes: Array<{ family: string; destination: string; gateway: string }>;
}) {
  const hostname =
    input.hostname
      .toLowerCase()
      .replace(/[^a-z0-9-]+/g, "-")
      .replace(/^-+|-+$/g, "")
      .slice(0, 63) || "netlab-node";
  if (input.templateKey === "vyos") {
    const commands = [
      `set system host-name '${hostname}'`,
      "set service ssh",
      `set system login user '${input.username}' authentication plaintext-password '${input.password}'`,
    ];
    if (input.ipv4Mode === "dhcpv4")
      commands.push(
        `set interfaces ethernet ${input.interfaceName} address dhcp`,
      );
    if (input.ipv4Mode === "static" && input.ipv4Address)
      commands.push(
        `set interfaces ethernet ${input.interfaceName} address '${input.ipv4Address}'`,
      );
    for (const route of input.routes) {
      if (route.family === "ipv4" && route.destination && route.gateway)
        commands.push(
          `set protocols static route '${route.destination}' next-hop '${route.gateway}'`,
        );
      if (route.family === "ipv6" && route.destination && route.gateway)
        commands.push(
          `set protocols static route6 '${route.destination}' next-hop '${route.gateway}'`,
        );
    }
    return `#cloud-config\n${JSON.stringify({ vyos_config_commands: commands }, null, 2)}\n`;
  }
  return standardLinuxCloudInit(hostname, input.username, input.password);
}

export function buildUbuntuPasswordCloudInit(
  username: string,
  password: string,
) {
  return standardLinuxCloudInit("ubuntu", username, password);
}
