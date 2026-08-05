# Research: 拓扑添加抽屉

## Decision 1: 增强现有 Sheet，而不是引入新的抽屉依赖

**Decision**: 扩展 `web/src/components/ui/sheet/Sheet.vue`，增加完整高度布局、独立滚动、固定 footer、受控关闭、键盘焦点和焦点恢复能力；保持现有 `side` 与基础 slot 用法兼容。

**Rationale**: 项目已经把 Sheet 用于紧凑视口下的 Devices、Inspector 和 Operations。增强现有基础组件可保持统一视觉与交互约定，避免出现创建抽屉和工作区抽屉两套行为，也不增加依赖。

**Alternatives considered**:

- 在创建组件中手写专用 fixed panel：改动较少，但会重复遮罩、Escape、焦点和滚动逻辑，后续容易再次分叉。
- 引入新的第三方 Drawer：项目已有可用基础依赖和设计体系，新增依赖无法提供足够额外价值。
- 继续使用 Dialog 只扩大宽度：无法解决保留画布上下文、固定操作区和右侧工作流的问题。

## Decision 2: 抽离资源目录供左侧 Palette 与右侧 Drawer 复用

**Decision**: 将模板加载、搜索、分类和 `PaletteSelection` 生成抽离为 `TopologyResourceCatalog.vue`；`DevicePalette.vue` 和创建抽屉复用同一目录组件。

**Rationale**: 现有左侧设备栏已经包含完整资源分类。直接复制到抽屉会产生模板过滤、名称和不可用状态的双重维护；复用目录可让用户保留熟悉的左侧入口，同时允许工具栏直接打开右侧选择步骤。

**Alternatives considered**:

- 抽屉只能接收左侧 Palette 的预选：实现最小，但不满足从添加入口直接在抽屉中查找资源，也不利于紧凑视口。
- 把 DevicePalette 整体移动到抽屉：会移除 EVE-NG 风格的长期可见设备栏并增加认知变化。

## Decision 3: 将草稿和负载构造从视图中分离

**Decision**: 用纯 TypeScript 草稿模型集中处理默认值、字段变更、脏状态、验证、模板切换和 `createNode`/`createNetworkObject` 请求负载构造；抽屉组件负责渲染和调用。

**Rationale**: 当前创建 Dialog 同时承担目录刷新、兼容性判断、表单状态、验证、负载构造和提交，文件持续增长。将确定性规则抽离后可直接单测长表单和类型切换，不需要依赖 Teleport 或 DOM 查询。

**Alternatives considered**:

- 仅把 Dialog 标签替换为 Sheet：能快速改变外观，但无法可靠实现脏数据切换确认、错误字段定位和持续扩展。
- 为每种资源建立完全独立表单：隔离度高，但会重复名称、模板、镜像、接口和错误处理逻辑，当前范围不需要如此拆分。

## Decision 4: 使用显式抽屉状态机

**Decision**: 抽屉采用 `closed → selecting → editing → submitting → succeeded/failed` 的显式状态；放弃确认作为覆盖在 selecting/editing 上的临时状态。

**Rationale**: 该流程需要区分尚未选择资源、编辑草稿、提交锁定和可恢复失败。显式状态可防止重复提交、多个抽屉实例和关闭时绕过确认。

**Alternatives considered**:

- 继续组合多个布尔值：符合当前代码，但容易出现 `open=false`、`busy=true` 或 selection 丢失等非法组合。
- 使用全局 Pinia store：抽屉草稿是单组件、单浏览器的短生命周期状态，不需要全局持久化或跨页面共享。

## Decision 5: 成功后关闭并由 Workspace 执行权威刷新

**Decision**: 抽屉提交成功后发出创建结果，由 `TopologyWorkspace.created()` 继续刷新活动实验室、合并响应资源、创建默认 placement 并选中新资源；抽屉本身不直接修改画布集合。

**Rationale**: 现有 Workspace 已封装创建后收敛逻辑。保留该边界可避免产生 UI-only 幽灵节点，并维持多客户端和事件流的一致性。

**Alternatives considered**:

- 抽屉直接把节点追加到画布：响应快但会绕过 Workspace 的 placement 和权威刷新规则。
- 成功后保持抽屉打开用于批量添加：用户未要求批量模式，且会增加草稿重置和重复提交歧义。

## Decision 6: 不修改后端创建合同

**Decision**: 继续使用现有模板/镜像查询、`POST /labs/{labId}/nodes` 和 `POST /labs/{labId}/network-objects`；不新增数据库字段、API 或 MCP 方法。

**Rationale**: 问题来自呈现、滚动和前端状态组织，现有后端已支持全部目标资源。保持合同不变可以缩小风险，并证明 UI 与自动化客户端仍观察相同资源。

**Alternatives considered**:

- 新增“创建草稿”服务端资源：会把浏览器本地便利状态变成共享状态，增加清理和并发复杂度。
- 新增批量创建接口：超出本次范围。

## Decision 7: 以真实浏览器矩阵验证而非仅做快照测试

**Decision**: 在组件测试之外，新增 Playwright 流程覆盖 1024×768、1366×768、1920×1080、键盘、独立滚动、脏数据确认、失败保留和真实资源创建。

**Rationale**: 本功能的核心风险是布局、焦点和点击行为，静态 DOM 测试无法证明操作区不会被裁切、页面不会跟随滚动或 Inspector 能正确恢复。

**Alternatives considered**:

- 仅保留现有 Dialog 单测：无法覆盖右侧布局和真实浏览器行为。
- 只做目标机手工测试：不可重复，无法阻止未来回归。

## Compatibility History Findings

- `Sheet.vue` 自恢复基线提交 `74dadf3` 起仅提供遮罩、方向和单一滚动 slot；增强时必须保持 `v-model`、`side`、`title` 以及 LaboratoryShell 中 Devices、Inspector、Operations 的现有调用兼容。
- `CreateTopologyResourceDialog.vue` 的提交 `1a45c22` 明确要求镜像按设备模板族隔离；抽屉迁移不得退回为仅按 QEMU/Docker runtime kind 过滤。
- `CreateTopologyResourceDialog.vue` 的提交 `96473ee` 增加 Docker 静态地址、SLAAC 和静态路由负载；草稿模型必须保留 `network_interfaces.routes` 构造和相关测试。
- `TopologyWorkspace.vue` 的创建回调负责权威刷新、合并响应资源、中心 placement 和选中状态；Drawer 不得直接建立 UI-only 节点或绕过该回调。
- Workspace 的 placement 是跨客户端共享状态，而 viewport、手工链路路径和 Drawer 草稿保持浏览器本地；打开 Drawer 不得调用 fit/reset/pan/zoom 或修改 placement。
- PC Console、network-object link 删除、Traffic Filter overlay 等后续提交均复用同一 Workspace 选择与 Operations 状态；关闭 Drawer 时只能恢复有效本地选择，不得覆盖 Drawer 打开期间收到的权威事件。
