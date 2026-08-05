# Tasks: 拓扑添加抽屉

**Input**: Design documents from `/specs/006-topology-add-drawer/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: 本功能明确要求组件、集成、键盘、可访问性、真实浏览器和目标机测试。所有测试任务应先于对应实现完成，并先证明当前行为不满足新合同。

**Organization**: 任务按用户故事组织，使右侧添加抽屉、长表单可靠性、关闭与工作区恢复可以分阶段独立验收。

## Format: `[ID] [P?] [Story] Description`

- **[P]**: 可在不同文件中并行执行，且不依赖尚未完成的任务
- **[Story]**: 对应 `spec.md` 中的用户故事
- 每个任务均包含明确文件路径

## Phase 1: Setup（兼容性调查与证据准备）

**Purpose**: 在修改既有 Sheet、创建流程和 Workspace 前确认历史意图，并建立本功能的验证记录。

- [x] T001 检查 `web/src/components/ui/sheet/Sheet.vue`、`web/src/features/topology/CreateTopologyResourceDialog.vue`、`web/src/features/topology/DevicePalette.vue`、`web/src/features/topology/TopologyWorkspace.vue` 的 `git log` 与 `git blame`，将必须保留的兼容行为记录到 `specs/006-topology-add-drawer/research.md`
- [x] T002 创建实现验证记录模板，包含本地测试、里程碑提交、候选摘要、部署、回滚和目标机结果字段，写入 `specs/006-topology-add-drawer/validation.md`

---

## Phase 2: Foundational（共享 Sheet 基础）

**Purpose**: 提供所有用户故事依赖的完整高度、独立滚动、固定头尾和受控关闭基础。

**⚠️ CRITICAL**: 本阶段完成前不得开始创建抽屉集成。

- [x] T003 [P] 为 right/left/bottom 布局、固定 header/footer、独立 body 滚动、Escape、遮罩关闭请求和关闭后焦点恢复编写失败测试于 `web/src/components/ui/sheet/Sheet.test.ts`
- [x] T004 扩展共享 Sheet 的描述、尺寸、header/body/footer、统一 close-request、焦点约束和向后兼容 props/slots 于 `web/src/components/ui/sheet/Sheet.vue`
- [x] T005 [P] 为紧凑视口 Devices、Inspector、Operations Sheet 的既有行为补充回归测试于 `web/src/components/shell/LaboratoryShell.test.ts`
- [x] T006 调整紧凑 Workspace 对增强 Sheet 的使用并保持原有 panel 行为于 `web/src/components/shell/LaboratoryShell.vue`
- [x] T007 运行 `Sheet.test.ts`、`LaboratoryShell.test.ts`、前端类型检查、lint 和格式检查，并把结果记录到 `specs/006-topology-add-drawer/validation.md`
- [x] T008 将 Phase 2 作为独立里程碑提交，确认提交后工作区干净并捕获提交 SHA，供 T020 写入 `specs/006-topology-add-drawer/validation.md`

**Checkpoint**: 共享 Sheet 可以承载创建抽屉，原有工作区 Sheet 没有退化。

---

## Phase 3: User Story 1 — 从右侧抽屉添加拓扑资源 (Priority: P1) 🎯 MVP

**Goal**: 用户可以从右侧抽屉查找或预选资源，完成 QEMU、Docker、PC 和网络对象创建，并在不刷新页面的情况下看到和选中新资源。

**Independent Test**: 在一个实验室中依次通过抽屉创建 QEMU、Docker、PC、Bridge、NAT Bridge、Lightweight L2 和 Lightweight L3，确认抽屉位于最右侧、画布 viewport 不变、成功资源立即出现并被选中。

### Tests for User Story 1

- [x] T009 [P] [US1] 为模板加载、搜索、QEMU/Docker/网络对象分类、不可用原因和选择事件编写失败测试于 `web/src/features/topology/TopologyResourceCatalog.test.ts`
- [x] T010 [P] [US1] 将现有七类资源创建请求和模板/镜像兼容性断言迁移为抽屉合同测试于 `web/src/features/topology/CreateTopologyResourceDrawer.test.ts`
- [x] T011 [P] [US1] 为工具栏打开选择步骤、DevicePalette 预选打开编辑步骤、单实例抽屉、成功刷新和新资源选择编写失败测试于 `web/src/features/topology/TopologyWorkspace.test.ts`
- [x] T012 [P] [US1] 为右侧抽屉可见性、画布 viewport 保持、七类资源创建和无需刷新显示编写 Playwright 场景于 `tests/e2e/journeys/topologyAddDrawer.spec.ts`

### Implementation for User Story 1

- [x] T013 [US1] 从 DevicePalette 抽离共享模板查询、搜索、分类和选择呈现到 `web/src/features/topology/TopologyResourceCatalog.vue`
- [x] T014 [US1] 重构左侧设备栏复用共享资源目录且保持现有 EVE-NG 风格入口于 `web/src/features/topology/DevicePalette.vue`
- [x] T015 [US1] 创建支持 selecting/editing/submitting/failed 状态、右侧固定布局和现有全部资源表单的 `web/src/features/topology/CreateTopologyResourceDrawer.vue`
- [x] T016 [US1] 将现有创建负载、模板刷新、镜像兼容性、cloud-init 和网络对象配置逻辑从 `web/src/features/topology/CreateTopologyResourceDialog.vue` 迁移到 `web/src/features/topology/CreateTopologyResourceDrawer.vue`
- [x] T017 [US1] 在 `web/src/features/topology/TopologyWorkspace.vue` 接入唯一抽屉实例、工具栏选择入口、DevicePalette 预选入口和现有 `created()` 权威刷新路径
- [x] T018 [US1] 更新拓扑页面对象的抽屉定位、资源选择和提交助手于 `tests/e2e/pages/TopologyPage.ts`
- [x] T019 [US1] 删除被替代的 `web/src/features/topology/CreateTopologyResourceDialog.vue` 与 `web/src/features/topology/CreateTopologyResourceDialog.test.ts`，并清理所有旧 Dialog 引用
- [x] T020 [US1] 运行 US1 组件、Workspace 和 Playwright 测试及前端构建，把结果和 T008 的 Sheet 里程碑 SHA 记录到 `specs/006-topology-add-drawer/validation.md`
- [x] T021 [US1] 将 US1 MVP 作为独立里程碑提交，确认提交后工作区干净并捕获提交 SHA，供 T031 写入 `specs/006-topology-add-drawer/validation.md`

**Checkpoint**: 用户故事 1 可独立交付；所有支持资源均可从右侧抽屉创建并收敛到权威拓扑。

---

## Phase 4: User Story 2 — 安全填写长配置表单 (Priority: P2)

**Goal**: 长表单滚动、默认值、脏状态、类型切换、字段错误、目录变化和重复提交均可预测且不丢失草稿。

**Independent Test**: 使用字段最多的 QEMU/Docker 模板填写到表单底部，触发字段错误、服务端错误、模板变化和快速双击提交，确认草稿保留、首个错误获得焦点且只创建一个资源。

### Tests for User Story 2

- [x] T022 [P] [US2] 为 Add Drawer Session、Resource Selection、Resource Create Draft 的初始化、规范化签名、脏状态和状态转换编写失败测试于 `web/src/features/topology/topologyResourceDraft.test.ts`
- [x] T023 [P] [US2] 为 QEMU、Docker、PC、Bridge、NAT、L2、L3 默认值、验证和请求负载构造编写表驱动失败测试于 `web/src/features/topology/topologyResourceDraft.test.ts`
- [x] T024 [P] [US2] 为长表单独立滚动、类型/模板切换确认、字段错误定位、结构化失败保留和提交锁编写失败测试于 `web/src/features/topology/CreateTopologyResourceDrawer.test.ts`
- [x] T025 [P] [US2] 扩展真实浏览器场景覆盖 1024×768 长表单、页面不滚动、固定操作区、失败恢复和双击提交于 `tests/e2e/journeys/topologyAddDrawer.spec.ts`

### Implementation for User Story 2

- [x] T026 [US2] 实现纯 TypeScript 草稿模型、默认值、规范化签名、验证、类型切换和创建负载构造于 `web/src/features/topology/topologyResourceDraft.ts`
- [x] T027 [US2] 重构抽屉使用草稿模型并保持滚动、折叠分组和仍适用字段值于 `web/src/features/topology/CreateTopologyResourceDrawer.vue`
- [x] T028 [US2] 实现首个无效字段定位、字段级中文错误、全局结构化错误、目录过期提示和可恢复重试于 `web/src/features/topology/CreateTopologyResourceDrawer.vue`
- [x] T029 [US2] 实现提交期间输入/提交锁和目录提交前复核，确保重复点击不产生重复请求于 `web/src/features/topology/CreateTopologyResourceDrawer.vue`
- [x] T030 [US2] 更新表单与对话框交互矩阵以覆盖抽屉长表单和错误恢复于 `tests/e2e/matrices/formAndDialogMatrix.spec.ts`
- [x] T031 [US2] 运行 US2 草稿、抽屉、交互矩阵和 Playwright 测试及前端构建，把结果和 T021 的 US1 里程碑 SHA 记录到 `specs/006-topology-add-drawer/validation.md`
- [ ] T032 [US2] 将 US2 作为独立里程碑提交，确认提交后工作区干净并捕获提交 SHA，供 T042 写入 `specs/006-topology-add-drawer/validation.md`

**Checkpoint**: 用户故事 2 可独立验收；最长表单和所有失败路径不会丢失有效草稿或创建重复资源。

---

## Phase 5: User Story 3 — 可预测地关闭和恢复工作区 (Priority: P3)

**Goal**: 用户通过关闭按钮、Escape、遮罩、资源切换或实验室切换离开时得到一致保护，关闭后 Inspector、选择和焦点恢复，键盘流程完整可用。

**Independent Test**: 在 Inspector 和底部 Operations 展开时打开抽屉，修改字段后分别使用关闭按钮、Escape、遮罩和实验室切换，验证放弃确认、继续编辑、焦点约束和 Workspace 状态恢复。

### Tests for User Story 3

- [ ] T033 [P] [US3] 为 dirty Sheet 关闭确认、所有 close reason 统一路径、alertdialog 焦点和取消放弃编写失败测试于 `web/src/components/ui/sheet/Sheet.test.ts`
- [ ] T034 [P] [US3] 为抽屉脏状态关闭、资源切换、实验室切换、关闭后触发焦点恢复编写失败测试于 `web/src/features/topology/CreateTopologyResourceDrawer.test.ts`
- [ ] T035 [P] [US3] 为 Inspector 折叠/宽度/选择快照和画布 viewport 不变编写失败测试于 `web/src/features/topology/TopologyWorkspace.test.ts`
- [ ] T036 [P] [US3] 为全键盘打开、选择、填写、提交、Escape、放弃确认和焦点恢复编写 Playwright 场景于 `tests/e2e/frontend_responsive_keyboard.spec.ts`
- [ ] T037 [P] [US3] 为抽屉和放弃确认运行 axe 可访问性规则编写测试于 `web/src/features/topology/CreateTopologyResourceDrawer.a11y.test.ts`

### Implementation for User Story 3

- [ ] T038 [US3] 在共享 Sheet 中实现 `preventClose`、close reason、受控放弃确认、焦点陷阱和触发焦点恢复于 `web/src/components/ui/sheet/Sheet.vue`
- [ ] T039 [US3] 在创建抽屉中连接脏状态、关闭/切换确认和实验室冲突保护于 `web/src/features/topology/CreateTopologyResourceDrawer.vue`
- [ ] T040 [US3] 在 Workspace 中记录并恢复 Inspector Snapshot、有效选择和触发焦点，同时保持画布和 Operations 状态于 `web/src/features/topology/TopologyWorkspace.vue`
- [ ] T041 [US3] 更新 LaboratoryShell 与工具栏的可访问名称、添加入口和紧凑视口行为于 `web/src/components/shell/LaboratoryShell.vue` 和 `web/src/features/laboratories/LaboratoryToolbar.vue`
- [ ] T042 [US3] 运行 US3 Sheet、Drawer、Workspace、axe 和键盘 Playwright 测试及前端构建，把结果和 T032 的 US2 里程碑 SHA 记录到 `specs/006-topology-add-drawer/validation.md`
- [ ] T043 [US3] 将 US3 作为独立里程碑提交，确认提交后工作区干净并捕获提交 SHA，供 T047 写入 `specs/006-topology-add-drawer/validation.md`

**Checkpoint**: 用户故事 3 可独立验收；所有关闭方式和键盘操作均安全，Workspace 上下文恢复且没有焦点陷阱。

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 完成全量回归、文档、干净构建、正式部署和目标机验收。

- [ ] T044 [P] 更新交互覆盖清单中的旧 Dialog 控件和新 Drawer 控件于 `tests/e2e/matrices/interaction-inventory.json`
- [ ] T045 [P] 更新现有创建流程页面对象与旅程中的 Dialog role/名称定位于 `tests/e2e/pages/TemplatePage.ts`、`tests/e2e/journeys/networkObjectLinks.spec.ts`、`tests/e2e/journeys/ubuntuBootstrapOverlay.spec.ts` 和 `tests/e2e/journeys/dockerStaticRoutes.spec.ts`
- [ ] T046 [P] 根据最终实现更新用户可见行为和实际测试命令于 `specs/006-topology-add-drawer/quickstart.md` 和 `specs/006-topology-add-drawer/contracts/ui-contract.md`
- [ ] T047 运行 `npm run lint`、`npm run format:check`、`npm test`、`npm run build`、相关 Go 合同测试和本地 Playwright 套件，并把全部结果、任何无关已知失败和 T043 的 US3 里程碑 SHA 记录到 `specs/006-topology-add-drawer/validation.md`
- [ ] T048 对 1024×768、1366×768、1920×1080 三种视口连续运行两次抽屉 Playwright 场景，记录无裁切、无重复创建、无 viewport 漂移结果到 `specs/006-topology-add-drawer/validation.md`
- [ ] T049 提交 T044-T048 涉及的文件和 `specs/006-topology-add-drawer/validation.md`，确认提交后工作区为空并捕获最终源 commit SHA
- [ ] T050 从 T049 的干净提交生成候选元数据，记录 source commit SHA、version、candidate ID、contract digest 和 built-at 到 `/var/tmp/netlab-candidates/006-topology-add-drawer/release.json`
- [ ] T051 从 T049 的干净提交构建部署产物、计算 binary digest，并保存到 `/var/tmp/netlab-candidates/006-topology-add-drawer/netlabd` 和 `/var/tmp/netlab-candidates/006-topology-add-drawer/release.json`
- [ ] T052 使用 `deploy/scripts/install.sh` 将 T051 产物正式部署到 `10.72.1.7`，把部署时间、发布信息、服务健康和回滚候选保存到 `/var/tmp/netlab-add-drawer-acceptance/deployment.json`
- [ ] T053 在 `10.72.1.7` 新建 `Topology Add Drawer Acceptance` Lab，通过真实浏览器抽屉创建 Ubuntu QEMU、BusyBox Docker、PC 和 Lightweight L2 或 NAT Bridge，并保存结果到 `/var/tmp/netlab-add-drawer-acceptance/resource-creation.json`
- [ ] T054 使用第二浏览器验证只同步成功资源、不共享抽屉草稿/滚动/错误状态，并把 API 最终资源对比保存到 `/var/tmp/netlab-add-drawer-acceptance/multi-client.json`
- [ ] T055 在目标机触发一次模板或镜像过期/不可用失败，验证草稿保留、中文下一步和零幽灵资源，并保存结果到 `/var/tmp/netlab-add-drawer-acceptance/failure-path.json`
- [ ] T056 删除目标验收 Lab 及其节点、网络对象和链路，验证拓扑、任务和运行资源清理，并保存结果到 `/var/tmp/netlab-add-drawer-acceptance/cleanup.json`
- [ ] T057 将 T049 源 SHA、T051 产物摘要和 T052-T056 目标机结果汇总到 `specs/006-topology-add-drawer/validation.md`，创建仅包含验收证据的最终文档提交并注明实际部署源仍为 T049

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 Setup**: 无依赖，可立即开始。
- **Phase 2 Foundational**: 依赖 Phase 1；阻塞全部用户故事。
- **US1 / Phase 3**: 依赖 Phase 2；形成可交付 MVP。
- **US2 / Phase 4**: 依赖 US1 的抽屉和目录，但草稿单测 T022/T023 可在 US1 收尾期间准备。
- **US3 / Phase 5**: 依赖 US1 的抽屉；Sheet 关闭测试 T033 和 axe 测试 T037 可与 US2 后段并行准备。
- **Phase 6 Polish**: 依赖 US1、US2、US3 及各里程碑提交全部完成。

### User Story Dependency Graph

```text
Setup
  └── Foundational Sheet
        └── US1 Right-side creation drawer (MVP)
              ├── US2 Long-form reliability
              └── US3 Close/focus/workspace restoration
                    └── Polish, clean build, deployment, target acceptance, evidence commit
```

US2 与 US3 在 US1 完成后可以大部分并行，但二者都可能修改 `CreateTopologyResourceDrawer.vue`，实际执行时应分支或串行整合以避免冲突。

### Within Each User Story

- 先编写测试并确认在旧实现上失败。
- 再实现纯模型和基础组件，随后接入视图与 Workspace。
- 先运行聚焦测试，再运行该故事相关构建和浏览器测试。
- 每个故事完成后必须创建聚焦 Git 提交，未提交改动不得进入下一部署阶段。

## Parallel Opportunities

### Foundational

```text
并行：T003 Sheet 合同测试 + T005 LaboratoryShell 回归测试
随后：T004/T006 实现与整合
```

### User Story 1

```text
并行：T009 Catalog 测试 + T010 Drawer 创建合同测试 + T011 Workspace 测试 + T012 E2E 场景
随后：T013 Catalog → T014 DevicePalette；T015/T016 Drawer → T017 Workspace → T018/T019 清理
```

### User Story 2

```text
并行：T022/T023 草稿模型测试 + T024 Drawer 失败路径测试 + T025 浏览器长表单场景
随后：T026 草稿实现 → T027/T028/T029 Drawer 集成；T030 可与后段并行
```

### User Story 3

```text
并行：T033 Sheet 关闭测试 + T034 Drawer 关闭测试 + T035 Workspace 恢复测试 + T036 键盘 E2E + T037 axe 测试
随后：T038 Sheet → T039 Drawer；T040 Workspace 与 T041 Shell/Toolbar 可并行
```

### Polish

```text
并行：T044 交互清单 + T045 旧 E2E 定位迁移 + T046 文档更新
随后：T047/T048 全量验证 → T049-T051 干净提交与构建 → T052-T056 目标机串行验收与清理 → T057 证据提交
```

## Implementation Strategy

### MVP First

1. 完成 Phase 1 和 Phase 2。
2. 完成 US1，交付可以创建全部资源的右侧抽屉。
3. 独立运行 US1 组件、Workspace 和真实浏览器验收。
4. 提交 MVP 里程碑后再增加长表单和关闭恢复能力。

### Incremental Delivery

1. **Milestone 1**: 共享 Sheet 基础且现有 Shell 无退化。
2. **Milestone 2**: US1 右侧创建抽屉和全部资源创建能力。
3. **Milestone 3**: US2 长表单、验证、错误恢复和提交锁。
4. **Milestone 4**: US3 关闭确认、焦点和 Workspace 恢复。
5. **Candidate**: 全量回归后从干净提交构建并部署目标机。

### Completion Definition

- 所有任务使用严格 checkbox、Task ID、可选 `[P]` 和用户故事标签格式。
- US1、US2、US3 均具有独立测试标准和里程碑提交。
- 所有旧 Dialog 测试和 E2E 定位已迁移，不保留双重创建界面。
- 本地构建和相关测试通过，任何无关失败均有可复现记录。
- `10.72.1.7` 上四类代表资源创建、多客户端隔离、失败恢复和清理均通过。
