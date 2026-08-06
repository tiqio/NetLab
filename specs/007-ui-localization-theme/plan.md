# Implementation Plan: 全页面中文化与明暗主题

**Branch**: `[007-ui-localization-theme]` | **Date**: 2026-08-06 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/007-ui-localization-theme/spec.md`

## Summary

将当前分散在各页面的中英文文案收敛为简体中文界面资源和统一术语/状态格式化规则，同时把现有单一深色变量扩展为浅色、深色两套语义主题。主题选择为浏览器本地显示状态，支持首次跟随设备偏好、用户手动覆盖、刷新保持和无存储回退；主题变化通过全局语义变量传递给页面、Shadcn 风格组件、拓扑画布和图表，不改变实验室共享状态、运行会话或任何 HTTP/MCP 合同。

## Technical Context

**Language/Version**: TypeScript 5.8.x、Vue 3.5.x；后端 Go 1.26.x 不变

**Primary Dependencies**: 现有 Vue Router、Pinia、Tailwind CSS 4、Shadcn Vue 风格组件、Lucide Vue、ECharts、xterm.js、noVNC；不新增运行时依赖

**Storage**: 不新增服务端或 SQLite 数据；主题偏好使用浏览器本地存储，语言固定为 `zh-CN`

**Testing**: Vitest、Vue Test Utils、axe-core、Playwright、现有交互覆盖矩阵、目标机真实浏览器验收

**Target Platform**: 现代桌面浏览器，最低支持视口 1024×768；SPA 最终部署到 `10.72.1.7`

**Project Type**: 现有 Vue 单页应用；仅调整前端显示状态、文案资源和视觉语义，后端合同不变

**Performance Goals**: 首屏在主要内容绘制前确定主题；手动主题切换 300ms 内完成可见界面更新；切换不触发网络资源重新加载或拓扑重新布局

**Constraints**: 简体中文为唯一产品语言；保留技术数据原文；主题选择不得同步到服务端；不得中断 Console、VNC、Capture、Traffic Filter 或运行任务；不得引入不可审计的固定颜色和散落英文产品文案

**Scale/Scope**: 3 个顶层路由、约 158 个前端组件/视图、全部共享 UI 基础组件、拓扑与 ECharts 可视化、三种代表性桌面视口、两种显式主题和系统跟随模式

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Design Gate

- **Shared state — PASS**: 语言固定为中文，主题偏好和已解析主题仅存在于当前浏览器；实验室、节点、链路、任务、抓包和运行状态继续以服务端快照及有序事件为权威。
- **Control parity — PASS**: 本功能不新增持久化资源或状态变更操作；界面现有动作继续调用相同 HTTP/MCP 应用命令，机器可读枚举和错误代码保持不变。
- **Runtime scope — PASS**: 不改变 QEMU、Docker、bridge、NAT、netns、镜像或单机运行时边界。
- **Live operations — PASS**: 主题和中文文案只改变显示；运行节点实时接线、QMP、Console、VNC、Capture 和 Traffic Filter 的会话、任务及回滚行为不变。
- **Safety and recovery — PASS**: 不新增特权操作或运行资源；本地存储失败采用当前会话回退，服务重启和事件重连继续恢复权威状态。
- **Verification — PASS**: 计划覆盖文案清单、术语映射、主题状态、CSS 语义、共享组件、拓扑、图表、键盘、axe、三视口浏览器和目标机运行会话连续性。
- **Image and secret hygiene — PASS**: 不新增镜像、凭据、抓包或秘密；技术内容原样展示，不写入源码之外的验收秘密。
- **Local-first delivery — PASS**: 按中文化基础、主题基础、复杂视图适配、全量验收四个本地里程碑提交；仅从干净提交构建并部署，记录 SHA、摘要、迁移状态和目标结果。

### Post-Design Re-check

- **PASS**: `research.md` 确定浏览器本地主题状态和无新依赖方案；`data-model.md` 不新增服务端实体或数据库迁移。
- **PASS**: UI 合同明确主题/语言不进入 API、MCP、事件或实验室快照，避免显示偏好污染共享状态。
- **PASS**: 快速验收覆盖运行中的 QEMU、Docker、链路、Console 和 Traffic Filter 在主题切换前后的连续性、权威状态与资源清理。
- **PASS**: 未发现需要宪法例外或超出单机产品边界的设计。

## Project Structure

### Documentation (this feature)

```text
specs/007-ui-localization-theme/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── localization-contract.md
│   └── theme-contract.md
└── tasks.md
```

### Source Code (repository root)

```text
web/
├── index.html
├── src/
│   ├── App.vue
│   ├── main.ts
│   ├── locales/
│   │   ├── zh-CN.ts
│   │   ├── terminology.ts
│   │   └── statusMessages.ts
│   ├── composables/
│   │   ├── useThemePreference.ts
│   │   └── useThemePreference.test.ts
│   ├── components/
│   │   ├── appearance/ThemeSwitcher.vue
│   │   ├── charts/EChart.vue
│   │   ├── common/
│   │   ├── shell/
│   │   └── ui/
│   ├── features/
│   │   ├── laboratories/
│   │   ├── topology/
│   │   ├── nodes/
│   │   ├── diagnostics/
│   │   ├── tasks/
│   │   └── templates/
│   ├── views/
│   │   ├── TemplatesView.vue
│   │   └── AutomationView.vue
│   └── styles/
│       ├── index.css
│       └── theme.css
└── tests/

tests/e2e/
├── matrices/
├── journeys/
├── frontend_responsive_keyboard.spec.ts
└── pages/

scripts/
└── check-ui-localization.sh
```

**Structure Decision**: 在现有 Vue SPA 内增加轻量、类型化的中文文案与主题状态模块，不引入新的国际化框架。共享文案、术语和状态映射集中在 `web/src/locales`；主题状态集中在 composable，视觉语义集中在 CSS 变量；各顶层页面复用同一主题切换组件。后端、数据库和运行时目录不修改。

## Implementation Strategy

### Milestone 1 — 中文术语与基础组件

- 建立中文术语表、状态/错误格式化规则和允许保留原文的技术词清单。
- 中文化共享按钮、Dialog、Sheet、确认、空态、结构化错误和导航基础组件。
- 增加文案扫描门禁，识别用户可见英文文字、英文可访问名称和遗漏占位符，并支持小型显式豁免清单。
- 本地门禁：文案资源单测、共享组件测试、扫描脚本、格式、静态检查；创建聚焦提交。

### Milestone 2 — 全局主题与页面入口

- 建立 `system/light/dark` 偏好状态和 `light/dark` 解析状态；处理本地存储、系统偏好监听、无存储回退和用户手动覆盖。
- 在首屏应用挂载前应用已解析主题，防止错误主题闪烁。
- 将现有深色变量拆为浅色/深色语义变量，禁止组件直接依赖主题专属固定颜色。
- 在拓扑工具栏、模板页和自动化页放置一致的中文主题切换入口，并补键盘与焦点测试。
- 本地门禁：主题状态单测、CSS/组件测试、首屏脚本测试、构建；创建聚焦提交。

### Milestone 3 — 复杂视图与全页面中文化

- 按路由和功能域中文化实验室、添加抽屉、拓扑、Inspector、节点操作、任务、Console、Capture、Wireshark、Traffic Filter、模板和自动化页面。
- 为拓扑图、Traffic Filter、资源图表和通用 EChart 提供主题调色板，主题变化时更新图表但保留数据、缩放、选中和拖动状态。
- 保持 xterm/noVNC 客户机内容原始配色，仅主题化外围导航、标签、边框和状态。
- 修复中文长度导致的溢出、重叠和不可点击问题，覆盖 1024×768 视口。
- 本地门禁：功能域组件测试、axe、视觉语义测试、三视口 Playwright；创建聚焦提交。

### Milestone 4 — 全量验证、候选与目标机验收

- 运行全量前端、Go 合同、格式、静态检查、构建和浏览器覆盖矩阵，确保 API/MCP 机器字段未改变。
- 从干净提交构建候选，记录 commit SHA、candidate ID、contract digest、binary digest、构建时间和无迁移变化的 schema state。
- 部署到 `10.72.1.7`，验证两种主题、不同浏览器隔离、系统跟随、运行 QEMU/Docker、链路、Console、Capture 和 Traffic Filter 会话连续性。
- 清理验收 Lab 和运行资源；失败时回到本地修复、测试、提交和重新部署。

## Complexity Tracking

无宪法违规。本功能不新增后端状态、数据库迁移、运行时适配器或外部依赖。
