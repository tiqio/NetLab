<script setup lang="ts">
import { computed, ref, watch } from "vue";
import {
  api,
  ApiError,
  type DeviceTemplate,
  type ImageVersion,
  type Node,
  type NodeInterface,
  type NetworkObject,
} from "@/api";
import { Button, Dialog, FormField, Input, Select } from "@/components/ui";
import LightweightSwitchConfigEditor from "@/features/nodes/LightweightSwitchConfigEditor.vue";
import {
  defaultLightweightSwitchConfig,
  validateLightweightSwitchConfig,
  type LightweightSwitchKind,
} from "@/features/nodes/lightweightSwitchConfig";
import type { PaletteSelection } from "./DevicePalette.vue";
import {
  buildUbuntuPasswordCloudInit,
  generateInitialPassword,
  supportsUbuntuPasswordBootstrap,
} from "./cloudInit";

const props = defineProps<{
  modelValue: boolean;
  laboratoryId: string;
  selection?: PaletteSelection;
}>();
const emit = defineEmits<{
  "update:modelValue": [boolean];
  created: [
    {
      node?: Node;
      interfaces?: NodeInterface[];
      networkObject?: NetworkObject;
    },
  ];
}>();
const name = ref("");
const templateId = ref("");
const versionId = ref("");
const imageVersionId = ref("");
const interfaceCount = ref(2);
const ipv4Mode = ref<"none" | "static" | "dhcpv4">("none");
const ipv4Address = ref("");
const ipv6Mode = ref<"none" | "static" | "slaac" | "dhcpv6">("none");
const ipv6Address = ref("");
type RouteDraft = {
  id: string;
  family: "ipv4" | "ipv6";
  destination: string;
  gateway: string;
  metric: string | number;
};
let routeSequence = 0;
const routes = ref<RouteDraft[]>([]);
function addRoute(family: "ipv4" | "ipv6") {
  routeSequence += 1;
  routes.value.push({
    id: `route-${routeSequence}`,
    family,
    destination: family === "ipv6" ? "::/0" : "0.0.0.0/0",
    gateway: "",
    metric: "",
  });
}
function removeRoute(id: string) {
  routes.value = routes.value.filter((route) => route.id !== id);
}
const cloudUsername = ref("ubuntu");
const cloudPassword = ref("");
const busy = ref(false);
const catalogLoading = ref(false);
const error = ref("");
const staleMessage = ref("");
const fieldErrors = ref<Record<string, string>>({});
const templates = ref<DeviceTemplate[]>([]);
const images = ref<ImageVersion[]>([]);
const lightweightSwitchConfig = ref<Record<string, unknown>>({});
const open = computed({
  get: () => props.modelValue,
  set: (value) => emit("update:modelValue", value),
});
const selectedTemplate = computed(() =>
  templates.value.find((item) => item.id === templateId.value),
);
const selectedVersion = computed(() =>
  selectedTemplate.value?.versions.find((item) => item.id === versionId.value),
);
const dockerSelected = computed(
  () => selectedTemplate.value?.runtime_kind === "docker",
);
const ubuntuPasswordBootstrap = computed(() =>
  supportsUbuntuPasswordBootstrap(
    selectedTemplate.value?.template_key,
    selectedVersion.value?.capabilities,
  ),
);
const networkConfigurable = computed(
  () => dockerSelected.value || ubuntuPasswordBootstrap.value,
);
const initialInterfaceName = computed(() => {
  const format = selectedVersion.value?.defaults.interface_name_format;
  return format?.includes("%d") ? format.replace("%d", "0") : "eth0";
});
watch(ubuntuPasswordBootstrap, (supported) => {
  if (supported && !cloudPassword.value)
    cloudPassword.value = generateInitialPassword();
  if (!supported) cloudPassword.value = "";
});
function lightweightConfig(
  kind: NonNullable<PaletteSelection["networkObjectKind"]>,
) {
  if (kind === "pc")
    return {
      hostname: name.value,
      interfaces: [{ name: "eth0", modes: ["dhcpv4", "dhcpv6", "slaac"] }],
    };
  if (kind === "nat_bridge")
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
  if (kind === "switch_l2" || kind === "switch_l3")
    return lightweightSwitchConfig.value;
  return { mtu: 1500, stp: false };
}
const selectedImage = computed(() =>
  images.value.find((item) => item.id === imageVersionId.value),
);
function imageMatchesTemplate(image: ImageVersion) {
  if (selectedTemplate.value?.runtime_kind !== "docker") return true;
  const key = selectedTemplate.value.template_key.toLowerCase();
  const name = image.name.toLowerCase();
  if (key.includes("ubuntu")) return name.includes("ubuntu");
  if (key.includes("busybox")) return name.includes("busybox");
  return true;
}
const compatibleImages = computed(() =>
  images.value
    .filter(
      (item) =>
        item.runtime_kind === selectedTemplate.value?.runtime_kind &&
        item.availability.toLowerCase() === "available" &&
        imageMatchesTemplate(item),
    )
    .sort((left, right) => {
      const leftPreferred = left.name.includes("network-tools") ? 1 : 0;
      const rightPreferred = right.name.includes("network-tools") ? 1 : 0;
      return rightPreferred - leftPreferred;
    }),
);
const imageHint = computed(() =>
  compatibleImages.value.length
    ? "Only available images matching the selected runtime can be used."
    : "No compatible image is available. Import an image from the Templates page before creating this device.",
);
function imageUnavailableReason(image: ImageVersion) {
  if (!selectedTemplate.value) return "select a template first";
  if (image.runtime_kind !== selectedTemplate.value.runtime_kind)
    return `requires ${image.runtime_kind.toUpperCase()}`;
  if (!imageMatchesTemplate(image)) return "not compatible with this template";
  if (image.availability.toLowerCase() !== "available")
    return image.availability || "unavailable";
  return "";
}
const canSubmit = computed(() => {
  if (!name.value.trim() || busy.value || catalogLoading.value) return false;
  if (props.selection?.networkObjectKind) return true;
  if (!selectedTemplate.value || !selectedVersion.value?.enabled) return false;
  if (!imageVersionId.value) return false;
  return Boolean(
    selectedImage.value && !imageUnavailableReason(selectedImage.value),
  );
});
const dirty = computed(() =>
  Boolean(
    name.value.trim() ||
    versionId.value ||
    imageVersionId.value ||
    interfaceCount.value !== 2 ||
    cloudPassword.value ||
    routes.value.length > 0,
  ),
);
function validate() {
  const next: Record<string, string> = {};
  if (!name.value.trim()) next.name = "Name is required.";
  if (name.value.trim().length > 120)
    next.name = "Name must be 120 characters or fewer.";
  const networkKind = props.selection?.networkObjectKind;
  if (networkKind === "switch_l2" || networkKind === "switch_l3") {
    const messages = validateLightweightSwitchConfig(
      networkKind,
      lightweightSwitchConfig.value,
    );
    if (messages.length) next.switchConfig = messages.join(" ");
  }
  if (!props.selection?.networkObjectKind) {
    if (!versionId.value) next.version = "Select an enabled template version.";
    if (!imageVersionId.value)
      next.image = "Select an available image version.";
    if (
      !Number.isInteger(Number(interfaceCount.value)) ||
      Number(interfaceCount.value) < 1 ||
      Number(interfaceCount.value) > 64
    )
      next.interfaces = "Interfaces must be an integer from 1 to 64.";
    if (ipv4Mode.value === "static" && !ipv4Address.value.includes("/"))
      next.ipv4Address = "Enter an IPv4 CIDR such as 192.0.2.10/24.";
    if (ipv6Mode.value === "static" && !ipv6Address.value.includes("/"))
      next.ipv6Address = "Enter an IPv6 CIDR such as 2001:db8::10/64.";
    for (const route of routes.value) {
      if (!route.destination.includes("/"))
        next[`route.${route.id}`] = "Enter a destination CIDR.";
      else if ((route.family === "ipv6") !== route.destination.includes(":"))
        next[`route.${route.id}`] =
          `Use an ${route.family === "ipv6" ? "IPv6" : "IPv4"} destination.`;
      else if (
        route.gateway &&
        (route.family === "ipv6") !== route.gateway.includes(":")
      )
        next[`route.${route.id}`] =
          "Gateway and destination must use the same address family.";
      else if (
        String(route.metric).trim() &&
        (!Number.isInteger(Number(route.metric)) || Number(route.metric) < 0)
      )
        next[`route.${route.id}`] = "Metric must be a non-negative integer.";
    }
    if (ubuntuPasswordBootstrap.value) {
      if (!/^[a-z_][a-z0-9_-]{0,31}$/.test(cloudUsername.value))
        next.cloudUsername =
          "Use a lowercase Linux username up to 32 characters.";
      if (cloudPassword.value.length < 12)
        next.cloudPassword =
          "Use an initial password of at least 12 characters.";
    }
  }
  fieldErrors.value = next;
  return Object.keys(next).length === 0;
}
async function loadCatalog(preserveSelection = true) {
  if (props.selection?.networkObjectKind) return;
  const previousTemplate = templateId.value;
  const previousVersion = versionId.value;
  const previousImage = imageVersionId.value;
  catalogLoading.value = true;
  try {
    const [nextTemplates, nextImages] = await Promise.all([
      api.listTemplates(),
      api.listImages(),
    ]);
    templates.value = nextTemplates;
    images.value = nextImages;
    templateId.value =
      (preserveSelection &&
        nextTemplates.some((item) => item.id === previousTemplate) &&
        previousTemplate) ||
      props.selection?.template?.id ||
      "";
    const template = nextTemplates.find((item) => item.id === templateId.value);
    versionId.value =
      (preserveSelection &&
        template?.versions.some((item) => item.id === previousVersion) &&
        previousVersion) ||
      props.selection?.version?.id ||
      template?.versions.find((item) => item.enabled)?.id ||
      "";
    const version = template?.versions.find(
      (item) => item.id === versionId.value,
    );
    imageVersionId.value =
      (preserveSelection &&
        nextImages.some((item) => item.id === previousImage) &&
        previousImage) ||
      version?.image_version_id ||
      compatibleImages.value[0]?.id ||
      "";
  } finally {
    catalogLoading.value = false;
  }
}
watch(
  () => props.selection,
  (selection) => {
    name.value = selection?.name || "";
    templateId.value = selection?.template?.id || "";
    versionId.value = selection?.version?.id || "";
    imageVersionId.value = selection?.version?.image_version_id || "";
    cloudUsername.value = "ubuntu";
    cloudPassword.value = supportsUbuntuPasswordBootstrap(
      selection?.template?.template_key,
      selection?.version?.capabilities,
    )
      ? generateInitialPassword()
      : "";
    const networkKind = selection?.networkObjectKind;
    lightweightSwitchConfig.value =
      networkKind === "switch_l2" || networkKind === "switch_l3"
        ? defaultLightweightSwitchConfig(networkKind)
        : {};
    routes.value = [];
  },
  { immediate: true },
);
watch(open, (value) => {
  if (value) void loadCatalog(false);
});
watch(templateId, () => {
  if (
    !selectedTemplate.value?.versions.some(
      (item) => item.id === versionId.value,
    )
  ) {
    versionId.value =
      selectedTemplate.value?.versions.find((item) => item.enabled)?.id || "";
    imageVersionId.value = "";
  }
  staleMessage.value = "";
});
watch(versionId, () => {
  const recommended = selectedVersion.value?.image_version_id;
  imageVersionId.value =
    recommended && images.value.some((item) => item.id === recommended)
      ? recommended
      : compatibleImages.value[0]?.id || "";
  staleMessage.value = "";
});
async function submit() {
  if (!props.selection || !canSubmit.value || !validate()) return;
  busy.value = true;
  error.value = "";
  try {
    if (props.selection.networkObjectKind) {
      const value = await api.createNetworkObject(props.laboratoryId, {
        name: name.value,
        kind: props.selection.networkObjectKind,
        config: lightweightConfig(props.selection.networkObjectKind),
      });
      emit("created", { networkObject: value.network_object });
    } else {
      const chosenTemplateId = templateId.value;
      const chosenVersionId = versionId.value;
      const chosenImageId = imageVersionId.value;
      await loadCatalog(true);
      const currentTemplate = templates.value.find(
        (item) => item.id === chosenTemplateId,
      );
      const currentVersion = currentTemplate?.versions.find(
        (item) => item.id === chosenVersionId,
      );
      const currentImage = images.value.find(
        (item) => item.id === chosenImageId,
      );
      if (
        !currentTemplate ||
        !currentVersion?.enabled ||
        (chosenImageId &&
          (!currentImage ||
            currentImage.runtime_kind !== currentTemplate.runtime_kind ||
            currentImage.availability.toLowerCase() !== "available"))
      ) {
        staleMessage.value =
          "The selected template or image changed on the server. Your other values are preserved; choose an available version and retry.";
        return;
      }
      const value = await api.createNode(props.laboratoryId, {
        name: name.value,
        kind: currentTemplate.runtime_kind,
        template_version_id: chosenVersionId,
        image_version_id: chosenImageId || undefined,
        interface_count: interfaceCount.value,
        config: networkConfigurable.value
          ? {
              network_interfaces: [
                {
                  name: initialInterfaceName.value,
                  modes: [ipv4Mode.value, ipv6Mode.value].filter(
                    (mode) => mode !== "none",
                  ),
                  addresses: [
                    ipv4Mode.value === "static" ? ipv4Address.value : "",
                    ipv6Mode.value === "static" ? ipv6Address.value : "",
                  ].filter(Boolean),
                  routes: routes.value.map((route) => ({
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
        bootstrap: ubuntuPasswordBootstrap.value
          ? {
              user_data: buildUbuntuPasswordCloudInit(
                cloudUsername.value,
                cloudPassword.value,
              ),
            }
          : undefined,
      });
      emit("created", value);
    }
    open.value = false;
  } catch (value) {
    if (value instanceof ApiError) {
      error.value = value.problem.message;
      const fields = value.problem.details?.fields;
      if (fields && typeof fields === "object")
        fieldErrors.value = Object.fromEntries(
          Object.entries(fields).map(([key, message]) => [
            key,
            String(message),
          ]),
        );
    } else error.value = value instanceof Error ? value.message : String(value);
  } finally {
    busy.value = false;
  }
}
</script>
<template>
  <Dialog
    v-model="open"
    :prevent-close="dirty && !busy"
    :title="`Add ${selection?.name || 'device'}`"
    description="The resource and confirmed placement are shared with every client; viewport and manual link routes remain local to this browser."
  >
    <form class="grid gap-3" @submit.prevent="submit">
      <FormField label="Name" :error="fieldErrors.name">
        <Input v-model="name" required maxlength="120" /> </FormField
      ><FormField
        v-if="
          selection?.networkObjectKind === 'switch_l2' ||
          selection?.networkObjectKind === 'switch_l3'
        "
        label="交换机配置"
        :error="fieldErrors.switchConfig"
        hint="端口、VLAN、接口地址和路由都将作为运行时配置保存。"
      >
        <LightweightSwitchConfigEditor
          v-model="lightweightSwitchConfig"
          :kind="selection.networkObjectKind as LightweightSwitchKind"
        /> </FormField
      ><FormField
        v-if="!selection?.networkObjectKind"
        label="Device template"
        :error="fieldErrors.template"
      >
        <Select v-model="templateId" :disabled="catalogLoading">
          <option value="">Select a template</option>
          <option v-for="item in templates" :key="item.id" :value="item.id">
            {{ item.display_name }} · {{ item.runtime_kind.toUpperCase() }}
          </option>
        </Select> </FormField
      ><FormField
        v-if="!selection?.networkObjectKind"
        label="Template version"
        :error="fieldErrors.version"
      >
        <Select v-model="versionId">
          <option value="">Select a version</option>
          <option
            v-for="version in selectedTemplate?.versions || []"
            :key="version.id"
            :value="version.id"
            :disabled="!version.enabled"
          >
            {{ version.version }}{{ version.enabled ? "" : " (disabled)" }}
          </option>
        </Select> </FormField
      ><FormField
        v-if="!selection?.networkObjectKind"
        label="Image version"
        :error="fieldErrors.image"
        :hint="imageHint"
      >
        <Select v-model="imageVersionId" :disabled="!selectedVersion">
          <option value="">Select an image version</option>
          <option
            v-for="image in images"
            :key="image.id"
            :value="image.id"
            :disabled="Boolean(imageUnavailableReason(image))"
          >
            {{ image.name }} {{ image.version
            }}{{
              imageUnavailableReason(image)
                ? ` (${imageUnavailableReason(image)})`
                : image.id === selectedVersion?.image_version_id
                  ? " (recommended)"
                  : ""
            }}
          </option>
        </Select> </FormField
      ><FormField
        v-if="!selection?.networkObjectKind"
        label="Interfaces (count)"
        :error="fieldErrors.interfaces"
        hint="Whole number from 1 to 64."
      >
        <Input
          v-model="interfaceCount"
          type="number"
          min="1"
          max="64"
          step="1"
        />
      </FormField>
      <div
        v-if="networkConfigurable"
        class="grid gap-3 rounded-md border border-border bg-muted/10 p-3"
      >
        <p class="text-xs font-medium">
          Initial {{ initialInterfaceName }} network configuration
        </p>
        <FormField label="IPv4 mode">
          <Select v-model="ipv4Mode" data-testid="docker-ipv4-mode">
            <option value="none">No automatic IPv4 configuration</option>
            <option value="static">Static IPv4</option>
            <option value="dhcpv4">DHCPv4</option>
          </Select>
        </FormField>
        <FormField
          v-if="ipv4Mode === 'static'"
          label="IPv4 CIDR"
          :error="fieldErrors.ipv4Address"
        >
          <Input
            v-model="ipv4Address"
            data-testid="docker-ipv4-address"
            placeholder="192.0.2.10/24"
          />
        </FormField>
        <div
          class="grid gap-2 rounded-md border border-border/60 p-2"
          data-testid="docker-route-editor"
        >
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div>
              <p class="text-xs font-medium">Static routes</p>
              <p class="text-[11px] text-muted-foreground">
                The gateway must be reachable through the static address on this
                interface.
              </p>
            </div>
            <div class="flex gap-2">
              <Button
                type="button"
                size="sm"
                variant="secondary"
                @click="addRoute('ipv4')"
                >Add IPv4 route</Button
              >
              <Button
                type="button"
                size="sm"
                variant="secondary"
                @click="addRoute('ipv6')"
                >Add IPv6 route</Button
              >
            </div>
          </div>
          <div
            v-for="(route, routeIndex) in routes"
            :key="route.id"
            class="grid gap-2 rounded-md bg-muted/20 p-2 md:grid-cols-[1.4fr_1fr_7rem_auto]"
          >
            <FormField
              label="Destination CIDR"
              :error="fieldErrors[`route.${route.id}`]"
            >
              <Input
                v-model="route.destination"
                :data-testid="`docker-route-${routeIndex}-destination`"
                :placeholder="route.family === 'ipv6' ? '::/0' : '0.0.0.0/0'"
              />
            </FormField>
            <FormField label="Gateway (optional)">
              <Input
                v-model="route.gateway"
                :data-testid="`docker-route-${routeIndex}-gateway`"
                :placeholder="
                  route.family === 'ipv6' ? '2001:db8::1' : '192.0.2.1'
                "
              />
            </FormField>
            <FormField label="Metric">
              <Input
                v-model="route.metric"
                :data-testid="`docker-route-${routeIndex}-metric`"
                type="number"
                min="0"
                step="1"
                placeholder="Default"
              />
            </FormField>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              :aria-label="`Remove route ${routeIndex + 1}`"
              @click="removeRoute(route.id)"
              >Remove</Button
            >
          </div>
          <p v-if="!routes.length" class="text-xs text-muted-foreground">
            No custom static routes.
          </p>
        </div>
        <FormField label="IPv6 mode">
          <Select v-model="ipv6Mode" data-testid="docker-ipv6-mode">
            <option value="none">Link-local only</option>
            <option value="static">Static IPv6</option>
            <option value="slaac">SLAAC</option>
            <option value="dhcpv6">DHCPv6</option>
          </Select>
        </FormField>
        <FormField
          v-if="ipv6Mode === 'static'"
          label="IPv6 CIDR"
          :error="fieldErrors.ipv6Address"
        >
          <Input
            v-model="ipv6Address"
            data-testid="docker-ipv6-address"
            placeholder="2001:db8::10/64"
          />
        </FormField>
        <p class="text-xs text-muted-foreground">
          <template v-if="dockerSelected">
            DHCP and diagnostic commands require a network-tools image.
            Addresses are applied after the interface enters the container
            network namespace.
          </template>
          <template v-else>
            The backend matches this logical interface to its generated MAC
            address and writes cloud-init network-config into the seed ISO.
          </template>
        </p>
      </div>
      <div
        v-if="ubuntuPasswordBootstrap"
        class="grid gap-3 rounded-md border border-border bg-muted/10 p-3"
      >
        <div class="flex items-center justify-between gap-2">
          <div>
            <p class="text-xs font-medium">Ubuntu initial login</p>
            <p class="text-[11px] text-muted-foreground">
              Injected once through the node-scoped cloud-init seed ISO and not
              saved in browser preferences.
            </p>
          </div>
          <Button
            type="button"
            size="sm"
            variant="secondary"
            @click="cloudPassword = generateInitialPassword()"
          >
            Regenerate
          </Button>
        </div>
        <FormField label="Initial username" :error="fieldErrors.cloudUsername">
          <Input v-model="cloudUsername" autocomplete="username" />
        </FormField>
        <FormField
          label="Initial password"
          :error="fieldErrors.cloudPassword"
          hint="Record this value before creating the node; it cannot be recovered from the UI later."
        >
          <Input
            v-model="cloudPassword"
            type="text"
            autocomplete="new-password"
            spellcheck="false"
          />
        </FormField>
      </div>
      <p v-if="error" role="alert" class="text-xs text-destructive">
        {{ error }}
      </p>
      <p v-if="staleMessage" role="alert" class="text-xs text-amber-300">
        {{ staleMessage }}
      </p>
      <div class="flex justify-end gap-2">
        <Button type="button" variant="secondary" @click="open = false">
          Cancel </Button
        ><Button type="submit" :disabled="!canSubmit">
          {{ busy || catalogLoading ? "Checking…" : "Add to topology" }}
        </Button>
      </div>
    </form>
  </Dialog>
</template>
