# Data Model: 统一拓扑端口连接体验

## 1. Connection Endpoint

用户与控制面共同使用的规范端点。

| Field | Type | Rules |
|---|---|---|
| `kind` | enum | `node_interface`、`network_object_port`、`network_object_access` |
| `laboratory_id` | ID | 两端必须属于同一实验室 |
| `resource_id` | ID | 节点或网络对象 ID |
| `resource_kind` | string | QEMU、Docker、PC、switch L2/L3、Bridge、NAT 等 |
| `port_id` | ID? | `node_interface` 必填；其他类型为空 |
| `port_name` | string? | 节点接口和命名对象端口必填；逻辑接入为空 |
| `display_name` | string | `资源名:端口` 或 `资源名:逻辑接入口` |
| `capabilities` | set | 可连接类别、L2/L3、VLAN 配置、抓包与运行时热接线能力 |
| `availability` | enum | `free`、`reserved`、`occupied`、`reconciling`、`unavailable`、`incompatible` |
| `unavailable_reason` | string? | 可操作的中文原因，不含秘密或载荷 |

### Validation

- `node_interface` 必须解析到当前实验室中存在且未被统一预留的 `NodeInterface`。
- `network_object_port` 必须存在于网络对象实际保存的 `ports` 或 `interfaces` 配置中。
- `network_object_access` 仅适用于声明多接入能力的 Bridge/NAT 等对象，不创建固定命名端口。
- 两端不得相同、跨实验室或违反资源自连规则。
- 端点组合必须能映射到唯一 backing kind。

## 2. Topology Connection

统一控制面与 UI 使用的连接资源；运行时仍由 backing model 拥有。

| Field | Type | Rules |
|---|---|---|
| `id` | ID | 全局唯一；沿用 backing record ID |
| `laboratory_id` | ID | 与两端一致 |
| `source` | ConnectionEndpoint | 保存规范方向；视觉可按无方向关系分组 |
| `target` | ConnectionEndpoint | 保存规范方向 |
| `backing_kind` | enum | `link`、`network_attachment`、`network_object_link`；服务端派生，只读 |
| `backing_id` | ID | 通常等于统一连接 ID |
| `config` | object | L2 PVID/tagged 等连接参数；按组合验证 |
| `revision` | integer | mutation 必须携带期望 revision |
| `desired_state` | enum | `connected`、`disconnected` |
| `observed_state` | enum | `pending`、`connected`、`failed`、`disconnecting`、`disconnected`、`cancelled` |
| `last_error` | Problem? | 阶段、清理状态和操作建议 |
| `capabilities` | set | select、delete、capture、wireshark、traffic_filter、reconnect 等 |
| `created_at` / `updated_at` | timestamp | UTC |

### Backing Mapping

| Endpoint A | Endpoint B | Backing Kind |
|---|---|---|
| node interface | node interface | `link` |
| node interface | named object port | `network_attachment` |
| node interface | Bridge/NAT access | `network_attachment` |
| named object port | named object port on another object | `network_object_link` |

未列出的组合返回 `endpoint_incompatible`，不得隐式创建额外端口或新网络对象。

## 3. Connection Draft

单浏览器瞬态状态，不持久化、不进入 laboratory snapshot。

| Field | Type | Rules |
|---|---|---|
| `mode` | enum | `idle`、`choosing_source`、`targeting`、`choosing_target`、`configuring`、`submitting` |
| `entry_point` | enum | `port_click`、`port_drag`、`resource_plus`、`keyboard` |
| `source` | ConnectionEndpoint? | 进入 targeting 前必须唯一 |
| `pointer_screen` | Point? | 拖拽时更新 |
| `pointer_world` | Point? | 用于规范目标解析 |
| `candidate_target` | ConnectionEndpoint/Resource? | hover 时派生 |
| `candidate_state` | enum | `connectable`、`unavailable`、`incompatible`、`ambiguous` |
| `chooser_anchor` | Point? | 多端口 chooser 的屏幕位置 |
| `target_candidates` | Endpoint[] | 只包含当前权威快照中合法候选 |
| `config` | object | 尚未提交的 VLAN 等参数 |

### State Transitions

```text
idle
  -> choosing_source
  -> targeting
  -> choosing_target | configuring
  -> submitting
  -> idle

choosing_source | targeting | choosing_target | configuring
  -> cancelled
  -> idle
```

窗口失焦、文档隐藏、实验室切换、Escape、空白点击或再次选择同一源端点均取消草稿且不发送 mutation。

## 4. Endpoint Reservation

SQLite 中跨 backing kind 的端点占用真相。

| Field | Type | Rules |
|---|---|---|
| `laboratory_id` | ID | 复合主键一部分 |
| `owner_type` | enum | `node_interface`、`network_object` |
| `owner_id` | ID | 接口 ID 或网络对象 ID |
| `port_name` | string | 节点接口实际名、对象命名端口；逻辑 access 不写唯一端口记录 |
| `resource_type` | enum | 三类 backing kind |
| `resource_id` | ID | 拥有此预留的连接 ID |
| `operation_id` | ID | 创建/删除 task ID，用于恢复和清理 |
| `state` | enum | `reserved`、`occupied`、`releasing` |
| `created_at` | timestamp | UTC |

### Invariants

- 复合主键确保一个具体端点最多属于一个未删除连接。
- 连接记录、两端预留、task、audit/outbox 和 laboratory revision 在同一事务提交。
- 失败创建只能删除属于当前 operation 的预留，不能释放其他连接占用。
- 服务恢复时 backing record 是最终真相；孤立预留被清理，缺失预留被安全重建或标记问题。

## 5. Connection Task

| Field | Type | Rules |
|---|---|---|
| `id` | ID | durable operation ID |
| `kind` | enum | `topology_connection_create`、`topology_connection_delete` |
| `idempotency_key` | string | 同 kind 下唯一；等价请求返回相同结果 |
| `request_fingerprint` | string | 规范端点、配置、实验室和 mutation 的摘要 |
| `state` | enum | `queued`、`running`、`succeeded`、`failed`、`cancelling`、`cancelled` |
| `progress` | integer | 0–100 |
| `connection_id` | ID | 创建预分配或删除目标 |
| `result` | object? | 最终统一连接和 laboratory revision |
| `problem` | Problem? | 结构化失败 |
| `cleanup_state` | enum | `not_required`、`pending`、`complete`、`failed` |
| `created_at` / `started_at` / `finished_at` | timestamp? | UTC |

### Idempotency

- 相同 key 与相同 fingerprint 返回同一 task 和连接结果。
- 相同 key 与不同端点、配置或 mutation 返回 `idempotency_conflict`。
- 客户端在 revision 刷新后重试冲突请求必须生成新 key。

## 6. Lightweight Port Set

### L2 Default

```text
ports = [
  {name: eth0, pvid: 1, tagged: []},
  {name: eth1, pvid: 1, tagged: []},
  {name: eth2, pvid: 1, tagged: []},
  {name: eth3, pvid: 1, tagged: []}
]
```

### L3 Default

```text
interfaces = [
  {name: eth0, addresses: []},
  {name: eth1, addresses: []},
  {name: eth2, addresses: []},
  {name: eth3, addresses: []}
]
```

### Rules

- 只在创建请求没有明确端口集合时应用默认值。
- SPA、HTTP 和 MCP 创建必须得到相同默认配置。
- 用户可在提交前增删、重命名和配置端口；名称必须唯一且符合现有限制。
- 更新、导入、恢复和数据库迁移不得自动补齐旧对象。

## 7. Audit Event

| Field | Type | Rules |
|---|---|---|
| `entry_point` | enum | SPA 手势、HTTP、MCP、兼容端点 |
| `laboratory_id` | ID | 必填 |
| `connection_id` | ID? | 提交后存在 |
| `source_summary` / `target_summary` | object | 类型、资源和端口摘要 |
| `config_summary` | object | VLAN 等非秘密字段 |
| `task_id` | ID? | mutation 对应任务 |
| `outcome` | enum | submitted、conflict、failed、cancelled、cleaned |
| `problem_code` | string? | 结构化错误码 |
| `cleanup_state` | string? | 最终清理摘要 |

不得包含终端内容、抓包载荷、密码、bootstrap secret 或与连接无关的配置。
