# UI Contract: 简体中文界面

## Scope

本合同适用于 SPA 内全部产品控制的用户可见文案和可访问名称。它不改变 HTTP、MCP、事件、导出文件或运行时机器值。

## Required Chinese surfaces

- 顶部导航、实验室切换、右键菜单和全局命令入口。
- 设备目录、添加抽屉、表单、字段错误和创建状态。
- 拓扑画布工具、节点/链路菜单、Inspector 和设置。
- 任务、Console、VNC、Capture、Wireshark、Traffic Filter 和诊断。
- 模板、镜像、自动化、审计、导入导出、空态和错误页。
- Dialog、Sheet、Toast、菜单、表格、分页、搜索、可访问名称和键盘公告。

## Translation boundary

### Translate

- 产品标题、字段标签、动作、状态、人类可读错误和帮助说明。
- 高风险确认中的对象、动作、影响和下一步。
- 后端机器状态的展示标签，例如 `running` → `运行中`。

### Retain verbatim

- 用户输入和资源名称。
- 命令、终端输出、VNC 画面和设备返回文本。
- IP/MAC、接口名、路径、URL、资源 ID、任务 ID、错误 code。
- 协议缩写、产品名、镜像名、厂商名和型号。

### Mixed presentation

后端或设备原始错误无法安全翻译时：

1. 先展示中文影响与下一步。
2. 再展示原始错误文本和 code。
3. 不修改原始字段内容，便于复制和自动化对照。

## Terminology contract

- `desired state` 固定为“期望状态”，`actual/observed state` 固定为“实际状态”。
- `link` 固定为“链路”，`interface` 固定为“接口”，`network object` 固定为“网络对象”。
- `capture` 在数据包语境固定为“抓包”；`Traffic Filter` 首次出现显示“Traffic Filter（流量过滤）”，紧凑位置可保留 `Traffic Filter`。
- `task` 固定为“任务”，`revision conflict` 固定为“版本冲突”。
- 技术缩写保留原文，可在首次出现时增加中文说明。

## Status presentation

每个机器状态映射必须包含：

- 中文短标签；
- 中文说明或下一步；
- 语义类别；
- 原始机器值可在详情或复制信息中查看。

未知状态不得显示空白，应显示“未知状态（原值）”。

## Localization coverage gate

文案扫描必须检查：

- Vue 模板可见文本；
- `aria-label`、`title`、`placeholder`；
- 面向用户的运行时状态和错误字符串；
- 测试中仍期待旧英文产品文案的断言。

扫描允许的英文必须来自受审计技术术语表。扫描失败输出文件、行号、文本和建议分类。

## Accessibility

- 可访问名称必须为中文，但资源名和快捷键可保留原文。
- 中文按钮不得只依赖图标表达含义。
- 动态状态变化使用中文公告，不重复朗读长原始错误。
- 中文文本换行后不得遮挡焦点、确认按钮或表单错误。

## Compatibility

- API/MCP 请求响应字段、枚举值、状态 code、审计动作和导出格式保持不变。
- 自动化测试应优先使用稳定 role、label 和 test ID；更新为中文 label 时必须同步页面对象。
- 中文化不得改变任何资源创建、更新、删除、连接或观测操作的提交负载。
