# Tasks: 全页面中文化与明暗主题

**Input**: Design documents from `specs/007-ui-localization-theme/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: 本功能规范明确要求自动化组件、可访问性、真实浏览器和目标机验收，因此各用户故事均采用测试先行，并在实现后运行独立门禁。

**Organization**: 任务按用户故事组织，以便中文化、主题切换和可访问性可以分别实现、验收和提交；所有更改先在本地完成并形成聚焦提交，再部署到 `10.72.1.7`。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可与同阶段其他任务并行，且写入文件不重叠
- **[US1]**: 完整简体中文界面
- **[US2]**: 全局浅色/深色/跟随系统主题
- **[US3]**: 两种主题下的可读性与可访问性

---

## Phase 1: Setup（测试与审计基础）

**Purpose**: 为文案扫描、主题测试和浏览器验收建立可复用基础，不改变后端、SQLite、HTTP 或 MCP 合同。

- [X] T001 创建前端文案扫描测试辅助工具并定义用户可见文本提取边界于 `web/src/test/localizationTestUtils.ts`
- [X] T002 [P] 创建可控 `matchMedia`、`localStorage` 与主题根元素测试夹具于 `web/src/test/themeTestUtils.ts`
- [X] T003 [P] 扩展页面对象的主题选择、当前主题和中文可访问名称定位能力于 `tests/e2e/pages/BasePage.ts`
- [X] T004 将主题入口、中文界面流程和三视口覆盖项登记到 `tests/e2e/matrices/interaction-inventory.json`

---

## Phase 2: Foundational（所有用户故事的阻塞前置）

**Purpose**: 建立集中式文案、术语、状态展示和主题语义边界，防止后续组件继续散落字符串或固定颜色。

**⚠️ CRITICAL**: 本阶段完成前不得开始用户故事实现。

- [X] T005 [P] 定义类型化简体中文文案目录和技术原文保留边界于 `web/src/locales/zh-CN.ts`
- [X] T006 [P] 定义核心术语、允许保留缩写和禁止冲突译法于 `web/src/locales/terminology.ts`
- [X] T007 [P] 定义期望/实际状态、任务、错误、超时和下一步建议格式化规则于 `web/src/locales/statusMessages.ts`
- [X] T008 汇总并导出文案、术语和状态资源于 `web/src/locales/index.ts`
- [X] T009 创建精确 allowlist 的用户可见英文扫描门禁于 `scripts/check-ui-localization.sh`
- [X] T010 为文案目录完整性、术语唯一性、技术原文保留和状态消息回退编写失败测试于 `web/src/locales/locales.test.ts`
- [X] T011 为语义颜色变量命名、禁止新增主题专属固定色和主题根属性编写静态测试于 `web/src/styles/theme.test.ts`

**Checkpoint**: 文案与视觉语义具有单一来源，扫描和测试可阻止新增未分类英文及固定主题颜色。

---

## Phase 3: User Story 1 - 使用完整中文界面操作网络实验室（Priority: P1）🎯 MVP

**Goal**: 用户可只依赖简体中文完成实验室、拓扑、节点、任务、终端、抓包、Traffic Filter、模板和自动化操作，同时技术数据保持原文。

**Independent Test**: 从创建实验室开始，完成添加资源、启动节点、打开终端、创建链路、查看任务、抓包、Traffic Filter 和删除资源；产品自有文案均为中文，高风险操作明确对象与影响，设备输出和标识保持原文。

### Tests for User Story 1

- [X] T012 [P] [US1] 为共享加载态、空态、确认、高风险动作、状态和结构化错误的中文合同编写失败测试于 `web/src/components/common/LocalizationContract.test.ts`
- [X] T013 [P] [US1] 为导航、命令面板、实验室工具栏和操作抽屉的中文标签及可访问名称编写失败测试于 `web/src/components/shell/ShellLocalization.test.ts`
- [X] T014 [P] [US1] 为添加资源、拓扑菜单、端口选择、Inspector 和节点操作的中文流程编写失败测试于 `web/src/features/topology/TopologyLocalization.test.ts`
- [X] T015 [P] [US1] 为 Console、Capture、Wireshark、Traffic Filter、任务与诊断的中文状态和错误上下文编写失败测试于 `web/src/features/diagnostics/DiagnosticsLocalization.test.ts`
- [X] T016 [P] [US1] 为模板、镜像导入和自动化页面的中文加载态、空态、失败态及重试入口编写失败测试于 `web/src/views/PageLocalization.test.ts`
- [X] T017 [P] [US1] 创建完整中文关键流程浏览器测试并校验技术数据不被翻译于 `tests/e2e/frontend_localization.spec.ts`

### Implementation for User Story 1

- [X] T018 [US1] 使用集中资源中文化确认、加载、空态、状态徽标、资源标识和结构化错误组件于 `web/src/components/common/ConfirmationDialog.vue`, `web/src/components/common/LoadingState.vue`, `web/src/components/common/EmptyState.vue`, `web/src/components/common/StatusBadge.vue`, `web/src/components/common/ResourceIdentity.vue`, `web/src/components/common/StatePresentation.vue`, `web/src/components/common/StructuredProblem.vue`
- [X] T019 [US1] 中文化应用导航、命令面板、实验室外壳和全局操作抽屉于 `web/src/components/shell/CommandPalette.vue`, `web/src/components/shell/LaboratoryShell.vue`, `web/src/components/shell/OperationsDrawer.vue`
- [X] T020 [P] [US1] 中文化实验室切换、重命名、复制、导入导出和删除确认于 `web/src/features/laboratories/LaboratoryToolbar.vue`, `web/src/features/laboratories/LaboratoryTransferDialog.vue`
- [X] T021 [US1] 中文化资源添加抽屉、设备目录、端口选择、节点与链路右键菜单于 `web/src/features/topology/CreateTopologyResourceDrawer.vue`, `web/src/features/topology/DevicePalette.vue`, `web/src/features/topology/TopologyResourceCatalog.vue`, `web/src/features/topology/PortChooser.vue`, `web/src/features/topology/LinkContextMenu.vue`
- [X] T022 [US1] 中文化拓扑工作区、画布操作、Inspector、接口编辑和节点编辑状态于 `web/src/features/topology/TopologyWorkspace.vue`, `web/src/features/topology/TopologyCanvas.vue`, `web/src/features/topology/TopologyInspector.vue`, `web/src/features/topology/InterfaceEditor.vue`, `web/src/features/topology/NodeEditor.vue`
- [X] T023 [P] [US1] 中文化 QEMU、Docker、PC、Lightweight L2/L3 的设置、能力、接口、资源、端口映射和命令操作于 `web/src/features/nodes/NodeConfigurationPanel.vue`, `web/src/features/nodes/NodeCapabilityPanel.vue`, `web/src/features/nodes/NodeOperationsPanel.vue`, `web/src/features/nodes/InterfaceOperations.vue`, `web/src/features/nodes/NodeResourcesEditor.vue`, `web/src/features/nodes/PortMappingsPanel.vue`, `web/src/features/nodes/GuestCommandPanel.vue`, `web/src/features/nodes/LightweightNodeEditor.vue`, `web/src/features/nodes/LightweightPCConfigurationPanel.vue`, `web/src/features/nodes/LightweightSwitchConfigEditor.vue`, `web/src/features/nodes/LightweightSwitchConfigurationPanel.vue`, `web/src/features/nodes/RuijieConfigurationPanel.vue`
- [X] T024 [P] [US1] 中文化任务搜索、进度、取消、重试、终态和诊断建议于 `web/src/features/tasks/TaskCenter.vue`
- [X] T025 [US1] 中文化 Console、Capture、Wireshark 集成、Traffic Filter 和诊断工作区并为原始错误附加中文上下文于 `web/src/features/diagnostics/ConsoleWorkspace.vue`, `web/src/features/diagnostics/GlobalConsoleWorkspace.vue`, `web/src/features/diagnostics/CapturePanel.vue`, `web/src/features/diagnostics/GlobalCaptureWorkspace.vue`, `web/src/features/diagnostics/TrafficFilterPanel.vue`, `web/src/features/diagnostics/DiagnosticsPanel.vue`
- [X] T026 [P] [US1] 中文化模板目录、模板选择、镜像导入、模板页和自动化页于 `web/src/features/templates/TemplateCatalog.vue`, `web/src/features/templates/TemplatePicker.vue`, `web/src/features/templates/ImageImportDialog.vue`, `web/src/views/TemplatesView.vue`, `web/src/views/AutomationView.vue`
- [X] T027 [US1] 创建响应式工作区样式并修复中文长文本在工具栏、添加抽屉、Inspector 和底部工作区中的截断与操作不可达问题于 `web/src/styles/workspace.css`, `web/src/styles/index.css`, `web/src/components/shell/LaboratoryShell.vue`, `web/src/features/topology/CreateTopologyResourceDrawer.vue`, `web/src/features/topology/TopologyInspector.vue`
- [X] T028 [US1] 运行 `scripts/check-ui-localization.sh`、US1 Vitest 与 `tests/e2e/frontend_localization.spec.ts` 并将覆盖率和遗留豁免记录到 `specs/007-ui-localization-theme/validation.md`
- [X] T029 [US1] 创建“中文术语与全页面文案”聚焦里程碑提交，提交范围记录于 `specs/007-ui-localization-theme/validation.md`

**Checkpoint**: User Story 1 可独立演示，核心流程无需英文产品说明即可完成。

---

## Phase 4: User Story 2 - 在白天与黑夜主题间切换（Priority: P2）

**Goal**: 在任意主要页面使用“跟随系统 / 浅色 / 深色”入口即时切换主题，刷新后保持且不同客户端相互隔离，不改变资源或运行会话。

**Independent Test**: 在拓扑、模板和自动化页面切换三态主题，检查所有已打开浮层同步更新；刷新、切换实验室和双浏览器验证持久化与隔离，并确认 Console、Capture 和 Traffic Filter 会话未中断。

### Tests for User Story 2

- [X] T030 [P] [US2] 为首次系统跟随、深色回退、手动覆盖、存储恢复、存储失败和系统主题变化编写失败测试于 `web/src/composables/useThemePreference.test.ts`
- [X] T031 [P] [US2] 为中文主题入口的键盘选择、焦点恢复和当前状态提示编写失败测试于 `web/src/components/appearance/ThemeSwitcher.test.ts`
- [X] T032 [P] [US2] 为首屏挂载前主题解析和无错误主题闪烁编写失败测试于 `web/src/themeBootstrap.test.ts`
- [X] T033 [P] [US2] 为 ECharts 主题更新后数据、缩放、图例和选择状态保持编写失败测试于 `web/src/components/charts/EChart.test.ts`
- [X] T034 [P] [US2] 为拓扑主题更新后节点坐标、路由点、缩放、选择和拖动状态保持编写失败测试于 `web/src/features/topology/topologyThemeContinuity.test.ts`
- [X] T035 [P] [US2] 为刷新、路由切换、双浏览器隔离和运行会话连续性创建浏览器测试于 `tests/e2e/frontend_theme_continuity.spec.ts`

### Implementation for User Story 2

- [X] T036 [US2] 实现 `system | light | dark` 偏好、`light | dark` 解析、`netlab.appearance.v1` 持久化和存储失败回退于 `web/src/composables/useThemePreference.ts`
- [X] T037 [US2] 在 Vue 挂载前解析并应用根元素 `data-theme` 与 `color-scheme` 于 `web/src/themeBootstrap.ts`, `web/src/main.ts`, `web/index.html`
- [X] T038 [US2] 将单一深色变量重构为浅色/深色语义调色板并覆盖表面、文字、边框、焦点、状态、拓扑和图表变量于 `web/src/styles/theme.css`
- [X] T039 [P] [US2] 实现可键盘操作且带中文可访问名称的三态主题切换器于 `web/src/components/appearance/ThemeSwitcher.vue`, `web/src/components/appearance/index.ts`
- [X] T040 [US2] 将统一主题入口接入拓扑实验室工具栏、模板页和自动化页于 `web/src/features/laboratories/LaboratoryToolbar.vue`, `web/src/views/TemplatesView.vue`, `web/src/views/AutomationView.vue`
- [X] T041 [US2] 使 Shadcn 风格按钮、表单、选择器、菜单、Dialog、Sheet、Tabs、Table、Tooltip 和调整手柄仅使用语义主题变量于 `web/src/components/ui/`
- [X] T042 [US2] 让 ECharts、资源图表、抓包图表和任务图表响应解析主题且保持实例交互状态于 `web/src/components/charts/EChart.vue`, `web/src/components/charts/CaptureVolumeChart.vue`, `web/src/components/charts/TaskProgressChart.vue`, `web/src/features/analytics/ResourceCharts.vue`
- [X] T043 [US2] 让拓扑画布、节点、网络对象、端口、链路、箭头、粒子、选择框和分组框使用主题语义且保持几何状态于 `web/src/features/topology/TopologyCanvas.vue`, `web/src/features/topology/TrafficPathOverlay.vue`, `web/src/features/topology/topologyVisualSemantics.ts`, `web/src/features/topology/topologySymbols.ts`
- [X] T044 [US2] 仅主题化 Console 与 VNC 外围标签、工具栏、边框和状态区并保持客户机 ANSI/远程画面配色于 `web/src/features/diagnostics/ConsoleWorkspace.vue`, `web/src/features/diagnostics/GlobalConsoleWorkspace.vue`, `web/src/features/nodes/NodeOperationsPanel.vue`
- [X] T045 [US2] 运行 US2 Vitest、构建和 `tests/e2e/frontend_theme_continuity.spec.ts` 并记录 300ms 切换、隔离和会话连续性证据于 `specs/007-ui-localization-theme/validation.md`
- [X] T046 [US2] 创建“全局主题与复杂视图适配”聚焦里程碑提交，提交范围记录于 `specs/007-ui-localization-theme/validation.md`

**Checkpoint**: User Story 2 可独立验收，主题只影响当前浏览器显示且不会触发资源 mutation 或会话重建。

---

## Phase 5: User Story 3 - 在两种主题下保持可读性与可访问性（Priority: P3）

**Goal**: 两种主题和三种桌面视口下均有清晰焦点、足够对比度、非纯颜色状态表达、舒适动画和可完成的键盘流程。

**Independent Test**: 在浅色和深色主题下仅用键盘完成导航、主题切换、打开添加抽屉、提交表单、选择拓扑资源和关闭确认；axe 无阻塞问题，减少动态效果时流量方向仍可识别。

### Tests for User Story 3

- [X] T047 [P] [US3] 扩展共享组件 axe 测试以覆盖两种主题、焦点、禁用和高风险确认语义于 `web/src/components/common/InteractionStateMatrix.test.ts`
- [X] T048 [P] [US3] 扩展拓扑可访问性测试以覆盖非纯颜色状态、方向标识、选择和键盘操作于 `web/src/features/topology/TopologyCanvas.a11y.test.ts`
- [X] T049 [P] [US3] 为减少动态效果下的流量粒子、状态动画和主题过渡编写失败测试于 `web/src/features/topology/topologyVisualSemantics.test.ts`
- [X] T050 [P] [US3] 为两种主题、三视口、中文长文本和全键盘流程创建浏览器矩阵于 `tests/e2e/frontend_localization_theme_accessibility.spec.ts`

### Implementation for User Story 3

- [X] T051 [US3] 为交互组件统一可见焦点、禁用、选中、成功、运行中、警告和失败的图标/文字/线型语义于 `web/src/styles/theme.css`, `web/src/components/common/StatusBadge.vue`, `web/src/components/common/StatePresentation.vue`
- [X] T052 [US3] 为主题切换器、菜单、抽屉、Dialog、Tabs 和表单补齐中文可访问名称、焦点恢复与键盘顺序于 `web/src/components/appearance/ThemeSwitcher.vue`, `web/src/components/ui/`, `web/src/features/topology/CreateTopologyResourceDrawer.vue`
- [X] T053 [US3] 为拓扑链路方向、流量粒子、活动保留和选择状态增加颜色之外的箭头、形状及线型表达于 `web/src/features/topology/TrafficPathOverlay.vue`, `web/src/features/topology/topologyVisualSemantics.ts`
- [X] T054 [US3] 实现 `prefers-reduced-motion` 下的低动态主题过渡和流量效果，同时保留方向可识别性于 `web/src/styles/theme.css`, `web/src/features/topology/TrafficPathOverlay.vue`
- [X] T055 [US3] 修复 1024×768、1366×768 和 1920×1080 下中文导航、工具栏、抽屉、Inspector、任务和底部工作区的溢出与重叠于 `web/src/components/shell/LaboratoryShell.vue`, `web/src/features/laboratories/LaboratoryToolbar.vue`, `web/src/features/tasks/TaskCenter.vue`, `web/src/features/topology/TopologyInspector.vue`, `web/src/styles/workspace.css`
- [X] T056 [US3] 运行 US3 Vitest、axe 和三视口 Playwright 矩阵并记录对比度、键盘、焦点和减少动态效果证据于 `specs/007-ui-localization-theme/validation.md`
- [X] T057 [US3] 创建“可访问性与响应式验收”聚焦里程碑提交，提交范围记录于 `specs/007-ui-localization-theme/validation.md`

**Checkpoint**: 三个用户故事均可独立工作，且两种主题不会降低关键流程的可读性或可操作性。

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 完成全量回归、合同不变性、可复现候选、目标机验收、清理与回滚证据。

- [X] T058 [P] 更新中文化、主题使用和技术原文保留说明于 `docs/frontend-localization-theme.md`
- [X] T059 [P] 扩展前端验收 schema/夹具以记录主题偏好、解析主题、视口、中文覆盖率和会话连续性于 `tests/e2e/fixtures/acceptanceTypes.ts`, `tests/e2e/fixtures/evidenceReporter.ts`
- [X] T060 运行 `go test ./...`、`web` 的 format/lint/Vitest/build 和 Playwright list，并将全量结果记录到 `specs/007-ui-localization-theme/validation.md`
- [X] T061 对比实现前后 OpenAPI、MCP、事件、SQLite schema 和前端生成类型摘要，确认机器字段与持久化合同未变化并记录于 `specs/007-ui-localization-theme/validation.md`
- [X] T062 扫描跟踪文件中的秘密、凭据、抓包和专有镜像并记录清洁结果于 `specs/007-ui-localization-theme/validation.md`
- [X] T063 验证工作区清洁后创建最终验收提交并记录 commit SHA、候选 ID、contract digest、binary digest、构建时间和无迁移状态于 `specs/007-ui-localization-theme/validation.md`
- [X] T064 从 T063 的干净提交构建部署产物并使用正式安装流程部署到 `10.72.1.7`，部署记录写入 `specs/007-ui-localization-theme/validation.md`
- [ ] T065 在 `10.72.1.7` 创建专用验收 Lab，验证 QEMU、Docker、链路、Console、Capture、Traffic Filter 在三态主题切换和双浏览器不同主题下持续可用，证据写入 `specs/007-ui-localization-theme/validation.md`
- [X] T066 删除目标机专用验收 Lab 并验证无遗留 QEMU、Docker、netns、bridge、capture、Traffic Filter 或 helper 资源，结果写入 `specs/007-ui-localization-theme/validation.md`
- [X] T067 使用上一已记录候选执行回滚演练，验证 SPA、既有实验室与运行会话恢复后将结果归档到 `specs/007-ui-localization-theme/validation.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 Setup**: 无依赖，可立即开始。
- **Phase 2 Foundational**: 依赖 Phase 1，阻塞全部用户故事。
- **Phase 3 US1**: 依赖 Phase 2；提供中文界面 MVP。
- **Phase 4 US2**: 依赖 Phase 2；可与 US1 分工并行，但合并时必须保留 US1 文案资源调用。
- **Phase 5 US3**: 依赖 Phase 2，验收时依赖 US1 与 US2 的完整中文和两套主题实现。
- **Phase 6 Polish**: 依赖所有计划交付的用户故事和里程碑提交。

### User Story Dependency Graph

```text
Setup → Foundational ─┬→ US1 中文界面（MVP） ─┐
                     └→ US2 全局主题 ────────┼→ US3 可访问性/响应式 → Polish → Target
                                              ┘
```

### Within Each User Story

- 测试任务必须先完成并确认在缺少实现时失败。
- 集中资源/状态模型先于组件接入。
- 共享组件先于功能域页面。
- 本地组件与浏览器验收通过后才创建该故事的聚焦提交。
- 目标机发现问题时回到本地修复、补测试、提交并重新部署，不直接修改目标源码。

### Parallel Opportunities

- Setup 的 T002、T003 可并行。
- Foundational 的 T005、T006、T007 可并行，之后由 T008 汇总。
- US1 的测试 T012–T017 可按文件并行；实现 T020、T023、T024、T026 可在共享基础组件完成后并行。
- US2 的测试 T030–T035 可并行；T039 可与 T038 并行，T042–T044 可在主题状态与变量完成后按功能域并行。
- US3 的测试 T047–T050 可并行；T053 和 T055 写入范围不重叠时可并行。
- Polish 的文档 T058 与验收夹具 T059 可并行，其余候选、部署、清理和回滚任务必须顺序执行。

---

## Parallel Example: User Story 1

```text
T012 共享组件中文合同测试
T013 Shell 与导航中文测试
T014 拓扑与节点中文测试
T015 诊断工作区中文测试
T016 模板与自动化中文测试
T017 完整中文浏览器流程
```

## Parallel Example: User Story 2

```text
T030 主题状态单元测试
T031 主题切换器交互测试
T032 首屏主题测试
T033 ECharts 连续性测试
T034 拓扑连续性测试
T035 双客户端与会话浏览器测试
```

## Parallel Example: User Story 3

```text
T047 共享状态与 axe 测试
T048 拓扑键盘与语义测试
T049 减少动态效果测试
T050 双主题三视口浏览器矩阵
```

---

## Implementation Strategy

### MVP First（User Story 1）

1. 完成 Phase 1 和 Phase 2。
2. 完成 Phase 3 的中文界面测试与实现。
3. 运行扫描、Vitest 和中文关键流程 Playwright。
4. 创建中文化里程碑提交并停止进行独立验收。

### Incremental Delivery

1. **中文化基础**: 集中资源、术语、错误和扫描门禁。
2. **中文界面 MVP**: 全部主要流程可仅依赖中文完成。
3. **全局主题**: 三态主题、首屏应用、持久化和客户端隔离。
4. **复杂视图与可访问性**: 拓扑、图表、Console/VNC 外壳、三视口和减少动态效果。
5. **候选与目标机**: 从干净提交构建、部署、真实运行会话验收、清理和回滚。

### Delivery Rules

- 每个里程碑先本地测试，再创建聚焦 Git 提交。
- 仅从已记录的干净提交构建候选产物。
- 不新增后端状态、数据库迁移或主题同步 API。
- 不把用户数据、设备输出、命令、地址、接口、镜像、厂商、协议、错误代码和资源 ID 翻译或写入文案目录。
- 不在源码、测试夹具或验收证据中保存凭据、专有镜像或抓包内容。

---

## Phase 7: Convergence

- [ ] T068 将 `web/src/components/`, `web/src/features/`, `web/src/views/` 中的产品自有英文文案直接迁移到 `web/src/locales/zh-CN.ts` 和领域化中文资源，移除对 `web/src/composables/useUiLocalization.ts` 挂载后文本改写的依赖，并将 `scripts/check-ui-localization.sh` 升级为覆盖文本节点、属性、运行时消息和精确技术术语豁免的全量门禁 per FR-001/FR-002/FR-024/SC-001 and plan: centralized text (contradicts)
- [ ] T069 将 `web/src/features/topology/TopologyCanvas.vue`, `web/src/features/topology/TrafficPathOverlay.vue`, `web/src/features/topology/topologyVisualSemantics.ts`, `web/src/features/topology/topologySymbols.ts` 中的固定颜色迁移到浅色/深色拓扑语义调色板，并增加节点、端口、链路、箭头、粒子、选择及流量状态双主题视觉测试 per FR-013/FR-015 and US3/AC2-3 (partial)
- [ ] T070 将 `web/src/features/analytics/ResourceCharts.vue`, `web/src/components/charts/CaptureVolumeChart.vue`, `web/src/components/charts/TaskProgressChart.vue` 的固定颜色迁移到图表语义调色板，修正 `web/src/components/charts/EChart.vue` 的主题配置合并以保留调用方 title/legend/tooltip/dataZoom/selection，并扩展 `web/src/components/charts/EChart.test.ts` per FR-016 and US2/AC2 and US3/AC3 (partial)
- [ ] T071 扩展 `tests/e2e/frontend_localization.spec.ts`, `tests/e2e/frontend_theme_continuity.spec.ts`, `tests/e2e/frontend_localization_theme_accessibility.spec.ts`，在两种主题和三视口下执行 axe、全键盘、中文长文本、资源添加、链路、任务、Console、Capture、Traffic Filter 及主题切换前后权威状态/会话连续性断言 per FR-024/FR-025 and SC-003/SC-004/SC-005 (partial)
- [ ] T072 修复 `tests/e2e/pages/TemplatePage.ts` 与相关 FormField/Select 稳定定位合同，随后在 `10.72.1.7` 创建并清理专用验收 Lab，完成 QEMU、Docker、链路、Console、Capture、Traffic Filter、三态主题和双浏览器隔离联合验收，并将证据写入 `specs/007-ui-localization-theme/validation.md` per FR-026 and T065 (missing)
- [X] T073 将 `web/src/components/appearance/ThemeSwitcher.vue` 作为共享控件接入 `web/src/features/laboratories/LaboratoryToolbar.vue`, `web/src/views/TemplatesView.vue`, `web/src/views/AutomationView.vue` 的正常布局，移除 `web/src/styles/workspace.css` 中可能遮挡操作的固定悬浮入口，并增加 1024×768 中文长文本可达性测试 per FR-007/FR-019 and plan: page entries (partial)
- [X] T074 使用安全的 `globalThis.matchMedia` 能力检测重构 `web/index.html` 和 `web/src/themeBootstrap.ts` 的首屏主题解析，并在 `web/src/themeBootstrap.test.ts` 增加无 matchMedia、无 localStorage、无明确系统偏好时深色回退且无挂载阻塞的测试 per FR-008/FR-021 (partial)
