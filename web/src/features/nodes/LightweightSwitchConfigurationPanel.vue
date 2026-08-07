<script setup lang="ts">
import { ref, watch } from "vue";
import { api, ApiError, type NetworkObject } from "@/api";
import { Button, FormField, Input } from "@/components/ui";
import LightweightSwitchConfigEditor from "./LightweightSwitchConfigEditor.vue";
import {
  validateLightweightSwitchConfig,
  type LightweightSwitchKind,
} from "./lightweightSwitchConfig";

const props = defineProps<{ networkObject: NetworkObject }>();
const emit = defineEmits<{ changed: [] }>();
const name = ref("");
const config = ref<Record<string, unknown>>({});
const busy = ref(false);
const error = ref("");
const success = ref("");

watch(
  () => props.networkObject,
  (value) => {
    name.value = value.name;
    config.value = JSON.parse(JSON.stringify(value.config || {})) as Record<
      string,
      unknown
    >;
    error.value = "";
    success.value = "";
  },
  { immediate: true, deep: true },
);

async function save() {
  error.value = "";
  success.value = "";
  if (!name.value.trim()) {
    error.value = "名称不能为空。";
    return;
  }
  const validation = validateLightweightSwitchConfig(
    props.networkObject.kind as LightweightSwitchKind,
    config.value,
  );
  if (validation.length) {
    error.value = validation.join(" ");
    return;
  }
  busy.value = true;
  try {
    await api.updateNetworkObject(props.networkObject, {
      name: name.value.trim(),
      config: config.value,
    });
    success.value = "配置更新任务已提交，运行时将实时重新应用。";
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
  <section class="panel-section">
    <h3>轻量交换机配置</h3>
    <div class="grid gap-3">
      <FormField label="名称">
        <Input v-model="name" maxlength="128" />
      </FormField>
      <LightweightSwitchConfigEditor
        v-model="config"
        :kind="networkObject.kind as LightweightSwitchKind"
      />
      <Button size="sm" :disabled="busy" @click="save">
        {{ busy ? "正在提交…" : "应用配置" }}
      </Button>
      <p class="text-[11px] text-muted-foreground">
        VLAN、地址、路由和转发配置会保存到数据库，并重新应用到该 netns。
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
