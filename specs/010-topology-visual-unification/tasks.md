# Tasks: 拓扑连接视觉与资源放置统一

**Input**: Design documents from `/specs/010-topology-visual-unification/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: 本功能改变连接状态展示、Traffic Filter 覆盖、创建 API/MCP、SQLite 事务和共享位置恢复；依照宪法，单元、合同、并发、恢复、清理、前端交互和目标机验收测试均为必需。

**Organization**: 任务按四个用户故事组织。每个故事先编写失败测试，再完成实现、运行独立门禁并形成聚焦 Git 提交。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可与同阶段其他任务并行，涉及不同文件且不依赖未完成任务。
- **[Story]**: 对应 `spec.md` 中的用户故事。
- 每项任务都包含准确文件路径。

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 固化兼容约束、基线和验收记录结构，避免实现期间重复踩过往拓扑交互问题。

- [X] T001 检查与连接渲染、Traffic Filter、placement 创建有关的 `git log`/`git blame`，将必须保留的兼容行为记录到 `specs/010-topology-visual-unification/implementation-notes.md`
- [X] T002 运行现有拓扑、placement、实验室 store 和目标 acceptance 聚焦测试，并把基线命令与结果记录到 `specs/010-topology-visual-unification/validation/baseline.md`
- [X] T003 [P] 建立四个里程碑的提交 SHA、测试结果、候选制品和目标机验收记录模板到 `specs/010-topology-visual-unification/validation/milestones.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 建立统一连接投影和视觉测试基础，后续状态、语义图例与恢复场景都只依赖这一套稳定类型。

**⚠️ CRITICAL**: 完成本阶段后才能进入任何用户故事实现。

- [X] T004 在 `web/src/features/topology/interactionTypes.ts` 定义 `ConnectionPresentation`、端点、状态视觉、语义标记、能力和 route group 公共类型
- [X] T005 [P] 在 `web/src/styles/theme.css` 增加连接成功、处理中、失败、断开、聚焦、抓包和 Traffic Filter 的浅色/深色语义 token，禁止按持久化连接类型分配基础状态颜色
- [X] T006 [P] 在 `web/src/features/topology/topologyConnectionFixtures.ts` 建立覆盖 `Link`、`NetworkAttachment`、`NetworkObjectLink`、平行接口和 Traffic Filter observation 的共享测试构造器
- [X] T007 [P] 在 `tests/e2e/fixtures/topologyVisualAssertions.ts` 增加连接样式、独立命中区域、语义图例和规范 placement 矩形的 Playwright 断言辅助函数

**Checkpoint**: 统一类型、主题 token 和测试夹具可供四个用户故事复用。

---

## Phase 3: User Story 1 - 一眼识别所有连接的状态 (Priority: P1) 🎯 MVP

**Goal**: 三类连接在待处理、已连接、失败、断开处理中使用同一基础视觉；选择、抓包和 Traffic Filter 只作为覆盖层；平行连接保持独立。

**Independent Test**: 在一个同时包含节点直连、Bridge/NAT attachment 和对象直连的实验室中构造全部状态与三条平行连接，验证状态视觉、中文名称、命中区域和 Traffic Filter 衰减完全一致。

### Tests for User Story 1

- [X] T008 [P] [US1] 为三类连接到统一投影、状态规范化、可读标签和能力映射编写失败单测到 `web/src/features/topology/topologyConnectionPresentation.test.ts`
- [X] T009 [P] [US1] 为跨连接类别 route group、稳定排序和 2/3/4 条平行线对称曲率编写失败单测到 `web/src/features/topology/topologyParallelRoutes.test.ts`
- [X] T010 [P] [US1] 扩展 `web/src/features/topology/topologyVisualSemantics.test.ts`，覆盖处理中、失败、断开、选中、聚焦、抓包和 Traffic Filter 的视觉优先级
- [X] T011 [P] [US1] 扩展 `web/src/features/topology/TopologyCanvas.test.ts`，验证三类连接共享渲染路径、端点标签和独立点击/右键区域
- [X] T012 [P] [US1] 扩展 `web/src/features/topology/TopologyCanvas.a11y.test.ts`、`web/src/features/topology/TopologyCanvas.performance.test.ts`、`web/src/features/diagnostics/GlobalCaptureWorkspace.test.ts` 与 `web/src/features/diagnostics/TrafficFilterPanel.test.ts`，覆盖非颜色状态、键盘聚焦、observation 更新不触发节点重布局且抓包/Wireshark 制品语义不变
- [X] T013 [P] [US1] 在 `tests/e2e/journeys/topologyVisualRecognition.spec.ts` 增加混合连接状态、50%/100%/200% 平行线选择和 Traffic Filter 衰减场景

### Implementation for User Story 1

- [X] T014 [US1] 在 `web/src/features/topology/topologyConnectionPresentation.ts` 实现三类持久化连接到 `ConnectionPresentation` 的单一规范化适配器
- [X] T015 [US1] 在 `web/src/features/topology/topologyVisualSemantics.ts` 实现状态优先的颜色、线型、端点标记和中文可访问描述，并保持覆盖层与基础状态分离
- [X] T016 [US1] 在 `web/src/features/topology/topologyParallelRoutes.ts` 实现无方向 route group、稳定组内排序和跨类型对称路径偏移
- [X] T017 [US1] 重构 `web/src/features/topology/TopologyCanvas.vue`，让 ECharts 基础边只消费统一投影和平行路径结果，并保持 `layout: none` 与权威节点坐标
- [X] T018 [US1] 更新 `web/src/features/topology/linkPresentation.ts`，统一使用 `资源名:端口 ↔ 资源名:端口` 标签、逻辑接入口名称和状态文本
- [X] T019 [US1] 更新 `web/src/features/topology/trafficPathTypes.ts` 与 `web/src/features/topology/TopologyCanvas.vue`，把 Traffic Filter 粒子、方向提示和 capture active 实现为有期限覆盖层且只归属具体连接 ID
- [X] T020 [US1] 更新 `web/src/features/topology/LinkContextMenu.vue` 与 `web/src/features/topology/TopologyInspector.vue`，让三类连接使用一致选择、检查、删除、抓包能力展示和禁用原因
- [X] T021 [US1] 运行 US1 的 Vitest、Playwright、lint 和 build 门禁，将结果写入 `specs/010-topology-visual-unification/validation/milestones.md` 并创建“统一连接状态视觉”聚焦 Git 提交

**Checkpoint**: US1 可独立交付；用户无需理解内部连接类型即可判断状态和观察平行线路。

---

## Phase 4: User Story 2 - 理解确有必要的连接差异 (Priority: P1)

**Goal**: 只为真实网络语义保留轻量标记，并通过左下角动态图例给出中文名称、原因、数量和非破坏性定位。

**Independent Test**: 在普通连接、NAT 上联和共享广播域之间切换，确认只有当前出现的特殊语义进入图例，失败状态始终优先，浅色/深色和键盘操作均可辨识。

### Tests for User Story 2

- [X] T022 [P] [US2] 为 NAT 管理上联、共享广播域和普通点到点连接的 marker 派生规则编写失败单测到 `web/src/features/topology/topologyConnectionSemantics.test.ts`
- [X] T023 [P] [US2] 为动态项生成、删除后消失、数量统计和匹配连接集合编写失败单测到 `web/src/features/topology/topologyConnectionLegend.test.ts`
- [X] T024 [P] [US2] 为折叠、滚动、hover/focus 强调、浅色/深色和 accessible name 编写失败组件测试到 `web/src/features/topology/TopologyConnectionLegend.test.ts`
- [X] T025 [P] [US2] 在 `tests/e2e/journeys/topologyVisualRecognition.spec.ts` 增加 NAT/Bridge 语义标记、动态图例删除和双主题键盘验收场景

### Implementation for User Story 2

- [X] T026 [US2] 在 `web/src/features/topology/topologyConnectionSemantics.ts` 实现 `managed-nat-uplink`、`shared-broadcast-domain` 和默认无标记规则，禁止按数据库类别产生 marker
- [X] T027 [US2] 在 `web/src/features/topology/topologyConnectionLegend.ts` 从当前 `ConnectionPresentation[]` 派生状态项、动态语义项、数量、分类依据和可聚焦连接集合，供图例与诊断审计复用
- [X] T028 [US2] 新建 `web/src/features/topology/TopologyConnectionLegend.vue`，实现中文说明、折叠/滚动、键盘聚焦和非破坏性匹配连接强调
- [X] T029 [US2] 更新 `web/src/features/topology/TopologyCanvas.vue` 与 `web/src/features/topology/TopologyWorkspace.vue`，渲染语义徽标并把动态图例放入左下角现有图例布局而不遮挡画布控件
- [X] T030 [P] [US2] 将状态、NAT 上联、共享广播域及差异原因文案集中到 `web/src/locales/zh-CN.ts` 与 `web/src/locales/terminology.ts`
- [X] T031 [US2] 运行 US2 的 Vitest、axe、Playwright、主题和布局门禁，将结果写入 `specs/010-topology-visual-unification/validation/milestones.md` 并创建“连接语义与动态图例”聚焦 Git 提交

**Checkpoint**: US2 可独立验收；未解释的特殊样式数量为零，普通拓扑不产生多余图例。

---

## Phase 5: User Story 3 - 新资源创建后自动落在空白位置 (Priority: P1)

**Goal**: QEMU、Docker、PC、Bridge、NAT、L2 和 L3 在创建事务内取得距离首选中心最近的权威无碰撞位置，连续创建和并发创建均不堆叠。

**Independent Test**: 在密集视口连续创建 20 个混合资源，验证规范足迹不相交、已有坐标不移动、冲突自动避让、失败不留占位且新资源可立即定位。

### Tests for User Story 3

- [X] T032 [P] [US3] 扩展 `internal/domain/topology_placement_test.go`，先覆盖 placement intent、受控 footprint class、坐标边界和首选坐标成对校验
- [X] T033 [P] [US3] 新建 `internal/domain/topology_placement_allocator_test.go`，先覆盖最近候选、稳定方形螺旋、clearance、密集扩展、候选耗尽和不移动既有 placement
- [X] T034 [P] [US3] 扩展 `internal/store/sqlite/topology_placement_repository_test.go`，先覆盖 SQLite 写事务中的碰撞裁决、20 次创建、并发创建、revision 和回滚
- [X] T035 [P] [US3] 扩展 `internal/store/sqlite/laboratory_repository_test.go`，先验证节点、接口、placement、实验室 revision 和 outbox 原子提交或全部回滚
- [X] T036 [P] [US3] 扩展 `internal/store/sqlite/network_state_repository_test.go`，先验证网络对象、placement、实验室 revision 和 outbox 原子提交或全部回滚
- [X] T037 [P] [US3] 新建 `internal/api/http/topology_creation_contract_test.go`，先覆盖节点/网络对象 placement intent、adjustment metadata、If-Match、幂等冲突和结构化错误
- [X] T038 [P] [US3] 扩展 `web/src/features/topology/CreateTopologyResourceDrawer.test.ts`，先验证创建请求携带画布中心 intent 且 UI 使用响应中的权威 placement
- [X] T039 [P] [US3] 扩展 `web/src/features/topology/TopologyWorkspace.test.ts` 与 `web/src/stores/laboratory.test.ts`，先验证创建后不再二次补写预测位置、权威位置收敛和“定位新资源”
- [X] T040 [P] [US3] 新建 `tests/e2e/journeys/topologyAuthoritativePlacement.spec.ts`，先覆盖 20 个混合资源、标签/端口/加号足迹、拥挤区域扩展和刷新后坐标保持

### Implementation for User Story 3

- [X] T041 [US3] 扩展 `internal/domain/topology_placement.go`，实现 `PlacementIntent`、`PlacementFootprint`、`PlacementAssignment`、受控 class 和完整验证规则
- [X] T042 [US3] 新建 `internal/domain/topology_placement_allocator.go`，实现版本化规范足迹、确定性方形螺旋、边界检查和首个安全候选选择
- [X] T043 [US3] 扩展 `internal/store/sqlite/topology_placement_repository.go`，提供事务内读取占用和提交初始 placement 的 repository primitive，复用现有 placement 唯一约束
- [X] T044 [US3] 修改 `internal/store/sqlite/laboratory_repository.go`，把节点、接口、初始 placement、实验室 revision 和含创建入口/请求位置/最终位置/调整原因的 outbox 审计写入合并为原子创建方法
- [X] T045 [US3] 修改 `internal/store/sqlite/network_repository.go`，把网络对象、初始 placement、实验室 revision 和含创建入口/请求位置/最终位置/调整原因的 outbox 审计写入合并为原子创建方法
- [X] T046 [US3] 扩展 `internal/app/command/node.go`，让节点创建接受 placement intent、expected revision 和 idempotency key，并返回 `PlacementAssignment`
- [X] T047 [US3] 扩展 `internal/app/reconcile/network_object_tasks.go`，让网络对象创建通过同一 placement allocator 和事务合同返回权威 assignment
- [X] T048 [US3] 更新 `internal/api/http/topology_handlers.go` 与 `internal/api/http/network_handlers.go`，解析 placement intent/If-Match/Idempotency-Key 并返回资源、权威 placement、实验室 revision 和调整原因
- [X] T049 [US3] 将 feature delta 合并到正式 `specs/001-network-simulator-platform/contracts/openapi.yaml`，同步节点与网络对象创建的 placement、revision、幂等和错误合同
- [X] T050 [US3] 更新 `web/src/api/generated.ts` 与 `web/src/api/index.ts`，加入 `PlacementIntent`、`PlacementAssignment` 和两个创建响应类型，并传递 If-Match/idempotency
- [X] T051 [US3] 更新 `web/src/features/topology/CreateTopologyResourceDrawer.vue`，从画布上下文提交 placement intent 并展示非阻断“已避让现有资源”反馈
- [X] T052 [US3] 更新 `web/src/features/topology/TopologyWorkspace.vue`，删除创建后调用 placement batch 的两阶段流程，以返回 assignment 为准且不移动已有资源
- [X] T053 [US3] 更新 `web/src/stores/laboratory.ts`，按实验室 revision 和 placement revision 幂等合并创建响应/事件，并提供新资源定位目标而不自动平移全图
- [X] T054 [US3] 运行 US3 的 Go 单元/SQLite 并发/HTTP 合同/Vitest/Playwright 门禁，将结果写入 `specs/010-topology-visual-unification/validation/milestones.md` 并创建“权威无碰撞初始位置”聚焦 Git 提交

**Checkpoint**: US3 可独立交付；连续创建 20 个资源无初始重叠，创建失败和取消无幽灵 placement。

---

## Phase 6: User Story 4 - 保持旧实验室稳定并支持一致控制面 (Priority: P2)

**Goal**: 升级不改动旧坐标和网络行为；SPA、HTTP、MCP 创建遵循相同位置合同；并发客户端、服务重启和导入恢复后保持一致。

**Independent Test**: 升级三个已有实验室，通过浏览器、HTTP 和 MCP 并发创建资源后重启服务，确认旧坐标变化为零、新位置一致恢复、图例重新派生且删除实验室后无 placement 或连接泄漏。

### Tests for User Story 4

- [X] T055 [P] [US4] 扩展 `internal/api/mcp/server_test.go`，先覆盖节点/网络对象创建工具的 placement intent、revision、幂等、权威 assignment 和结构化冲突
- [X] T056 [P] [US4] 扩展 `internal/app/query/topology_placement_test.go` 并新建 `internal/app/query/laboratory_test.go`，先验证重启查询保留旧坐标和全部新 placement
- [X] T057 [P] [US4] 扩展 `internal/store/sqlite/import_repository_test.go`，先验证导入旧实验室不重新计算已有坐标且新资源避开旧重叠区域
- [X] T058 [P] [US4] 扩展 `web/src/stores/laboratory.runtimeTruth.test.ts`，先验证乱序/重复 placement 事件、多客户端创建和刷新恢复最终收敛
- [X] T059 [P] [US4] 扩展 `tests/e2e/journeys/concurrentClients.spec.ts`，先覆盖两个浏览器加 HTTP/MCP 的 10 组并发创建和 2 秒布局收敛
- [X] T060 [P] [US4] 新建 `tests/e2e/journeys/topologyPlacementRecovery.spec.ts`，先覆盖旧坐标升级对比、服务重启、旧连接视觉恢复和实验室删除清理
- [X] T061 [P] [US4] 扩展 `acceptance/t225-service-restart.sh`，先加入 placement 数量/坐标摘要、连接状态、孤立 placement 和删除后资源泄漏断言

### Implementation for User Story 4

- [X] T062 [US4] 更新 `internal/api/mcp/network_tools.go`，让节点和网络对象创建工具使用与 HTTP 相同的 application command、placement intent、revision 和 idempotency 语义
- [X] T063 [US4] 更新 `internal/api/mcp/topology_placement_tools.go` 与 `specs/001-network-simulator-platform/contracts/mcp-tools.md`，返回权威 assignment 并明确手工移动工具不承担首次放置
- [X] T064 [US4] 更新 `internal/app/query/laboratory.go` 与 `internal/app/query/topology_placement.go`，保证 snapshot 在资源创建事件后包含权威 placement 且排序稳定
- [X] T065 [US4] 更新 `internal/api/stream/events.go` 与 `web/src/stores/laboratory.ts`，让资源创建/placement 事件按 sequence 幂等收敛且不产生 `(0,0)` 临时事实
- [X] T066 [US4] 更新 `internal/store/sqlite/import_repository.go`，导入时原样保留已有 placement，并为缺失 placement 的旧资源采用兼容 fallback 而不重排已定位资源
- [X] T067 [US4] 更新 `web/src/features/topology/topologyLayout.ts` 与 `web/src/features/topology/TopologyCanvas.vue`，仅对真正缺失 placement 的旧资源使用稳定 fallback，收到权威位置后局部收敛且不触发全图 relayout
- [X] T068 [US4] 在 `acceptance/frontend-acceptance.sh` 增加混合资源、控制面并发创建、重启恢复和删除清理场景，并将资源台账限制到专用 acceptance Lab
- [X] T069 [US4] 运行 US4 的 MCP/查询/导入/store/并发/recovery/leak 门禁，将结果写入 `specs/010-topology-visual-unification/validation/milestones.md` 并创建“控制面一致与恢复”聚焦 Git 提交

**Checkpoint**: 四个故事全部可用；旧实验室坐标不变，所有控制面和客户端共享同一权威位置与连接视觉。

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: 完成合同同步、全量质量门禁、可复现制品、目标机混合拓扑验收和回滚证据。

- [X] T070 [P] 更新 `specs/010-topology-visual-unification/quickstart.md`，使最终命令、测试文件、候选记录和目标机端口与实现一致
- [X] T071 [P] 更新 `acceptance/README.md`，说明视觉统一、20 资源 placement、并发客户端、Traffic Filter 衰减、重启和清理验收方法
- [X] T072 运行 `go test ./...` 及仓库规定的格式、静态分析、合同、恢复和资源泄漏测试，并把结果写入 `specs/010-topology-visual-unification/validation/final-local.md`
- [X] T073 运行 `npm run format:check`、`npm run lint`、`npm run build`、`npm test`、`npm run test:acceptance-unit` 和本地 Playwright，并把结果写入 `specs/010-topology-visual-unification/validation/final-local.md`
- [ ] T074 从干净 Git 提交构建部署候选，记录 commit SHA、制品摘要、迁移状态、合同版本和上一制品到 `specs/010-topology-visual-unification/validation/deployment.md`
- [ ] T075 将已记录候选部署到 `10.72.1.7`，按 `specs/010-topology-visual-unification/quickstart.md` 验证混合连接、平行线、动态图例、20 资源、双客户端/API/MCP、Traffic Filter 衰减、重启恢复和实验室删除清理
- [ ] T076 将目标机结果、证据路径和资源清理摘要写入 `specs/010-topology-visual-unification/validation/target-acceptance.md`；若失败则回滚上一制品并返回本地修复、测试和新提交

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1** 无依赖，先建立历史与验证基线。
- **Phase 2** 依赖 Phase 1，完成后才能开始任何用户故事。
- **US1** 依赖 Phase 2，是连接视觉 MVP。
- **US2** 依赖 US1 的 `ConnectionPresentation` 和基础状态视觉，但不依赖 placement 实现。
- **US3** 依赖 Phase 2；后端 placement 工作可在 US1/US2 前端稳定后并行准备，但合并前必须保留统一创建合同。
- **US4** 依赖 US3 的权威创建结果，并复用 US1/US2 的投影与图例恢复规则。
- **Polish** 依赖四个用户故事及其里程碑提交全部完成。

### User Story Dependency Graph

```text
Setup → Foundation → US1 ──→ US2
                     │
                     └────→ US3 → US4 → Polish/Target Deployment
```

### Within Each User Story

1. 先编写并运行该故事的失败测试。
2. 实现领域/投影模型，再实现服务或派生逻辑。
3. 接入 HTTP/MCP/store/UI 和事件同步。
4. 运行故事独立门禁并更新验证记录。
5. 创建干净、聚焦的里程碑 Git 提交后才能进入部署候选。

### Requirement Coverage

| Requirement range | Primary tasks |
|---|---|
| FR-001–FR-009 统一状态、覆盖层、平行线和标签 | T008–T020 |
| FR-010–FR-015 真实语义标记与动态图例 | T022–T031 |
| FR-016–FR-025 无碰撞初始位置、可见性和并发裁决 | T032–T053 |
| FR-026–FR-029 幂等、失败清理、恢复和删除同步 | T034–T069 |
| FR-030 抓包、Wireshark 与 Traffic Filter 制品兼容 | T012–T020、T060–T061、T075 |
| FR-031 审计创建入口、位置调整、状态和图例依据 | T027、T044–T048、T063、T076 |
| FR-032 容量与权限边界不扩大 | T037、T055、T072、T075 |
| FR-033 旧实验室坐标、端点、网络与运行状态兼容 | T056–T067、T075 |
| FR-034 键盘、辅助技术和非颜色表达 | T010–T013、T024–T030、T073、T075 |

---

## Parallel Opportunities

### User Story 1

```text
T008 Connection projection tests
T009 Parallel route tests
T010 Visual priority tests
T011 Canvas interaction tests
T012 Accessibility/performance tests
T013 Playwright recognition tests
```

测试夹具完成后可并行编写；实现阶段 `T014`、`T015`、`T016` 可按文件分工，`T017` 在三者完成后统一接入。

### User Story 2

```text
T022 Semantic marker tests
T023 Legend derivation tests
T024 Legend component tests
T025 End-to-end legend tests
```

`T026` 与 `T030` 可并行，`T027` 依赖 `T026`，`T028` 依赖 `T027`，最后由 `T029` 集成到画布。

### User Story 3

```text
T032 Domain validation tests
T033 Allocator tests
T034 Placement transaction tests
T035 Node atomic-create tests
T036 Network-object atomic-create tests
T037 HTTP contract tests
T038 Drawer tests
T039 Workspace/store tests
T040 Twenty-resource E2E tests
```

领域、SQLite、HTTP 和前端测试可以并行编写；实现时节点 repository `T044` 与网络对象 repository `T045` 可并行，API 类型 `T050` 可在正式合同 `T049` 确定后独立推进。

### User Story 4

```text
T055 MCP parity tests
T056 Query recovery tests
T057 Import compatibility tests
T058 Store convergence tests
T059 Concurrent-client E2E tests
T060 Restart E2E tests
T061 Host restart script assertions
```

MCP、查询、导入、前端 store 和 acceptance 测试可并行；`T062`、`T064`、`T066` 可按后端文件分工，`T065` 与 `T067` 在事件/查询合同稳定后集成。

---

## Implementation Strategy

### MVP First

1. 完成 Setup 与 Foundation。
2. 只实现 US1，统一所有连接的状态视觉、标签、覆盖层和平行线路。
3. 运行 US1 独立测试并形成第一个可演示提交。
4. 在不部署未提交代码的前提下评审视觉语义，再继续 US2 和 placement。

### Incremental Delivery

1. **M1 / US1**: 统一基础状态、选择、Traffic Filter 和平行线。
2. **M2 / US2**: 加入真实语义 marker 和动态图例。
3. **M3 / US3**: 加入创建事务内权威无碰撞 placement。
4. **M4 / US4**: 完成 HTTP/MCP/SPA 一致性、并发、旧数据和重启恢复。
5. **Release**: 全量门禁、干净制品、`10.72.1.7` 验收和可验证回滚。

### Completion Definition

- 所有任务保持 `- [ ] TNNN [P?] [US?] 描述 + 文件路径` 格式。
- 四个用户故事均有可独立执行的测试标准和对应里程碑提交。
- 连接状态、语义、观察覆盖和 placement 都只有一个权威来源。
- 目标机只部署来自干净已记录提交的制品，且删除 acceptance Lab 后资源清理率为 100%。
