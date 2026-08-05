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
    status.value = `Imported ${value.name}:${value.version}`;
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
    title="Import image reference"
    description="Import a server-local or registry reference. Browser uploads and embedded proprietary bytes are not supported."
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
      ><FormField
        label="Source reference"
        hint="Do not include credentials in the reference."
      >
        <Input v-model="source" required /> </FormField
      ><FormField label="License notes">
        <Input v-model="license" required />
      </FormField>
      <div class="grid grid-cols-2 gap-2">
        <FormField
          label="Console username"
          hint="Optional; enables serial auto-login."
        >
          <Input v-model="consoleUsername" autocomplete="off" />
        </FormField>
        <FormField
          label="Console password"
          hint="Stored only on the NetLab host."
        >
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
      <Button>Import and validate</Button>
    </form>
  </Dialog>
</template>
