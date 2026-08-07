# UI Contract: 统一拓扑连接与资源放置

## Purpose

本合同定义全部拓扑连接如何显示、选择、解释和叠加观察效果，以及新资源如何出现在不遮挡已有资源的位置。合同适用于浅色/深色主题、鼠标/键盘操作和浏览器刷新后的共享状态。

## Connection Normalization

1. `Link`、`NetworkAttachment` 和 `NetworkObjectLink` 在进入 ECharts/SVG 渲染前必须转换为同一 `ConnectionPresentation`。
2. 同一实际状态必须使用同一基础颜色、线型、宽度和状态文本，不得因持久化类型不同而改变。
3. 线路名称采用 `资源名:端口 ↔ 资源名:端口`；内部 ID 只能作为次级诊断信息。
4. 所有连接使用同一选择、右键、删除、抓包和 Traffic Filter 命中模型；不支持的能力显示明确禁用原因。
5. 连接端点、route group 和组内顺序在刷新、主题切换和多客户端之间保持稳定。

## Status Matrix

| Actual state | Base line | Required secondary cue | Chinese text |
|---|---|---|---|
| connected/running/active | success solid | normal endpoints | 已连接 |
| queued/pending/provisioning/starting/stopping | transition dashed | transition icon | 处理中 |
| failed/error/degraded | danger dotted/broken | warning marker | 失败/异常 |
| deleting/disconnecting | muted long-dashed | removal marker | 正在断开 |
| unknown/stopped | neutral | explicit state marker | 未知/未连接 |

颜色 token 必须来自主题变量；每种状态至少有一种非颜色线索。

## Semantic Marker Rules

1. 默认点到点连接不显示类型徽标。
2. NAT 管理上联显示 `managed-nat-uplink` 标记，说明该端点由 NetLab 管理地址转换和出口访问。
3. 共享 Bridge/交换域仅在多接入语义需要解释时显示 `shared-broadcast-domain` 标记。
4. 标记不得替换状态颜色，不得改变 Traffic Filter 方向，不得遮挡端口、节点名或线路选择区域。
5. 新标记必须同时增加中文名称、原因、图例样例、浅色/深色和无障碍测试。

## Visual Priority

从低到高的绘制顺序固定为：

1. 基础状态线；
2. 语义辅助描边或徽标；
3. hover、selected 和 keyboard focus；
4. capture active；
5. Traffic Filter 粒子和有期限方向提示；
6. 当前连接预览。

高层覆盖不得删除或伪造低层实际状态。Traffic Filter 停止并超过衰减时间后必须完全恢复基础边。

## Parallel Connections

1. 所有三类连接共享无方向 `route_group_key` 和稳定组内排序。
2. 两条连接使用对称偏移；三条及以上沿中心线对称分布，中心线最多一条。
3. 50%、100%、200% 缩放下每条连接都必须有独立可见路径和命中区域。
4. 激活一条线路的 Traffic Filter 不得点亮同端点间的其他接口线路。
5. 方向粒子沿实际 source → target 观察方向移动；双向流量可同时显示两个方向，但不能用永久双箭头掩盖发起方向。

## Dynamic Legend

1. 左下角状态图例解释基础状态；特殊语义图例只显示当前 topology 中实际存在的 marker。
2. 每项包含样例、中文名称、原因和匹配数量。
3. 聚焦或点击语义项可突出匹配连接；离开后恢复原状态。
4. 最后一条匹配连接删除后，对应语义项在共享快照更新后消失。
5. 图例不得重复资源类型颜色说明，也不得解释内部数据库连接类型。

## Authoritative Placement

1. 新建节点或网络对象时，UI 将当前画布中心作为可选 `placement_intent`，而不是最终坐标。
2. 创建响应中的 `placement_assignment.placement` 是唯一权威位置，必须覆盖临时预测。
3. 服务端调整位置时，UI 可显示非阻断提示“已避让现有资源”，但不得要求再次确认。
4. 新资源必须避开已有资源主体、名称、端口、连接加号和规范 clearance；现有资源不得被自动移动。
5. 未提供 intent 的 API/MCP 创建也必须获得可见且不重叠的权威位置。
6. 新资源在当前视口外时，UI 提供“定位新资源”，不得自动平移到导致用户失去上下文的位置。
7. 创建失败、取消或删除后不得显示幽灵节点、临时占位或永久碰撞框。

## Theme and Accessibility

1. 浅色和深色主题下状态颜色、语义徽标、焦点和文本均满足现有对比度门禁。
2. SVG 图标和线条使用主题 token，不保留浅色主题下不可见的硬编码白色细节。
3. 连接和图例均有包含端点、状态和语义的 accessible name。
4. 键盘可聚焦平行线路和图例项，焦点轮廓不被 canvas、底部面板或检查器裁剪。
5. 状态、方向和特殊语义均不能只依赖颜色或动画。

## Stability and Performance

1. ECharts graph 使用固定 `x/y` 和 `layout: none`；任何连接、hover、Traffic Filter 或主题更新不得启动 force layout。
2. 端口位置与节点位置分离计算，连接 option 更新不得重写节点坐标。
3. 投影、route grouping 和 legend 使用稳定键及 computed/memoized 派生。
4. 连续创建、观察更新和面板切换不得导致节点回弹、全图收缩或拖动后漂移。
5. 新资源共享事件到达后 2 秒内，所有活动客户端显示相同位置。

## Test Hooks

允许增加稳定的 `data-testid`、`data-connection-id`、`data-route-group`、`data-semantic-marker` 和 `data-placement-resource-id` 作为自动化测试接口；这些属性不得进入 API、MCP 或实验室持久化模型。
