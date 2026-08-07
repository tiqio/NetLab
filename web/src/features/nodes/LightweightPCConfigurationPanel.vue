<script setup lang="ts">
import { ref, watch } from "vue";
import { Plus, Trash2 } from "lucide-vue-next";
import { api, ApiError, type NetworkObject } from "@/api";
import { Button, FormField, Input } from "@/components/ui";

type AddressMode = "static" | "dhcpv4" | "dhcpv6" | "slaac";
interface RouteRow {
  id: string;
  destination: string;
  gateway: string;
  metric: number;
}
interface InterfaceRow {
  id: string;
  name: string;
  modes: AddressMode[];
  addresses: string;
  dns: string;
  routes: RouteRow[];
}

const props = defineProps<{ networkObject: NetworkObject }>();
const emit = defineEmits<{ changed: [] }>();
const name = ref("");
const hostname = ref("");
const interfaces = ref<InterfaceRow[]>([]);
const busy = ref(false);
const error = ref("");
const success = ref("");
let sequence = 0;

function nextId(prefix: string) {
  sequence += 1;
  return `${prefix}-${sequence}`;
}

function values(value: unknown) {
  return Array.isArray(value) ? value.map(String).join(", ") : "";
}

function normalizeInterface(raw: Record<string, unknown>, index: number) {
  const routes = Array.isArray(raw.routes) ? raw.routes : [];
  return {
    id: nextId("interface"),
    name: String(raw.name || `eth${index}`),
    modes: Array.isArray(raw.modes)
      ? (raw.modes.filter((mode) =>
          ["static", "dhcpv4", "dhcpv6", "slaac"].includes(String(mode)),
        ) as AddressMode[])
      : [],
    addresses: values(raw.addresses),
    dns: values(raw.dns),
    routes: routes.map((route) => {
      const value = route as Record<string, unknown>;
      return {
        id: nextId("route"),
        destination: String(value.destination || ""),
        gateway: String(value.gateway || ""),
        metric: Number(value.metric || 0),
      };
    }),
  };
}

watch(
  () => props.networkObject,
  (value) => {
    const config = value.config || {};
    const rows = Array.isArray(config.interfaces) ? config.interfaces : [];
    name.value = value.name;
    hostname.value = String(config.hostname || value.name);
    interfaces.value = rows.length
      ? rows.map((row, index) =>
          normalizeInterface(row as Record<string, unknown>, index),
        )
      : [normalizeInterface({ name: "eth0", modes: ["dhcpv4"] }, 0)];
    error.value = "";
    success.value = "";
  },
  { immediate: true, deep: true },
);

function toggleMode(row: InterfaceRow, mode: AddressMode, enabled: boolean) {
  row.modes = enabled
    ? Array.from(new Set([...row.modes, mode]))
    : row.modes.filter((value) => value !== mode);
}

function splitValues(value: string) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function addInterface() {
  interfaces.value.push(
    normalizeInterface(
      { name: `eth${interfaces.value.length}`, modes: ["dhcpv4"] },
      interfaces.value.length,
    ),
  );
}

function addRoute(row: InterfaceRow) {
  row.routes.push({
    id: nextId("route"),
    destination: "0.0.0.0/0",
    gateway: "",
    metric: 0,
  });
}

function validate() {
  if (!name.value.trim()) return "名称不能为空。";
  if (!interfaces.value.length) return "PC 至少需要一个接口。";
  const names = new Set<string>();
  for (const row of interfaces.value) {
    if (!/^[A-Za-z0-9_.-]{1,15}$/.test(row.name))
      return `接口名 ${row.name || "(空)"} 无效。`;
    if (names.has(row.name)) return `接口名 ${row.name} 重复。`;
    names.add(row.name);
    if (!row.modes.length) return `${row.name} 至少选择一种地址模式。`;
    if (row.modes.includes("static") && !splitValues(row.addresses).length)
      return `${row.name} 启用静态地址后必须填写 CIDR。`;
    if (row.routes.some((route) => !route.destination.trim()))
      return `${row.name} 的路由目标不能为空。`;
  }
  return "";
}

async function save() {
  error.value = validate();
  success.value = "";
  if (error.value) return;
  busy.value = true;
  try {
    await api.updateNetworkObject(props.networkObject, {
      name: name.value.trim(),
      config: {
        hostname: hostname.value.trim() || name.value.trim(),
        interfaces: interfaces.value.map((row) => ({
          name: row.name,
          modes: row.modes,
          addresses: row.modes.includes("static")
            ? splitValues(row.addresses)
            : [],
          dns: splitValues(row.dns),
          routes: row.routes.map((route) => ({
            destination: route.destination.trim(),
            ...(route.gateway.trim() ? { gateway: route.gateway.trim() } : {}),
            ...(route.metric > 0 ? { metric: route.metric } : {}),
          })),
        })),
      },
    });
    success.value = "PC 网络配置已提交，地址、路由和 DHCP 将实时重新应用。";
    emit("changed");
  } catch (value) {
    error.value =
      value instanceof ApiError
        ? value.problem.message
        : value instanceof Error
          ? value.message
          : String(value);
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <section class="panel-section" data-testid="lightweight-pc-configuration">
    <h3>PC 网络配置</h3>
    <div class="grid gap-3">
      <div class="grid grid-cols-2 gap-2">
        <FormField label="名称"
          ><Input v-model="name" maxlength="128"
        /></FormField>
        <FormField label="主机名"
          ><Input v-model="hostname" maxlength="63"
        /></FormField>
      </div>
      <article
        v-for="(row, index) in interfaces"
        :key="row.id"
        class="grid gap-3 rounded-md border border-border bg-background/40 p-3"
        :data-testid="`pc-interface-${index}`"
      >
        <div class="flex items-center justify-between gap-2">
          <strong class="text-xs">接口 {{ index + 1 }}</strong>
          <Button
            variant="ghost"
            size="sm"
            :disabled="interfaces.length === 1"
            @click="interfaces.splice(index, 1)"
          >
            <Trash2 :size="13" /> 删除接口
          </Button>
        </div>
        <FormField label="接口名称" hint="用于连线，例如 eth0、eth1。">
          <Input v-model="row.name" maxlength="15" />
        </FormField>
        <fieldset class="grid grid-cols-2 gap-2 text-xs">
          <legend class="col-span-2 mb-1 text-muted-foreground">
            地址获取方式
          </legend>
          <label
            v-for="mode in [
              ['dhcpv4', 'DHCPv4'],
              ['dhcpv6', 'DHCPv6'],
              ['slaac', 'IPv6 SLAAC'],
              ['static', '静态地址'],
            ] as const"
            :key="mode[0]"
            class="flex items-center gap-2 rounded border border-border/70 px-2 py-1.5"
          >
            <input
              type="checkbox"
              :checked="row.modes.includes(mode[0])"
              @change="
                toggleMode(
                  row,
                  mode[0],
                  ($event.target as HTMLInputElement).checked,
                )
              "
            />
            {{ mode[1] }}
          </label>
        </fieldset>
        <FormField
          v-if="row.modes.includes('static')"
          label="静态地址"
          hint="多个 CIDR 用逗号分隔，例如 192.0.2.10/24, 2001:db8::10/64。"
        >
          <Input v-model="row.addresses" placeholder="192.0.2.10/24" />
        </FormField>
        <FormField label="DNS 服务器" hint="多个地址用逗号分隔。">
          <Input
            v-model="row.dns"
            placeholder="1.1.1.1, 2606:4700:4700::1111"
          />
        </FormField>
        <div class="grid gap-2">
          <div class="flex items-center justify-between">
            <span class="text-xs text-muted-foreground">静态路由</span>
            <Button variant="ghost" size="sm" @click="addRoute(row)">
              <Plus :size="13" /> 添加路由
            </Button>
          </div>
          <div
            v-for="(route, routeIndex) in row.routes"
            :key="route.id"
            class="grid grid-cols-[1fr_1fr_70px_30px] gap-1"
          >
            <Input
              v-model="route.destination"
              aria-label="路由目标"
              placeholder="0.0.0.0/0"
            />
            <Input
              v-model="route.gateway"
              aria-label="下一跳"
              placeholder="192.0.2.1"
            />
            <Input
              v-model.number="route.metric"
              aria-label="路由跃点值"
              type="number"
              min="0"
            />
            <Button
              variant="ghost"
              size="icon"
              :aria-label="`删除路由 ${routeIndex + 1}`"
              @click="row.routes.splice(routeIndex, 1)"
            >
              <Trash2 :size="13" />
            </Button>
          </div>
        </div>
      </article>
      <Button variant="outline" size="sm" @click="addInterface">
        <Plus :size="13" /> 添加网络接口
      </Button>
      <Button size="sm" :disabled="busy" @click="save">
        {{ busy ? "正在提交…" : "应用 PC 配置" }}
      </Button>
      <p class="text-[11px] text-muted-foreground">
        与 EVE-NG 的 VPCS 使用习惯类似：接口负责 DHCP
        或静态地址；高级场景可同时配置 IPv6 SLAAC、DHCPv6、DNS 和静态路由。
      </p>
      <p v-if="success" class="text-xs text-[color:var(--success)]">
        {{ success }}
      </p>
      <p v-if="error" role="alert" class="text-xs text-destructive">
        {{ error }}
      </p>
    </div>
  </section>
</template>
