<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  Box,
  CheckCircle2,
  Search,
  ShieldCheck,
  XCircle,
} from "lucide-vue-next";
import { api, type DeviceTemplate, type ImageVersion } from "@/api";
import { Input, Select, Table } from "@/components/ui";
import StatusBadge from "@/components/common/StatusBadge.vue";
const templates = ref<DeviceTemplate[]>([]);
const images = ref<ImageVersion[]>([]);
const query = ref("");
const runtime = ref("all");
const loading = ref(false);
const error = ref("");
const filtered = computed(() =>
  templates.value.filter(
    (item) =>
      (runtime.value === "all" || item.runtime_kind === runtime.value) &&
      `${item.display_name} ${item.template_key}`
        .toLowerCase()
        .includes(query.value.toLowerCase()),
  ),
);
const imageFor = (id?: string) => images.value.find((item) => item.id === id);
async function load() {
  loading.value = true;
  try {
    [templates.value, images.value] = await Promise.all([
      api.listTemplates(),
      api.listImages(),
    ]);
  } catch (value) {
    error.value = value instanceof Error ? value.message : String(value);
  } finally {
    loading.value = false;
  }
}
onMounted(load);
defineExpose({ refresh: load });
</script>
<template>
  <section>
    <header class="flex items-center gap-2 border-b border-border p-3">
      <div class="relative max-w-sm flex-1">
        <Search
          :size="14"
          class="absolute left-2 top-2 text-muted-foreground"
        /><Input v-model="query" class="pl-7" placeholder="搜索模板或系列" />
      </div>
      <Select v-model="runtime" class="max-w-36">
        <option value="all">全部运行时</option>
        <option value="qemu">QEMU</option>
        <option value="docker">Docker</option>
      </Select>
    </header>
    <p v-if="loading" role="status" class="p-4 text-sm text-muted-foreground">
      正在加载模板目录…
    </p>
    <p v-if="error" role="alert" class="p-4 text-destructive">
      {{ error }}
    </p>
    <div class="grid gap-3 p-3 lg:grid-cols-2">
      <article
        v-for="template in filtered"
        :key="template.id"
        class="rounded-lg border border-border bg-card p-3"
      >
        <div class="flex items-start gap-2">
          <span
            class="grid h-9 w-9 place-items-center rounded bg-accent text-primary"
            ><Box :size="18"
          /></span>
          <div>
            <h2 class="font-semibold">
              {{ template.display_name }}
            </h2>
            <p class="text-xs text-muted-foreground">
              {{ template.template_key }} ·
              {{ template.runtime_kind.toUpperCase() }}
            </p>
          </div>
        </div>
        <div class="mt-3 overflow-x-auto">
          <Table>
            <thead class="text-muted-foreground">
              <tr>
                <th>Version</th>
                <th>Image</th>
                <th>Capabilities</th>
                <th>Consoles</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="version in template.versions"
                :key="version.id"
                class="border-t border-border"
              >
                <td class="py-2">
                  {{ version.version }}
                </td>
                <td>
                  <template v-if="imageFor(version.image_version_id)">
                    <div>
                      {{ imageFor(version.image_version_id)?.version }}
                    </div>
                    <code class="text-[9px] text-muted-foreground">{{
                      imageFor(version.image_version_id)?.digest
                    }}</code> </template
                  ><span v-else class="text-muted-foreground">none</span>
                </td>
                <td>{{ version.capabilities.join(", ") || "basic" }}</td>
                <td>{{ version.console_modes.join(", ") || "none" }}</td>
                <td>
                  <StatusBadge
                    :state="
                      version.readiness?.status ||
                      (version.enabled ? 'available' : 'disabled')
                    "
                  />
                  <p
                    v-if="version.readiness"
                    class="mt-1 text-[10px] text-muted-foreground"
                  >
                    {{
                      version.readiness.genuine_workload
                        ? "Genuine workload"
                        : "Mechanics or operator asset only"
                    }}
                    <span v-if="version.readiness.exception_id">
                      · exception {{ version.readiness.exception_id }}</span
                    >
                  </p>
                </td>
              </tr>
            </tbody>
          </Table>
        </div>
      </article>
    </div>
    <div
      v-if="!loading && !filtered.length"
      class="grid min-h-40 place-items-center text-sm text-muted-foreground"
    >
      <span><XCircle :size="16" class="mr-1 inline" />没有匹配的模板。</span>
    </div>
    <footer class="border-t border-border p-3 text-xs text-muted-foreground">
      <ShieldCheck :size="13" class="mr-1 inline" />镜像来源、摘要、许可说明、可用性和校验结果均以服务器为准。
      <CheckCircle2 :size="13" class="ml-2 mr-1 inline text-green-400" />浏览器不会存储镜像内容或凭据。
    </footer>
  </section>
</template>
