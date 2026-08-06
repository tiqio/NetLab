<script setup lang="ts">
import { ref } from "vue";
import { RouterLink } from "vue-router";
import { ArrowLeft, Upload } from "lucide-vue-next";
import { Button } from "@/components/ui";
import { ThemeSwitcher } from "@/components/appearance";
import TemplateCatalog from "@/features/templates/TemplateCatalog.vue";
import ImageImportDialog from "@/features/templates/ImageImportDialog.vue";
const catalog = ref<InstanceType<typeof TemplateCatalog>>();
const importOpen = ref(false);
</script>
<template>
  <main class="h-full overflow-auto bg-background netlab-scrollbar">
    <header
      class="sticky top-0 z-10 flex min-h-12 flex-wrap items-center gap-3 border-b border-border bg-card px-3 py-2"
    >
      <RouterLink
        to="/"
        class="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft :size="15" /> 工作区
      </RouterLink>
      <h1 class="font-semibold">模板与镜像</h1>
      <div
        class="ml-auto flex min-w-0 flex-wrap items-center justify-end gap-2"
      >
        <ThemeSwitcher />
        <Button size="sm" @click="importOpen = true">
          <Upload :size="14" /> 导入镜像引用
        </Button>
      </div>
    </header>
    <TemplateCatalog ref="catalog" /><ImageImportDialog
      v-model="importOpen"
      @imported="catalog?.refresh()"
    />
  </main>
</template>
