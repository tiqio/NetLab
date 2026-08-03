export interface TemplateVersion {
  id: string;
  version: string;
  capabilities: string[];
  supported_nic_drivers: string[];
  console_modes: string[];
  enabled: boolean;
}
export interface DeviceTemplate {
  id: string;
  template_key: string;
  display_name: string;
  runtime_kind: "qemu" | "docker";
  versions: TemplateVersion[];
}
export interface ImageVersion {
  id: string;
  name: string;
  version: string;
  digest: string;
  availability: string;
  license_status: string;
}
async function json<T>(response: Response): Promise<T> {
  if (!response.ok) throw new Error(await response.text());
  return response.json() as Promise<T>;
}
export const templateApi = {
  list: () => fetch("/api/v1/templates").then(json<DeviceTemplate[]>),
  images: () => fetch("/api/v1/images").then(json<ImageVersion[]>),
  importImage: (body: Record<string, unknown>) =>
    fetch("/api/v1/images", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }).then(json<ImageVersion>),
};
