<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  Box,
  Network,
  Search,
  Server,
  Shield,
  TerminalSquare,
} from "lucide-vue-next";
import { api, type DeviceTemplate, type TemplateVersion } from "@/api";
import { Button, Input } from "@/components/ui";

export interface PaletteSelection {
  kind: "qemu" | "docker" | "pc" | "switch_l2" | "switch_l3";
  name: string;
  description?: string;
  template?: DeviceTemplate;
  version?: TemplateVersion;
  networkObjectKind?:
    "pc" | "switch_l2" | "switch_l3" | "bridge" | "nat_bridge";
}
defineOptions({ name: "TopologyResourceCatalog" });
const emit = defineEmits<{ choose: [PaletteSelection] }>();
const templates = ref<DeviceTemplate[]>([]);
const query = ref("");
const loading = ref(false);
const loadError = ref("");
const lightweight: PaletteSelection[] = [
  {
    kind: "pc",
    name: "PC",
    description: "Linux netns 主机 · IPv4 / IPv6",
    networkObjectKind: "pc",
  },
  {
    kind: "switch_l2",
    name: "轻量级二层交换机",
    description: "Linux 网桥 + VLAN 过滤 · 无需 KVM",
    networkObjectKind: "switch_l2",
  },
  {
    kind: "switch_l3",
    name: "轻量级三层交换机",
    description: "Linux netns 路由 · 无需 KVM",
    networkObjectKind: "switch_l3",
  },
  {
    kind: "switch_l2",
    name: "网桥",
    description: "非托管 Linux 网桥",
    networkObjectKind: "bridge",
  },
  {
    kind: "switch_l3",
    name: "NAT 网桥",
    description: "通过 DHCP 和 NAT 访问互联网",
    networkObjectKind: "nat_bridge",
  },
];
const networkDevices = computed(() =>
  [
    {
      key: "ruijie-switch",
      name: "锐捷二层交换机",
      description: "QEMU 设备 · 锐捷交换机 V1.06 · KVM",
    },
    {
      key: "ruijie-router",
      name: "锐捷三层路由器",
      description: "QEMU 设备 · 锐捷路由器 V1.06 · KVM",
    },
  ]
    .map((shortcut) => ({
      ...shortcut,
      template: templates.value.find(
        (item) => item.template_key === shortcut.key,
      ),
    }))
    .filter(
      (item) =>
        item.template &&
        `${item.key} ${item.name} ${item.description}`
          .toLowerCase()
          .includes(query.value.toLowerCase()),
    ),
);
const filtered = computed(() =>
  templates.value.filter(
    (item) =>
      !["ruijie-router", "ruijie-switch"].includes(item.template_key) &&
      `${item.display_name} ${item.template_key} ${item.runtime_kind} ${item.versions.map((version) => String(version.runtime_options?.description || "")).join(" ")}`
        .toLowerCase()
        .includes(query.value.toLowerCase()),
  ),
);
const filteredLightweight = computed(() =>
  lightweight.filter((item) =>
    `${item.name} ${item.networkObjectKind || ""} ${item.description || ""}`
      .toLowerCase()
      .includes(query.value.toLowerCase()),
  ),
);
async function loadTemplates() {
  loading.value = true;
  loadError.value = "";
  try {
    templates.value = (await api.listTemplates()) || [];
  } catch (value) {
    templates.value = [];
    loadError.value = `无法加载设备模板：${value instanceof Error ? value.message : String(value)}`;
  } finally {
    loading.value = false;
  }
}
onMounted(loadTemplates);
function iconFor(name: string) {
  const lower = name.toLowerCase();
  if (lower.includes("forti")) return Shield;
  if (lower.includes("vyos") || lower.includes("fancy")) return Network;
  if (lower.includes("ubuntu")) return TerminalSquare;
  return Server;
}
function enabledVersion(template: DeviceTemplate) {
  return template.versions.find((version) => version.enabled);
}
</script>
<template>
  <div class="flex h-full min-h-0 flex-col">
    <header class="border-b border-border p-3">
      <h2
        class="mb-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground"
      >
        添加设备
      </h2>
      <div class="relative">
        <Search
          :size="14"
          class="absolute left-2 top-2 text-muted-foreground"
        /><Input
          v-model="query"
          class="pl-7"
          placeholder="搜索模板"
          aria-label="搜索设备模板"
        />
      </div>
    </header>
    <div class="flex-1 overflow-auto p-2 netlab-scrollbar">
      <p v-if="loading" role="status" class="p-2 text-xs text-muted-foreground">
        正在加载模板…
      </p>
      <div
        v-if="loadError"
        role="alert"
        class="m-2 rounded-md border border-destructive/40 p-3 text-xs text-destructive"
      >
        <p>{{ loadError }}</p>
        <Button
          class="mt-2"
          size="sm"
          variant="secondary"
          @click="loadTemplates"
        >
          重试加载
        </Button>
      </div>
      <section>
        <h3
          class="px-2 py-2 text-[10px] font-semibold uppercase tracking-widest text-muted-foreground"
        >
          锐捷设备
        </h3>
        <Button
          v-for="item in networkDevices"
          :key="item.key"
          variant="ghost"
          class="palette-item h-auto justify-start"
          @click="
            emit('choose', {
              kind: 'qemu',
              name: item.name,
              template: item.template,
              version: item.template?.versions.find(
                (version) => version.enabled,
              ),
            })
          "
        >
          <Network :size="18" class="text-emerald-300" />
          <span class="min-w-0 text-left">
            <strong class="block truncate text-xs">{{ item.name }}</strong>
            <small class="text-[10px] text-muted-foreground">{{
              item.description
            }}</small>
          </span>
        </Button>
      </section>
      <section>
        <h3
          class="px-2 py-2 text-[10px] font-semibold uppercase tracking-widest text-muted-foreground"
        >
          虚拟设备
        </h3>
        <Button
          v-for="template in filtered"
          :key="template.id"
          variant="ghost"
          class="palette-item h-auto justify-start"
          :disabled="!enabledVersion(template)"
          @click="
            emit('choose', {
              kind: template.runtime_kind,
              name: template.display_name,
              template,
              version: enabledVersion(template),
            })
          "
        >
          <component
            :is="iconFor(template.display_name)"
            :size="18"
            class="text-primary"
          /><span class="min-w-0 text-left"
            ><strong class="block truncate text-xs">{{
              template.display_name
            }}</strong
            ><small class="text-[10px] text-muted-foreground"
              >{{ template.runtime_kind.toUpperCase() }} ·
              {{ template.versions.length }} 个版本 ·
              {{ enabledVersion(template) ? "可用" : "没有已启用版本" }}</small
            ></span
          >
        </Button>
      </section>
      <section>
        <h3
          class="px-2 py-2 text-[10px] font-semibold uppercase tracking-widest text-muted-foreground"
        >
          轻量网络对象
        </h3>
        <Button
          v-for="item in filteredLightweight"
          :key="item.name"
          variant="ghost"
          class="palette-item h-auto justify-start"
          @click="emit('choose', item)"
        >
          <Box :size="18" class="text-sky-300" />
          <span class="min-w-0 text-left">
            <strong class="block truncate text-xs">{{ item.name }}</strong>
            <small class="text-[10px] text-muted-foreground">{{
              item.description
            }}</small>
          </span>
        </Button>
      </section>
    </div>
  </div>
</template>
<style scoped>
.palette-item {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 0.65rem;
  border-radius: 0.35rem;
  padding: 0.55rem 0.5rem;
}
.palette-item:hover {
  background: var(--accent);
}
</style>
