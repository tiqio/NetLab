# Implementation Plan: 统一拓扑端口连接体验

**Branch**: `[009-unified-link-interaction]` | **Date**: 2026-08-10 | **Spec**: `specs/009-unified-link-interaction/spec.md`

**Input**: Feature specification from `/specs/009-unified-link-interaction/spec.md`

**Note**: 本计划与 `specs/010-topology-visual-unification/` 协同：009 统一端点选择、连接命令与生命周期，010 已建立的连接展示投影、平行线路、状态视觉、Traffic Filter 覆盖和权威 placement 保持不变。

## Summary

为 QEMU、Docker、PC、轻量 L2/L3、Bridge 和 NAT 建立统一连接端点与连接命令。SPA 的端口拖拽、右上角加号、键盘选择和资源主体投放共享同一个客户端连接草稿控制器；HTTP 与 MCP 使用同一应用命令，将节点链路、网络附件和网络对象链路映射到既有持久化及运行时模型。SQLite 在一个事务内验证实验室 revision、预留全部具体端点、创建对应连接记录、持久化幂等任务并写出站事件，从而串行裁决并发端口争用。新建轻量 L2/L3 在服务端创建默认配置时生成 `eth0`–`eth3`，已有对象、导入数据和更新请求不自动扩口。

## Technical Context

**Language/Version**: Go 1.26.x；TypeScript 5.8.x；Vue 3.5.x

**Primary Dependencies**: Gin 1.12.x、modernc.org/sqlite 1.54.x、Pinia 3.0.x、ECharts 6.1.x、Vite 7.0.x、Vitest 3.2.x、Playwright 1.54.x；既有 QEMU、Docker、Linux netlink/nftables/runtime adapters

**Storage**: SQLite WAL；保留 `links`、`network_attachments`、`network_object_links`，扩展统一端点预留、幂等任务、outbox 和审计记录

**Testing**: `go test`、Go 合同/SQLite/恢复/运行时测试、Vitest、Vue Test Utils、axe-core、Playwright、本地 acceptance、`10.72.1.7` 特权验收与资源泄漏检查

**Target Platform**: 单台 x86_64 Linux 主机，cgroup v2、KVM/QEMU、Docker、Linux bridge、network namespace、netlink、nftables 和抓包工具

**Project Type**: Go Web 服务 + Vue 单页应用 + MCP 控制面

**Performance Goals**: 端口拖拽预览在正常浏览器负载下保持 60 FPS；连接提交后 3 秒内进入已连接或明确失败；两个客户端在 2 秒内收敛；50 次拖拽不触发节点移动或视口漂移

**Constraints**: 不合并三类运行时连接模型；不停止正在运行的资源；不自动修改既有轻量设备端口；不扩大宿主机权限；连接草稿不进入共享状态；连接创建、删除和恢复必须幂等、可取消、可审计、可清理

**Scale/Scope**: 单实验室混合 QEMU、Docker、PC、轻量 L2/L3、Bridge、NAT；至少覆盖五类连接组合、平行连接、两个浏览器与 HTTP/MCP 并发，以及 20 次连续连接/删除/重连循环

## Constitution Check

*GATE: Phase 0 前检查通过；Phase 1 设计后再次通过。*

- **PASS — Shared state**: 服务端权威状态为三类持久化连接、统一端点预留、实验室 revision、任务和 outbox；连接草稿、指针位置和端口选择器只存在于当前浏览器。SQLite 事务、revision 与唯一预留共同裁决并发。
- **PASS — Control parity**: SPA、HTTP 与 MCP 均提交相同 `ConnectionEndpoint`、配置、revision 和 idempotency key，并进入 `TopologyConnectionCommandService`；旧专用端点作为兼容包装器复用同一命令。
- **PASS — Runtime scope**: 节点链路、附件和对象链路继续由既有 QEMU、Docker、bridge、NAT、namespace 与 Linux 网络适配器执行，不引入多机调度或新特权。
- **PASS — Live operations**: 连接创建/删除以 durable task 表示排队、执行、成功、失败和取消；运行中的资源不因接线停止。抓包、Wireshark 和 Traffic Filter 继续消费 010 的统一连接展示与既有制品语义。
- **PASS — Safety and recovery**: 端点能力、实验室归属、端口存在性、占用和容量在提交前后验证；事务失败不留下连接、预留或 outbox；reconciler 在服务/主机恢复时采用归属资源并清理失败任务残留。
- **PASS — Verification**: 计划包含领域、命令、SQLite、HTTP/MCP 合同、前端控制器、手势互斥、无障碍、并发、恢复、抓包/Traffic Filter、特权运行时和泄漏测试。
- **PASS — Image and secret hygiene**: 功能不新增镜像、凭据或抓包样本；测试只使用仓库允许的模板元数据与短生命周期生成流量，证据记录不包含终端内容或包载荷。
- **PASS — Local-first delivery**: 四个本地里程碑分别运行聚焦门禁并形成独立提交；仅从干净提交构建候选，记录 SHA、摘要、迁移状态和回滚制品后部署 `10.72.1.7`，禁止目标机直接改源码。

### Post-Design Re-check

- **PASS — Domain boundaries preserved**: `ConnectionEndpoint`、兼容规则、连接状态和轻量默认端口位于 domain/application；Gin、SQLite、ECharts 和宿主机命令不进入领域规则。
- **PASS — Persistence and runtime ownership preserved**: 统一控制面不替换三张连接表，也不改变 runtime reconciler 的资源所有权；统一端点预留只负责跨类型占用裁决。
- **PASS — Failure is observable and reversible**: 每次变更返回 durable task 与统一连接快照；失败问题包含端点、阶段、清理状态和操作建议，等价重试返回同一结果。
- **PASS — Existing laboratories remain stable**: 旧连接加载后仅使用统一入口与 010 视觉；已有轻量设备不补端口，placement 和连接关系不改变。
- **PASS — Contracts remain symmetric**: OpenAPI、MCP 与 UI contract 使用相同端点类型、配置、任务、错误、revision 和 idempotency 语义。

## Project Structure

### Documentation (this feature)

```text
specs/009-unified-link-interaction/
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
│   ├── topology_connection.go
│   └── lightweight_ports.go
├── app/
│   ├── command/
│   │   ├── link.go
│   │   ├── topology_connection.go
│   │   └── topology_connection_tasks.go
│   └── reconcile/
│       ├── network_objects.go
│       ├── topology.go
│       └── topology_connections.go
├── api/
│   ├── http/
│   │   ├── topology_connection_handlers.go
│   │   └── routes.go
│   └── mcp/
│       └── topology_connection_tools.go
└── store/sqlite/
    ├── migrations/
    ├── laboratory_repository.go
    ├── network_repository.go
    └── topology_connection_repository.go

web/
├── src/
│   ├── api/
│   │   ├── generated.ts
│   │   └── index.ts
│   ├── stores/laboratory.ts
│   ├── features/nodes/lightweightSwitchConfig.ts
│   └── features/topology/
│       ├── TopologyCanvas.vue
│       ├── TopologyWorkspace.vue
│       ├── PortChooser.vue
│       ├── interactionTypes.ts
│       ├── topologyConnectionController.ts
│       ├── topologyEndpointCompatibility.ts
│       ├── topologyGeometry.ts
│       └── topologyInteractionController.ts
├── acceptance/
└── tests/

tests/e2e/
├── fixtures/
└── journeys/
```

**Structure Decision**: 保留现有 Go 分层、三类连接 repository/runtime 路径和 Vue topology feature。新增统一连接领域投影、应用命令、HTTP/MCP 适配器和前端草稿控制器；兼容端点转发到统一命令，避免 SPA 或外部客户端继续复制底层类别判断。

## Desired State and Transactions

1. 客户端提交两个规范化端点、可选连接配置、实验室 revision 和 idempotency key。
2. 应用命令按端点组合确定 backing kind，但不要求调用方选择 `link`、`network_attachment` 或 `network_object_link`。
3. SQLite 单事务校验实验室、资源、端口、能力和 revision；为两个具体端点写唯一预留；创建 backing record、operation task、审计摘要和 outbox；更新实验室 revision。
4. 提交后 task runner 调用既有 runtime adapter；统一连接的 observed state 从 backing record 与任务结果派生。
5. 失败或取消执行补偿清理并释放仅属于失败操作的预留；成功连接保留预留直到删除完成。
6. 删除命令解析统一连接 ID，进入同一任务语义；运行时删除成功后删除 backing record 和端点预留并发布 outbox。

## State Transitions

- **Connection Draft**: `idle → choosing_source → targeting → choosing_target/configuring → submitting → idle`；任一选择态可通过 Escape、空白点击、同源再次选择、实验室切换、窗口失焦进入 `cancelled → idle`。
- **Topology Connection**: `pending → connected | failed`；删除为 `connected|failed → disconnecting → disconnected`；提交前取消不创建共享连接，提交后取消由 task 进入 `cancelling → cancelled|failed|succeeded` 并以最终权威状态为准。
- **Endpoint Availability**: `free → reserved → occupied`；事务失败回到 `free`，删除成功回到 `free`，恢复期间可暂为 `reconciling` 但不得被新请求占用。

## Implementation Milestones

### Milestone 1 — Unified Endpoint and Command Foundation

- 定义统一端点、连接、配置、能力、状态和错误模型。
- 扩展 SQLite 端点预留覆盖节点接口与网络对象端口，并通过同一事务创建 backing record、task、audit 和 outbox。
- 新增统一 HTTP/MCP create/list/get/delete 合同；旧连接端点复用统一应用命令。
- 先完成领域、repository、幂等、revision、并发冲突与合同失败测试，再实现代码并形成聚焦提交。

### Milestone 2 — Unified Canvas Interaction

- 将点击端口、端口拖拽、资源右上角加号、资源主体投放和键盘流程统一到 `TopologyConnectionController`。
- 使用 SVG overlay pointer capture 和独立阈值隔离端口拖拽、节点拖动、框选、平移及缩放；预览线始终锚定源端点。
- 目标端口唯一时自动选择，多端口时在操作位置附近打开 chooser；L2 配置在提交前确认。
- 验证节点到节点、节点到对象、对象到对象、节点到 Bridge/NAT 及取消/错误反馈并形成聚焦提交。

### Milestone 3 — Four-Port Defaults and Live Lifecycle

- 服务端创建默认与 SPA 编辑器统一生成轻量 L2/L3 的 `eth0`–`eth3`；API/MCP 缺省配置获得相同结果，已有对象和导入数据保持原样。
- 连接创建、删除、取消和恢复统一返回 durable task；补齐运行中资源、部分失败、服务重启、并发端口争用、审计和清理测试。
- 验证抓包、Wireshark、Traffic Filter、平行线路和 010 视觉无回归并形成聚焦提交。

### Milestone 4 — Acceptance, Candidate and Target Validation

- 运行 Go 全量、前端格式/静态分析/单元/acceptance-unit、Playwright 本地场景和资源泄漏检查。
- 从干净提交构建候选，记录 commit SHA、制品 digest、迁移状态、合同版本、构建时间和上一制品。
- 部署到 `10.72.1.7`，验证混合运行资源、两客户端/API/MCP 并发、50 次拖拽、20 次连接循环、抓包/Traffic Filter、重启恢复和实验室删除清理。
- 任一失败回滚上一候选，并返回本地修复、测试、提交和重新构建。

## Agent Context Update

当前仓库没有 `.specify/scripts/bash/update-agent-context.sh`。本次规划手工更新根目录 `AGENTS.md`，将 009 设为活动功能，并记录 010 视觉兼容依赖、统一命令/端点预留、客户端草稿隔离和新建轻量设备四端口规则。

## Complexity Tracking

无宪法例外。保留三类 backing model 是既有运行时所有权边界，不属于新增复杂度偏差。
