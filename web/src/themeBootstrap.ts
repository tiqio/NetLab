import { THEME_STORAGE_KEY, type ResolvedTheme, type ThemePreference } from "@/composables/useThemePreference";

export function resolveInitialTheme(): { preference: ThemePreference; resolved: ResolvedTheme } {
  let preference: ThemePreference = "system";
  try {
    const value = localStorage.getItem(THEME_STORAGE_KEY);
    if (value === "system" || value === "light" || value === "dark") preference = value;
  } catch {
    preference = "system";
  }
  const system = globalThis.matchMedia?.("(prefers-color-scheme: light)").matches ? "light" : "dark";
  return { preference, resolved: preference === "system" ? system : preference };
}

export function bootstrapTheme(): void {
  const { resolved } = resolveInitialTheme();
  document.documentElement.dataset.theme = resolved;
  document.documentElement.style.colorScheme = resolved;
}

