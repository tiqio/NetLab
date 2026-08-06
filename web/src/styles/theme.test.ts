import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

describe("semantic theme", () => {
  const css = readFileSync(
    resolve(process.cwd(), "src/styles/theme.css"),
    "utf8",
  );

  it("定义浅色和深色根主题", () => {
    expect(css).toContain(':root[data-theme="dark"]');
    expect(css).toContain(':root[data-theme="light"]');
  });

  it("为拓扑和图表提供语义变量", () => {
    expect(css).toContain("--topology-active");
    expect(css).toContain("--chart-grid");
  });
});
