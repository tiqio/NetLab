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
      "Network object task did not include the accepted resource",
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
    <h2>Lightweight nodes</h2>
    <label
      >Kind
      <Select v-model="kind">
        <option value="pc">PC</option>
        <option value="switch_l2">Layer-2 switch</option>
        <option value="switch_l3">Layer-3 switch</option>
        <option value="bridge">Bridge</option>
        <option value="nat_bridge">NAT bridge</option>
      </Select></label
    >
    <label>Name <input v-model="name" /></label>
    <fieldset v-if="kind === 'pc'">
      <legend>Dual-stack addressing</legend>
      <label
        ><input v-model="modes" type="checkbox" value="static" /> Static</label
      ><label
        ><input v-model="modes" type="checkbox" value="dhcpv4" /> DHCPv4</label
      ><label
        ><input v-model="modes" type="checkbox" value="dhcpv6" /> DHCPv6</label
      ><label
        ><input v-model="modes" type="checkbox" value="slaac" /> IPv6
        SLAAC</label
      ><label>IPv4 <input v-model="ipv4" /></label
      ><label>IPv6 <input v-model="ipv6" /></label>
      <label>DNS servers <input v-model="dns" /></label>
    </fieldset>
    <fieldset v-if="kind === 'nat_bridge'">
      <legend>NAT</legend>
      <label>Prefix <input v-model="natPrefix" /></label
      ><label>IPv6 prefix <input v-model="natIPv6Prefix" /></label
      ><label>Uplink <input v-model="uplink" /></label>
    </fieldset>
    <fieldset v-if="kind === 'bridge'">
      <legend>Bridge</legend>
      <label>MTU <input v-model.number="bridgeMTU" type="number" /></label>
      <label><input v-model="bridgeSTP" type="checkbox" /> STP</label>
    </fieldset>
    <fieldset v-if="kind === 'switch_l2'">
      <legend>VLAN port</legend>
      <label>Port <input v-model="switchPort" /></label>
      <label>PVID <input v-model.number="pvid" type="number" /></label>
      <label>Tagged VLANs <input v-model="taggedVLANs" /></label>
    </fieldset>
    <fieldset v-if="kind === 'switch_l3'">
      <legend>Routing</legend>
      <label>Interface <input v-model="switchPort" /></label>
      <label>Addresses <input v-model="l3Addresses" /></label>
      <label>Route <input v-model="l3Route" /></label>
      <label>Gateway <input v-model="l3Gateway" /></label>
    </fieldset>
    <Button type="button" @click="create">Create network object</Button>
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
