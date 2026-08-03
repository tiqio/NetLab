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
const emit = defineEmits<{ choose: [PaletteSelection] }>();
const templates = ref<DeviceTemplate[]>([]);
const query = ref("");
const loading = ref(false);
const lightweight: PaletteSelection[] = [
  {
    kind: "pc",
    name: "PC",
    description: "Linux netns host · IPv4 / IPv6",
    networkObjectKind: "pc",
  },
  {
    kind: "switch_l2",
    name: "Lightweight L2 Switch",
    description: "Linux bridge + VLAN filtering · No KVM",
    networkObjectKind: "switch_l2",
  },
  {
    kind: "switch_l3",
    name: "Lightweight L3 Switch",
    description: "Linux netns routing · No KVM",
    networkObjectKind: "switch_l3",
  },
  {
    kind: "switch_l2",
    name: "Bridge",
    description: "Unmanaged Linux bridge",
    networkObjectKind: "bridge",
  },
  {
    kind: "switch_l3",
    name: "NAT bridge",
    description: "Internet access with DHCP and NAT",
    networkObjectKind: "nat_bridge",
  },
];
const networkDevices = computed(() =>
  [
    {
      key: "ruijie-switch",
      name: "Ruijie L2 Switch",
      description: "QEMU appliance · Ruijie Switch V1.06 · KVM",
    },
    {
      key: "ruijie-router",
      name: "Ruijie L3 Router",
      description: "QEMU appliance · Ruijie Router V1.06 · KVM",
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
        `${item.name} ${item.description}`
          .toLowerCase()
          .includes(query.value.toLowerCase()),
    ),
);
const filtered = computed(() =>
  templates.value.filter(
    (item) =>
      !["ruijie-router", "ruijie-switch"].includes(item.template_key) &&
      `${item.display_name} ${item.runtime_kind}`
        .toLowerCase()
        .includes(query.value.toLowerCase()),
  ),
);
onMounted(async () => {
  loading.value = true;
  try {
    templates.value = await api.listTemplates();
  } finally {
    loading.value = false;
  }
});
function iconFor(name: string) {
  const lower = name.toLowerCase();
  if (lower.includes("forti")) return Shield;
  if (lower.includes("vyos") || lower.includes("fancy")) return Network;
  if (lower.includes("ubuntu")) return TerminalSquare;
  return Server;
}
</script>
<template>
  <div class="flex h-full flex-col">
    <header class="border-b border-border p-3">
      <h2
        class="mb-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground"
      >
        Add device
      </h2>
      <div class="relative">
        <Search
          :size="14"
          class="absolute left-2 top-2 text-muted-foreground"
        /><Input
          v-model="query"
          class="pl-7"
          placeholder="Search templates"
          aria-label="Search device templates"
        />
      </div>
    </header>
    <div class="flex-1 overflow-auto p-2 netlab-scrollbar">
      <p v-if="loading" role="status" class="p-2 text-xs text-muted-foreground">
        Loading templates…
      </p>
      <section>
        <h3
          class="px-2 py-2 text-[10px] font-semibold uppercase tracking-widest text-muted-foreground"
        >
          Ruijie appliances
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
          Virtual devices
        </h3>
        <Button
          v-for="template in filtered"
          :key="template.id"
          variant="ghost"
          class="palette-item h-auto justify-start"
          @click="
            emit('choose', {
              kind: template.runtime_kind,
              name: template.display_name,
              template,
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
              {{ template.versions.length }} versions</small
            ></span
          >
        </Button>
      </section>
      <section>
        <h3
          class="px-2 py-2 text-[10px] font-semibold uppercase tracking-widest text-muted-foreground"
        >
          Lightweight
        </h3>
        <Button
          v-for="item in lightweight"
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
