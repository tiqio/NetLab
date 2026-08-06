# Implementation Plan: 全页面视觉与中文化治理

**Branch**: `[008-ui-overlap-remediation]` | **Date**: 2026-08-06 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/008-ui-overlap-remediation/spec.md`

**Note**: 本计划只治理前端布局、可读性、中文化和交互可达性，不改变服务器权威资源、API、MCP、数据库或运行时生命周期。

## Summary

全面审计并修复 NetLab SPA 在拓扑、检查器、上下文菜单、抽屉、对话框、底部工作区和图表中的元素碰撞、低对比度、不可点击区域与中英文混杂。实施基于现有 Vue 组件、Shadcn 风格基础组件、语义主题变量和 ECharts，自上而下建立布局区域、确定性拓扑端口几何、统一菜单与错误呈现、集中中文术语及自动文案扫描，并通过 Vitest、Playwright、axe、几何碰撞断言和目标机高密度 Lab 验收形成持续门禁。

## Technical Context

**Language/Version**: TypeScript 5.8.3、Vue 3.5.18、CSS/Tailwind CSS 4.3；Go 1.26.x 后端仅执行既有合同回归，不新增后端实现

**Primary Dependencies**: Vue Router 4.5、Pinia 3.0、现有 Shadcn Vue/Reka UI 基础组件、ECharts 6.1、Lucide Vue、xterm.js 5.5、noVNC 1.6

**Storage**: 不新增服务端存储或 SQLite 迁移；继续使用现有浏览器主题与工作区偏好存储，审计证据输出到测试结果目录且不得包含敏感数据

**Testing**: Vitest 3.2、Vue Test Utils 2.4、Playwright 1.54、axe-core 4.12、现有中文文案扫描脚本、前端构建与既有 Go 单元/合同回归

**Target Platform**: 现代桌面浏览器，最小支持视口 1024×768；本地 Linux 开发环境及部署在 `10.72.1.7` 的单机 NetLab 服务

**Project Type**: 现有 Go 后端 + Vue SPA Web 应用；本功能为前端横切治理

**Performance Goals**: 拖动、缩放、菜单打开和流量效果保持交互流畅；面板尺寸或主题变化后在一个稳定渲染周期内完成重排；不因审计辅助逻辑持续轮询布局或引入可感知输入延迟

**Constraints**: 不改变 API/MCP/事件字段和服务器权威状态；不产生页面级横向滚动；支持浅色/深色、1024×768、1366×768、1920×1080 与关键流程 125% 显示比例；保留 EVE-NG 类工作流；技术原文和用户数据不得被机械翻译

**Scale/Scope**: 3 个顶层路由、约 66 个 Vue 组件、拓扑与 6 类主要网络对象、任务/终端/抓包/流量过滤底部工作区、全部上下文菜单/抽屉/对话框，以及约 69 个现有组件测试和 43 个浏览器规格的增量回归

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

### Pre-Research Gate

- **PASS — Shared state**: 本功能只改变客户端展示、布局和本地偏好；实验室、节点、链路、接口、任务、抓包和流量过滤继续以服务器状态为准，并保留现有 revision、事件与并发客户端语义。
- **PASS — Control parity**: 不新增仅 UI 可用的资源操作；重排后的启动、停止、删除、终端、抓包和流量过滤按钮继续调用现有应用命令/API，MCP 合同不变。
- **PASS — Runtime scope**: 不增加 QEMU、Docker、网络命名空间、网桥、NAT 或宿主机运行时能力，也不改变设备模板范围。
- **PASS — Live operations**: 运行中连线、QMP 网卡变更、Console、Capture 和 Traffic Filter 的业务行为保持不变；验收专门验证布局变化不会中断或误报这些会话。
- **PASS — Safety and recovery**: 不创建新的特权资源；测试资源继续通过现有验收资源台账拥有和清理，目标机只删除本功能创建的专用 Lab。
- **PASS — Verification**: 设计包含组件单测、文案扫描、浏览器几何/键盘/axe 检查、现有合同回归、目标机真实 Lab 验收和清理基线比较。
- **PASS — Image and secret hygiene**: 不引入镜像、凭据、bootstrap 数据或抓包；截图和证据必须排除终端敏感输出、密码和数据包内容。
- **PASS — Local-first delivery**: 五个独立里程碑均要求本地门禁和聚焦 Git 提交后才能部署；候选记录 commit SHA 与产物摘要，禁止在 `10.72.1.7` 直接改源码。

## Project Structure

### Documentation (this feature)

```text
specs/008-ui-overlap-remediation/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── acceptance-evidence.schema.json
│   ├── localization-contract.md
│   └── visual-layout-contract.md
└── tasks.md
```

### Source Code (repository root)

```text
web/
├── src/
│   ├── locales/
│   │   ├── zh-CN.ts
│   │   ├── terminology.ts
│   │   └── statusMessages.ts
│   ├── components/
│   │   ├── charts/
│   │   ├── common/
│   │   ├── shell/
│   │   └── ui/
│   ├── features/
│   │   ├── analytics/
│   │   ├── diagnostics/
│   │   ├── laboratories/
│   │   ├── nodes/
│   │   ├── tasks/
│   │   ├── templates/
│   │   └── topology/
│   ├── styles/
│   │   ├── index.css
│   │   ├── theme.css
│   │   └── workspace.css
│   └── views/
├── playwright.config.ts
├── playwright.target.config.ts
└── vitest.config.ts

tests/e2e/
├── fixtures/
├── journeys/
├── matrices/
├── pages/
├── frontend_localization.spec.ts
├── frontend_localization_theme_accessibility.spec.ts
├── frontend_responsive_keyboard.spec.ts
└── frontend_topology_workspace.spec.ts

scripts/
├── check-ui-localization.mjs
├── check-ui-localization.sh
├── compare-frontend-acceptance-runs.mjs
└── run-constitution-acceptance.sh
```

**Structure Decision**: 在现有 SPA 内完成横切治理，不新建前端应用或后端模块。共享布局与可读状态优先进入 `components/common`、`components/ui` 和语义样式文件；拓扑几何留在 `features/topology` 的纯计算与渲染边界；中文术语和状态映射集中在 `locales`；回归检测扩展现有 Vitest、Playwright 和文案扫描设施。

## Implementation Strategy

### Milestone 1 — 审计基线与共享视觉原语

- 建立页面/状态/视口/主题审计矩阵和严重度记录格式，补充稳定的测试选择器与碰撞检测辅助函数。
- 统一文字层级、间距、局部滚动、截断/展开、焦点轮廓、禁用/危险/选中状态和命中区域规则。
- 修复 Dialog、Sheet、Dropdown、Tooltip、FormField、EmptyState、StructuredProblem 等共享组件，使后续功能域继承正确布局和对比度。
- 本地门禁：共享组件 Vitest、主题测试、axe、格式、静态检查、前端构建和文案扫描。
- 里程碑提交：聚焦提交共享视觉原语与审计工具，记录提交 SHA 和覆盖矩阵。

### Milestone 2 — 拓扑节点、端口、链路与流量效果

- 将节点内部划分为身份、状态、图形、端口轨道和连接入口，建立 1/2/4/8/16 接口确定性几何测试。
- 修复端口标签与加号碰撞、鼠标经过偏移、拖动后漂移、平行链路重合、标签遮挡和未活动链路误高亮。
- 约束方向箭头、粒子和 Traffic Filter 高亮图层；停止流量后按合同衰减，不引发布局收缩或节点移动。
- 验证缩放、平移、适配、重置、节点拖动和运行中连线时的稳定命中区域。
- 本地门禁：几何纯函数测试、TopologyCanvas 组件测试、三视口浏览器交互、键盘选择和高密度拓扑截图证据。
- 里程碑提交：聚焦提交拓扑视觉与交互修复。

### Milestone 3 — 检查器、图表、菜单和底部工作区

- 重构检查器标题、资源图、指标、状态标志和操作按钮为明确布局区，支持窄宽、长文本和滚动。
- 让资源图和 ECharts 按容器尺寸重排，避免 vCPU、CPU 配额、内存、上下行文字覆盖绘图区。
- 统一节点、链路、网络对象和空画布右键菜单的碰撞避让、层级、语义颜色、键盘焦点和危险操作呈现。
- 修复任务、终端、抓包、流量过滤和诊断工作区标签、工具栏、会话列表、空状态和内容滚动关系。
- 本地门禁：检查器/图表/菜单/工作区组件测试、边缘菜单浏览器测试、面板展开收起循环、axe 和页面溢出检查。
- 里程碑提交：聚焦提交检查器、菜单与工作区治理。

### Milestone 4 — 全页面中文化与错误呈现

- 按初步审计清单清理 Tasks、Console、Capture、Serial、Reconnect、Close、Select、Remove、running 等非白名单英文，并继续扫描所有页面、动态字符串和无障碍文本。
- 固化核心中文术语和技术名称白名单，扩展文案扫描以覆盖模板文本、属性、动态消息与常见拼接模式。
- 为 Wireshark 辅助程序及其他已知错误提供中文摘要、操作建议和可展开原始详情；未知状态采用中文类别加原始值。
- 更新受影响测试断言和页面对象，防止测试继续依赖旧英文标签。
- 本地门禁：中文化扫描零非白名单结果、术语单测、代表性错误分支测试、三路由浏览器遍历和无障碍名称检查。
- 里程碑提交：聚焦提交全页面中文化与错误呈现。

### Milestone 5 — 全量回归、候选构建与目标机验收

- 运行格式、静态检查、全部前端单测、构建、浏览器矩阵和适用的 Go 单元/合同测试，确认 API/MCP/事件和权威状态无差异。
- 从干净提交构建候选，记录 commit SHA、candidate ID、前端/服务产物摘要、合同摘要、无迁移状态和构建时间。
- 部署到 `10.72.1.7`，创建验收专用高密度 Lab，覆盖 QEMU、Docker、PC、Bridge、NAT Bridge、Lightweight L2/L3、多接口、多链路、失败状态和 Traffic Filter。
- 在浅色/深色、1024×768、1366×768、1920×1080 与关键流程 125% 显示比例下验证重叠、对比度、中文化、键盘、菜单边界、终端/抓包/流量过滤连续性和跨客户端权威状态。
- 删除专用 Lab 及其进程、容器、接口、链路、抓包和过滤资源，对比验收前后资源基线；失败时回到本地形成新测试、新提交和新候选。
- 里程碑提交：验收证据与部署记录形成独立文档提交，不包含敏感截图、凭据或抓包内容。

## Post-Design Constitution Check

- **PASS — Shared state and control parity**: `data-model.md` 仅定义客户端审计和证据对象，不新增共享业务实体；`visual-layout-contract.md` 明确布局不得改变任何操作结果，客户端显示偏好保持隔离。
- **PASS — Runtime and live operations**: 合同将拓扑、Console、Capture 和 Traffic Filter 视为既有行为，验收只确认视觉治理不打断运行中操作。
- **PASS — Safety and recovery**: `acceptance-evidence.schema.json` 记录资源基线与清理结果，禁止保存凭据、终端敏感输出和包内容；目标机资源使用现有所有权台账。
- **PASS — Verification**: `quickstart.md` 覆盖共享组件、拓扑几何、检查器、菜单、中文化、axe、三视口、125% 显示比例、全量合同回归和目标机清理。
- **PASS — Image and secret hygiene**: 不新增镜像或秘密，验收复用合法现有模板；证据只保存布局结果、定位信息和脱敏截图。
- **PASS — Local-first delivery**: 五个里程碑都有本地质量门禁和聚焦提交，目标部署要求干净 commit、产物摘要、回滚候选及失败回流。
- **PASS — No exception required**: 未发现需要宪法豁免、数据库迁移、新特权适配器或 API/MCP 变更的设计。

## Complexity Tracking

无宪法违规。本功能复用既有前端依赖、样式体系、中文资源和验收框架，不引入新的运行时服务、持久化模型或外部依赖。
