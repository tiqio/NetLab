<script setup lang="ts">
import { computed, ref, watch } from "vue";
import {
  AlertTriangle,
  CheckCircle2,
  Download,
  FileArchive,
  ShieldCheck,
  Upload,
} from "lucide-vue-next";
import {
  api,
  type Artifact,
  type ImageVersion,
  type Laboratory,
  type OperationTask,
} from "@/api";
import { Button, Dialog, Textarea } from "@/components/ui";
import StatusBadge from "@/components/common/StatusBadge.vue";
import StructuredProblem from "@/components/common/StructuredProblem.vue";

type TransferMode = "export" | "import";
type ExportBundle = {
  schema_version?: number;
  nodes?: Array<{ image_digest?: string }>;
  redaction?: Record<string, boolean>;
};

const props = defineProps<{
  modelValue: boolean;
  mode: TransferMode;
  laboratory?: Laboratory;
}>();
const emit = defineEmits<{
  "update:modelValue": [boolean];
  changed: [];
  status: [string];
}>();

const transferText = ref("");
const task = ref<OperationTask>();
const artifact = ref<Artifact>();
const images = ref<ImageVersion[]>([]);
const error = ref("");
const parsedBundle = ref<ExportBundle>();
let pollTimer: ReturnType<typeof setTimeout> | undefined;

const redaction = computed(() => parsedBundle.value?.redaction || {});
const referencedDigests = computed(() =>
  Array.from(
    new Set(
      (parsedBundle.value?.nodes || [])
        .map((node) => node.image_digest)
        .filter((value): value is string => Boolean(value)),
    ),
  ),
);
const missingImages = computed(() => {
  const available = new Set(images.value.map((image) => image.digest));
  return referencedDigests.value.filter((digest) => !available.has(digest));
});
const artifactURL = computed(() =>
  artifact.value ? api.downloadArtifact(artifact.value.id) : "",
);

function close() {
  emit("update:modelValue", false);
}

function reset() {
  task.value = undefined;
  artifact.value = undefined;
  error.value = "";
  parsedBundle.value = undefined;
  if (pollTimer) clearTimeout(pollTimer);
}

function parseBundle() {
  error.value = "";
  try {
    parsedBundle.value = JSON.parse(transferText.value) as ExportBundle;
  } catch {
    parsedBundle.value = undefined;
    error.value = "Import bundle must be valid JSON.";
  }
}

function readArtifact(value: OperationTask) {
  const result = value.result || {};
  const candidate = result.artifact;
  if (candidate && typeof candidate === "object")
    artifact.value = candidate as Artifact;
}

async function pollTask(id: string) {
  try {
    const value = await api.getTask(id);
    task.value = value;
    readArtifact(value);
    emit(
      "status",
      `${props.mode === "export" ? "Export" : "Import"}: ${value.state}`,
    );
    if (["queued", "running", "cancelling"].includes(value.state)) {
      pollTimer = setTimeout(() => void pollTask(id), 500);
    } else if (value.state === "succeeded") {
      emit("changed");
    }
  } catch (value) {
    error.value = value instanceof Error ? value.message : String(value);
  }
}

async function startExport() {
  if (!props.laboratory) return;
  reset();
  try {
    const value = await api.exportLab(props.laboratory.id);
    task.value = value.task;
    emit("status", `Export queued: ${value.task.id}`);
    await pollTask(value.task.id);
  } catch (value) {
    error.value = value instanceof Error ? value.message : String(value);
  }
}

async function startImport() {
  parseBundle();
  if (!parsedBundle.value || error.value) return;
  if (missingImages.value.length) return;
  try {
    const value = await api.importLab(parsedBundle.value);
    task.value = value.task;
    emit("status", `Import queued: ${value.task.id}`);
    await pollTask(value.task.id);
  } catch (value) {
    error.value = value instanceof Error ? value.message : String(value);
  }
}

watch(
  () => props.modelValue,
  async (open) => {
    if (!open) {
      if (pollTimer) clearTimeout(pollTimer);
      return;
    }
    reset();
    if (props.mode === "import") {
      try {
        images.value = await api.listImages();
      } catch (value) {
        error.value = value instanceof Error ? value.message : String(value);
      }
    }
  },
);
</script>

<template>
  <Dialog
    :model-value="modelValue"
    :title="mode === 'export' ? '导出实验室' : '导入实验室'"
    :description="
      mode === 'export'
        ? '为此实验室创建持久化且已脱敏的元数据包。'
        : '校验并导入已脱敏的 NetLab 元数据包。'
    "
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="grid gap-3">
      <section
        class="rounded-md border border-border bg-muted/30 p-3 text-xs text-muted-foreground"
      >
        <ShieldCheck :size="15" class="mr-1 inline text-green-400" />导出内容不包含镜像数据、凭据、引导密钥和抓包文件。
      </section>

      <template v-if="mode === 'import'">
        <Textarea
          v-model="transferText"
          aria-label="实验室导出 JSON"
          rows="12"
          class="w-full rounded border border-input bg-background p-2 font-mono text-xs"
          @input="parseBundle"
        />
        <section
          v-if="parsedBundle"
          class="rounded-md border border-border p-3"
        >
          <h3 class="text-sm font-semibold">数据包元信息</h3>
          <p class="mt-1 text-xs text-muted-foreground">
            Schema version {{ parsedBundle.schema_version || "unknown" }} ·
            {{ referencedDigests.length }} referenced image digest(s)
          </p>
          <dl class="mt-2 grid grid-cols-2 gap-1 text-xs">
            <template v-for="(excluded, key) in redaction" :key="key">
              <dt>{{ String(key).replaceAll("_", " ") }}</dt>
              <dd :class="excluded ? 'text-green-400' : 'text-amber-300'">
                {{ excluded ? "excluded" : "not declared" }}
              </dd>
            </template>
          </dl>
        </section>
        <section
          v-if="missingImages.length"
          role="alert"
          class="rounded-md border border-amber-400/50 bg-amber-400/10 p-3 text-xs"
        >
          <h3 class="font-semibold text-amber-300">
            <AlertTriangle :size="14" class="mr-1 inline" />Missing images
          </h3>
          <p class="mt-1 text-muted-foreground">
            Import is blocked until these server-authoritative image digests are
            available:
          </p>
          <code
            v-for="digest in missingImages"
            :key="digest"
            class="mt-1 block"
          >
            {{ digest }}
          </code>
        </section>
      </template>

      <section v-if="task" class="rounded-md border border-border p-3 text-xs">
        <div class="flex items-center justify-between gap-2">
          <span class="font-semibold">Durable task {{ task.id }}</span>
          <StatusBadge :state="task.state" />
        </div>
        <progress
          class="mt-2 h-1.5 w-full"
          :value="task.progress_current"
          :max="task.progress_total || 1"
        />
        <StructuredProblem
          v-if="task.error"
          class="mt-2"
          :problem="task.error"
        />
      </section>

      <section
        v-if="artifact"
        class="rounded-md border border-green-500/40 bg-green-500/10 p-3 text-xs"
      >
        <h3 class="font-semibold text-green-300">
          <CheckCircle2 :size="14" class="mr-1 inline" />Export artifact ready
        </h3>
        <p class="mt-1 text-muted-foreground">
          {{ artifact.kind }} · {{ artifact.media_type }} ·
          {{ artifact.size_bytes }} bytes
        </p>
        <p class="mt-1 break-all font-mono text-[10px]">
          {{ artifact.sha256 }}
        </p>
        <a
          :href="artifactURL"
          download
          class="mt-2 inline-flex h-7 items-center gap-2 rounded-md bg-primary px-2.5 text-xs font-medium text-primary-foreground"
        >
          <FileArchive :size="14" />Download artifact
        </a>
      </section>

      <p v-if="error" role="alert" class="text-xs text-destructive">
        {{ error }}
      </p>
    </div>
    <template #footer>
      <Button variant="secondary" @click="close">Close</Button>
      <Button
        v-if="mode === 'export'"
        :disabled="
          !laboratory ||
          Boolean(task && ['queued', 'running'].includes(task.state))
        "
        @click="startExport"
      >
        <Download :size="14" />Create export
      </Button>
      <Button
        v-else
        :disabled="
          !parsedBundle ||
          Boolean(missingImages.length) ||
          Boolean(task && ['queued', 'running'].includes(task.state))
        "
        @click="startImport"
      >
        <Upload :size="14" />Import
      </Button>
    </template>
  </Dialog>
</template>
