# Implementation Plan: 拓扑连接视觉与资源放置统一

**Branch**: `[010-topology-visual-unification]` | **Date**: 2026-08-07 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/010-topology-visual-unification/spec.md`

**Note**: 本功能依赖 `specs/009-unified-link-interaction/spec.md` 约定的统一连线入口；本计划负责统一连接展示、状态语义、动态图例和新资源的共享无碰撞放置，不合并既有运行时连接模型，也不自动重排已有实验室。

## Summary

为节点直连、节点到 Bridge/NAT/轻量对象及网络对象直连建立统一的 `ConnectionPresentation` 投影，让相同实际状态始终使用相同颜色、线型和可访问文本，仅以轻量语义标记解释 NAT 上联、共享广播域等真实差异；同时把新资源位置从“客户端创建后补写坐标”升级为创建命令内的服务器权威位置分配，使用固定世界坐标足迹、确定性最近候选搜索和 SQLite 事务避免浏览器、API 与 MCP 并发创建重叠。现有坐标、三类持久化连接、运行时实现、抓包和 Traffic Filter 能力保持不变，前端只消费统一投影并叠加选择、流量和抓包效果。

## Technical Context

**Language/Version**: Go 1.26.x；TypeScript 5.8.3；Vue 3.5.18；CSS/Tailwind CSS 4.3

**Primary Dependencies**: Gin 1.12.x、SQLite WAL、Pinia 3.0、ECharts 6.1、Vue Router 4.5、现有 Shadcn Vue/Reka UI 基础组件、Lucide Vue、xterm.js、noVNC

**Storage**: 复用 SQLite `topology_placements` 与实验室 revision/outbox；扩展创建事务写入权威位置，不新增连接持久化表；连接展示和动态图例为派生数据

**Testing**: Go `testing` 的领域、应用、SQLite 合同、并发、恢复和清理测试；Vitest、Vue Test Utils、Playwright、axe-core、前端构建/格式/中文化扫描；目标机特权集成与资源泄漏验收

**Target Platform**: 单台 x86_64 Linux 主机，具备 KVM/QEMU、Docker、cgroup v2、netlink、nftables 和抓包工具；最终部署目标 `10.72.1.7`；现代桌面浏览器最小视口 1024×768

**Project Type**: 现有 Go HTTP/MCP 服务 + Vue SPA；本功能横跨领域投影、创建命令、SQLite repository、HTTP/MCP 合同和拓扑画布

**Performance Goals**: 连接状态在 3 秒内可识别；新资源创建成功后 2 秒内可见或可定位；并发客户端 2 秒内收敛；连续创建 20 个混合资源无初始重叠；三条平行连接在 50%、100%、200% 缩放下保持独立可选；拖动和 Traffic Filter 动画不触发全图重新布局

**Constraints**: 保留已有资源坐标；不做自动全局重排；不改变 VLAN、路由、NAT 或运行时接线语义；所有创建入口共享同一命令和 revision/idempotency 规则；碰撞判断不得依赖浏览器像素测量；浅色/深色和键盘操作不能仅靠颜色表达状态

**Scale/Scope**: QEMU、Docker、PC、Bridge、NAT、轻量 L2、轻量 L3 七类资源；`Link`、`NetworkAttachment`、`NetworkObjectLink` 三类连接；节点与网络对象创建 API/MCP；已有 placement 移动接口；拓扑画布、图例、选择、抓包和 Traffic Filter 覆盖层

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

### Pre-Research Gate

- **PASS — Shared state**: 连接实际状态继续来自服务器权威快照；创建位置在节点或网络对象创建事务内计算并持久化，浏览器、API、MCP 与并发客户端都消费同一权威坐标，手工拖动仍使用实验室 revision 和 placement revision。
- **PASS — Control parity**: UI、HTTP API 和 MCP 的资源创建均接受相同可选 `placement_intent` 并返回权威 placement；不引入 UI 专属位置写入或视觉状态修改路径。
- **PASS — Runtime scope**: 不增加运行时类型和特权原语；保留 QEMU、Docker、Linux bridge、NAT 与 netns 的既有三类连接持久化和 reconciliation。
- **PASS — Live operations**: 运行中连线、QMP 热插拔、抓包和 Traffic Filter 行为不变；统一展示只读取实际状态，并保证临时观察效果结束后恢复基础状态。
- **PASS — Safety and recovery**: 位置分配在 SQLite 事务中校验实验室 revision、资源所有权、幂等键、坐标边界和碰撞；失败或取消不留下 placement/预留；服务重启从持久化位置和连接快照恢复。
- **PASS — Verification**: 设计覆盖投影单测、主题/无障碍/缩放测试、API/MCP 合同、SQLite 并发创建、失败回滚、服务重启恢复、目标机真实混合拓扑和删除后泄漏检查。
- **PASS — Image and secret hygiene**: 不新增或修改设备镜像，不保存凭据、bootstrap secret、抓包内容或目标机敏感数据；验收只记录资源 ID、状态、坐标和制品摘要。
- **PASS — Local-first delivery**: 四个实现里程碑分别要求本地测试和聚焦 Git 提交；仅从干净提交构建带摘要的候选制品并部署到 `10.72.1.7`，禁止直接修改目标机源码，保留上一候选用于回滚。

### Post-Design Gate

- **PASS — Authority remains singular**: `PlacementIntent` 是瞬时请求，`TopologyPlacement` 才是共享事实；视觉投影不产生第二套可写状态。
- **PASS — Contracts remain symmetric**: OpenAPI 与 MCP 使用相同 placement intent、footprint class、adjustment metadata、revision 和 idempotency 语义。
- **PASS — Runtime ownership unchanged**: 三类连接和宿主机资源所有权不迁移；仅增加派生展示适配器和创建事务内 placement 写入。
- **PASS — Failure is reversible**: 创建、位置计算和 outbox 写入作为原子工作流提交；冲突返回结构化错误，重试不会产生幽灵资源或永久占位。
- **PASS — Acceptance is measurable**: `quickstart.md` 覆盖 20 资源、双客户端/API 并发、平行线、主题、Traffic Filter 衰减、重启恢复、旧坐标不变和实验室删除清理。

## Project Structure

### Documentation (this feature)

```text
specs/010-topology-visual-unification/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── openapi.yaml
│   ├── mcp-tools.json
│   └── ui-contract.md
└── tasks.md
```

### Source Code (repository root)

```text
internal/
├── domain/
│   ├── models.go
│   ├── operations.go
│   └── topology_placement.go
├── app/
│   ├── command/
│   │   ├── node.go
│   │   ├── topology_placement.go
│   │   └── network_objects.go
│   └── query/
├── api/
│   ├── http/
│   │   ├── node_handlers.go
│   │   ├── network_object_handlers.go
│   │   └── topology_placement_handlers.go
│   └── mcp/
└── store/sqlite/
    ├── laboratory_repository.go
    ├── network_repository.go
    └── topology_placement_repository.go

web/
├── src/
│   ├── api/generated.ts
│   ├── stores/laboratories.ts
│   └── features/topology/
│       ├── TopologyCanvas.vue
│       ├── TopologyWorkspace.vue
│       ├── topologyVisualSemantics.ts
│       ├── linkPresentation.ts
│       ├── topologyLayout.ts
│       ├── topologyPlacementBatch.ts
│       ├── topologyConnectionController.ts
│       └── topologyGeometry.ts
└── tests/acceptance/
```

**Structure Decision**: 保留现有 Go 分层与 Vue feature 目录。后端只扩展创建命令和 placement repository 的原子能力；前端在 `features/topology` 内增加统一连接投影和权威位置收敛逻辑，不把 ECharts、Gin、SQLite 或运行时实现泄漏到领域规则。

## Implementation Milestones

### Milestone 1 — Unified Connection Presentation

- 建立覆盖三类连接的规范化展示投影、统一状态优先级和跨类型平行路径分组。
- 让 ECharts 基础边只消费投影；选择、抓包和 Traffic Filter 作为不改变基础状态的覆盖层。
- 增加单元、组件、缩放、方向、主题与无障碍测试。
- 本地门禁通过后创建独立 Git 提交并记录 SHA。

### Milestone 2 — Semantic Markers and Dynamic Legend

- 为确有差异的 NAT 管理上联、共享广播域等添加非颜色语义标记和中文解释。
- 左下角图例仅显示当前实验室实际出现的标记；状态图例与资源类型图例不重复。
- 验证无特殊差异时不产生噪声，浅色/深色与键盘读取一致。
- 本地门禁通过后创建独立 Git 提交并记录 SHA。

### Milestone 3 — Authoritative Collision-Free Creation

- 为节点和网络对象创建合同增加可选 placement intent，并在同一 SQLite 事务内选择、校验和提交最终位置。
- 使用规范化足迹和确定性方形螺旋搜索；已有坐标永不移动；API/MCP 无坐标创建也自动取得位置。
- 前端以返回位置为准，支持创建后定位；保留现有 placement batch 用于拖动。
- 增加并发、幂等、revision、失败回滚、密集拓扑和旧数据兼容测试。
- 本地门禁通过后创建独立 Git 提交并记录 SHA。

### Milestone 4 — Recovery and Target Validation

- 验证服务重启、浏览器刷新、双客户端和 outbox 事件后连接视觉与位置收敛。
- 在 `10.72.1.7` 以混合资源 Lab 验证全部连接、20 次连续创建、平行线、Traffic Filter 衰减及删除清理。
- 记录候选 ID、commit SHA、制品摘要、部署时间、测试结果和上一候选回滚信息。
- 目标验收修复必须回到本地形成新提交和新制品。

## Agent Context Update

当前 Spec Kit 分发不包含 `.specify/scripts/bash/update-agent-context.sh` 或等价 CLI。已按仓库既有规划惯例手工更新根目录 `AGENTS.md`：活动功能指向本规格、计划和合同，记录 `009` 依赖，并补充服务器权威初始位置、保留既有坐标和连接展示/运行时模型分离规则；现有技术栈、安全、测试和本地优先交付约束继续有效。

## Complexity Tracking

无宪法例外或需保留的复杂度偏差。
