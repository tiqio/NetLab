<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from "vue";
import { RouterLink } from "vue-router";
import {
  Check,
  ChevronDown,
  Copy,
  Download,
  Edit3,
  FlaskConical,
  Menu,
  MoreVertical,
  Plus,
  RefreshCw,
  Search,
  Trash2,
  Upload,
} from "lucide-vue-next";
import { api, ApiError, type Laboratory } from "@/api";
import { Button, Dialog, FormField, Input, Select } from "@/components/ui";
import { ThemeSwitcher } from "@/components/appearance";
import LaboratoryTransferDialog from "./LaboratoryTransferDialog.vue";

const props = withDefaults(
  defineProps<{
    labs: Laboratory[];
    active?: Laboratory;
    eventStatus: string;
    loading?: boolean;
    showHeading?: boolean;
    paletteAvailable?: boolean;
  }>(),
  { showHeading: true, active: undefined, paletteAvailable: true },
);
const emit = defineEmits<{
  select: [string];
  deleteAccepted: [string];
  refresh: [];
  changed: [];
  togglePalette: [];
  openCreate: [];
  openCommands: [];
}>();
const createOpen = ref(false);
const renameOpen = ref(false);
const deleteOpen = ref(false);
const transferOpen = ref(false);
const transferMode = ref<"export" | "import">("import");
const switcherOpen = ref(false);
const switcher = ref<HTMLElement>();
const contextMenu = ref<HTMLElement>();
const contextMenuOpen = ref(false);
const contextMenuX = ref(0);
const contextMenuY = ref(0);
const actionLab = ref<Laboratory>();
const laboratoryQuery = ref("");
const name = ref("");
const renameName = ref("");
const renameRevision = ref(0);
const renameBusy = ref(false);
const renameError = ref("");
const policy = ref<"auto_restore" | "remain_stopped">("auto_restore");
const status = ref("");
const filteredLabs = computed(() => {
  const query = laboratoryQuery.value.trim().toLowerCase();
  if (!query) return props.labs;
  return props.labs.filter((lab) =>
    `${lab.name} ${lab.description || ""}`.toLowerCase().includes(query),
  );
});

function toggleSwitcher() {
  switcherOpen.value = !switcherOpen.value;
  if (switcherOpen.value)
    void nextTick(() => switcher.value?.querySelector("input")?.focus());
}
function closeSwitcher() {
  switcherOpen.value = false;
  contextMenuOpen.value = false;
  laboratoryQuery.value = "";
}
function selectLab(id: string) {
  closeSwitcher();
  if (id !== props.active?.id) emit("select", id);
}
function openCreate() {
  closeSwitcher();
  createOpen.value = true;
}
function handleOutsidePointer(event: PointerEvent) {
  if (
    switcherOpen.value &&
    !switcher.value?.contains(event.target as globalThis.Node) &&
    !contextMenu.value?.contains(event.target as globalThis.Node)
  )
    closeSwitcher();
}
function handleEscape(event: KeyboardEvent) {
  if (contextMenuOpen.value && event.key === "Escape") {
    event.preventDefault();
    contextMenuOpen.value = false;
    return;
  }
  if (switcherOpen.value && event.key === "Escape") {
    event.preventDefault();
    closeSwitcher();
  }
}
onMounted(() => {
  window.addEventListener("pointerdown", handleOutsidePointer);
  window.addEventListener("keydown", handleEscape);
});
onBeforeUnmount(() => {
  window.removeEventListener("pointerdown", handleOutsidePointer);
  window.removeEventListener("keydown", handleEscape);
});

function positionContextMenu(x: number, y: number) {
  const menuWidth = 208;
  const menuHeight = 224;
  const margin = 8;
  contextMenuX.value = Math.max(
    margin,
    Math.min(x, window.innerWidth - menuWidth - margin),
  );
  contextMenuY.value = Math.max(
    margin,
    Math.min(y, window.innerHeight - menuHeight - margin),
  );
  contextMenuOpen.value = true;
}

function openLabContext(lab: Laboratory, event: MouseEvent) {
  event.preventDefault();
  event.stopPropagation();
  actionLab.value = lab;
  positionContextMenu(event.clientX, event.clientY);
}

function openLabActions(lab: Laboratory, event: MouseEvent) {
  event.preventDefault();
  event.stopPropagation();
  actionLab.value = lab;
  const bounds = (event.currentTarget as HTMLElement).getBoundingClientRect();
  positionContextMenu(bounds.right - 208, bounds.bottom + 4);
}

function openGlobalContext(event: MouseEvent) {
  if ((event.target as HTMLElement).closest('[data-laboratory-row="true"]'))
    return;
  event.preventDefault();
  actionLab.value = undefined;
  positionContextMenu(event.clientX, event.clientY);
}

function finishContextAction() {
  contextMenuOpen.value = false;
  switcherOpen.value = false;
  laboratoryQuery.value = "";
}

async function createLab() {
  if (!name.value.trim()) return;
  const lab = await api.createLab({
    name: name.value.trim(),
    description: "",
    recovery_policy: policy.value,
  });
  createOpen.value = false;
  name.value = "";
  emit("select", lab.id);
}
async function duplicateLab() {
  if (!actionLab.value) return;
  const laboratory = actionLab.value;
  finishContextAction();
  const value = await api.duplicateLab(
    laboratory.id,
    `${laboratory.name} copy`,
  );
  status.value = `复制任务已进入队列：${value.task.id}`;
  emit("changed");
}
function openRename() {
  if (!actionLab.value) return;
  const laboratory = actionLab.value;
  finishContextAction();
  renameName.value = laboratory.name;
  renameRevision.value = laboratory.revision;
  renameError.value = "";
  renameOpen.value = true;
}
async function renameLab() {
  if (!actionLab.value || !renameName.value.trim() || renameBusy.value) return;
  const laboratory = actionLab.value;
  renameBusy.value = true;
  renameError.value = "";
  try {
    await api.updateLab({
      ...laboratory,
      name: renameName.value.trim(),
      revision: renameRevision.value,
    });
    renameOpen.value = false;
    emit("changed");
  } catch (error) {
    if (error instanceof ApiError && error.status === 409) {
      const latest = await api.getLab(laboratory.id);
      renameRevision.value = latest.laboratory.revision;
      renameError.value =
        "The laboratory changed elsewhere. Your name is preserved; review it and retry against the refreshed revision.";
      emit("changed");
    } else {
      renameError.value =
        error instanceof Error ? error.message : "无法重命名实验室";
    }
  } finally {
    renameBusy.value = false;
  }
}
function openTransfer(mode: "export" | "import") {
  if (mode === "export" && !actionLab.value) return;
  finishContextAction();
  transferMode.value = mode;
  transferOpen.value = true;
}
async function deleteLab() {
  if (!actionLab.value) return;
  const laboratory = actionLab.value;
  const deletingId = laboratory.id;
  const value = await api.deleteLab(laboratory);
  status.value = `删除任务已进入队列：${value.task.id}`;
  deleteOpen.value = false;
  emit("deleteAccepted", deletingId);
}
function openDelete() {
  if (!actionLab.value) return;
  finishContextAction();
  deleteOpen.value = true;
}
</script>
<template>
  <header
    class="flex h-[var(--panel-toolbar-height)] items-center gap-2 border-b border-border bg-card px-2"
    aria-label="实验室工具栏"
  >
    <Button
      variant="ghost"
      size="icon"
      aria-label="切换设备面板"
      :disabled="!paletteAvailable"
      :title="
        paletteAvailable ? '显示或隐藏设备模板' : '添加设备前请创建或选择实验室'
      "
      autofocus
      @click="$emit('togglePalette')"
    >
      <Menu :size="17" />
    </Button>
    <Button
      variant="secondary"
      size="sm"
      aria-label="添加资源"
      :disabled="!paletteAvailable"
      @click="$emit('openCreate')"
    >
      <Plus :size="14" /> 添加
    </Button>
    <div class="flex items-center gap-2 pr-2">
      <span
        class="grid h-7 w-7 place-items-center rounded bg-primary text-primary-foreground"
        ><FlaskConical :size="17"
      /></span>
      <h1 v-if="showHeading" class="font-semibold tracking-wide">NetLab</h1>
    </div>
    <div ref="switcher" class="relative min-w-0 sm:w-72">
      <Button
        variant="outline"
        class="w-full min-w-0 justify-between gap-2 px-3"
        role="combobox"
        aria-label="实验室"
        aria-haspopup="listbox"
        :aria-expanded="switcherOpen"
        data-testid="laboratory-switcher"
        @click="toggleSwitcher"
      >
        <span class="min-w-0 truncate text-left">
          {{ active?.name || "选择实验室" }}
        </span>
        <span
          class="flex shrink-0 items-center gap-1 text-[10px] text-muted-foreground"
        >
          {{ labs.length }} <ChevronDown :size="13" />
        </span>
      </Button>
      <section
        v-show="switcherOpen"
        class="absolute left-0 top-full z-40 mt-1 w-[min(24rem,calc(100vw-1rem))] overflow-hidden rounded-md border border-border bg-popover shadow-2xl"
        aria-label="实验室切换器"
      >
        <header class="flex items-center gap-2 border-b border-border p-2">
          <div class="relative min-w-0 flex-1">
            <Search
              :size="14"
              class="pointer-events-none absolute left-2 top-2 text-muted-foreground"
            />
            <Input
              v-model="laboratoryQuery"
              aria-label="搜索实验室"
              class="pl-7"
              placeholder="搜索实验室"
            />
          </div>
          <Button size="sm" data-testid="new-laboratory" @click="openCreate">
            <Plus :size="14" /> 新建
          </Button>
        </header>
        <div
          role="listbox"
          aria-label="实验室列表"
          class="max-h-72 overflow-y-auto p-1 netlab-scrollbar"
          @contextmenu="openGlobalContext"
        >
          <div
            v-for="lab in filteredLabs"
            :key="lab.id"
            data-laboratory-row="true"
            :data-laboratory-id="lab.id"
            class="group flex w-full items-center rounded hover:bg-accent"
            :class="active?.id === lab.id && 'bg-accent/70'"
            @contextmenu="openLabContext(lab, $event)"
          >
            <button
              role="option"
              :aria-selected="active?.id === lab.id"
              class="flex min-w-0 flex-1 items-center gap-2 px-2 py-2 text-left"
              @click="selectLab(lab.id)"
            >
              <Check
                :size="14"
                class="shrink-0"
                :class="active?.id === lab.id ? 'text-primary' : 'opacity-0'"
              />
              <span class="min-w-0 flex-1">
                <strong class="block truncate text-xs font-medium">{{
                  lab.name
                }}</strong>
                <small class="block truncate text-[10px] text-muted-foreground">
                  {{
                    lab.recovery_policy === "auto_restore"
                      ? "自动恢复"
                      : "保持停止"
                  }}
                  · revision {{ lab.revision }}
                </small>
              </span>
            </button>
            <Button
              variant="ghost"
              size="icon"
              class="mr-1 h-7 w-7 shrink-0 opacity-60 group-hover:opacity-100"
              :aria-label="`${lab.name} 的操作`"
              :title="`管理 ${lab.name}`"
              @click="openLabActions(lab, $event)"
            >
              <MoreVertical :size="14" />
            </Button>
          </div>
          <p
            v-if="!filteredLabs.length"
            class="px-3 py-6 text-center text-xs text-muted-foreground"
          >
            没有匹配“{{ laboratoryQuery }}”的实验室。
          </p>
        </div>
        <footer
          class="border-t border-border px-3 py-2 text-[10px] text-muted-foreground"
        >
          右键单击实验室可重命名、复制、导出、导入或删除。触摸设备请使用
          <MoreVertical :size="11" class="inline" /> 按钮。
        </footer>
      </section>
    </div>
    <div class="ml-auto flex items-center gap-2">
      <ThemeSwitcher />
      <RouterLink
        to="/templates"
        class="hidden text-xs text-muted-foreground hover:text-foreground md:inline"
      >
        Templates </RouterLink
      ><RouterLink
        to="/automation"
        class="hidden text-xs text-muted-foreground hover:text-foreground md:inline"
      >
        Automation </RouterLink
      ><span
        role="status"
        class="hidden text-xs text-muted-foreground xl:block"
        >{{ status }}</span
      ><span
        class="inline-flex items-center gap-1 text-xs"
        :class="
          eventStatus === 'connected'
            ? 'text-[color:var(--success)]'
            : 'text-[color:var(--warning)]'
        "
        ><span class="h-2 w-2 rounded-full bg-current" />{{ eventStatus }}</span
      ><Button
        variant="ghost"
        size="icon"
        aria-label="刷新"
        :disabled="loading"
        @click="$emit('refresh')"
      >
        <RefreshCw :size="16" :class="loading && 'animate-spin'" /> </Button
      ><Button variant="ghost" size="sm" @click="$emit('openCommands')">
        ⌘K
      </Button>
    </div>
  </header>
  <Teleport to="body">
    <section
      v-if="contextMenuOpen"
      ref="contextMenu"
      role="menu"
      :aria-label="actionLab ? `${actionLab.name} 的操作` : '实验室操作'"
      class="fixed z-[70] w-52 rounded-md border border-border bg-popover p-1 shadow-2xl"
      :style="{ left: `${contextMenuX}px`, top: `${contextMenuY}px` }"
      @contextmenu.prevent
    >
      <p
        v-if="actionLab"
        class="truncate border-b border-border px-2 py-1.5 text-[10px] font-medium text-muted-foreground"
      >
        {{ actionLab.name }}
      </p>
      <Button
        v-if="actionLab"
        variant="ghost"
        size="sm"
        class="w-full justify-start"
        role="menuitem"
        @click="openRename"
      >
        <Edit3 :size="13" /> 重命名
      </Button>
      <Button
        v-if="actionLab"
        variant="ghost"
        size="sm"
        class="w-full justify-start"
        role="menuitem"
        @click="duplicateLab"
      >
        <Copy :size="13" /> 复制
      </Button>
      <Button
        v-if="actionLab"
        variant="ghost"
        size="sm"
        class="w-full justify-start"
        role="menuitem"
        @click="openTransfer('export')"
      >
        <Download :size="13" /> 导出
      </Button>
      <Button
        variant="ghost"
        size="sm"
        class="w-full justify-start"
        role="menuitem"
        @click="openTransfer('import')"
      >
        <Upload :size="13" /> 导入
      </Button>
      <Button
        v-if="actionLab"
        variant="ghost"
        size="sm"
        class="w-full justify-start text-red-300 hover:text-red-200"
        role="menuitem"
        @click="openDelete"
      >
        <Trash2 :size="13" /> 删除
      </Button>
    </section>
  </Teleport>
  <Dialog
    v-model="createOpen"
    title="创建实验室"
    description="创建由服务器管理并在客户端间共享的实验室。"
  >
    <form class="grid gap-3" @submit.prevent="createLab">
      <FormField data-field="name" label="名称">
        <Input v-model="name" required autofocus /> </FormField
      ><FormField label="恢复策略">
        <Select v-model="policy">
          <option value="auto_restore">宿主机重启后自动恢复</option>
          <option value="remain_stopped">保持停止</option>
        </Select> </FormField
      ><Button type="submit"> 创建实验室 </Button>
    </form>
  </Dialog>
  <LaboratoryTransferDialog
    v-model="transferOpen"
    :mode="transferMode"
    :laboratory="actionLab"
    @changed="$emit('changed')"
    @status="status = $event"
  />
  <Dialog
    v-model="renameOpen"
    :prevent-close="
      renameName.trim() !== (actionLab?.name || '') && !renameBusy
    "
    title="重命名实验室"
    description="名称会同步到所有已连接的浏览器。"
  >
    <form class="grid gap-3" @submit.prevent="renameLab">
      <FormField label="名称" :error="renameError">
        <Input v-model="renameName" required maxlength="120" />
      </FormField>
      <div class="flex justify-end gap-2">
        <Button type="button" variant="secondary" @click="renameOpen = false">
          取消
        </Button>
        <Button type="submit" :disabled="renameBusy || !renameName.trim()">
          {{ renameBusy ? "正在保存…" : "保存名称" }}
        </Button>
      </div>
    </form>
  </Dialog>
  <Dialog
    v-model="deleteOpen"
    title="删除实验室"
    :description="actionLab ? `删除 ${actionLab.name} 及其拥有的资源？` : ''"
  >
    <p class="text-sm text-muted-foreground">
      此操作不可恢复，清理进度会作为持久化任务持续显示。
    </p>
    <template #footer>
      <Button variant="secondary" @click="deleteOpen = false"> 取消 </Button
      ><Button variant="destructive" @click="deleteLab"> 删除 </Button>
    </template>
  </Dialog>
</template>
