<script setup lang="ts">
import { ref } from "vue";
import {
  nodeOperationsApi,
  type NodeInterface,
} from "../../api/nodeOperations";
import { Button, Select } from "@/components/ui";

const props = defineProps<{
  nodeId: string;
  interfaces: NodeInterface[];
  running: boolean;
}>();
const emit = defineEmits<{ changed: [] }>();
const driver = ref("virtio-net-pci");
const status = ref("");

async function addInterface() {
  status.value = "正在添加接口…";
  try {
    const value = await nodeOperationsApi.addInterface(
      props.nodeId,
      driver.value,
    );
    status.value = value.task
      ? `热添加任务已排队：${value.task.id}`
      : "接口已添加";
    emit("changed");
  } catch (error) {
    status.value = error instanceof Error ? error.message : String(error);
  }
}

async function removeInterface(value: NodeInterface) {
  if (value.desired_link_id) {
    status.value = "移除此接口前请先断开链路。";
    return;
  }
  try {
    await nodeOperationsApi.removeInterface(value);
    status.value = props.running ? "热移除任务已排队" : "接口已移除";
    emit("changed");
  } catch (error) {
    status.value = error instanceof Error ? error.message : String(error);
  }
}
</script>

<template>
  <section>
    <h3>网络接口</h3>
    <label
      >网卡驱动
      <Select v-model="driver">
        <option value="virtio-net-pci">VirtIO</option>
        <option value="e1000">Intel e1000</option>
        <option value="e1000e">Intel e1000e</option>
        <option value="vmxnet3">VMXNET3</option>
      </Select></label
    >
    <Button type="button" @click="addInterface">
      {{ running ? "热添加接口" : "添加接口" }}
    </Button>
    <ul>
      <li v-for="item in interfaces" :key="item.id">
        {{ item.name }} · {{ item.driver }} · {{ item.mac_address }}
        <Button
          variant="ghost"
          size="sm"
          type="button"
          :disabled="Boolean(item.desired_link_id)"
          @click="removeInterface(item)"
        >
          移除
        </Button>
      </li>
    </ul>
    <p role="status">
      {{ status }}
    </p>
  </section>
</template>
