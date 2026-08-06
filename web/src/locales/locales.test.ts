import { describe, expect, it } from "vitest";
import { exactTranslations, localizeState, problemContext, technicalTerms, terminology } from "@/locales";

describe("简体中文资源", () => {
  it("为核心术语提供唯一中文名称", () => {
    expect(new Set(Object.values(terminology)).size).toBe(Object.values(terminology).length);
  });

  it("保留技术缩写并翻译产品动作", () => {
    expect(technicalTerms).toContain("QEMU");
    expect(exactTranslations["Start capture"]).toBe("开始抓包");
  });

  it("本地化已知状态并保留未知机器值", () => {
    expect(localizeState("running")).toBe("运行中");
    expect(localizeState("vendor-state")).toBe("vendor-state");
  });

  it("在保留原始错误时提供中文上下文", () => {
    expect(problemContext("operation_timeout")).toContain("超时");
    expect(problemContext("revision_conflict")).toContain("刷新");
  });
});

