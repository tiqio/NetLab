# Implementation Plan: 拓扑添加抽屉

**Branch**: `[006-topology-add-drawer]` | **Date**: 2026-08-05 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/006-topology-add-drawer/spec.md`

## Summary

将当前居中的 `CreateTopologyResourceDialog` 替换为工作区最右侧的创建抽屉。方案保持现有节点和网络对象创建接口、模板与镜像兼容性检查、字段默认值以及拓扑刷新逻辑不变，同时增强共享 Sheet 基础组件，使其支持完整高度、独立滚动、固定头尾、脏数据关闭保护、键盘焦点约束和关闭后焦点恢复。资源目录将从 `DevicePalette` 中抽离为可复用视图，使用户既可从左侧设备栏直接预选资源，也可在右侧抽屉内搜索并切换资源类型。

## Technical Context

**Language/Version**: TypeScript 5.8.3、Vue 3.5.18；后端 Go 1.26.x 合同保持不变

**Primary Dependencies**: Vite 7.0.6、Pinia 3.0.3、现有 Shadcn Vue 风格组件、Reka UI 2.10.1、Lucide Vue、现有类型化 HTTP 客户端

**Storage**: 不新增持久化；抽屉会话、草稿、滚动位置和焦点快照仅保存在当前浏览器内存，成功资源继续由现有服务端和 SQLite 状态管理

**Testing**: Vitest、Vue Test Utils、Testing Library DOM、axe-core、Playwright、现有前端契约/交互覆盖矩阵及目标机浏览器验收

**Target Platform**: 现代桌面浏览器，最低支持视口 1024×768；最终 SPA 由单机 NetLab 服务部署在 `10.72.1.7`

**Project Type**: Vue 单页应用与既有 Go HTTP 服务组成的 Web 应用；本功能主要修改前端

**Performance Goals**: 抽屉开关和类型切换不引起可感知画布重排；长表单滚动保持流畅；提交后 3 秒内通过权威刷新或事件使新资源出现在拓扑；禁止重复提交

**Constraints**: 不新增前端依赖；不改变现有创建 API/MCP 合同；不持久化未提交草稿；不改变画布视口、节点坐标或其他客户端布局；必须兼容现有左侧设备栏、右侧 Inspector 和底部 Operations 区域

**Scale/Scope**: 7 类图形化资源入口、现有全部模板与镜像版本、最长 QEMU/Docker 配置表单、最多 64 个声明接口；覆盖 1024×768、1366×768 和 1920×1080 视口

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Pre-Design Gate

- **Shared state — PASS**: 抽屉状态和草稿是浏览器本地瞬时状态；节点、网络对象、接口、位置、任务和实际状态仍以服务端实验室快照及有序事件为准。并发客户端不会共享抽屉状态，创建冲突继续使用既有服务端校验。
- **Control parity — PASS**: 抽屉只调用现有 `createNode`、`createNetworkObject`、模板和镜像查询；HTTP 与 MCP 可继续创建相同资源，最终结果通过相同权威资源模型观察。无新增 UI-only 资源生命周期。
- **Runtime scope — PASS**: 不改变 QEMU、Docker、bridge、NAT 或 netns 运行时，不引入新设备类型或集群能力。
- **Live operations — PASS**: 不影响运行节点、实时接线、Console、Capture 或 Traffic Filter；新资源创建后仍由现有工作区和 Inspector 提供这些操作。
- **Safety and recovery — PASS**: 复用现有幂等、配额、模板兼容性、镜像许可和结构化错误语义；客户端提交锁阻止重复点击，失败保留草稿且不创建幽灵图形。
- **Verification — PASS**: 计划包含共享 Sheet 单测、抽屉组件测试、API 负载测试、工作区集成测试、键盘/可访问性测试、真实浏览器视口测试和 `10.72.1.7` 代表性资源验收。
- **Image and secret hygiene — PASS**: 不新增镜像或凭据文件；镜像继续通过现有可用性、兼容性和许可状态过滤。测试使用现有合法模板或无秘密夹具。
- **Local-first delivery — PASS**: 分三个本地里程碑测试并提交；仅从干净提交构建带 SHA 和摘要的产物部署到目标机，失败返回本地修复并重新提交。

### Post-Design Re-check

- **PASS**: `research.md`、`data-model.md` 和 contracts 保持抽屉为本地会话、服务端资源为权威状态，没有新增数据库表、运行时资源或后端旁路。
- **PASS**: 设计明确关闭、提交、错误、冲突和刷新状态；所有创建动作仍映射到现有类型化接口，成功后从权威快照收敛。
- **PASS**: 快速验收覆盖本地单元/集成、真实浏览器、目标机部署、失败恢复和删除清理；没有需要例外批准的宪法违规。

## Project Structure

### Documentation (this feature)

```text
specs/006-topology-add-drawer/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── backend-parity.md
│   └── ui-contract.md
└── tasks.md
```

### Source Code (repository root)

```text
web/src/
├── components/ui/sheet/
│   ├── Sheet.vue
│   ├── Sheet.test.ts
│   └── index.ts
├── features/topology/
│   ├── CreateTopologyResourceDrawer.vue
│   ├── CreateTopologyResourceDrawer.test.ts
│   ├── TopologyResourceCatalog.vue
│   ├── TopologyResourceCatalog.test.ts
│   ├── DevicePalette.vue
│   ├── TopologyWorkspace.vue
│   ├── TopologyWorkspace.test.ts
│   ├── topologyResourceDraft.ts
│   └── topologyResourceDraft.test.ts
└── components/shell/
    ├── LaboratoryShell.vue
    └── LaboratoryShell.test.ts

tests/e2e/
├── journeys/topologyAddDrawer.spec.ts
├── matrices/formAndDialogMatrix.spec.ts
├── frontend_responsive_keyboard.spec.ts
└── pages/TopologyPage.ts
```

**Structure Decision**: 在现有 Vue SPA 内完成聚焦前端改造。共享交互能力放在 `components/ui/sheet`，资源目录和创建状态放在 `features/topology`，Workspace 仅负责打开来源、成功后的权威刷新、选择新资源以及与 Inspector 的协调。后端和数据库目录不修改，除非实现时发现现有接口合同缺陷并另行规格化。

## Implementation Strategy

### Milestone 1 — Accessible Sheet foundation

- 扩展 `Sheet.vue`，保持现有左侧、右侧和底部 Sheet 默认行为兼容。
- 增加描述、可配置宽度、固定 header/footer、独立 body 滚动、关闭请求拦截、Escape、遮罩点击、焦点约束和焦点恢复。
- 将放弃确认作为受控关闭流程，不允许子组件直接把 `modelValue` 写成 false 绕过脏数据保护。
- 为现有 LaboratoryShell 紧凑模式补回归测试，确保 Devices、Inspector 和 Operations Sheet 不退化。
- 本地门禁：相关 Vitest、类型检查、lint、格式检查；提交独立里程碑。

### Milestone 2 — Resource drawer and draft model

- 从 `DevicePalette.vue` 抽离 `TopologyResourceCatalog.vue`，集中负责模板查询、搜索、分类、不可用状态和 `PaletteSelection` 输出。
- 建立 `topologyResourceDraft.ts`，集中处理初始默认值、草稿签名、脏状态、类型切换、字段验证和创建负载构造，避免这些规则依赖抽屉 DOM。
- 将 `CreateTopologyResourceDialog.vue` 替换为 `CreateTopologyResourceDrawer.vue`；支持 `selecting → editing → submitting → failed/succeeded` 状态，并保留现有 QEMU、Docker、cloud-init、PC、Bridge、NAT、L2/L3 配置能力。
- 服务端字段错误映射到具体控件；提交失败保留草稿；成功仅发出权威创建结果并请求关闭。
- 本地门禁：目录、草稿和抽屉组件测试，现有创建请求测试全部迁移并保持覆盖；提交独立里程碑。

### Milestone 3 — Workspace integration and acceptance

- `TopologyWorkspace.vue` 统一管理抽屉打开来源：工具栏添加入口可从选择步骤打开，左侧 DevicePalette 可带预选资源直接进入编辑步骤。
- 打开抽屉时记录 Inspector 和焦点上下文；关闭后恢复，且不修改画布 viewport、节点位置或底部 Operations 状态。
- 创建成功后复用现有 `created()` 权威刷新与 placement 流程，选中新资源并确保无需整页刷新。
- 更新 E2E 页面对象和交互矩阵，覆盖长表单独立滚动、脏数据确认、重复提交、结构化失败、三种视口、键盘操作及真实后端创建。
- 本地门禁：完整前端测试、构建、相关后端合同测试与本地 E2E；提交独立里程碑。

### Deployment and target validation

- 从第三个干净里程碑提交构建候选产物，记录 commit SHA、candidate ID、contract digest、binary digest 和构建时间。
- 使用正式安装脚本部署到 `10.72.1.7`，不在目标机修改源文件。
- 新建专用验收 Lab，分别创建 Ubuntu QEMU、BusyBox Docker、PC 和 Lightweight L2 或 NAT Bridge；验证抽屉滚动、成功选中、事件刷新、错误恢复和删除清理。
- 在 1024×768 与 1920×1080 浏览器视口重复关键流程，并验证另一个浏览器只看到成功资源，不看到当前抽屉草稿。
- 失败时使用安装器回滚到前一候选产物，验证 SPA、已有 Lab 和创建接口恢复，再回到本地修复。

## Complexity Tracking

无宪法违规或超出单机产品边界的复杂度。本功能不新增后端状态、数据库迁移、运行时适配器或外部依赖。
