# Research: 统一拓扑端口连接体验

## 1. 使用统一控制面，不合并三类持久化连接

**Decision**: 新增 `TopologyConnection` 应用投影与命令，根据端点组合映射到现有 `Link`、`NetworkAttachment` 或 `NetworkObjectLink`。

**Rationale**: 用户不应识别底层类别，但三类模型拥有不同运行时资源、reconciliation 和删除路径。统一命令可消除 UI/API/MCP 分叉，同时保护 010 已统一的视觉投影和现有恢复逻辑。

**Alternatives considered**:

- 合并为单表：拒绝；迁移和运行时回归范围过大。
- 只在 SPA 判断类别：拒绝；HTTP/MCP 仍不对称，并发和幂等无法统一。
- 保留 Inspector 附件入口作为唯一节点到对象路径：拒绝；不满足统一拖拽与加号体验。

## 2. 使用带判别字段的统一端点

**Decision**: 端点类型为 `node_interface`、`network_object_port` 和 `network_object_access`。端点携带实验室、资源、具体接口/端口、显示名、能力和可用性。

**Rationale**: 普通节点和轻量设备有具体端口；Bridge/NAT 是多接入逻辑目标，不应伪造固定用户端口。判别联合能同时支持 UI、OpenAPI、MCP 和审计。

**Alternatives considered**:

- 统一成一个字符串：拒绝；难以验证归属、跨实验室和能力。
- 为 Bridge/NAT 永久生成固定端口：拒绝；改变多接入语义和容量模型。
- 让调用方直接提交 backing kind：拒绝；继续暴露内部类别。

## 3. 跨连接类别使用统一端点预留

**Decision**: 扩展 `topology_endpoint_reservations`，同时预留节点接口和网络对象命名端口；Bridge/NAT 逻辑接入不使用单端口唯一约束，但仍受对象容量和宿主机限制。

**Rationale**: 现有节点链路依赖 `interfaces.desired_link_id`，对象端口使用独立预留；节点附件可能被前端误判为空闲。统一预留使节点链路、附件和对象链路在一个 SQLite 事务中互斥。

**Alternatives considered**:

- 只补前端占用判断：拒绝；不能裁决并发客户端。
- 将附件 ID 写入 `desired_link_id`：拒绝；字段语义和旧代码都假定节点链路。
- 使用进程内锁：拒绝；服务重启后失效，且不与事务状态一致。

## 4. 创建、删除和取消全部使用 durable task

**Decision**: 统一连接命令返回连接快照与 durable task；附件路径从同步 `void` 结果提升为同一任务语义。任务记录幂等、进度、错误、取消和清理结果。

**Rationale**: 实时接线涉及 QMP、tap、bridge、namespace 和 nftables 等多步骤操作，HTTP 请求生命周期不足以表达最终状态。统一任务也满足多客户端和 MCP 观察需求。

**Alternatives considered**:

- 所有操作同步等待：拒绝；超时和部分失败难以恢复。
- 仅节点链路使用任务：拒绝；不同底层类别继续产生不同反馈。
- 失败后由用户手工刷新：拒绝；无法证明资源已清理。

## 5. 客户端连接草稿保持本地

**Decision**: `ConnectionDraft` 只保存在 `TopologyWorkspace`/controller，包含源端点、指针、候选目标、chooser 位置和待确认配置；不写 Pinia 共享快照、数据库或 outbox。

**Rationale**: 拖拽预览是瞬态手势，不应传播给其他浏览器。提交后只展示服务端返回的 pending 连接和 task。

**Alternatives considered**:

- 将预览广播给所有客户端：拒绝；产生干扰和无意义冲突。
- 在画布组件内分散多个布尔状态：拒绝；点击、拖拽和键盘会继续漂移。
- 提交前创建临时连接记录：拒绝；取消会产生幽灵资源清理负担。

## 6. 使用 SVG overlay pointer capture 隔离端口拖拽

**Decision**: 端口 SVG hit area 在 pointer down 后捕获指针，超过独立阈值才进入拖拽；期间禁用节点拖动、框选和平移，并由统一 controller 更新预览和目标反馈。

**Rationale**: 当前端口已经位于 ECharts 上方的 SVG overlay，直接在该层捕获手势能避免 ECharts 节点 draggable 与画布 roam 同时响应。

**Alternatives considered**:

- 依赖 HTML5 drag-and-drop：拒绝；SVG、触控和坐标转换行为不稳定。
- 在 ECharts graph node 上模拟端口：拒绝；端口标签与独立命中区难以稳定。
- 全局 document mousemove：拒绝；窗口失焦和组件卸载更易遗留状态。

## 7. 资源主体投放使用确定性候选解析

**Decision**: 画布以 010 的规范 footprint 和当前端口 overlay 解析指针下资源；一个可用端口自动选择，多个端口打开就近 chooser，无可用端口或不兼容时不提交。

**Rationale**: 资源主体投放必须与缩放无关且不能依赖浏览器 DOM 测量；010 已提供稳定世界坐标和足迹类。

**Alternatives considered**:

- 总是打开 Inspector：拒绝；丢失当前拖拽上下文。
- 自动选择第一个端口：拒绝；多端口设备会产生不可预期连接。
- 使用节点图标像素边界：拒绝；不同主题和缩放下不稳定。

## 8. 加号、点击、拖拽和键盘共享同一状态机

**Decision**: 加号仅负责选择或请求选择源端点；之后全部入口调用同一兼容性、目标选择、配置、提交和取消函数。

**Rationale**: 统一状态机才能保证源端口唯一时自动选择、多端口 chooser、Escape/空白取消和错误反馈一致。

**Alternatives considered**:

- 为加号保留独立流程：拒绝；会再次形成两套规则。
- 仅支持拖拽：拒绝；破坏既有习惯和键盘可访问性。
- 端口单击立即创建：拒绝；无法处理配置确认和多端口目标。

## 9. 四端口默认值必须服务端权威

**Decision**: 新建 `switch_l2` 和 `switch_l3` 在 config 未明确提供端口集合时由应用/domain 默认生成 `eth0`–`eth3`；SPA 使用同一规则初始化可编辑草稿。更新、导入和恢复不应用默认扩口。

**Rationale**: 仅前端默认无法覆盖 HTTP/MCP。只在创建缺省配置时应用可保证新资源一致，同时不修改既有实验室。

**Alternatives considered**:

- 仅修改 Vue 默认表单：拒绝；控制面不对称。
- 数据库迁移为全部旧对象补端口：拒绝；可能改变 VLAN、路由和占用。
- 运行时按需创建端口：拒绝；绕过保存配置和容量声明。

## 10. 兼容端点转发到统一命令

**Decision**: 保留现有 `/links`、`/network-objects/{id}/attachments` 和 `/network-object-links` 路由及 MCP 工具，但将 mutation 适配到统一应用命令；新客户端使用 `/connections` 与 `netlab.topology_connections.*`。

**Rationale**: 现有自动化和 SPA 流程不能被突然破坏，同时所有 mutation 必须共享 validation、idempotency、task 和 audit 语义。

**Alternatives considered**:

- 立即删除旧端点：拒绝；破坏兼容性。
- 长期维护两套服务：拒绝；状态和错误会继续漂移。
- 只新增查询投影：拒绝；mutation 仍不统一。

## 11. 010 视觉语义是不可回归合同

**Decision**: 提交成功后，画布继续从三类 backing records 构建 010 `ConnectionPresentation`；预览线、候选反馈和 chooser 是交互覆盖，不改变基础状态颜色、平行路由、抓包或 Traffic Filter 语义。

**Rationale**: 009 改变入口和生命周期，不应重写已经验证的视觉及观察模型。

**Alternatives considered**:

- 为新统一连接增加第四类画布边：拒绝；会重复和破坏投影。
- 用预览线替换 pending 连接：拒绝；其他客户端看不到权威状态。
- 按入口区分颜色：拒绝；入口不是连接健康状态。

## 12. 审计记录入口与端点摘要，不记录载荷

**Decision**: 审计记录 `entry_point`、规范端点、配置摘要、task、冲突码和清理结果；禁止记录终端内容、包载荷、凭据或无关文本。

**Rationale**: 统一入口需要诊断加号/拖拽/API/MCP 的行为差异，同时遵守秘密和抓包数据边界。

**Alternatives considered**:

- 记录完整请求与运行时命令：拒绝；可能包含敏感配置。
- 不记录入口：拒绝；无法分析交互或自动化故障。
- 将审计仅写日志：拒绝；不耐重启且无法与 durable task 对齐。

## 13. 本地里程碑和目标候选严格分离

**Decision**: 端点/命令、画布交互、默认端口/恢复、全量验收四个里程碑分别测试并提交；目标机只部署干净提交构建的带摘要候选。

**Rationale**: 连接功能同时触及 UI、SQLite、任务和特权运行时，需要可回滚的独立切片。

**Alternatives considered**:

- 完成后一次性提交：拒绝；难以定位运行时回归。
- 直接在目标机调试：拒绝；违反宪法与交付流程。
- 跳过无法本地执行的特权测试：拒绝；必须记录可重复目标验收。
