<script setup lang="ts">
import { ref } from "vue";
import { RouterLink } from "vue-router";
import { ArrowLeft, Upload } from "lucide-vue-next";
import { Button } from "@/components/ui";
import TemplateCatalog from "@/features/templates/TemplateCatalog.vue";
import ImageImportDialog from "@/features/templates/ImageImportDialog.vue";
const catalog = ref<InstanceType<typeof TemplateCatalog>>();
const importOpen = ref(false);
</script>
<template>
  <main class="h-full overflow-auto bg-background netlab-scrollbar">
    <header
      class="sticky top-0 z-10 flex h-12 items-center gap-3 border-b border-border bg-card px-3"
    >
      <RouterLink
        to="/"
        class="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft :size="15" /> Workspace
      </RouterLink>
      <h1 class="font-semibold">Templates and images</h1>
      <Button class="ml-auto" size="sm" @click="importOpen = true">
        <Upload :size="14" /> Import image reference
      </Button>
    </header>
    <TemplateCatalog ref="catalog" /><ImageImportDialog
      v-model="importOpen"
      @imported="catalog?.refresh()"
    />
  </main>
</template>
