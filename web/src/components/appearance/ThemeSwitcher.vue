<script setup lang="ts">
import { computed } from "vue";
import { MonitorCog, Moon, Sun } from "lucide-vue-next";
import { zhCN } from "@/locales";
import {
  useThemePreference,
  type ThemePreference,
} from "@/composables/useThemePreference";

const { preference, resolvedTheme, setPreference } = useThemePreference();
const options = computed<Array<{ value: ThemePreference; label: string }>>(
  () => [
    {
      value: "system",
      label: `${zhCN.appearance.system}（当前${resolvedTheme.value === "light" ? "浅色" : "深色"}）`,
    },
    { value: "light", label: zhCN.appearance.light },
    { value: "dark", label: zhCN.appearance.dark },
  ],
);
const resolvedLabel = computed(() =>
  resolvedTheme.value === "light" ? "浅色主题" : "深色主题",
);
</script>

<template>
  <div
    class="theme-switcher"
    role="group"
    :aria-label="zhCN.appearance.label"
    :data-resolved-theme="resolvedTheme"
    :title="`当前生效：${resolvedLabel}`"
  >
    <MonitorCog v-if="preference === 'system'" :size="16" aria-hidden="true" />
    <Sun v-else-if="resolvedTheme === 'light'" :size="16" aria-hidden="true" />
    <Moon v-else :size="16" aria-hidden="true" />
    <label class="sr-only" for="netlab-theme-preference">{{
      zhCN.appearance.label
    }}</label>
    <select
      id="netlab-theme-preference"
      :value="preference"
      :aria-label="zhCN.appearance.label"
      @change="
        setPreference(
          ($event.target as HTMLSelectElement).value as ThemePreference,
        )
      "
    >
      <option
        v-for="option in options"
        :key="option.value"
        :value="option.value"
      >
        {{ option.label }}
      </option>
    </select>
    <span
      class="h-2 w-2 shrink-0 rounded-full border border-border"
      :class="resolvedTheme === 'light' ? 'bg-amber-400' : 'bg-indigo-400'"
      aria-hidden="true"
    />
    <span class="sr-only" role="status">当前生效：{{ resolvedLabel }}</span>
  </div>
</template>
