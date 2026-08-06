<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import {
  api,
  ApiError,
  type DeviceTemplate,
  type ImageVersion,
  type Node,
  type NodeInterface,
  type NetworkObject,
} from "@/api";
import { Button, FormField, Input, Select, Sheet } from "@/components/ui";
import LightweightSwitchConfigEditor from "@/features/nodes/LightweightSwitchConfigEditor.vue";
import { type LightweightSwitchKind } from "@/features/nodes/lightweightSwitchConfig";
import TopologyResourceCatalog, {
  type PaletteSelection,
} from "./TopologyResourceCatalog.vue";
import {
  generateInitialPassword,
  supportsUbuntuPasswordBootstrap,
} from "./cloudInit";
import {
  buildResourceCreateRequest,
  createResourceDraft,
  draftSignature,
  validateResourceDraft,
  type ResourceCreateDraft,
} from "./topologyResourceDraft";

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
  selectionChanged: [PaletteSelection | undefined];
}>();
const name = ref("");
const templateId = ref("");
const templateSelectKey = ref(0);
const versionId = ref("");
const imageVersionId = ref("");
const cpuCount = ref(1);
const cpuQuotaMicros = ref(0);
const memoryMiB = ref(512);
const storageGiB = ref(0);
const interfaceLimit = ref(64);
const processLimit = ref(4096);
const nicDriver = ref("");
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
let submitLocked = false;
const catalogLoading = ref(false);
const catalogError = ref("");
const error = ref("");
const staleMessage = ref("");
const fieldErrors = ref<Record<string, string>>({});
const templates = ref<DeviceTemplate[]>([]);
const images = ref<ImageVersion[]>([]);
const lightweightSwitchConfig = ref<Record<string, unknown>>({});
const initialSignature = ref("");
const expandedSections = ref({
  resources: true,
  network: true,
  bootstrap: true,
});
const sheetRef = ref<{
  requestClose: (reason: "button" | "overlay" | "escape") => void;
  requestDiscardConfirmation: (action: () => void) => void;
}>();
const open = computed({
  get: () => props.modelValue,
  set: (value) => emit("update:modelValue", value),
});
const selectedTemplate = computed(() =>
  templates.value.find((item) => item.id === templateId.value),
);
const compatibleTemplates = computed(() =>
  templates.value.filter(
    (item) =>
      item.runtime_kind === props.selection?.kind &&
      item.versions.some((version) => version.enabled),
  ),
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
const currentDraft = computed<ResourceCreateDraft>(() => ({
  name: name.value,
  templateId: templateId.value,
  templateVersionId: versionId.value,
  imageVersionId: imageVersionId.value,
  cpuCount: Number(cpuCount.value),
  cpuQuotaMicros: Number(cpuQuotaMicros.value),
  memoryMiB: Number(memoryMiB.value),
  storageGiB: Number(storageGiB.value),
  interfaceLimit: Number(interfaceLimit.value),
  processLimit: Number(processLimit.value),
  nicDriver: nicDriver.value,
  interfaceCount: Number(interfaceCount.value),
  ipv4Mode: ipv4Mode.value,
  ipv4Address: ipv4Address.value,
  ipv6Mode: ipv6Mode.value,
  ipv6Address: ipv6Address.value,
  routes: routes.value,
  cloudUsername: cloudUsername.value,
  cloudPassword: cloudPassword.value,
  networkObjectConfig: lightweightSwitchConfig.value,
}));
const dirty = computed(
  () =>
    Boolean(props.selection) &&
    draftSignature(currentDraft.value) !== initialSignature.value,
);
function requestDrawerClose() {
  sheetRef.value?.requestClose("button");
}
function changeResource() {
  const change = () => emit("selectionChanged", undefined);
  if (dirty.value) sheetRef.value?.requestDiscardConfirmation(change);
  else change();
}
function requestExternalDiscard(action: () => void) {
  if (dirty.value) sheetRef.value?.requestDiscardConfirmation(action);
  else action();
}
defineExpose({ isDirty: () => dirty.value, requestExternalDiscard });
const selectedImage = computed(() =>
  images.value.find((item) => item.id === imageVersionId.value),
);
function imageMatchesTemplate(image: ImageVersion) {
  if (!selectedTemplate.value || !selectedVersion.value) return false;
  return Boolean(
    selectedVersion.value.compatible_image_version_ids?.includes(image.id),
  );
}
const templateCompatibleImages = computed(() =>
  images.value.filter((item) => imageMatchesTemplate(item)),
);
const compatibleImages = computed(() =>
  templateCompatibleImages.value
    .filter(
      (item) =>
        item.runtime_kind === selectedTemplate.value?.runtime_kind &&
        item.availability.toLowerCase() === "available" &&
        item.license_status.toLowerCase() === "reviewed",
    )
    .sort((left, right) => {
      const leftPreferred = left.name.includes("network-tools") ? 1 : 0;
      const rightPreferred = right.name.includes("network-tools") ? 1 : 0;
      return rightPreferred - leftPreferred;
    }),
);
const imageHint = computed(() =>
  compatibleImages.value.length
    ? "Only reviewed images assigned to this device family are shown."
    : `No reviewed ${selectedTemplate.value?.display_name || "compatible"} image is available. Import the correct image from the Templates page first.`,
);
function imageUnavailableReason(image: ImageVersion) {
  if (!selectedTemplate.value) return "select a template first";
  if (image.runtime_kind !== selectedTemplate.value.runtime_kind)
    return `requires ${image.runtime_kind.toUpperCase()}`;
  if (!imageMatchesTemplate(image)) return "not compatible with this template";
  if (image.availability.toLowerCase() !== "available")
    return image.availability || "unavailable";
  if (image.license_status.toLowerCase() !== "reviewed")
    return `license ${image.license_status || "unreviewed"}`;
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
async function focusFirstError() {
  await nextTick();
  const firstKey = Object.keys(fieldErrors.value)[0];
  if (!firstKey) return;
  const selector = firstKey.startsWith("route.")
    ? `[data-route-id="${CSS.escape(firstKey.slice(6))}"] input`
    : `[data-field="${CSS.escape(firstKey)}"] input, [data-field="${CSS.escape(firstKey)}"] select, [data-field="${CSS.escape(firstKey)}"] button`;
  const element = document.querySelector<HTMLElement>(selector);
  element?.scrollIntoView?.({ block: "center", behavior: "smooth" });
  element?.focus();
}
async function validate() {
  if (!props.selection) return false;
  fieldErrors.value = validateResourceDraft(
    props.selection,
    currentDraft.value,
    {
      template: selectedTemplate.value,
      version: selectedVersion.value,
      image: selectedImage.value,
    },
  );
  if (Object.keys(fieldErrors.value).length) await focusFirstError();
  return Object.keys(fieldErrors.value).length === 0;
}
async function loadCatalog(preserveSelection = true) {
  if (props.selection?.networkObjectKind) return true;
  const previousTemplate = templateId.value;
  const previousVersion = versionId.value;
  const previousImage = imageVersionId.value;
  catalogLoading.value = true;
  catalogError.value = "";
  try {
    const [nextTemplates, nextImages] = await Promise.all([
      api.listTemplates(),
      api.listImages(),
    ]);
    templates.value = nextTemplates.filter(
      (item) => item.runtime_kind === props.selection?.kind,
    );
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
        version?.compatible_image_version_ids?.includes(previousImage) &&
        previousImage) ||
      version?.image_version_id ||
      compatibleImages.value[0]?.id ||
      "";
    await nextTick();
    if (!preserveSelection)
      initialSignature.value = draftSignature(currentDraft.value);
    return true;
  } catch (value) {
    catalogError.value = `无法加载模板和镜像目录：${value instanceof Error ? value.message : String(value)}`;
    return false;
  } finally {
    catalogLoading.value = false;
  }
}
watch(
  () => props.selection,
  (selection) => {
    if (!selection) {
      initialSignature.value = "";
      return;
    }
    const draft = createResourceDraft(selection, generateInitialPassword);
    name.value = draft.name;
    templateId.value = draft.templateId;
    versionId.value = draft.templateVersionId;
    imageVersionId.value = draft.imageVersionId;
    cpuCount.value = draft.cpuCount;
    cpuQuotaMicros.value = draft.cpuQuotaMicros;
    memoryMiB.value = draft.memoryMiB;
    storageGiB.value = draft.storageGiB;
    interfaceLimit.value = draft.interfaceLimit;
    processLimit.value = draft.processLimit;
    nicDriver.value = draft.nicDriver;
    interfaceCount.value = draft.interfaceCount;
    ipv4Mode.value = draft.ipv4Mode;
    ipv4Address.value = draft.ipv4Address;
    ipv6Mode.value = draft.ipv6Mode;
    ipv6Address.value = draft.ipv6Address;
    cloudUsername.value = draft.cloudUsername;
    cloudPassword.value = draft.cloudPassword;
    lightweightSwitchConfig.value = draft.networkObjectConfig;
    routes.value = draft.routes;
    fieldErrors.value = {};
    error.value = "";
    staleMessage.value = "";
    initialSignature.value = draftSignature(draft);
  },
  { immediate: true },
);
watch(
  open,
  (value) => {
    if (value) void loadCatalog(false);
  },
  { immediate: true },
);
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
  const defaults = selectedVersion.value?.defaults;
  if (defaults) {
    cpuCount.value = defaults.cpu_count || 1;
    cpuQuotaMicros.value = defaults.cpu_quota_micros || 0;
    memoryMiB.value = defaults.memory_mib || 512;
    storageGiB.value = defaults.disk_gib || 0;
    interfaceCount.value = defaults.interfaces || 1;
  }
  nicDriver.value = selectedVersion.value?.supported_nic_drivers[0] || "";
  staleMessage.value = "";
});
watch(versionId, () => {
  const recommended = selectedVersion.value?.image_version_id;
  imageVersionId.value =
    recommended && images.value.some((item) => item.id === recommended)
      ? recommended
      : compatibleImages.value[0]?.id || "";
  const defaults = selectedVersion.value?.defaults;
  if (defaults) {
    cpuCount.value = defaults.cpu_count || 1;
    cpuQuotaMicros.value = defaults.cpu_quota_micros || 0;
    memoryMiB.value = defaults.memory_mib || 512;
    storageGiB.value = defaults.disk_gib || 0;
    interfaceCount.value = defaults.interfaces || 1;
  }
  nicDriver.value = selectedVersion.value?.supported_nic_drivers[0] || "";
  staleMessage.value = "";
});
function requestTemplateChange(value: string | number | undefined) {
  const nextTemplateId = String(value || "");
  if (nextTemplateId === templateId.value) return;
  const apply = () => {
    templateId.value = nextTemplateId;
  };
  if (dirty.value) {
    templateSelectKey.value += 1;
    sheetRef.value?.requestDiscardConfirmation(apply);
  } else apply();
}
const serverFieldMap: Record<string, string> = {
  name: "name",
  template_id: "template",
  template_version_id: "version",
  image_version_id: "image",
  interface_count: "interfaces",
  cpu_count: "cpuCount",
  cpu_quota_micros: "cpuQuotaMicros",
  memory_mib: "memoryMiB",
  storage_gib: "storageGiB",
  interface_limit: "interfaceLimit",
  process_limit: "processLimit",
  nic_driver: "nicDriver",
};
async function submit() {
  if (!props.selection || !canSubmit.value || busy.value || submitLocked)
    return;
  submitLocked = true;
  error.value = "";
  if (!(await validate())) {
    submitLocked = false;
    return;
  }
  busy.value = true;
  try {
    if (props.selection.networkObjectKind) {
      const create = buildResourceCreateRequest(
        props.selection,
        currentDraft.value,
        {},
      );
      if (create.kind !== "network-object") return;
      const value = await api.createNetworkObject(
        props.laboratoryId,
        create.request,
      );
      emit("created", { networkObject: value.network_object });
    } else {
      const chosenTemplateId = templateId.value;
      const chosenVersionId = versionId.value;
      const chosenImageId = imageVersionId.value;
      if (!(await loadCatalog(true))) {
        staleMessage.value =
          "无法重新验证模板和镜像目录，未提交创建请求。请重试目录加载后再次提交。";
        return;
      }
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
            currentImage.availability.toLowerCase() !== "available" ||
            currentImage.license_status.toLowerCase() !== "reviewed" ||
            !currentVersion.compatible_image_version_ids?.includes(
              chosenImageId,
            )))
      ) {
        staleMessage.value =
          "The selected template or image changed on the server. Your other values are preserved; choose an available version and retry.";
        return;
      }
      const create = buildResourceCreateRequest(
        props.selection,
        currentDraft.value,
        {
          template: currentTemplate,
          version: currentVersion,
          image: currentImage,
        },
      );
      if (create.kind !== "node") return;
      const value = await api.createNode(props.laboratoryId, create.request);
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
            serverFieldMap[key] || key,
            String(message),
          ]),
        );
      if (Object.keys(fieldErrors.value).length) {
        busy.value = false;
        await focusFirstError();
      }
    } else error.value = value instanceof Error ? value.message : String(value);
  } finally {
    busy.value = false;
    submitLocked = false;
  }
}
</script>
<template>
  <Sheet
    ref="sheetRef"
    v-model="open"
    size="min(92vw, 580px)"
    :prevent-close="dirty"
    :title="selection ? `Add ${selection.name}` : 'Add resource'"
    description="资源及确认位置会共享给所有客户端；画布视口、手工链路路径和当前抽屉草稿仅保存在当前浏览器。"
  >
    <TopologyResourceCatalog
      v-if="!selection"
      @choose="emit('selectionChanged', $event)"
    />
    <form
      v-else
      id="topology-resource-create-form"
      class="grid gap-3"
      @submit.prevent="submit"
    >
      <fieldset :disabled="busy" class="contents">
        <div
          class="flex items-center justify-between gap-3 rounded-md border border-border bg-muted/20 p-2"
        >
          <div class="min-w-0">
            <p class="truncate text-xs font-semibold">{{ selection.name }}</p>
            <p class="truncate text-[11px] text-muted-foreground">
              {{ selection.description || selection.kind }}
            </p>
          </div>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            @click="changeResource"
          >
            更换资源
          </Button>
        </div>
        <FormField data-field="name" label="Name" :error="fieldErrors.name">
          <Input
            v-model="name"
            data-testid="create-resource-name"
            required
            maxlength="120"
          /> </FormField
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
          <Select
            :key="templateSelectKey"
            :model-value="templateId"
            :disabled="catalogLoading"
            @update:model-value="requestTemplateChange"
          >
            <option value="">Select a template</option>
            <option
              v-for="item in compatibleTemplates"
              :key="item.id"
              :value="item.id"
            >
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
              v-for="image in templateCompatibleImages"
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
          </Select>
        </FormField>
        <details
          v-if="!selection?.networkObjectKind"
          class="rounded-md border border-border bg-muted/10 p-3"
          :open="expandedSections.resources"
          @toggle="
            expandedSections.resources = (
              $event.target as HTMLDetailsElement
            ).open
          "
        >
          <summary class="cursor-pointer text-xs font-semibold">
            计算与运行资源
          </summary>
          <div class="mt-3 grid gap-3 sm:grid-cols-2">
            <FormField
              data-field="cpuCount"
              label="vCPU（核心）"
              :error="fieldErrors.cpuCount"
            >
              <Input v-model="cpuCount" type="number" min="1" step="1" />
            </FormField>
            <FormField
              data-field="cpuQuotaMicros"
              label="CPU 配额（微秒/100ms）"
              :error="fieldErrors.cpuQuotaMicros"
              hint="0 表示使用模板/平台默认值。"
            >
              <Input
                v-model="cpuQuotaMicros"
                type="number"
                min="0"
                step="1000"
              />
            </FormField>
            <FormField
              data-field="memoryMiB"
              label="内存（MiB）"
              :error="fieldErrors.memoryMiB"
            >
              <Input v-model="memoryMiB" type="number" min="64" step="64" />
            </FormField>
            <FormField
              data-field="storageGiB"
              label="存储（GiB）"
              :error="fieldErrors.storageGiB"
            >
              <Input v-model="storageGiB" type="number" min="0" step="1" />
            </FormField>
            <FormField
              data-field="interfaceLimit"
              label="接口上限"
              :error="fieldErrors.interfaceLimit"
            >
              <Input
                v-model="interfaceLimit"
                type="number"
                min="1"
                max="64"
                step="1"
              />
            </FormField>
            <FormField
              data-field="processLimit"
              label="进程上限"
              :error="fieldErrors.processLimit"
            >
              <Input v-model="processLimit" type="number" min="1" step="1" />
            </FormField>
            <FormField
              data-field="nicDriver"
              label="网卡驱动"
              :error="fieldErrors.nicDriver"
              hint="仅显示当前模板版本支持的驱动。"
            >
              <Select
                v-model="nicDriver"
                :disabled="!selectedVersion?.supported_nic_drivers.length"
              >
                <option value="">使用模板默认值</option>
                <option
                  v-for="driver in selectedVersion?.supported_nic_drivers || []"
                  :key="driver"
                  :value="driver"
                >
                  {{ driver }}
                </option>
              </Select>
            </FormField>
          </div>
        </details>
        <p
          v-if="catalogError"
          role="alert"
          class="rounded-md border border-destructive/40 p-2 text-xs text-destructive"
        >
          {{ catalogError }}
          <Button
            type="button"
            size="sm"
            variant="secondary"
            class="ml-2"
            @click="loadCatalog(true)"
            >重试</Button
          >
        </p>
        <FormField
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
                  The gateway must be reachable through the static address on
                  this interface.
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
              :data-route-id="route.id"
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
                Injected once through the node-scoped cloud-init seed ISO and
                not saved in browser preferences.
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
          <FormField
            label="Initial username"
            :error="fieldErrors.cloudUsername"
          >
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
      </fieldset>
      <p v-if="error" role="alert" class="text-xs text-destructive">
        {{ error }}
      </p>
      <p v-if="staleMessage" role="alert" class="text-xs text-amber-300">
        {{ staleMessage }}
      </p>
    </form>
    <template v-if="selection" #footer>
      <Button type="button" variant="secondary" @click="requestDrawerClose">
        取消
      </Button>
      <Button
        type="submit"
        form="topology-resource-create-form"
        aria-label="Add to topology"
        :disabled="!canSubmit"
      >
        {{ busy || catalogLoading ? "检查中…" : "添加到拓扑" }}
      </Button>
    </template>
  </Sheet>
</template>
