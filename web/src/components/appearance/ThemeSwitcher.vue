<script setup lang="ts">
import { MonitorCog, Moon, Sun } from "lucide-vue-next";
import { zhCN } from "@/locales";
import {
  useThemePreference,
  type ThemePreference,
} from "@/composables/useThemePreference";

const { preference, resolvedTheme, setPreference } = useThemePreference();
const options: Array<{ value: ThemePreference; label: string }> = [
  { value: "system", label: zhCN.appearance.system },
  { value: "light", label: zhCN.appearance.light },
  { value: "dark", label: zhCN.appearance.dark },
];
</script>

<template>
  <div class="theme-switcher" role="group" :aria-label="zhCN.appearance.label">
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
  </div>
</template>
