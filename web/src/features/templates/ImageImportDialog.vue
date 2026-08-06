<script setup lang="ts">
import { ref } from "vue";
import { api, type ImageVersion } from "@/api";
import { Button, Dialog, FormField, Input, Select } from "@/components/ui";
const open = defineModel<boolean>({ required: true });
const emit = defineEmits<{ imported: [ImageVersion] }>();
const name = ref("");
const version = ref("");
const runtime = ref<"qemu" | "docker">("qemu");
const source = ref("");
const license = ref("");
const consoleUsername = ref("");
const consolePassword = ref("");
const status = ref("");
async function submit() {
  try {
    const value = await api.importImage({
      name: name.value,
      version: version.value,
      runtime_kind: runtime.value,
      source_reference: source.value,
      license_notes: license.value,
      console_username: consoleUsername.value || undefined,
      console_password: consoleUsername.value
        ? consolePassword.value
        : undefined,
    });
    status.value = `已导入 ${value.name}:${value.version}`;
    emit("imported", value);
    open.value = false;
  } catch (error) {
    status.value = error instanceof Error ? error.message : String(error);
  }
}
</script>
<template>
  <Dialog
    v-model="open"
    title="导入镜像引用"
    description="导入服务器本地路径或镜像仓库引用。不支持浏览器上传或嵌入专有镜像数据。"
  >
    <form class="grid gap-3" @submit.prevent="submit">
      <div class="grid grid-cols-2 gap-2">
        <FormField label="Name"> <Input v-model="name" required /> </FormField
        ><FormField label="Version">
          <Input v-model="version" required />
        </FormField>
      </div>
      <FormField label="Runtime">
        <Select v-model="runtime">
          <option value="qemu">QEMU</option>
          <option value="docker">Docker</option>
        </Select> </FormField
      ><FormField label="来源引用" hint="引用中不得包含凭据。">
        <Input v-model="source" required /> </FormField
      ><FormField label="许可说明">
        <Input v-model="license" required />
      </FormField>
      <div class="grid grid-cols-2 gap-2">
        <FormField label="终端用户名" hint="可选，用于启用串口自动登录。">
          <Input v-model="consoleUsername" autocomplete="off" />
        </FormField>
        <FormField label="终端密码" hint="仅存储在 NetLab 宿主机上。">
          <Input
            v-model="consolePassword"
            type="password"
            autocomplete="new-password"
          />
        </FormField>
      </div>
      <p role="status" class="text-xs text-muted-foreground">
        {{ status }}
      </p>
      <Button>导入并校验</Button>
    </form>
  </Dialog>
</template>
