export function supportsUbuntuPasswordBootstrap(
  templateKey: string | undefined,
  capabilities: string[] | undefined,
) {
  const normalizedCapabilities = new Set(
    (capabilities || []).map((value) =>
      value.toLowerCase().replaceAll("-", "_"),
    ),
  );
  return (
    Boolean(templateKey?.toLowerCase().includes("ubuntu")) &&
    normalizedCapabilities.has("cloud_init")
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

export function buildUbuntuPasswordCloudInit(
  username: string,
  password: string,
) {
  return `#cloud-config\n${JSON.stringify(
    {
      users: ["default"],
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
