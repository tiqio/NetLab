<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { RefreshCw, ShieldCheck } from "lucide-vue-next";
import { api, type DeviceTemplate, type TemplateVersion } from "@/api";
import { Button, FormField, Select } from "@/components/ui";

const emit = defineEmits<{ select: [templateId: string, versionId: string] }>();
const templates = ref<DeviceTemplate[]>([]);
const selectedTemplate = ref("");
const selectedVersion = ref("");
const loading = ref(false);
const error = ref("");
const staleMessage = ref("");

const current = computed(() =>
  templates.value.find((item) => item.id === selectedTemplate.value),
);
const version = computed(() =>
  current.value?.versions.find((item) => item.id === selectedVersion.value),
);
const canChoose = computed(
  () => Boolean(version.value?.enabled) && !loading.value,
);

function reconcileSelection(previousVersion?: TemplateVersion) {
  if (selectedTemplate.value && !current.value) {
    selectedTemplate.value = "";
    selectedVersion.value = "";
    staleMessage.value = "The selected template is no longer available.";
    return;
  }
  if (selectedVersion.value && (!version.value || !version.value.enabled)) {
    selectedVersion.value = "";
    staleMessage.value = previousVersion
      ? `Version ${previousVersion.version} is no longer available.`
      : "The selected version is no longer available.";
  }
}

async function load() {
  const previousVersion = version.value;
  loading.value = true;
  error.value = "";
  try {
    templates.value = await api.listTemplates();
    reconcileSelection(previousVersion);
  } catch (value) {
    error.value = value instanceof Error ? value.message : String(value);
  } finally {
    loading.value = false;
  }
}

async function choose() {
  const templateId = selectedTemplate.value;
  const versionId = selectedVersion.value;
  await load();
  if (
    templateId === selectedTemplate.value &&
    versionId === selectedVersion.value &&
    canChoose.value
  ) {
    emit("select", templateId, versionId);
  }
}

watch(selectedTemplate, () => {
  selectedVersion.value = "";
  staleMessage.value = "";
});
watch(selectedVersion, (value) => {
  if (value) staleMessage.value = "";
});
onMounted(load);
</script>

<template>
  <section class="grid gap-3 rounded-md border border-border bg-card p-3">
    <div class="flex items-center justify-between gap-2">
      <h3 class="text-sm font-semibold">设备模板</h3>
      <Button
        variant="ghost"
        size="icon"
        aria-label="刷新模板"
        :disabled="loading"
        @click="load"
      >
        <RefreshCw :size="15" :class="loading && 'animate-spin'" />
      </Button>
    </div>
    <FormField label="Template">
      <Select v-model="selectedTemplate" :disabled="loading">
        <option value="">Select</option>
        <option v-for="item in templates" :key="item.id" :value="item.id">
          {{ item.display_name }} · {{ item.runtime_kind.toUpperCase() }}
        </option>
      </Select>
    </FormField>
    <FormField label="Version">
      <Select
        v-model="selectedVersion"
        :disabled="loading || !selectedTemplate"
      >
        <option value="">Select</option>
        <option
          v-for="item in current?.versions || []"
          :key="item.id"
          :value="item.id"
          :disabled="!item.enabled"
        >
          {{ item.version }}{{ item.enabled ? "" : " (unavailable)" }}
        </option>
      </Select>
    </FormField>
    <p v-if="error" role="alert" class="text-xs text-destructive">
      {{ error }}
    </p>
    <p v-if="staleMessage" role="alert" class="text-xs text-amber-300">
      {{ staleMessage }} Refresh the catalog and choose another version.
    </p>
    <div v-if="version" class="rounded bg-muted/50 p-2 text-xs">
      <p>{{ version.capabilities.join(", ") || "Basic runtime" }}</p>
      <p class="mt-1 text-muted-foreground">
        NICs: {{ version.supported_nic_drivers.join(", ") || "default" }} ·
        Consoles: {{ version.console_modes.join(", ") || "none" }}
      </p>
    </div>
    <Button :disabled="!canChoose" @click="choose"> 使用模板 </Button>
    <p class="text-[11px] text-muted-foreground">
      <ShieldCheck
        :size="12"
        class="mr-1 inline"
      />提交前会根据服务器实时模板目录重新校验所选内容。
    </p>
  </section>
</template>
