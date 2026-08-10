# UI Contract: 统一拓扑端口连接体验

## 1. Supported Resources

- QEMU、Docker 和普通 PC 节点以具体 `NodeInterface` 作为端点。
- 轻量 PC、L2 和 L3 网络对象以实际保存的命名端口作为端点。
- Bridge 和 NAT 以设备主体上的逻辑接入口作为多接入目标。
- 用户不选择 backing kind；提交结果由端点组合决定。

## 2. Port Visibility

- 资源被选中、悬停、键盘聚焦或连接模式激活时展示端口/逻辑接入口。
- 每个端点提供名称、可用性、兼容状态、命中区和可访问名称。
- `free` 显示可连接反馈；`occupied`、`reserved`、`unavailable` 和 `incompatible` 使用非颜色状态与原因。
- 连接模式期间只强调与源端点兼容且当前可用的目标。

## 3. Unified Plus Entry

- 所有具有空闲端口或逻辑接入能力的资源在右上角显示一致的加号。
- 一个可用源端点时自动选择并进入 targeting。
- 多个可用源端点时打开源端口 chooser。
- 无可用端点时保留资源选择，不创建草稿，并播报原因。
- 加号之后使用与端口拖拽和键盘完全相同的目标、配置、提交和失败流程。

## 4. Direct Port Drag

- `pointerdown` 发生在端口独立 hit area；超过拖拽阈值后进入 `port_drag`。
- 预览线的起点始终来自源端口当前屏幕坐标，终点跟随指针。
- 拖拽期间节点不可移动，ECharts roam、框选和画布平移不响应同一指针。
- pointer capture 保证离开端口或资源主体后仍能收到 move/up/cancel。
- 指针经过目标时显示 `connectable`、`unavailable`、`incompatible` 或 `ambiguous` 反馈。
- `pointerup` 在具体兼容端口上直接进入配置/提交；在资源主体上按目标端口数量自动选择或打开 chooser；其他位置取消。

## 5. Click and Keyboard Flow

- 单击或 Enter/Space 选择源端点，再选择目标端点，与拖拽产生相同草稿和请求。
- Tab/方向导航可聚焦资源、加号、端口和 chooser 项。
- Enter/Space 确认，Escape 取消当前草稿或 chooser。
- 状态变化通过 live region 播报：源已选择、目标可用、需要选端口、正在提交、任务成功、冲突或失败。
- 不要求键盘用户使用指针坐标；chooser 锚点可退化为画布中心附近的可见位置。

## 6. Resource Body Drop

- 目标资源只有一个兼容可用端点时自动选择。
- 多个兼容可用端点时在操作位置附近打开 chooser，保留源端点与草稿。
- 无可用端点时显示“无可用端口”，不发送 mutation。
- Bridge/NAT 逻辑 access 可直接作为目标，不展示虚构固定端口列表。

## 7. Configuration Confirmation

- 普通节点链路和默认对象链路可走快速确认。
- 目标为轻量 L2 命名端口时，提交前允许确认 PVID 和 Tagged VLAN，默认使用该端口保存配置。
- 配置校验失败时保留草稿和 chooser 上下文，不创建共享连接。
- 用户取消配置时回到 targeting 或完全取消，不产生 topology mutation。

## 8. Submission and Authority

- SPA 提交统一 `ConnectionEndpoint` 请求、laboratory revision 和新的 idempotency key。
- 提交期间显示 pending 状态，但不得在 store 中伪造最终 backing record。
- 接收 `202` 后以返回的统一连接和 task 为准，并通过共享事件/刷新收敛到 backing snapshot。
- `revision_conflict`、`endpoint_occupied` 或 `endpoint_missing` 时刷新拓扑，清除失效草稿并提供重新选择动作；重试必须使用新 key。
- 等价网络重试使用原 key并返回相同 task/connection，不绘制重复线路。

## 9. Cancellation Boundaries

- Escape、点击空白、再次选择同一源端点、切换实验室、窗口失焦、文档隐藏或组件卸载取消未提交草稿。
- 未提交取消不得调用 API，也不得写入 Pinia 共享 topology。
- 已提交任务的取消使用正式 task cancel；UI 随后查询最终连接状态，不假定回滚成功。

## 10. Existing Connection Operations

- 三类连接继续使用 010 的 `ConnectionPresentation`、状态优先级、平行线路和动态语义图例。
- 选择、右键、删除、抓包、Wireshark 和 Traffic Filter 的入口及反馈不按 backing kind 区分。
- 非活动平行连接不得因邻接连接流量而高亮。
- 删除任务执行期间连接显示 disconnecting；成功后移除，失败后恢复权威状态和问题说明。

## 11. Four-Port Creation

- 新建轻量 L2/L3 的编辑器初始显示 `eth0`、`eth1`、`eth2`、`eth3`。
- 四个端口可分别重命名、删除、增加和配置；提交时必须至少一个端口且名称唯一。
- API/MCP 缺省创建后画布显示恰好四个端口。
- 加载旧对象时严格显示已保存端口，不因 UI 默认值自动扩展。

## 12. Gesture Invariants

- 一次 pointer sequence 最多提交一种操作：端口连接、节点移动、框选或画布平移。
- 端口拖拽不改变 viewport、resource placement 或 selection rectangle。
- 50%、100%、200% 缩放下预览锚点和目标命中保持与可见端口一致。
- pointer cancellation 后不得保留预览线、目标强调、chooser 或跨实验室源端点。

## 13. Test Selectors

- `[data-topology-connector][data-resource-id]`
- `[data-interface-id][data-port-hit-area]` 或端口父元素的现有稳定 selector
- `[data-connection-preview]`
- `[data-connection-target-state]`
- `[data-port-chooser][data-mode=source|target]`
- `[data-connection-task-state]`
- `[data-lightweight-port-name]`

选择器必须表达语义，不依赖主题颜色或构建后 class 名。
