# Tasks: 全页面视觉与中文化治理

**Input**: Design documents from `/specs/008-ui-overlap-remediation/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: 本功能明确要求完整的视觉、交互、中文化、键盘、对比度和目标机验收，因此每个用户故事均先补失败测试，再实施修复。

**Organization**: 任务按用户故事组织，使拓扑、检查器、中文化、响应式和持续回归可以独立实现、测试和提交。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可在不同文件上并行执行，且不依赖同阶段未完成任务
- **[Story]**: 对应 `spec.md` 中的用户故事
- 每个任务均包含明确文件路径

## Phase 1: Setup（审计与执行基线）

**Purpose**: 固定兼容性约束、审计范围、测试数据和证据合同，避免实现过程中遗漏页面或破坏历史行为。

- [X] T001 检查拓扑、检查器、主题和中文化相关历史提交及 `git blame`，将兼容性约束补充到 `specs/008-ui-overlap-remediation/research.md`
- [X] T002 建立覆盖 3 个路由、共享组件和全部功能域的初始视觉/中文化审计台账到 `specs/008-ui-overlap-remediation/validation/audit-inventory.md`
- [X] T003 [P] 将双主题、三视口、100%/125%、空/正常/高密度状态编码为场景清单 `tests/e2e/matrices/ui-overlap-scenarios.json`
- [X] T004 [P] 为 VisualAuditScenario、LayoutRegion、VisualFinding、LocalizationAuditItem 和 AcceptanceEvidence 定义测试类型 `tests/e2e/fixtures/visualAuditTypes.ts`
- [X] T005 [P] 添加验收证据 Schema 严格编译和有效/无效样例测试 `tests/e2e/fixtures/visualAuditSchema.test.ts`
- [X] T006 在 `specs/008-ui-overlap-remediation/validation/milestones.md` 建立里程碑提交、质量门禁、候选摘要和目标机结果记录模板

---

## Phase 2: Foundational（共享阻断基础）

**Purpose**: 建立所有用户故事共同依赖的布局语义、碰撞断言、证据输出、文案扫描和共享组件行为。

**⚠️ CRITICAL**: 本阶段完成前不得开始用户故事实现。

- [X] T007 [P] 为文字层级、状态、焦点、表面、边框、图表和拓扑补齐跨主题语义变量 `web/src/styles/theme.css`
- [X] T008 [P] 为明确布局区、局部滚动、换行、截断、操作区和无页面横向溢出补充共享规则 `web/src/styles/workspace.css`
- [X] T009 [P] 实现边界框相交、命中区域冲突、视口裁切和页面溢出断言 `tests/e2e/fixtures/layoutAssertions.ts`
- [X] T010 [P] 为布局断言增加容差、允许覆盖和失败定位单测 `tests/e2e/fixtures/layoutAssertions.test.ts`
- [X] T011 [P] 扩展验收类型和运行摘要以承载视觉、中文化、截图与清理结果 `tests/e2e/fixtures/acceptanceTypes.ts`
- [X] T012 [P] 扩展证据写入与 Schema 校验流程 `tests/e2e/fixtures/acceptanceFixture.ts`
- [X] T013 [P] 强化截图、日志和报告的凭据/终端/抓包脱敏规则 `tests/e2e/fixtures/artifactPolicy.ts`
- [X] T014 [P] 扩展英文扫描覆盖模板文本、属性、动态消息和无障碍名称 `scripts/check-ui-localization.mjs`
- [X] T015 先补共享组件的布局、焦点、禁用/危险状态和长文本失败测试 `web/src/components/common/InteractionStateMatrix.test.ts`
- [X] T016 修复共享空状态、资源身份、状态和错误展示的布局与可读性 `web/src/components/common/EmptyState.vue`、`web/src/components/common/ResourceIdentity.vue`、`web/src/components/common/StatePresentation.vue`、`web/src/components/common/StatusBadge.vue`、`web/src/components/common/StructuredProblem.vue`
- [X] T017 修复基础控件的命中区、碰撞避让、焦点与主题状态 `web/src/components/ui/button/Button.vue`、`web/src/components/ui/dialog/Dialog.vue`、`web/src/components/ui/dropdown-menu/DropdownMenu.vue`、`web/src/components/ui/form/FormField.vue`、`web/src/components/ui/select/Select.vue`、`web/src/components/ui/sheet/Sheet.vue`、`web/src/components/ui/tabs/Tabs.vue`、`web/src/components/ui/tooltip/Tooltip.vue`
- [X] T018 运行共享基础门禁并将结果记录到 `specs/008-ui-overlap-remediation/validation/milestones.md`
- [X] T019 创建共享基础里程碑的聚焦 Git 提交，并把提交 SHA 写入 `specs/008-ui-overlap-remediation/validation/milestones.md`

**Checkpoint**: 共享布局、文案和验收基础可被所有用户故事复用。

---

## Phase 3: User Story 1 - 清晰操作拓扑节点与链路（Priority: P1）🎯 MVP

**Goal**: 让节点图形、名称、状态、端口、连接入口、链路、方向和流量效果在高密度拓扑中保持稳定、可辨和可点击。

**Independent Test**: 创建含 1、2、4、8、16 接口节点及多条平行链路的拓扑，执行拖动、缩放、平移、适配、重置和 Traffic Filter，确认端口坐标稳定、命中区不相交且只有活动链路显示流量。

### Tests for User Story 1

- [X] T020 [P] [US1] 为 1、2、4、8、16 接口的端口轨道、标签位置和独立命中区增加失败测试 `web/src/features/topology/topologyGeometry.test.ts`
- [X] T021 [P] [US1] 为 hover、选择、拖动、缩放和重置后的节点内部坐标稳定性增加失败测试 `web/src/features/topology/TopologyCanvas.test.ts`
- [X] T022 [P] [US1] 为平行链路路径、端点显示名和未活动链路不继承高亮增加失败测试 `web/src/features/topology/linkPresentation.test.ts`
- [X] T023 [P] [US1] 为方向粒子、停止流量后的衰减和减少动态效果增加失败测试 `web/src/features/topology/TrafficPathOverlay.test.ts`
- [X] T024 [P] [US1] 增加高密度拓扑拖动、端口点击、平行链路和流量切换浏览器场景 `tests/e2e/journeys/topologyVisualRecognition.spec.ts`

### Implementation for User Story 1

- [X] T025 [US1] 在 `web/src/features/topology/topologyGeometry.ts` 实现确定性端口轨道、节点布局区域和缩放标签优先级
- [X] T026 [US1] 在 `web/src/features/topology/TopologyCanvas.vue` 分离身份、状态、图形、端口、标签和连接入口，并阻止装饰层截获点击
- [X] T027 [US1] 在 `web/src/features/topology/linkPresentation.ts` 实现平行链路偏移、可读端点名称和活动链路精确匹配
- [X] T028 [US1] 在 `web/src/features/topology/TrafficPathOverlay.vue` 修复方向粒子图层、停止衰减和非闪烁展示
- [X] T029 [US1] 在 `web/src/features/topology/TopologyCanvas.vue` 和 `web/src/styles/workspace.css` 修复节点拖动漂移、画布收缩和缩放后命中区失配
- [X] T030 [US1] 运行 US1 聚焦测试与三视口浏览器验收，并把结果记录到 `specs/008-ui-overlap-remediation/validation/milestones.md`
- [X] T031 [US1] 创建拓扑治理里程碑的聚焦 Git 提交，并把提交 SHA 写入 `specs/008-ui-overlap-remediation/validation/milestones.md`

**Checkpoint**: US1 可独立演示为稳定、清晰、可连线且流量方向正确的拓扑画布。

---

## Phase 4: User Story 2 - 清晰阅读检查器与上下文菜单（Priority: P1）

**Goal**: 让检查器资源图、指标、状态、按钮、菜单和底部工作区在窄宽与复杂状态下保持清晰、可滚动和可操作。

**Independent Test**: 在双主题和最小视口依次打开所有资源检查器与视口四角的右键菜单，再切换任务、终端、抓包和流量过滤，确认无重叠、裁切、低对比度和不可到达内容。

### Tests for User Story 2

- [X] T032 [P] [US2] 为检查器长名称、失败状态、资源图和多按钮窄宽布局增加失败测试 `web/src/features/topology/TopologyInspector.test.ts`
- [X] T033 [P] [US2] 为资源图容器尺寸、指标区和主题切换重排增加失败测试 `web/src/features/analytics/ResourceCharts.test.ts`
- [X] T034 [P] [US2] 为 ECharts 零尺寸保护、ResizeObserver 重排和主题连续性增加失败测试 `web/src/components/charts/EChart.test.ts`
- [X] T035 [P] [US2] 为链路菜单视口碰撞、禁用/危险状态和键盘焦点增加失败测试 `web/src/features/topology/LinkContextMenu.test.ts`
- [X] T036 [P] [US2] 为任务/终端/抓包/流量过滤标签持久、内容滚动和切换连续性增加失败测试 `web/src/components/shell/OperationsDrawer.test.ts`
- [X] T037 [P] [US2] 增加检查器、菜单边缘和底部工作区浏览器矩阵 `tests/e2e/frontend_diagnostics.spec.ts`

### Implementation for User Story 2

- [X] T038 [US2] 在 `web/src/features/topology/TopologyInspector.vue` 分离标题、身份、状态、资源图、指标、表单和操作区域并加入局部滚动
- [X] T039 [US2] 在 `web/src/features/analytics/ResourceCharts.vue` 和 `web/src/components/charts/EChart.vue` 实现容器驱动尺寸、图例避让和主题重排
- [X] T040 [US2] 在 `web/src/features/topology/LinkContextMenu.vue` 与 `web/src/features/topology/TopologyWorkspace.vue` 统一节点/链路/网络对象/画布菜单定位和状态呈现
- [X] T041 [US2] 在 `web/src/components/shell/OperationsDrawer.vue` 和 `web/src/components/shell/LaboratoryShell.vue` 修复标签、固定区、滚动区和拓扑画布边界
- [X] T042 [US2] 在 `web/src/features/tasks/TaskCenter.vue`、`web/src/features/diagnostics/GlobalConsoleWorkspace.vue`、`web/src/features/diagnostics/GlobalCaptureWorkspace.vue` 和 `web/src/features/diagnostics/TrafficFilterPanel.vue` 修复会话列表、工具栏和空/长历史布局
- [X] T043 [US2] 运行 US2 聚焦测试、axe 和菜单边缘验收，并把结果记录到 `specs/008-ui-overlap-remediation/validation/milestones.md`
- [X] T044 [US2] 创建检查器与工作区里程碑的聚焦 Git 提交，并把提交 SHA 写入 `specs/008-ui-overlap-remediation/validation/milestones.md`

**Checkpoint**: US2 可独立验收所有检查器、菜单和底部工作区的可读性与可达性。

---

## Phase 5: User Story 3 - 使用完整一致的中文界面（Priority: P1）

**Goal**: 清除非白名单英文产品文案，统一核心术语，并为错误提供中文摘要、操作建议和原始详情。

**Independent Test**: 遍历实验室、模板、自动化、拓扑、检查器、菜单、任务、终端、抓包、流量过滤和诊断的默认/空/失败状态，扫描结果中无非白名单英文，代表性流程可仅依据中文完成。

### Tests for User Story 3

- [X] T045 [P] [US3] 为终端空状态、Serial/Reconnect/Close 和无障碍名称增加中文断言 `web/src/features/diagnostics/GlobalConsoleWorkspace.test.ts`
- [X] T046 [P] [US3] 为 Wireshark 无响应、来源不匹配、程序未找到和启动失败增加中文错误分层测试 `web/src/features/diagnostics/CapturePanel.test.ts`
- [X] T047 [P] [US3] 为任务、模板、镜像导入、接口选择和链路菜单的旧英文文案增加失败测试 `web/src/locales/locales.test.ts`
- [X] T048 [P] [US3] 为页面可见文本、属性、动态消息和无障碍名称增加扫描回归测试 `scripts/check-ui-localization.test.mjs`
- [X] T049 [P] [US3] 增加三路由及终端/抓包/流量过滤中文化浏览器旅程 `tests/e2e/frontend_localization.spec.ts`

### Implementation for User Story 3

- [X] T050 [US3] 在 `web/src/locales/terminology.ts` 和 `web/src/locales/zh-CN.ts` 固化终端、抓包、流量过滤、任务、检查器、诊断及通用操作术语
- [X] T051 [P] [US3] 中文化底部标签、终端和诊断的标签、空状态、按钮及辅助文本 `web/src/components/shell/OperationsDrawer.vue`、`web/src/features/diagnostics/ConsoleWorkspace.vue`、`web/src/features/diagnostics/GlobalConsoleWorkspace.vue`、`web/src/features/diagnostics/DiagnosticsPanel.vue`
- [X] T052 [P] [US3] 中文化抓包与流量过滤的状态、错误、图表及操作文案 `web/src/features/diagnostics/CapturePanel.vue`、`web/src/features/diagnostics/GlobalCaptureWorkspace.vue`、`web/src/features/diagnostics/TrafficFilterPanel.vue`、`web/src/features/diagnostics/TrafficFilterChart.vue`
- [X] T053 [P] [US3] 中文化模板、目录、镜像导入和实验室传输的通用产品文案 `web/src/features/templates/TemplatePicker.vue`、`web/src/features/templates/TemplateCatalog.vue`、`web/src/features/templates/ImageImportDialog.vue`、`web/src/features/laboratories/LaboratoryTransferDialog.vue`
- [X] T054 [P] [US3] 中文化接口编辑、接口选择、添加抽屉、链路菜单和拓扑动态提示 `web/src/features/topology/InterfaceEditor.vue`、`web/src/features/topology/PortChooser.vue`、`web/src/features/topology/CreateTopologyResourceDrawer.vue`、`web/src/features/topology/LinkContextMenu.vue`、`web/src/features/topology/TopologyCanvas.vue`
- [X] T055 [P] [US3] 中文化 TaskCenter 分页、状态和空结果文案 `web/src/features/tasks/TaskCenter.vue`
- [X] T056 [US3] 在 `web/src/components/common/StructuredProblem.vue` 和 `web/src/locales/statusMessages.ts` 实现中文摘要、处理建议和可展开原始详情
- [X] T057 [US3] 更新终端、抓包、任务、拓扑和流量过滤页面对象以使用中文可访问名称 `tests/e2e/pages/ConsolePage.ts`、`tests/e2e/pages/CapturePage.ts`、`tests/e2e/pages/TaskCenterPage.ts`、`tests/e2e/pages/TopologyPage.ts`、`tests/e2e/pages/TrafficFilterPage.ts`
- [X] T058 [US3] 运行中文扫描、US3 组件测试和浏览器验收，并把结果记录到 `specs/008-ui-overlap-remediation/validation/milestones.md`
- [X] T059 [US3] 创建全页面中文化里程碑的聚焦 Git 提交，并把提交 SHA 写入 `specs/008-ui-overlap-remediation/validation/milestones.md`

**Checkpoint**: US3 可独立证明产品文案全中文、技术名词准确保留且错误可操作。

---

## Phase 6: User Story 4 - 在不同窗口和主题下保持布局稳定（Priority: P2）

**Goal**: 在双主题、三视口、125% 显示比例和重复面板切换下保持无页面溢出、无主题残色和稳定工作区。

**Independent Test**: 在 1024×768、1366×768、1920×1080 的浅色/深色组合中遍历三路由，并以 125% 完成六个关键流程；重复主题、面板、路由和刷新 20 次无重叠或 Reset 恢复需求。

### Tests for User Story 4

- [X] T060 [P] [US4] 扩展双主题三视口的页面溢出、焦点和 axe 矩阵 `tests/e2e/frontend_localization_theme_accessibility.spec.ts`
- [X] T061 [P] [US4] 增加 20 次主题/面板/路由/刷新循环的布局稳定测试 `tests/e2e/frontend_theme_continuity.spec.ts`
- [X] T062 [P] [US4] 增加关键创建、连接、检查器、菜单、终端和抓包的 125% 验收场景 `tests/e2e/matrices/viewportInputMatrix.spec.ts`
- [X] T063 [P] [US4] 为检查器和操作抽屉尺寸边界增加失败测试 `web/src/components/shell/laboratoryShellSizing.test.ts`

### Implementation for User Story 4

- [X] T064 [US4] 在 `web/src/components/shell/laboratoryShellSizing.ts` 和 `web/src/components/shell/LaboratoryShell.vue` 约束面板尺寸、窄视口 Sheet 切换和页面级溢出
- [X] T065 [P] [US4] 修复实验室工具栏、设备添加抽屉和长表单在 1024×768/125% 下的换行与滚动 `web/src/features/laboratories/LaboratoryToolbar.vue` 和 `web/src/features/topology/CreateTopologyResourceDrawer.vue`
- [X] T066 [P] [US4] 修复模板页和自动化页在中文长文本及主题切换下的网格与局部滚动 `web/src/views/TemplatesView.vue` 和 `web/src/views/AutomationView.vue`
- [X] T067 [US4] 修复图表、菜单、下拉框和拓扑语义色在主题切换后的残留颜色 `web/src/styles/theme.css`
- [X] T068 [US4] 运行 US4 三视口、双主题、125% 和 20 次循环验收，并把结果记录到 `specs/008-ui-overlap-remediation/validation/milestones.md`
- [ ] T069 [US4] 创建响应式与主题稳定里程碑的聚焦 Git 提交，并把提交 SHA 写入 `specs/008-ui-overlap-remediation/validation/milestones.md`

**Checkpoint**: US4 可独立证明所有主页面在支持矩阵中稳定、可读、可滚动和可操作。

---

## Phase 7: User Story 5 - 持续发现视觉与文案回归（Priority: P3）

**Goal**: 将场景清单、碰撞检查、中文扫描、证据 Schema 和目标机结果接入持续验收，使后续改动无法悄悄重新引入问题。

**Independent Test**: 对审计矩阵执行自动验收，故意注入重叠和英文文案时测试失败；恢复后生成符合 Schema 的通过证据，并在目标机清理全部验收资源。

### Tests for User Story 5

- [ ] T070 [P] [US5] 为审计场景完整性、重复 ID、矩阵覆盖和白名单约束增加测试 `tests/e2e/matrices/uiOverlapScenarioInventory.test.ts`
- [ ] T071 [P] [US5] 增加通用页面碰撞、裁切、命中区、焦点和非白名单英文检测规格 `tests/e2e/matrices/uiVisualAudit.spec.ts`
- [ ] T072 [P] [US5] 为验收证据汇总、Schema 校验和失败发现关联增加测试 `tests/e2e/fixtures/visualAuditEvidence.test.ts`
- [ ] T073 [P] [US5] 扩展证据脱敏测试，拒绝密码、终端敏感输出和数据包内容 `tests/e2e/fixtures/artifactPolicy.test.ts`

### Implementation for User Story 5

- [ ] T074 [US5] 在 `tests/e2e/fixtures/visualAudit.ts` 实现场景装载、布局采样、允许覆盖、严重度分类和证据生成
- [ ] T075 [US5] 将视觉审计摘要接入现有 acceptance run summary `tests/e2e/fixtures/acceptanceFixture.ts`
- [ ] T076 [US5] 将中文化扫描、视觉矩阵和证据 Schema 校验接入本地验收入口 `acceptance/frontend-acceptance.sh`
- [ ] T077 [US5] 扩展前端产物合规检查以验证脱敏视觉证据和禁止敏感内容 `scripts/check-frontend-artifacts.sh`
- [ ] T078 [US5] 运行 US5 注入失败/恢复通过测试，并把自动化回归结果记录到 `specs/008-ui-overlap-remediation/validation/milestones.md`
- [ ] T079 [US5] 创建持续视觉与中文化回归里程碑的聚焦 Git 提交，并把提交 SHA 写入 `specs/008-ui-overlap-remediation/validation/milestones.md`

**Checkpoint**: US5 可独立阻止布局碰撞、低对比度和英文文案回归。

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: 完成全量门禁、干净候选、目标机高密度验收、资源清理和可追溯交付。

- [ ] T080 [P] 将最终术语、允许英文范围、错误分层和布局约束补充到用户文档 `docs/operations/ui-workspace.md`
- [ ] T081 [P] 将审计覆盖率、已修复发现和允许覆盖理由回填到 `specs/008-ui-overlap-remediation/validation/audit-inventory.md`
- [ ] T082 按 `specs/008-ui-overlap-remediation/quickstart.md` 运行 lint、格式、全部前端单测、构建、浏览器矩阵、Go 单元/合同/安全和产物合规门禁
- [ ] T083 在干净提交上构建候选并将 commit SHA、candidate ID、二进制摘要、合同摘要、构建时间和无迁移状态记录到 `specs/008-ui-overlap-remediation/validation/candidate.md`
- [ ] T084 使用已构建候选部署到 `10.72.1.7`，禁止目标机源码修改，并将部署与回滚信息记录到 `specs/008-ui-overlap-remediation/validation/target-acceptance.md`
- [ ] T085 在 `10.72.1.7` 创建专用高密度 Lab，执行双主题、三视口、125%、QEMU/Docker/PC/Bridge/NAT/L2/L3、平行链路、失败状态、终端、抓包和 Traffic Filter 验收并输出 `specs/008-ui-overlap-remediation/validation/acceptance-evidence.json`
- [ ] T086 删除验收 Lab 及其进程、容器、netns、接口、链路、抓包和过滤资源，确认用户 Lab 基线不变并更新 `specs/008-ui-overlap-remediation/validation/target-acceptance.md`
- [ ] T087 使用 `specs/008-ui-overlap-remediation/contracts/acceptance-evidence.schema.json` 校验最终证据并运行仓库敏感内容扫描
- [ ] T088 创建最终验收与文档的聚焦 Git 提交，并将最终提交 SHA 与产物摘要记录到 `specs/008-ui-overlap-remediation/validation/milestones.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 Setup**: 无依赖，可立即开始。
- **Phase 2 Foundational**: 依赖 Phase 1；完成后所有用户故事共享同一布局、证据和扫描基础。
- **US1 / Phase 3**: 依赖 Phase 2；是建议 MVP。
- **US2 / Phase 4**: 依赖 Phase 2，可与 US1 的非共享文件工作并行；最终需要 US1 的拓扑选择与菜单状态保持兼容。
- **US3 / Phase 5**: 依赖 Phase 2，可与 US1/US2 的实现并行，但最终文案替换应在相关组件布局稳定后合并。
- **US4 / Phase 6**: 依赖 US1、US2、US3 的主要布局和文案结果，负责组合矩阵收敛。
- **US5 / Phase 7**: 依赖 Phase 2 的证据基础，可提前开发；完整场景基线必须在 US1–US4 完成后固化。
- **Phase 8 Polish**: 依赖全部用户故事及其里程碑提交。

### User Story Dependency Graph

```text
Setup -> Foundation -> US1 ─┐
                         US2 ├-> US4 -> US5 final baseline -> Polish/Target
                         US3 ┘
Foundation -------------------------> US5 infrastructure
```

- **US1**: 独立交付拓扑清晰度，不依赖其他故事。
- **US2**: 独立交付检查器、菜单和工作区清晰度，复用共享基础。
- **US3**: 独立交付中文化和错误呈现，复用共享基础。
- **US4**: 组合验证 US1–US3 在视口、主题和缩放变化下的稳定性。
- **US5**: 可先实现审计基础，最终基线依赖 US1–US4 的完成状态。

## Parallel Opportunities

### Setup and Foundation

```text
T003 场景 JSON ─┐
T004 审计类型   ├─> T005 Schema tests
T007 主题变量   ├─> T015-T017 共享组件
T009/T010 断言  ├─> 各 Story 浏览器测试
T011-T013 证据  ┘
```

### User Story 1

```text
T020 几何测试 -> T025 几何实现 -> T026 Canvas
T022 链路测试 -> T027 链路实现
T023 流量测试 -> T028 Overlay 实现
T024 浏览器场景可与以上组件测试并行编写
```

### User Story 2

```text
T032 检查器测试 -> T038 检查器布局
T033/T034 图表测试 -> T039 图表实现
T035 菜单测试 -> T040 菜单实现
T036 工作区测试 -> T041/T042 工作区实现
```

### User Story 3

```text
T045-T049 测试可并行编写
T051 终端/诊断、T052 抓包/过滤、T053 模板、T054 拓扑、T055 任务可在不同文件域并行实现
T050 术语表完成后统一合并，T056 负责错误分层收口
```

### User Story 4 and 5

```text
T060-T063 响应式测试可并行
T065 与 T066 在不同页面并行，T064/T067 负责共享壳与主题收口
T070-T073 回归基础测试可并行，T074-T077 按证据流水线顺序集成
```

## Implementation Strategy

### MVP First

1. 完成 Phase 1 和 Phase 2。
2. 完成 US1，使拓扑节点、端口、链路和 Traffic Filter 首先达到稳定可用。
3. 运行 US1 独立验收并形成聚焦提交。
4. 在不影响 US1 的前提下继续 US2 和 US3。

### Incremental Delivery

1. **Foundation**: 审计、断言、证据、语义样式和共享组件。
2. **US1**: 拓扑核心视觉与交互。
3. **US2**: 检查器、菜单和底部工作区。
4. **US3**: 全页面中文化与错误呈现。
5. **US4**: 双主题、三视口和 125% 组合稳定性。
6. **US5**: 自动审计和持续回归门禁。
7. **Final**: 干净候选、`10.72.1.7` 验收、清理和可追溯记录。

### Milestone Rule

- 每个阶段先写失败测试，再实施修复。
- 每个用户故事通过其独立测试后必须创建聚焦 Git 提交。
- 未提交或工作树不干净的产物不得部署到 `10.72.1.7`。
- 目标机失败必须返回本地补测试、修复和新提交，不得直接修改目标机源码。
- 所有验收证据必须脱敏，并确保专用测试资源残留数量为 0。
