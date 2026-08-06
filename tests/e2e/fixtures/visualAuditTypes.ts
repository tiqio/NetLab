export type VisualTheme = "light" | "dark";
export type VisualDensity = "empty" | "normal" | "dense";
export type VisualResult = "passed" | "failed" | "waived";
export type VisualSeverity = "blocking" | "serious" | "moderate" | "cosmetic";

export interface VisualAuditViewport {
  width: number;
  height: number;
}

export interface VisualAuditScenarioDefinition {
  id: string;
  surface: string;
  density: VisualDensity;
  states: string[];
}

export interface VisualAuditScenarioInventory {
  schema_version: "1.0";
  themes: VisualTheme[];
  viewports: VisualAuditViewport[];
  display_scales: number[];
  scenarios: VisualAuditScenarioDefinition[];
}

export interface LayoutBounds extends VisualAuditViewport {
  x: number;
  y: number;
}

export interface LayoutRegionObservation {
  id: string;
  role: string;
  bounds: LayoutBounds;
  interactive: boolean;
  may_overlap: string[];
}

export interface VisualFinding {
  id: string;
  scenario_id: string;
  kind:
    | "overlap"
    | "clipping"
    | "overflow"
    | "low-contrast"
    | "hidden-focus"
    | "hit-target-conflict"
    | "layout-shift"
    | "untranslated-text"
    | "terminology-mismatch";
  severity: VisualSeverity;
  region_ids: string[];
  message: string;
}

export interface LocalizationAuditItem {
  id: string;
  surface: string;
  source_text: string;
  classification: "product-copy" | "technical-term" | "user-data" | "raw-diagnostic";
  allow_english: boolean;
  result: "pending" | "pass" | "fail" | "waived";
}
