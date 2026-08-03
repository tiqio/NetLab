<script setup lang="ts">
import { ref } from "vue";
import { api, type Node, type NodeInterface } from "@/api";
import { Button, FormField, Select } from "@/components/ui";
const props = defineProps<{ node: Node; interfaces: NodeInterface[] }>();
const emit = defineEmits<{ changed: [] }>();
const driver = ref("virtio-net-pci");
const status = ref("");
const busy = ref(false);
async function add() {
  busy.value = true;
  try {
    const value = await api.addInterface(props.node.id, driver.value);
    status.value = `已添加接口 ${"interface" in value ? value.interface.name : value.name}`;
    emit("changed");
  } catch (error) {
    status.value = error instanceof Error ? error.message : String(error);
  } finally {
    busy.value = false;
  }
}
async function remove(item: NodeInterface) {
  busy.value = true;
  try {
    await api.removeInterface(item.id, item.revision);
    status.value = `${item.name} 已进入移除队列`;
    emit("changed");
  } catch (error) {
    status.value = error instanceof Error ? error.message : String(error);
  } finally {
    busy.value = false;
  }
}
</script>
<template>
  <section class="panel-section">
    <div>
      <h3>接口操作</h3>
      <p class="text-xs text-muted-foreground">
        运行中的 QEMU 节点将通过 QMP 热添加或移除网卡。
      </p>
    </div>
    <div class="grid grid-cols-[1fr_auto] gap-2">
      <FormField label="新接口驱动">
        <Select v-model="driver">
          <option value="virtio-net-pci">virtio-net-pci</option>
          <option value="e1000">e1000</option>
          <option value="vmxnet3">vmxnet3</option>
          <option value="rtl8139">rtl8139</option>
        </Select> </FormField
      ><Button
        class="self-end"
        size="sm"
        aria-label="添加接口"
        :disabled="busy"
        @click="add"
      >
        添加接口
      </Button>
    </div>
    <ul class="mt-2 grid gap-1">
      <li
        v-for="item in interfaces"
        :key="item.id"
        class="flex items-center justify-between rounded bg-muted px-2 py-1 text-xs"
      >
        <span
          >{{ item.name }} · {{ item.driver }} ·
          {{ item.operational_state }}</span
        ><Button
          variant="ghost"
          size="sm"
          :disabled="busy || Boolean(item.desired_link_id)"
          @click="remove(item)"
        >
          移除
        </Button>
      </li>
    </ul>
    <p role="status" class="mt-1 text-xs text-muted-foreground">
      {{ status }}
    </p>
  </section>
</template>
