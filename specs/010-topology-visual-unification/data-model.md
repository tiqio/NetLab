# Data Model: 拓扑连接视觉与资源放置统一

## Overview

本功能不合并现有三类连接，也不持久化 UI 线条样式。连接视觉和图例由服务器快照与前端主题确定性派生；新资源位置继续使用既有 `TopologyPlacement` 持久化，但首次位置必须在资源创建事务内权威分配。

## Existing Persisted Entities

### TopologyPlacement

共享且持久化的最终资源位置。

| Field | Type | Rules |
|---|---|---|
| `laboratory_id` | ID | 必须存在且拥有资源 |
| `resource_id` | ID | 节点或网络对象 ID；实验室内唯一 |
| `resource_type` | `node \| network_object` | 必须与真实资源类型一致 |
| `x` | float64 | 有限世界坐标，绝对值不超过领域上限 |
| `y` | float64 | 有限世界坐标，绝对值不超过领域上限 |
| `revision` | int64 | placement 乐观并发版本 |

**Lifecycle**: 资源创建时插入；拖动时 revisioned update；资源或实验室删除时级联删除；服务重启后原样恢复。

### Existing Connection Sources

- `Link`: 节点接口 ↔ 节点接口。
- `NetworkAttachment`: 节点接口 ↔ 网络对象端口。
- `NetworkObjectLink`: 网络对象端口 ↔ 网络对象端口。

三者继续保持现有持久化、所有权、desired/actual state 和 reconciliation 规则。

## Derived Presentation Entities

### ConnectionPresentation

画布、图例、选择、抓包和 Traffic Filter 共用的只读连接投影。

| Field | Type | Description |
|---|---|---|
| `id` | ID | 原始连接资源 ID |
| `persisted_kind` | enum | `node_link`, `network_attachment`, `network_object_link`；仅用于能力和调试，不直接决定颜色 |
| `source` | `ConnectionEndpointPresentation` | 规范化源端点 |
| `target` | `ConnectionEndpointPresentation` | 规范化目标端点 |
| `desired_state` | string | 原始 desired state |
| `actual_state` | string | 原始 actual/observed state |
| `status_visual` | `ConnectionStatusVisual` | 由实际状态派生 |
| `semantic_markers` | marker[] | 仅真实网络语义差异 |
| `route_group_key` | string | 无方向规范端点对，用于平行线分组 |
| `route_index` | integer | 稳定排序后的组内位置 |
| `route_count` | integer | 同端点路径数 |
| `label` | string | `资源:端口 ↔ 资源:端口` 中文可读名称 |
| `capabilities` | object | `selectable`, `deletable`, `capturable`, `traffic_filterable` |
| `accessibility_label` | string | 含端点、状态、语义和能力的说明 |

**Identity rule**: 投影 ID 等于原始连接 ID，不产生可写别名。

### ConnectionEndpointPresentation

| Field | Type | Description |
|---|---|---|
| `resource_id` | ID | 节点或网络对象 ID |
| `resource_type` | enum | `node`, `bridge`, `nat`, `pc`, `l2`, `l3` 等展示类型 |
| `resource_name` | string | 当前共享名称 |
| `port_id` | ID | 接口或网络对象端口 ID |
| `port_name` | string | `eth0`, `ens0`, `port1` 等 |
| `endpoint_key` | string | `resource_id/port_id` 规范键 |
| `symbol_role` | enum? | 可选 `nat`, `shared-domain`, `router` 等端点辅助符号 |

### ConnectionStatusVisual

| State family | Color role | Line style | Non-color cue |
|---|---|---|---|
| connected/running/active | `connection-success` | solid | normal endpoint markers |
| queued/pending/provisioning/starting/stopping | `connection-transition` | dashed | transition text/icon |
| failed/error/degraded | `connection-danger` | dotted or broken | warning marker and error label |
| deleting/disconnecting | `connection-muted` | long-dashed | removal marker |
| unknown/stopped | `connection-neutral` | solid or muted | explicit state text |

`selected`, `focused`, `capture_active` 和 `traffic_active` 是覆盖层，不写入此实体。

### ConnectionSemanticMarker

| Field | Type | Rules |
|---|---|---|
| `key` | string enum | 初始支持 `managed-nat-uplink`, `shared-broadcast-domain` |
| `label_zh` | string | 面向用户的中文名称 |
| `reason_zh` | string | 为什么视觉上保留差异 |
| `icon` | semantic token | 使用现有 Lucide/拓扑符号 token，不持久化 SVG |
| `pattern` | enum | `endpoint_badge`, `center_badge`, `secondary_stroke` |

标记不能覆盖基础状态颜色，新增标记必须同步 UI 合同和测试。

### TopologyLegendItem

| Field | Type | Description |
|---|---|---|
| `key` | string | 状态或语义键 |
| `category` | `status \| semantic` | 图例分组 |
| `label_zh` | string | 中文名称 |
| `description_zh` | string | 状态说明或差异原因 |
| `visual_sample` | object | 与连接实际渲染共用的 token |
| `connection_ids` | ID[] | 当前匹配连接集合 |
| `count` | integer | 当前连接数量 |

**Derivation**: 状态项可固定展示；语义项只在 `count > 0` 时出现。图例不持久化。

## Placement Request and Allocation Entities

### PlacementIntent

创建命令中的瞬时请求。

| Field | Type | Rules |
|---|---|---|
| `preferred_x` | float64? | 成对提供；有限且在世界坐标范围内 |
| `preferred_y` | float64? | 成对提供；有限且在世界坐标范围内 |
| `footprint_class` | enum? | 省略时由资源类型/模板推导 |
| `viewport_hint` | object? | 可选可见区域提示，只影响候选优先级，不构成持久化状态 |

API/MCP 调用方不能提交任意宽高。无 intent 时使用实验室默认创建锚点与确定性序列。

### PlacementFootprint

版本化的服务端规范足迹。

| Field | Type | Rules |
|---|---|---|
| `class` | enum | `node-standard`, `node-wide`, `network-object-standard`, `network-object-wide` 等受控值 |
| `width` | world units | 覆盖主体和左右端口余量 |
| `height` | world units | 覆盖主体、名称、状态和连接入口 |
| `clearance_x` | world units | 与邻近资源的最小水平间距 |
| `clearance_y` | world units | 与邻近资源的最小垂直间距 |
| `version` | integer | 算法/合同版本；已有 placement 不因升级重算 |

### PlacementCandidate

分配算法内部值，不持久化。

| Field | Type | Description |
|---|---|---|
| `x`, `y` | float64 | 候选中心 |
| `ring` | integer | 相对首选中心的搜索环 |
| `ordinal` | integer | 环内稳定顺序 |
| `bounds` | rectangle | 足迹加 clearance 后边界 |
| `collision_ids` | ID[] | 冲突 placement，用于内部诊断 |
| `within_bounds` | boolean | 是否在领域坐标范围内 |

候选按 `(ring, ordinal)` 升序检查，首个无碰撞候选获选。

### PlacementAssignment

创建响应和审计中的结果。

| Field | Type | Description |
|---|---|---|
| `placement` | `TopologyPlacement` | 已提交权威坐标 |
| `requested_center` | point? | 调用方首选位置 |
| `assigned_center` | point | 最终位置 |
| `adjusted` | boolean | 最终位置是否偏离首选 |
| `reason` | enum | `preferred_available`, `collision_avoided`, `default_anchor`, `viewport_adjusted` |
| `footprint_class` | enum | 实际采用的受控足迹 |
| `algorithm_version` | integer | 用于审计和可复现测试 |

## Relationships

```text
CreateNode/CreateNetworkObject
        │ includes optional
        ▼
 PlacementIntent ──resolve──▶ PlacementCandidate
        │                          │
        │                          ▼ first valid
        └──────────────────▶ PlacementAssignment
                                   │
                 same transaction  ▼
 Resource + TopologyPlacement + Laboratory revision + Outbox event

Link / NetworkAttachment / NetworkObjectLink
        └──derive──▶ ConnectionPresentation
                         ├──derive──▶ TopologyLegendItem[]
                         └──render──▶ base edge + interaction/traffic overlays
```

## Placement Claim Lifecycle

1. **requested**: 创建命令通过校验并解析 intent、资源类型和默认 footprint。
2. **calculating**: 在写事务内读取实验室已有 placements，按稳定顺序测试候选。
3. **committed**: 资源、placement、实验室 revision 和 outbox 原子提交，返回 assignment。
4. **rejected**: revision 冲突、请求越界、资源验证失败或候选耗尽；不写资源和 placement。
5. **cancelled**: 若上层任务在提交前取消，则事务回滚；提交后按正常资源删除工作流清理。

## Connection Presentation Transitions

```text
pending/queued ──reconcile──▶ connected
      │                           │
      └────────failure──────────▶ failed

connected ──delete request──▶ disconnecting ──cleanup──▶ removed
connected ──runtime loss────▶ failed/degraded ──retry──▶ connected
```

- Traffic Filter observation 不改变上述状态，只产生有期限的覆盖层。
- Capture active 不改变基础状态，只显示捕获徽标。
- 语义标记由端点/网络对象类型派生，状态变化不删除其含义。

## Validation Rules

1. 坐标必须有限、成对提供且在 `MaxPlacementCoordinate` 范围内。
2. `footprint_class` 必须受支持，并与资源类型兼容。
3. 新 placement 的规范边界加 clearance 后不得与同实验室已提交 placement 相交。
4. 计算只移动新资源，绝不修改已有 placement。
5. 并发创建必须在相同实验室 revision/事务序列下产生不同位置或明确冲突。
6. 相同 idempotency key 和等价请求返回同一资源与 placement；同 key 不同请求返回冲突。
7. 创建失败、取消和实验室删除后不得残留 placement、预留或孤立 outbox 事件。
8. `route_group_key` 必须与端点方向无关；组内排序在刷新和客户端之间稳定。
9. 连接颜色只能来自 `status_visual`；semantic marker、selected、capture、traffic 不能改写实际状态文本。
10. 所有图例语义项必须至少关联一条当前连接，连接消失后相应项同步消失。

## Persistence Summary

| Entity | Persisted | Authority |
|---|---|---|
| Existing three connection models | Yes | Server/runtime reconciliation |
| `TopologyPlacement` | Yes | Server/SQLite |
| `PlacementIntent` | No | Request only |
| `PlacementCandidate` | No | Transaction-local |
| `PlacementAssignment` metadata | Response/audit only | Server command result |
| `ConnectionPresentation` | No | Deterministic projection |
| `ConnectionSemanticMarker` | No | Versioned application/UI rules |
| `TopologyLegendItem` | No | Current topology projection |
