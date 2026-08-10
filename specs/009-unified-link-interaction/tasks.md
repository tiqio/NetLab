# Tasks: 统一拓扑端口连接体验

**Input**: Design documents from `/specs/009-unified-link-interaction/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: 本功能改变网络连接 mutation、SQLite 端点占用、durable task、运行中资源接线、HTTP/MCP 合同、画布手势、抓包/Traffic Filter 入口和恢复清理；依照宪法，领域、合同、并发、故障、恢复、资源泄漏、前端交互、无障碍和目标机测试全部必需，并先于对应实现编写。

**Organization**: 任务按四个用户故事组织。每个故事先编写失败测试，再完成实现、运行独立门禁并形成聚焦 Git 提交。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可与同阶段其他任务并行，涉及不同文件且不依赖未完成任务。
- **[Story]**: 对应 `spec.md` 中的用户故事。
- 每项任务都包含准确文件路径。

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 固化现有连接行为、基线和里程碑证据，避免统一入口破坏 010 视觉、运行时所有权或旧自动化兼容性。

- [X] T001 检查节点链路、网络附件、对象链路、端口 overlay、加号入口、轻量默认配置和连接恢复相关 `git log`/`git blame`，将兼容约束记录到 `specs/009-unified-link-interaction/implementation-notes.md`
- [X] T002 运行现有连接、网络对象、任务、拓扑交互、抓包、Traffic Filter 和恢复聚焦测试，将命令与结果记录到 `specs/009-unified-link-interaction/validation/baseline.md`
- [X] T003 [P] 建立端点/命令、画布交互、四端口/恢复、最终候选四个里程碑的 SHA 与测试记录模板到 `specs/009-unified-link-interaction/validation/milestones.md`
- [X] T004 [P] 建立合同版本、迁移状态、候选摘要、目标机证据和回滚信息模板到 `specs/009-unified-link-interaction/validation/deployment-template.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 建立所有故事共用的统一端点、跨类型占用、durable task、合同和前端类型基础。

**⚠️ CRITICAL**: 完成本阶段后才能进入任何用户故事实现。

### Tests for Foundational Infrastructure

- [X] T005 [P] 为规范端点验证、允许的 backing 映射、自连/跨实验室/不兼容拒绝编写失败领域测试到 `internal/domain/topology_connection_test.go`
- [X] T006 [P] 为节点接口与对象端口跨 `Link`/`NetworkAttachment`/`NetworkObjectLink` 唯一预留、事务回滚和迁移回填编写失败 SQLite 测试到 `internal/store/sqlite/topology_connection_repository_test.go`
- [X] T007 [P] 为统一创建命令的 revision、幂等 fingerprint、不同 key 并发争用和结构化问题编写失败测试到 `internal/app/command/topology_connection_test.go`
- [X] T008 [P] 为统一创建/list/get/delete OpenAPI 合同及旧专用路由兼容语义编写失败合同测试到 `internal/api/http/topology_connection_contract_test.go`
- [X] T009 [P] 为 `netlab.topology_connections.*` 与旧 MCP 工具对称语义编写失败测试到 `internal/api/mcp/topology_connection_tools_test.go`
- [X] T010 [P] 为统一前端端点类型、可用性、能力与 backing 映射编写失败单测到 `web/src/features/topology/topologyEndpointCompatibility.test.ts`

### Implementation for Foundational Infrastructure

- [X] T011 在 `internal/domain/topology_connection.go` 定义 `ConnectionEndpoint`、`TopologyConnection`、配置、能力、状态、映射和结构化验证规则
- [X] T012 在 `internal/store/sqlite/migrations/0013_unified_topology_connections.sql` 扩展端点预留的 operation/state 字段并回填节点链路、网络附件和对象链路具体端点
- [X] T013 在 `internal/store/sqlite/topology_connection_repository.go` 实现统一查询、原子预留、backing record 创建/删除、revision、task、audit/outbox 和失败回滚事务
- [X] T014 在 `internal/app/command/topology_connection.go` 实现调用方不选择 backing kind 的统一 create/list/get/delete 应用服务
- [X] T015 在 `internal/app/command/topology_connection_tasks.go` 实现规范 request fingerprint、durable task 创建、幂等结果和取消入口
- [X] T016 在 `internal/app/reconcile/topology_connections.go` 增加统一连接到既有 link/attachment/object-link runtime service 的适配边界
- [X] T017 在 `internal/api/http/topology_connection_handlers.go` 与 `internal/api/http/routes.go` 注册统一 HTTP handlers、revision/idempotency headers 和 problem 响应
- [X] T018 在 `internal/api/mcp/topology_connection_tools.go` 与 `internal/api/mcp/tools.go` 注册统一 MCP create/list/get/delete 工具
- [X] T019 在 `web/src/api/generated.ts` 与 `web/src/api/index.ts` 增加合同生成类型和统一连接 API client
- [X] T020 在 `web/src/features/topology/interactionTypes.ts` 与 `web/src/features/topology/topologyEndpointCompatibility.ts` 定义共享端点、草稿、候选状态和兼容性类型
- [X] T021 运行 T005–T010 及迁移回放测试并将基础阶段结果记录到 `specs/009-unified-link-interaction/validation/milestones.md`
- [X] T022 将统一端点与命令基础作为聚焦 Git 提交，并把 commit SHA 写入 `specs/009-unified-link-interaction/validation/milestones.md`

**Checkpoint**: UI、HTTP 和 MCP 可使用同一端点合同，SQLite 能跨三类 backing model 串行占用具体端点。

---

## Phase 3: User Story 1 - 从任意端口直接拖拽连接 (Priority: P1) 🎯 MVP

**Goal**: 用户从普通节点或轻量设备空闲端口拖到兼容端口、资源主体、Bridge 或 NAT，即可完成统一连接，不再进入 Inspector 判断底层类别。

**Independent Test**: 在包含 QEMU、Docker、PC、轻量 L2/L3、Bridge 和 NAT 的实验室中，仅使用端口拖拽完成节点到节点、节点到轻量对象、对象到对象、节点到 Bridge/NAT 五类连接；验证预览锚点、目标反馈、端口 chooser、最终连接、取消和错误路径。

### Tests for User Story 1

- [X] T023 [P] [US1] 扩展连接草稿状态机失败测试，覆盖直接端口 drop、资源主体多端口 chooser、逻辑 access、同源取消和不兼容目标到 `web/src/features/topology/topologyConnectionController.test.ts`
- [X] T024 [P] [US1] 扩展手势互斥失败测试，覆盖 pointer capture、拖拽阈值、端口拖拽不触发节点移动/框选/平移和 pointer cancel 到 `web/src/features/topology/topologyInteractionController.test.ts`
- [X] T025 [P] [US1] 扩展画布组件、无障碍和性能失败测试，覆盖源锚定预览、目标状态、Bridge/NAT access、50%/100%/200% 命中和 50 次拖拽到 `web/src/features/topology/TopologyCanvas.test.ts`、`web/src/features/topology/TopologyCanvas.a11y.test.ts` 与 `web/src/features/topology/TopologyCanvas.performance.test.ts`
- [X] T026 [P] [US1] 扩展工作区失败测试，覆盖五类端点组合、资源主体 chooser、L2 VLAN 配置、冲突刷新和不产生 optimistic backing record 到 `web/src/features/topology/TopologyWorkspace.test.ts`
- [X] T027 [P] [US1] 为直接拖拽使用统一 HTTP 命令及 HTTP/MCP 最终状态对称编写失败合同测试到 `internal/api/http/topology_connection_contract_test.go` 与 `internal/api/mcp/topology_connection_tools_test.go`
- [X] T028 [P] [US1] 增加混合资源端口拖拽、目标反馈、取消和缩放稳定性的 Playwright 旅程到 `tests/e2e/journeys/unifiedPortConnection.spec.ts`

### Implementation for User Story 1

- [X] T029 [US1] 在 `web/src/features/topology/topologyConnectionController.ts` 实现统一 source/target 草稿、candidate 状态、chooser/configuring/submitting 转换和本地取消
- [X] T030 [US1] 在 `web/src/features/topology/topologyGeometry.ts` 与 `web/src/features/topology/topologyEndpointCompatibility.ts` 实现基于 010 footprint 的资源主体命中、最近端口和目标兼容解析
- [X] T031 [US1] 在 `web/src/features/topology/TopologyCanvas.vue` 实现端口 SVG pointer capture、源锚定预览线、候选反馈、逻辑 access 和互斥手势事件
- [X] T032 [US1] 在 `tests/e2e/fixtures/topologyConnectionAssertions.ts` 增加预览锚点、目标状态、端口命中、视口稳定和无幽灵线断言辅助函数
- [X] T033 [US1] 在 `web/src/features/topology/TopologyWorkspace.vue` 用统一草稿替换分离的 `pendingEndpoint`/`pendingObjectPort` 流程，并提交节点链路、附件和对象链路到统一 API
- [X] T034 [US1] 在 `web/src/features/topology/PortChooser.vue` 支持节点接口、对象命名端口和逻辑 access 候选，同时保留操作位置与草稿上下文
- [X] T035 [US1] 在 `web/src/stores/laboratory.ts` 合并统一连接 task/event 响应并以权威 snapshot 收敛，禁止客户端伪造 backing 连接
- [X] T036 [US1] 在 `web/src/features/topology/topologyConnectionPresentation.ts` 与 `web/src/features/topology/topologyVisualSemantics.test.ts` 验证 009 提交结果继续复用 010 状态、平行路由、抓包和 Traffic Filter 视觉合同
- [ ] T037 [US1] 运行 T023–T028、五类连接聚焦 Go 测试和 Playwright 旅程，并记录结果到 `specs/009-unified-link-interaction/validation/milestones.md`
- [X] T038 [US1] 将直接端口拖拽 MVP 作为聚焦 Git 提交，并把 commit SHA 写入 `specs/009-unified-link-interaction/validation/milestones.md`

**Checkpoint**: US1 可独立演示；用户仅通过拖拽即可创建五类受支持连接，取消和失败均不留下共享状态。

---

## Phase 4: User Story 2 - 使用统一加号快速连接 (Priority: P1)

**Goal**: 普通节点和所有轻量网络对象使用位置一致的右上角加号选择源端点，之后复用与拖拽相同的目标、配置、提交和取消流程。

**Independent Test**: 分别在 QEMU、轻量 L2、轻量 L3、Bridge 和 NAT 上使用加号开始连接，验证单端口自动选择、多端口 chooser、键盘完成、Escape/空白取消及最终结果与拖拽一致。

### Tests for User Story 2

- [X] T039 [P] [US2] 扩展画布失败测试，覆盖节点与网络对象一致的 connector 位置、capacity gate、hover/selection 可见性和逻辑 access 到 `web/src/features/topology/TopologyCanvas.test.ts`
- [X] T040 [P] [US2] 扩展工作区失败测试，覆盖单源自动选择、多源 chooser、加号后复用目标流程和取消到 `web/src/features/topology/TopologyWorkspace.test.ts`
- [X] T041 [P] [US2] 扩展键盘与无障碍失败测试，覆盖资源聚焦、加号、源/目标选择、live region、Enter/Space/Escape 到 `web/src/features/topology/topologyKeyboardController.test.ts` 与 `web/src/features/topology/TopologyCanvas.a11y.test.ts`
- [X] T042 [P] [US2] 增加普通节点和轻量对象加号、键盘连接及与拖拽结果对称的 Playwright 旅程到 `tests/e2e/journeys/unifiedPlusConnection.spec.ts`

### Implementation for User Story 2

- [X] T043 [US2] 在 `web/src/features/topology/TopologyCanvas.vue` 将单节点 connector overlay 泛化为所有具有空闲端点或逻辑 access 的资源并附加稳定 resource selector
- [X] T044 [US2] 在 `web/src/features/topology/TopologyWorkspace.vue` 将 `startConnection` 泛化为统一资源 source 解析，并复用 US1 草稿、chooser、配置和提交路径
- [X] T045 [US2] 在 `web/src/features/topology/PortChooser.vue` 增加 source/target 模式标题、就近锚定、兼容性说明和键盘焦点恢复
- [X] T046 [US2] 在 `web/src/features/topology/topologyKeyboardController.ts` 实现资源到 connector、端口、chooser 的无指针导航和取消动作
- [X] T047 [US2] 在 `web/src/features/topology/TopologyWorkspace.vue` 与 `web/src/features/topology/topologySelection.ts` 统一命令面板 `begin_connection`、空白取消和 selection 恢复语义
- [ ] T048 [US2] 运行 T039–T042、前端无障碍门禁和加号/键盘 Playwright 旅程，并记录结果到 `specs/009-unified-link-interaction/validation/milestones.md`
- [X] T049 [US2] 将统一加号与键盘入口作为聚焦 Git 提交，并把 commit SHA 写入 `specs/009-unified-link-interaction/validation/milestones.md`

**Checkpoint**: US2 可独立演示；加号不是第二套连接实现，所有后续状态和提交结果与端口拖拽一致。

---

## Phase 5: User Story 3 - 创建默认四端口轻量交换设备 (Priority: P1)

**Goal**: 新建轻量 L2/L3 在 SPA、HTTP 和 MCP 缺省创建时获得可编辑、可独立连接的 `eth0`–`eth3`，旧设备保持保存的端口集合。

**Independent Test**: 通过 SPA、HTTP 和 MCP 分别新建轻量 L2/L3，验证恰好四个默认端口、配置可编辑、四端口可分别占用与释放；加载和导入旧单端口对象时端口数量不变。

### Tests for User Story 3

- [X] T050 [P] [US3] 为 L2/L3 四端口默认、显式配置优先、唯一名称和旧配置不扩展编写失败领域测试到 `internal/domain/lightweight_ports_test.go`
- [ ] T051 [P] [US3] 为 SPA/HTTP/MCP 缺省创建一致、幂等重试和返回配置编写失败合同测试到 `internal/api/http/topology_creation_contract_test.go` 与 `internal/api/mcp/server_test.go`
- [X] T052 [P] [US3] 为创建草稿、配置编辑器、drawer 可增删改端口和四端口提交编写失败前端测试到 `web/src/features/topology/topologyResourceDraft.test.ts`、`web/src/features/topology/CreateTopologyResourceDrawer.test.ts` 与 `web/src/features/nodes/lightweightSwitchConfig.test.ts`
- [ ] T053 [P] [US3] 为旧单端口对象升级、导入导出和恢复不扩口编写失败测试到 `internal/app/command/import_test.go`、`internal/app/command/export_test.go` 与 `internal/app/reconcile/recovery_coordinator_test.go`
- [ ] T054 [P] [US3] 增加 SPA/HTTP/MCP 新建四端口、四端口独立连接和旧对象保持的 Playwright 旅程到 `tests/e2e/journeys/lightweightFourPorts.spec.ts`

### Implementation for User Story 3

- [X] T055 [US3] 在 `internal/domain/lightweight_ports.go` 实现仅创建时应用的 L2/L3 `eth0`–`eth3` 默认配置与验证辅助函数
- [X] T056 [US3] 在 `internal/app/reconcile/network_objects.go` 与 `internal/app/reconcile/network_object_tasks.go` 对 SPA/HTTP/MCP 创建统一应用服务端默认值，显式配置、更新、导入和恢复不补端口
- [X] T057 [US3] 在 `web/src/features/nodes/lightweightSwitchConfig.ts` 与 `web/src/features/topology/topologyResourceDraft.ts` 将新建 L2/L3 草稿初始化为四端口并保持可编辑
- [X] T058 [US3] 在 `web/src/features/nodes/LightweightSwitchConfigEditor.vue` 与 `web/src/features/topology/CreateTopologyResourceDrawer.vue` 支持创建前增加、删除、重命名和分别配置四个默认端口
- [ ] T059 [US3] 在 `internal/app/command/import.go` 与 `internal/app/command/export.go` 保持实际保存端口集合并禁止导入/恢复路径调用创建默认扩展
- [ ] T060 [US3] 运行 T050–T054、创建合同、导入导出和四端口 Playwright 旅程，并记录结果到 `specs/009-unified-link-interaction/validation/milestones.md`
- [ ] T061 [US3] 将服务端权威四端口默认和兼容保护作为聚焦 Git 提交，并把 commit SHA 写入 `specs/009-unified-link-interaction/validation/milestones.md`

**Checkpoint**: US3 可独立演示；所有创建控制面默认一致，旧实验室不会因升级自动改变端口或连接。

---

## Phase 6: User Story 4 - 实时、安全地完成统一连接 (Priority: P2)

**Goal**: 运行中资源可通过统一入口增删连接；并发、失败、取消、重启和主机恢复保持权威状态、durable task、观察能力和资源清理一致。

**Independent Test**: 两个浏览器与 HTTP/MCP 同时争用端口，在运行中的 QEMU、Docker、PC、L2/L3、Bridge/NAT 上执行创建、删除、取消、失败和重启；验证最多一个成功、2 秒收敛、3 秒明确状态、抓包/Traffic Filter 可用和删除实验室后零泄漏。

### Tests for User Story 4

- [ ] T062 [P] [US4] 为统一 create/delete task 的 queued/running/succeeded/failed/cancelling/cancelled、进度、最终查询和幂等结果编写失败测试到 `internal/app/command/topology_connection_tasks_test.go`
- [ ] T063 [P] [US4] 为事务中 task、backing record、预留、audit/outbox、revision 各故障点的全量回滚和 operation-owned cleanup 编写失败测试到 `internal/store/sqlite/topology_connection_repository_test.go`
- [ ] T064 [P] [US4] 为运行中节点链路、附件和对象链路适配、部分 runtime 失败与补偿编写失败测试到 `internal/app/reconcile/topology_connections_test.go`
- [ ] T065 [P] [US4] 为两个 HTTP 客户端与 MCP 同端口争用、兼容路由对称、冲突刷新和不同 key 重试编写失败测试到 `internal/api/http/topology_connection_concurrency_test.go` 与 `internal/api/mcp/topology_connection_tools_test.go`
- [ ] T066 [P] [US4] 为事件顺序、多客户端 store 收敛、提交后删除源/目标和最终 task 状态编写失败前端测试到 `web/src/stores/laboratory.connectionEvents.test.ts` 与 `web/src/features/topology/TopologyWorkspace.test.ts`
- [ ] T067 [P] [US4] 为三类统一连接的选择、删除、抓包、Wireshark 和 Traffic Filter 能力无回归编写失败测试到 `web/src/features/topology/TopologyInspector.test.ts`、`web/src/features/diagnostics/GlobalCaptureWorkspace.test.ts` 与 `web/src/features/diagnostics/TrafficFilterPanel.test.ts`
- [ ] T068 [P] [US4] 增加双浏览器、HTTP、MCP 十组并发端口争用和 2 秒收敛 Playwright 旅程到 `tests/e2e/journeys/unifiedConnectionConcurrency.spec.ts`
- [ ] T069 [P] [US4] 增加连接创建/删除/取消、服务重启、恢复和实验室删除零泄漏 Playwright 旅程到 `tests/e2e/journeys/unifiedConnectionRecovery.spec.ts`
- [ ] T070 [P] [US4] 扩展特权服务重启脚本，覆盖混合 backing connection 身份采用、端点预留、孤立资源和泄漏断言到 `acceptance/t225-service-restart.sh`

### Implementation for User Story 4

- [ ] T071 [US4] 在 `internal/app/command/topology_connection_tasks.go` 实现 create/delete/cancel task runner、进度、最终状态检查和 operation-owned 补偿清理
- [ ] T072 [US4] 在 `internal/app/reconcile/topology_connections.go` 与 `internal/app/reconcile/recovery_coordinator.go` 实现服务重启采用、缺失/孤立预留修复、部分资源清理和结构化恢复问题
- [ ] T073 [US4] 在 `internal/api/http/topology_handlers.go`、`internal/api/http/network_handlers.go`、`internal/api/mcp/tools.go` 与 `internal/api/mcp/network_tools.go` 将旧专用 mutation 适配到统一应用命令和 durable task 语义
- [ ] T074 [US4] 在 `internal/app/audit/service.go` 与 `internal/domain/topology_connection.go` 记录 entry point、端点/配置摘要、task、冲突和 cleanup，排除终端内容、包载荷和秘密
- [ ] T075 [US4] 在 `web/src/stores/laboratory.ts` 与 `web/src/features/topology/TopologyWorkspace.vue` 实现 ordered event 收敛、revision 冲突刷新、新 key 重试和已提交任务取消/最终查询
- [ ] T076 [US4] 在 `web/src/features/topology/TopologyInspector.vue`、`web/src/features/topology/LinkContextMenu.vue` 与 `web/src/features/diagnostics/GlobalCaptureWorkspace.vue` 统一三类连接的删除、抓包、Wireshark 和 Traffic Filter 能力入口
- [ ] T077 [US4] 在 `internal/app/reconcile/laboratory_deletion.go` 与 `internal/store/sqlite/topology_connection_repository.go` 清理实验室拥有的 backing records、预留、失败 operation 和运行时资源
- [ ] T078 [US4] 运行 T062–T070、20 次连接循环、恢复、抓包/Traffic Filter 和资源泄漏聚焦门禁，并记录结果到 `specs/009-unified-link-interaction/validation/milestones.md`
- [ ] T079 [US4] 将实时连接、并发、恢复和清理作为聚焦 Git 提交，并把 commit SHA 写入 `specs/009-unified-link-interaction/validation/milestones.md`

**Checkpoint**: US4 可独立验收；共享连接状态、任务、观察和清理不因入口或 backing kind 不同而分叉。

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: 同步合同和文档，完成全量质量门禁、可复现候选、目标机混合拓扑验收和回滚证据。

- [ ] T080 [P] 根据最终实现同步 `specs/009-unified-link-interaction/contracts/openapi.yaml`、`specs/009-unified-link-interaction/contracts/mcp-tools.json` 与 `web/src/api/generated.ts`
- [ ] T081 [P] 更新 `specs/009-unified-link-interaction/quickstart.md` 与 `acceptance/README.md`，记录最终测试文件、命令、目标端口变量、证据和回滚流程
- [ ] T082 [P] 运行键盘、axe、50 次拖拽性能、50%/100%/200% 缩放和 010 视觉回归检查，并将结果写入 `specs/009-unified-link-interaction/validation/final-local.md`
- [ ] T083 运行 `go test ./...` 及仓库规定的格式、静态分析、合同、并发、恢复和资源泄漏测试，并将结果写入 `specs/009-unified-link-interaction/validation/final-local.md`
- [ ] T084 运行 `npm run format:check`、`npm run lint`、`npm run build`、`npm test`、`npm run test:acceptance-unit` 和本地 Playwright，并将结果写入 `specs/009-unified-link-interaction/validation/final-local.md`
- [ ] T085 运行 `specs/001-network-simulator-platform/quickstart.md` 要求的适用特权 integration、recovery 和 leak 门禁，记录无法本地执行项目的原因与目标机命令到 `specs/009-unified-link-interaction/validation/final-local.md`
- [ ] T086 从干净 Git 提交构建部署候选，记录 commit SHA、制品摘要、迁移状态、合同版本和上一制品到 `specs/009-unified-link-interaction/validation/deployment.md`
- [ ] T087 将已记录候选部署到 `10.72.1.7`，按 `specs/009-unified-link-interaction/quickstart.md` 验证五类连接、加号、键盘、四端口、50 次拖拽、双客户端/API/MCP、运行中资源、抓包、Traffic Filter、重启和清理
- [ ] T088 将目标机结果、证据路径和资源清理摘要写入 `specs/009-unified-link-interaction/validation/target-acceptance.md`；若失败则回滚上一制品并返回本地修复、测试和新提交

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 — Setup**: 无依赖，可立即执行。
- **Phase 2 — Foundational**: 依赖 Phase 1；阻塞全部用户故事。
- **Phase 3 — US1**: 依赖 Phase 2；推荐作为 MVP 首先交付。
- **Phase 4 — US2**: 依赖 Phase 2；逻辑上可与 US1 并行，但 `TopologyCanvas.vue`、`TopologyWorkspace.vue` 和 `PortChooser.vue` 必须按文件顺序协调。
- **Phase 5 — US3**: 依赖 Phase 2；可与 US1/US2 并行，最终端口连接验收复用统一端点基础。
- **Phase 6 — US4**: 依赖 US1、US2、US3；统一验证所有入口、默认端口和 backing kind 的实时生命周期。
- **Phase 7 — Polish**: 依赖全部用户故事完成。

### User Story Dependency Graph

```text
Setup
  ↓
Foundational
  ├──→ US1 Direct Port Drag ──┐
  ├──→ US2 Unified Plus ──────┼──→ US4 Live/Safe Lifecycle ──→ Polish/Deploy
  └──→ US3 Four-Port Defaults ┘
```

### Within Each User Story

1. 编写并确认测试失败。
2. 实现领域/服务端或控制器基础。
3. 实现 HTTP/MCP/SPA 适配与组件集成。
4. 运行故事独立门禁和清理检查。
5. 形成聚焦 Git 提交并记录 SHA。

### File Coordination Rules

- `web/src/features/topology/TopologyCanvas.vue` 的 US1 与 US2 任务顺序执行。
- `web/src/features/topology/TopologyWorkspace.vue` 的 US1、US2 与 US4 任务顺序执行。
- `internal/store/sqlite/topology_connection_repository.go` 的 foundational 与 US4 清理任务顺序执行。
- `internal/app/command/topology_connection_tasks.go` 的 foundational 与 US4 生命周期任务顺序执行。
- `internal/api/mcp/tools.go` 和既有兼容工具变更在统一工具注册后执行。

---

## Parallel Execution Examples

### Foundational

- T005、T006、T007、T008、T009、T010 可并行编写失败测试。
- T017、T018、T019 可在 T011–T016 完成后并行实现不同适配层。

### User Story 1

- T023、T024、T025、T026、T027、T028 可并行编写。
- T032 可与 T029–T031 并行；T033–T036 在控制器和画布事件稳定后顺序集成。

### User Story 2

- T039、T040、T041、T042 可并行编写。
- T045 与 T046 可并行；T043、T044、T047 因共享画布/工作区状态顺序执行。

### User Story 3

- T050、T051、T052、T053、T054 可并行编写。
- T055、T057 可并行；T056、T058、T059 分别在服务端默认和前端草稿稳定后执行。

### User Story 4

- T062、T063、T064、T065、T066、T067、T068、T069、T070 可并行编写。
- T072、T074、T076 可在 T071 的 task 生命周期合同稳定后并行；T073、T075、T077 按共享文件和最终语义顺序集成。

---

## Implementation Strategy

### MVP First

1. 完成 Phase 1 和 Phase 2。
2. 完成 US1 的五类直接端口拖拽、取消和错误反馈。
3. 运行 US1 独立单元、合同和 Playwright 门禁。
4. 形成可演示的 MVP 提交，不等待 US2–US4。

### Incremental Delivery

1. **MVP**: 统一端点基础 + 直接拖拽。
2. **Interaction parity**: 增加加号和键盘入口，不复制提交逻辑。
3. **Creation parity**: 增加服务端权威四端口默认并保护旧对象。
4. **Operational hardening**: 增加并发、durable task、恢复、观察和清理。
5. **Release**: 全量本地门禁、干净候选、目标机验收和回滚记录。

### Success Conditions

- 每个用户故事都有独立失败测试、实现任务、局部门禁和聚焦提交。
- UI、HTTP、MCP 与兼容 mutation 最终调用同一应用命令。
- 具体端点跨三类连接最多被一个未删除连接占用。
- 新轻量 L2/L3 默认四端口，旧对象端口集合零变化。
- 010 连接视觉、平行线、抓包和 Traffic Filter 行为无回归。
- 服务/主机恢复和实验室删除后无幽灵连接、永久占用或运行时资源泄漏。
