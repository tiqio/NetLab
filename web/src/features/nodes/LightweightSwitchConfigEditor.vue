<script setup lang="ts">
import { nextTick, ref, watch } from "vue";
import { Button, FormField, Input } from "@/components/ui";
import {
  defaultLightweightSwitchConfig,
  splitList,
  type L2PortDraft,
  type L3InterfaceDraft,
  type L3RouteDraft,
  type LightweightSwitchKind,
} from "./lightweightSwitchConfig";

const props = defineProps<{
  kind: LightweightSwitchKind;
  modelValue: Record<string, unknown>;
}>();
const emit = defineEmits<{
  "update:modelValue": [value: Record<string, unknown>];
}>();
const vlanFiltering = ref(true);
const l2Ports = ref<L2PortDraft[]>([]);
const l3Interfaces = ref<L3InterfaceDraft[]>([]);
const l3Routes = ref<L3RouteDraft[]>([]);
const forwardIPv4 = ref(true);
const forwardIPv6 = ref(true);
let syncing = false;

function load(value: Record<string, unknown>) {
  syncing = true;
  const fallback = defaultLightweightSwitchConfig(props.kind);
  const source = Object.keys(value || {}).length ? value : fallback;
  if (props.kind === "switch_l2") {
    vlanFiltering.value = source.vlan_filtering !== false;
    l2Ports.value = (Array.isArray(source.ports) ? source.ports : []).map(
      (raw) => {
        const port = raw as { name?: string; pvid?: number; tagged?: number[] };
        return {
          name: String(port.name || ""),
          pvid: Number(port.pvid || 0),
          tagged: Array.isArray(port.tagged) ? port.tagged.join(",") : "",
        };
      },
    );
  } else {
    l3Interfaces.value = (
      Array.isArray(source.interfaces) ? source.interfaces : []
    ).map((raw) => {
      const iface = raw as { name?: string; addresses?: string[] };
      return {
        name: String(iface.name || ""),
        addresses: Array.isArray(iface.addresses)
          ? iface.addresses.join(",")
          : "",
      };
    });
    l3Routes.value = (Array.isArray(source.routes) ? source.routes : []).map(
      (raw) => {
        const route = raw as {
          destination?: string;
          gateway?: string;
          metric?: number;
        };
        return {
          destination: String(route.destination || ""),
          gateway: String(route.gateway || ""),
          metric: Number(route.metric || 0),
        };
      },
    );
    forwardIPv4.value = source.forward_ipv4 !== false;
    forwardIPv6.value = source.forward_ipv6 !== false;
  }
  syncing = false;
}

function publish() {
  if (syncing) return;
  if (props.kind === "switch_l2") {
    emit("update:modelValue", {
      vlan_filtering: vlanFiltering.value,
      ports: l2Ports.value.map((port) => ({
        name: port.name.trim(),
        pvid: Number(port.pvid || 0),
        tagged: splitList(port.tagged).map(Number),
      })),
    });
    return;
  }
  emit("update:modelValue", {
    interfaces: l3Interfaces.value.map((iface) => ({
      name: iface.name.trim(),
      addresses: splitList(iface.addresses),
    })),
    routes: l3Routes.value.map((route) => ({
      destination: route.destination.trim(),
      gateway: route.gateway.trim(),
      metric: Number(route.metric || 0),
    })),
    forward_ipv4: forwardIPv4.value,
    forward_ipv6: forwardIPv6.value,
  });
}

function publishAfterUpdate() {
  void nextTick(publish);
}

function addL2Port() {
  l2Ports.value.push({
    name: `eth${l2Ports.value.length}`,
    pvid: 1,
    tagged: "",
  });
  publish();
}
function addL3Interface() {
  l3Interfaces.value.push({
    name: `eth${l3Interfaces.value.length}`,
    addresses: "",
  });
  publish();
}
function addRoute() {
  l3Routes.value.push({ destination: "0.0.0.0/0", gateway: "", metric: 0 });
  publish();
}

watch(
  () => [props.kind, props.modelValue] as const,
  ([, value]) => load(value),
  { immediate: true, deep: true },
);
</script>

<template>
  <div class="grid gap-3" data-testid="lightweight-switch-config">
    <template v-if="kind === 'switch_l2'">
      <label class="flex items-center gap-2 text-xs">
        <input v-model="vlanFiltering" type="checkbox" @change="publish" />
        启用 VLAN Filtering
      </label>
      <div
        v-for="(port, index) in l2Ports"
        :key="index"
        class="grid gap-2 rounded-md border border-border p-2"
      >
        <div class="grid grid-cols-[1fr_90px] gap-2">
          <FormField label="端口名称">
            <Input
              v-model="port.name"
              placeholder="eth0"
              @update:model-value="publishAfterUpdate"
            />
          </FormField>
          <FormField label="PVID">
            <Input
              v-model="port.pvid"
              type="number"
              min="0"
              max="4094"
              @update:model-value="publishAfterUpdate"
            />
          </FormField>
        </div>
        <FormField label="Tagged VLAN" hint="逗号分隔，例如 10,20,30">
          <Input
            v-model="port.tagged"
            placeholder="10,20"
            @update:model-value="publishAfterUpdate"
          />
        </FormField>
        <Button
          v-if="l2Ports.length > 1"
          type="button"
          size="sm"
          variant="destructive"
          @click="
            l2Ports.splice(index, 1);
            publish();
          "
        >
          删除端口
        </Button>
      </div>
      <Button type="button" size="sm" variant="outline" @click="addL2Port">
        添加二层端口
      </Button>
    </template>
    <template v-else>
      <div
        v-for="(iface, index) in l3Interfaces"
        :key="index"
        class="grid gap-2 rounded-md border border-border p-2"
      >
        <FormField label="接口名称">
          <Input
            v-model="iface.name"
            placeholder="eth0"
            @update:model-value="publishAfterUpdate"
          />
        </FormField>
        <FormField label="IPv4 / IPv6 地址" hint="逗号分隔并使用 CIDR">
          <Input
            v-model="iface.addresses"
            placeholder="192.0.2.1/24,2001:db8::1/64"
            @update:model-value="publishAfterUpdate"
          />
        </FormField>
        <Button
          v-if="l3Interfaces.length > 1"
          type="button"
          size="sm"
          variant="destructive"
          @click="
            l3Interfaces.splice(index, 1);
            publish();
          "
        >
          删除接口
        </Button>
      </div>
      <Button type="button" size="sm" variant="outline" @click="addL3Interface">
        添加三层接口
      </Button>
      <div
        v-for="(route, index) in l3Routes"
        :key="`route-${index}`"
        class="grid gap-2 rounded-md border border-border p-2"
      >
        <FormField label="路由目标">
          <Input
            v-model="route.destination"
            placeholder="0.0.0.0/0"
            @update:model-value="publishAfterUpdate"
          />
        </FormField>
        <div class="grid grid-cols-[1fr_90px] gap-2">
          <FormField label="下一跳">
            <Input
              v-model="route.gateway"
              placeholder="192.0.2.254"
              @update:model-value="publishAfterUpdate"
            />
          </FormField>
          <FormField label="Metric">
            <Input
              v-model="route.metric"
              type="number"
              min="0"
              @update:model-value="publishAfterUpdate"
            />
          </FormField>
        </div>
        <Button
          type="button"
          size="sm"
          variant="destructive"
          @click="
            l3Routes.splice(index, 1);
            publish();
          "
        >
          删除路由
        </Button>
      </div>
      <Button type="button" size="sm" variant="outline" @click="addRoute">
        添加静态路由
      </Button>
      <label class="flex items-center gap-2 text-xs">
        <input v-model="forwardIPv4" type="checkbox" @change="publish" />
        启用 IPv4 转发
      </label>
      <label class="flex items-center gap-2 text-xs">
        <input v-model="forwardIPv6" type="checkbox" @change="publish" />
        启用 IPv6 转发
      </label>
    </template>
  </div>
</template>
