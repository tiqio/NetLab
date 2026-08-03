import { computed, onBeforeUnmount, ref, watch, type Ref } from "vue";
import type { BottomTab, WorkspacePreferences } from "@/types/workspace";
import { randomUUID } from "@/lib/uuid";

const STORAGE_PREFIX = "netlab.workspace.v1.";
const now = () => new Date().toISOString();
const clamp = (value: number, minimum: number, maximum: number) =>
  Math.min(maximum, Math.max(minimum, value));

export function defaultWorkspacePreferences(
  laboratoryId: string,
): WorkspacePreferences {
  return {
    schemaVersion: 1,
    laboratoryId,
    updatedAt: now(),
    panels: {
      devicePalette: { collapsed: false, size: 260 },
      inspector: { collapsed: false, size: 340 },
      bottomDrawer: { collapsed: false, size: 250 },
    },
    viewport: { centerX: 0, centerY: 0, zoom: 1 },
    groups: [],
    linkRoutes: {},
    labelDensity: "comfortable",
    reducedMotion: false,
    activeBottomTab: "tasks",
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function parseWorkspacePreferences(
  value: unknown,
  laboratoryId: string,
): WorkspacePreferences {
  if (
    !isRecord(value) ||
    value.schemaVersion !== 1 ||
    value.laboratoryId !== laboratoryId
  )
    return defaultWorkspacePreferences(laboratoryId);
  const result = defaultWorkspacePreferences(laboratoryId);
  const panels = isRecord(value.panels) ? value.panels : {};
  const panelRanges: Record<
    keyof WorkspacePreferences["panels"],
    readonly [number, number]
  > = {
    devicePalette: [180, 640],
    inspector: [180, 640],
    bottomDrawer: [160, 720],
  };
  for (const key of Object.keys(panelRanges) as Array<
    keyof WorkspacePreferences["panels"]
  >) {
    const range = panelRanges[key];
    const panel = isRecord(panels[key]) ? panels[key] : {};
    result.panels[key] = {
      collapsed: typeof panel.collapsed === "boolean" ? panel.collapsed : false,
      size: clamp(
        typeof panel.size === "number" ? panel.size : result.panels[key].size,
        range[0],
        range[1],
      ),
    };
  }
  if (isRecord(value.viewport)) {
    result.viewport = {
      centerX: Number.isFinite(value.viewport.centerX)
        ? Number(value.viewport.centerX)
        : 0,
      centerY: Number.isFinite(value.viewport.centerY)
        ? Number(value.viewport.centerY)
        : 0,
      zoom: clamp(
        Number.isFinite(value.viewport.zoom) ? Number(value.viewport.zoom) : 1,
        0.1,
        8,
      ),
    };
  }
  if (Array.isArray(value.groups)) {
    result.groups = value.groups.flatMap((candidate) => {
      if (
        !isRecord(candidate) ||
        typeof candidate.id !== "string" ||
        typeof candidate.label !== "string" ||
        !Array.isArray(candidate.memberResourceIds)
      )
        return [];
      return [
        {
          id: candidate.id,
          label: candidate.label,
          memberResourceIds: candidate.memberResourceIds.filter(
            (id): id is string => typeof id === "string",
          ),
          collapsed: candidate.collapsed === true,
        },
      ];
    });
  }
  if (isRecord(value.linkRoutes)) {
    for (const [id, points] of Object.entries(value.linkRoutes)) {
      if (!Array.isArray(points)) continue;
      result.linkRoutes[id] = points.flatMap((point) =>
        isRecord(point) && Number.isFinite(point.x) && Number.isFinite(point.y)
          ? [{ x: Number(point.x), y: Number(point.y) }]
          : [],
      );
    }
  }
  if (
    ["comfortable", "compact", "minimal"].includes(String(value.labelDensity))
  )
    result.labelDensity =
      value.labelDensity as WorkspacePreferences["labelDensity"];
  result.reducedMotion = value.reducedMotion === true;
  if (
    ["tasks", "console", "captures", "traffic-filter"].includes(
      String(value.activeBottomTab),
    )
  )
    result.activeBottomTab = value.activeBottomTab as BottomTab;
  result.updatedAt =
    typeof value.updatedAt === "string" ? value.updatedAt : now();
  return result;
}

export function loadWorkspacePreferences(
  laboratoryId: string,
  storage: Storage = localStorage,
) {
  try {
    return parseWorkspacePreferences(
      JSON.parse(storage.getItem(`${STORAGE_PREFIX}${laboratoryId}`) || "null"),
      laboratoryId,
    );
  } catch {
    return defaultWorkspacePreferences(laboratoryId);
  }
}

export function removeWorkspacePreferences(
  laboratoryId: string,
  storage: Pick<Storage, "removeItem"> = localStorage,
) {
  try {
    storage.removeItem(`${STORAGE_PREFIX}${laboratoryId}`);
  } catch {
    return;
  }
}

export function pruneLinkRoutes(
  preferences: WorkspacePreferences,
  authoritativeLinkIds: Iterable<string>,
) {
  const available = new Set(authoritativeLinkIds);
  let changed = false;
  for (const linkId of Object.keys(preferences.linkRoutes)) {
    if (available.has(linkId)) continue;
    delete preferences.linkRoutes[linkId];
    changed = true;
  }
  return changed;
}

export function useWorkspacePreferences(laboratoryId: Ref<string | undefined>) {
  const preferences = ref(
    defaultWorkspacePreferences(laboratoryId.value || "pending"),
  );
  let timer: ReturnType<typeof setTimeout> | undefined;
  const storageAvailable = ref(true);

  function persist() {
    if (!laboratoryId.value) return;
    preferences.value.updatedAt = now();
    try {
      localStorage.setItem(
        `${STORAGE_PREFIX}${laboratoryId.value}`,
        JSON.stringify(preferences.value),
      );
      storageAvailable.value = true;
    } catch {
      storageAvailable.value = false;
    }
  }
  function schedulePersist() {
    clearTimeout(timer);
    timer = setTimeout(persist, 120);
  }
  function setPanel(
    panel: keyof WorkspacePreferences["panels"],
    value: Partial<WorkspacePreferences["panels"][typeof panel]>,
  ) {
    preferences.value.panels[panel] = {
      ...preferences.value.panels[panel],
      ...value,
    };
    schedulePersist();
  }
  function setViewport(value: Partial<WorkspacePreferences["viewport"]>) {
    preferences.value.viewport = { ...preferences.value.viewport, ...value };
    schedulePersist();
  }
  function setActiveBottomTab(value: BottomTab) {
    preferences.value.activeBottomTab = value;
    schedulePersist();
  }
  function createGroup(resourceIds: string[]) {
    const unique = [...new Set(resourceIds)];
    if (unique.length < 2) return;
    preferences.value.groups.push({
      id: randomUUID(),
      label: `Group ${preferences.value.groups.length + 1}`,
      memberResourceIds: unique,
      collapsed: false,
    });
    schedulePersist();
  }
  function toggleGroup(groupId: string) {
    const group = preferences.value.groups.find((item) => item.id === groupId);
    if (!group) return;
    group.collapsed = !group.collapsed;
    schedulePersist();
  }
  function setLinkRoute(
    linkId: string,
    points: Array<{ x: number; y: number }>,
  ) {
    if (points.length) preferences.value.linkRoutes[linkId] = points;
    else delete preferences.value.linkRoutes[linkId];
    schedulePersist();
  }
  function setLabelDensity(value: WorkspacePreferences["labelDensity"]) {
    preferences.value.labelDensity = value;
    schedulePersist();
  }
  function setReducedMotion(value: boolean) {
    preferences.value.reducedMotion = value;
    schedulePersist();
  }

  watch(
    laboratoryId,
    (id) => {
      if (id) preferences.value = loadWorkspacePreferences(id);
    },
    { immediate: true },
  );
  onBeforeUnmount(() => {
    clearTimeout(timer);
    persist();
  });
  return {
    preferences,
    storageAvailable: computed(() => storageAvailable.value),
    setPanel,
    setViewport,
    setActiveBottomTab,
    createGroup,
    toggleGroup,
    setLinkRoute,
    setLabelDensity,
    setReducedMotion,
    persist,
  };
}
