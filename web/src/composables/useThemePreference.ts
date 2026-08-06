import { computed, onBeforeUnmount, ref } from "vue";

export type ThemePreference = "system" | "light" | "dark";
export type ResolvedTheme = "light" | "dark";
export const THEME_STORAGE_KEY = "netlab.appearance.v1";

function systemTheme(): ResolvedTheme {
  return globalThis.matchMedia?.("(prefers-color-scheme: light)").matches
    ? "light"
    : "dark";
}

function readPreference(): ThemePreference {
  try {
    const value = localStorage.getItem(THEME_STORAGE_KEY);
    return value === "light" || value === "dark" || value === "system"
      ? value
      : "system";
  } catch {
    return "system";
  }
}

const preference = ref<ThemePreference>(readPreference());
const systemResolved = ref<ResolvedTheme>(systemTheme());

export function applyResolvedTheme(theme: ResolvedTheme): void {
  document.documentElement.dataset.theme = theme;
  document.documentElement.style.colorScheme = theme;
}

export function useThemePreference() {
  const media = globalThis.matchMedia?.("(prefers-color-scheme: light)");
  const onSystemChange = (event: MediaQueryListEvent) => {
    systemResolved.value = event.matches ? "light" : "dark";
    if (preference.value === "system") applyResolvedTheme(systemResolved.value);
  };
  media?.addEventListener?.("change", onSystemChange);

  const resolvedTheme = computed<ResolvedTheme>(() =>
    preference.value === "system" ? systemResolved.value : preference.value,
  );

  const setPreference = (value: ThemePreference) => {
    preference.value = value;
    try {
      localStorage.setItem(THEME_STORAGE_KEY, value);
    } catch {
      // Current-session state remains authoritative when storage is unavailable.
    }
    applyResolvedTheme(resolvedTheme.value);
  };

  applyResolvedTheme(resolvedTheme.value);
  onBeforeUnmount(() => media?.removeEventListener?.("change", onSystemChange));

  return { preference, resolvedTheme, setPreference };
}
