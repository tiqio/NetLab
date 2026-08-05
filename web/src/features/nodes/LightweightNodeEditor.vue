<script setup lang="ts">
import { computed, ref } from "vue";
import { api } from "../../api";
import { Button, Select } from "@/components/ui";

type Kind = "pc" | "switch_l2" | "switch_l3" | "bridge" | "nat_bridge";
const props = defineProps<{ laboratoryId: string }>();
const emit = defineEmits<{ created: [id: string] }>();
const kind = ref<Kind>("pc");
const name = ref("pc1");
const ipv4 = ref("192.0.2.10/24");
const ipv6 = ref("2001:db8::10/64");
const dns = ref("192.0.2.53,2001:db8::53");
const modes = ref(["static", "slaac"]);
const natPrefix = ref("10.10.0.0/24");
const uplink = ref("eth0");
const natIPv6Prefix = ref("2001:db8:10::/64");
const bridgeMTU = ref(1500);
const bridgeSTP = ref(false);
const switchPort = ref("eth0");
const pvid = ref(1);
const taggedVLANs = ref("10,20");
const l3Addresses = ref("192.0.2.1/24,2001:db8::1/64");
const l3Route = ref("0.0.0.0/0");
const l3Gateway = ref("192.0.2.254");
const status = ref("");
const lastCreatedId = ref("");

const config = computed(() => {
  if (kind.value === "pc")
    return {
      hostname: name.value,
      interfaces: [
        {
          name: "eth0",
          modes: modes.value,
          addresses: modes.value.includes("static")
            ? [ipv4.value, ipv6.value]
            : [],
          dns: splitValues(dns.value),
        },
      ],
    };
  if (kind.value === "nat_bridge")
    return {
      ipv4_prefix: natPrefix.value,
      ipv6_prefix: natIPv6Prefix.value,
      uplink: uplink.value,
    };
  if (kind.value === "switch_l2")
    return {
      vlan_filtering: true,
      ports: [
        {
          name: switchPort.value,
          pvid: pvid.value,
          tagged: splitValues(taggedVLANs.value).map(Number),
        },
      ],
    };
  if (kind.value === "switch_l3")
    return {
      interfaces: [
        { name: switchPort.value, addresses: splitValues(l3Addresses.value) },
      ],
      routes: [{ destination: l3Route.value, gateway: l3Gateway.value }],
      forward_ipv4: true,
      forward_ipv6: true,
    };
  return { mtu: bridgeMTU.value, stp: bridgeSTP.value };
});

function splitValues(value: string) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

async function create() {
  const envelope = await api.createNetworkObject(props.laboratoryId, {
    name: name.value,
    kind: kind.value,
    config: config.value,
  });
  const value = envelope.network_object;
  if (!value)
    throw new Error(
      "网络对象任务未返回已接受的资源",
    );
  lastCreatedId.value = value.id;
  status.value = `${value.kind} ${value.name} is ${value.observed_state}`;
  emit("created", value.id);
}

async function diagnostics(id: string) {
  status.value = JSON.stringify(await api.getNetworkObjectDiagnostics(id));
}
</script>

<template>
  <section>
    <h2>轻量网络对象</h2>
    <label
      >Kind
      <Select v-model="kind">
        <option value="pc">PC</option>
        <option value="switch_l2">二层交换机</option>
        <option value="switch_l3">三层交换机</option>
        <option value="bridge">网桥</option>
        <option value="nat_bridge">NAT 网桥</option>
      </Select></label
    >
    <label>名称 <input v-model="name" /></label>
    <fieldset v-if="kind === 'pc'">
      <legend>双栈地址配置</legend>
      <label
        ><input v-model="modes" type="checkbox" value="static" /> 静态地址</label
      ><label
        ><input v-model="modes" type="checkbox" value="dhcpv4" /> DHCPv4</label
      ><label
        ><input v-model="modes" type="checkbox" value="dhcpv6" /> DHCPv6</label
      ><label
        ><input v-model="modes" type="checkbox" value="slaac" /> IPv6
        SLAAC</label
      ><label>IPv4 <input v-model="ipv4" /></label
      ><label>IPv6 <input v-model="ipv6" /></label>
      <label>DNS 服务器 <input v-model="dns" /></label>
    </fieldset>
    <fieldset v-if="kind === 'nat_bridge'">
      <legend>NAT 配置</legend>
      <label>IPv4 前缀 <input v-model="natPrefix" /></label
      ><label>IPv6 前缀 <input v-model="natIPv6Prefix" /></label
      ><label>上联接口 <input v-model="uplink" /></label>
    </fieldset>
    <fieldset v-if="kind === 'bridge'">
      <legend>网桥配置</legend>
      <label>MTU <input v-model.number="bridgeMTU" type="number" /></label>
      <label><input v-model="bridgeSTP" type="checkbox" /> STP</label>
    </fieldset>
    <fieldset v-if="kind === 'switch_l2'">
      <legend>VLAN 端口</legend>
      <label>端口 <input v-model="switchPort" /></label>
      <label>PVID <input v-model.number="pvid" type="number" /></label>
      <label>Tagged VLAN <input v-model="taggedVLANs" /></label>
    </fieldset>
    <fieldset v-if="kind === 'switch_l3'">
      <legend>路由配置</legend>
      <label>接口 <input v-model="switchPort" /></label>
      <label>地址 <input v-model="l3Addresses" /></label>
      <label>路由 <input v-model="l3Route" /></label>
      <label>网关 <input v-model="l3Gateway" /></label>
    </fieldset>
    <Button type="button" @click="create">创建网络对象</Button>
    <Button
      type="button"
      :disabled="!lastCreatedId"
      @click="diagnostics(lastCreatedId)"
    >
      Refresh diagnostics
    </Button>
    <p role="status">
      {{ status }}
    </p>
  </section>
</template>
