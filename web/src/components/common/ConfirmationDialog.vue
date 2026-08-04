<script setup lang="ts">
import { AlertTriangle } from "lucide-vue-next";
import { Button, Dialog } from "@/components/ui";
const open = defineModel<boolean>({ required: true });
defineProps<{
  title: string;
  resource: string;
  description: string;
  impact?: string;
  confirmLabel?: string;
  cancelLabel?: string;
}>();
defineEmits<{ confirm: [] }>();
</script>
<template>
  <Dialog v-model="open" :title="title" :description="description">
    <div class="grid gap-3 text-sm">
      <p class="rounded border border-destructive/40 bg-destructive/10 p-3">
        <AlertTriangle :size="15" class="mr-1 inline text-destructive" />{{
          resource
        }}
      </p>
      <p v-if="impact" class="text-xs text-muted-foreground">{{ impact }}</p>
      <div class="flex justify-end gap-2">
        <Button variant="secondary" @click="open = false">{{
          cancelLabel || "Cancel"
        }}</Button>
        <Button variant="destructive" @click="$emit('confirm')">{{
          confirmLabel || "Confirm"
        }}</Button>
      </div>
    </div>
  </Dialog>
</template>
