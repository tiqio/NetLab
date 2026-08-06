import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

describe("semantic theme", () => {
  const css = readFileSync(
    resolve(process.cwd(), "src/styles/theme.css"),
    "utf8",
  );
  const indexCss = readFileSync(
    resolve(process.cwd(), "src/styles/index.css"),
    "utf8",
  );
  const workspaceCss = readFileSync(
    resolve(process.cwd(), "src/styles/workspace.css"),
    "utf8",
  );

  it("定义浅色和深色根主题", () => {
    expect(css).toContain(':root[data-theme="dark"]');
    expect(css).toContain(':root[data-theme="light"]');
  });

  it("为拓扑和图表提供语义变量", () => {
    for (const token of [
      "--topology-running",
      "--topology-failed",
      "--topology-transition",
      "--topology-selected",
      "--topology-traffic",
      "--topology-port",
      "--chart-grid",
      "--chart-track",
      "--chart-series-primary",
      "--chart-danger",
    ]) {
      expect(css.match(new RegExp(`${token}:`, "g"))).toHaveLength(2);
    }
  });

  it("让原生下拉框和选项跟随当前主题", () => {
    expect(indexCss).toContain("select option");
    expect(indexCss).toContain("background-color: var(--popover)");
    expect(indexCss).toContain("color: var(--popover-foreground)");
    expect(indexCss).toContain(':root[data-theme="light"] select');
    expect(indexCss).toContain(':root[data-theme="dark"] select');
    expect(workspaceCss).toContain(".theme-switcher option");
  });
});
