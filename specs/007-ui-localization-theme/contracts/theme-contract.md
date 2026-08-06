# UI Contract: 明暗主题

## Preference contract

浏览器存储键：`netlab.appearance.v1`

```json
{
  "schemaVersion": 1,
  "preference": "system",
  "explicit": false
}
```

允许的 `preference`：`system`、`light`、`dark`。

无记录、记录损坏或存储不可用时按 `system` 处理；设备偏好不可用时解析为 `dark`。

## Root document contract

- 根文档始终具有 `data-theme="light"` 或 `data-theme="dark"`。
- 根文档 `color-scheme` 与解析主题一致。
- 主题属性在主要应用内容绘制前设置。
- 运行时变更只更新主题属性和相关显示状态，不重新加载页面。

## Theme switcher contract

- 所有顶层页面提供一致的“外观”入口。
- 控件提供“跟随系统”“浅色”“深色”三个中文选项，并明确当前解析结果。
- 可通过键盘打开、选择、关闭并恢复焦点。
- 主题变化应具有中文状态反馈，但不得弹出阻塞确认。
- 存储失败时显示非阻塞中文提示，并保持当前会话选择。

## Isolation contract

- 主题不进入实验室快照、WorkspacePreferences、HTTP、MCP、事件流、审计或导入导出。
- 两个浏览器可以同时使用不同主题。
- 切换实验室和路由不改变显式主题选择。

## Continuity contract

切换主题前后必须保持：

- 当前路由和实验室；
- 节点、网络对象、接口、链路和任务 ID；
- 期望/实际运行状态；
- 拓扑中心、缩放、节点坐标、选择和链路路径；
- 未提交表单草稿和滚动位置；
- Console/VNC/Capture/Traffic Filter 会话及其连接状态。

## Visual semantic contract

- 浅色和深色主题必须覆盖共享表面、文字、边框、输入、焦点和状态 token。
- 拓扑与图表使用主题语义调色板，不读取固定单主题颜色。
- 成功、警告、失败、运行、选中和流量方向均提供非颜色线索。
- 减少动态效果启用时，主题过渡和流量动画不得强烈闪烁。
- xterm ANSI 内容和 noVNC 客户机画面不被反色或滤镜处理。

## System preference behavior

- `preference=system` 时监听设备主题变化并实时更新。
- `preference=light|dark` 时忽略设备主题变化。
- 用户重新选择“跟随系统”后立即按当前设备偏好解析。

## Failure behavior

- 本地存储读失败：使用系统/深色回退并继续启动。
- 本地存储写失败：当前页面继续使用选择，提示刷新后可能恢复默认。
- 系统媒体查询不可用：`system` 解析为 `dark`。
- 图表主题更新失败：保留原数据并显示中文诊断，不得清空拓扑或关闭会话。
