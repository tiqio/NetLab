# UI Contract: Topology Add Drawer

## Purpose

定义拓扑添加抽屉、资源目录和 Workspace 之间的可测试交互边界。

## CreateTopologyResourceDrawer

### Inputs

| Input | Required | Meaning |
|---|---:|---|
| `modelValue` | yes | 当前抽屉是否打开 |
| `laboratoryId` | yes | 权威创建目标实验室 |
| `selection` | no | 来自左侧设备栏的预选资源；缺省时从资源选择步骤开始 |

### Outputs

| Event | Payload | Meaning |
|---|---|---|
| `update:modelValue` | boolean | 受控打开状态变更 |
| `created` | `{ node?, interfaces?, networkObject? }` | 服务端已经接受并返回的权威资源 |
| `selectionChanged` | Resource Selection or null | 抽屉内资源选择发生变化，供 Workspace 更新上下文 |
| `closeRequested` | close reason | overlay、escape、button、laboratory-change 或 success |

### Required behavior

- 打开且没有 selection 时，显示可搜索、分组的资源目录。
- 打开且有 selection 时，直接显示该资源的配置表单，并提供“更换资源”入口。
- body 独立滚动；header 与 footer 固定可见。
- `submitting` 时主提交操作不可再次触发，关闭操作不得制造未知创建结果。
- 验证失败时聚焦第一个无效字段；服务端失败保留草稿。
- 成功事件只能包含服务端响应中的资源，不允许发送临时 ID。
- dirty 时，overlay、Escape、关闭按钮、切换资源和切换实验室均进入放弃确认。

## TopologyResourceCatalog

### Inputs

| Input | Required | Meaning |
|---|---:|---|
| `query` | no | 外部受控搜索内容 |
| `compact` | no | 左侧 Palette 或抽屉选择步骤的密度模式 |
| `selectedKey` | no | 当前选中项，用于可见状态而不是共享状态 |

### Outputs

| Event | Payload | Meaning |
|---|---|---|
| `choose` | Resource Selection | 用户选择可创建资源 |
| `catalogState` | loading/ready/error | 模板目录当前状态 |

### Required behavior

- QEMU、Docker 和网络对象按清晰类别展示。
- 搜索同时匹配显示名称、模板 key、运行时类型和描述。
- 不可用资源可以展示，但必须说明原因并禁止进入提交状态。
- 同一模板过滤和默认版本规则在 DevicePalette 与 Drawer 中必须一致。

## Sheet

### Backward-compatible inputs

- `modelValue`
- `side`: left/right/bottom
- `title`

### New optional inputs/slots

- `description`
- `preventClose`
- `size` 或等价宽度约束
- default/body slot
- footer slot
- close-confirmation slot or controlled close callback

### Required behavior

- right/left Sheet 使用完整可用高度；bottom Sheet 保持现有高度约束。
- header 和 footer 不参与 body 滚动。
- Escape、overlay 和关闭按钮使用同一 close-request 路径。
- `preventClose` 时不得直接修改 modelValue；必须先确认。
- 打开时焦点进入 Sheet；关闭时焦点返回原触发元素。
- aria label/title、modal 语义和焦点范围满足键盘验收。

## Workspace Integration

- 工具栏添加入口打开 Drawer 的 selecting 状态。
- 左侧 DevicePalette 的 `choose` 打开同一 Drawer 的 editing 状态，不创建第二实例。
- Drawer 打开或关闭不得调用画布 fit/reset/pan/zoom API。
- `created` 继续进入现有 Workspace `created()` 路径：刷新权威快照、合并响应、确认 placement、选中新资源。
- Inspector 关闭/宽度/选择快照仅作为浏览器本地恢复上下文；资源事件仍可在 Drawer 打开时刷新画布。

## Accessibility Contract

- 抽屉具有可访问名称和可选描述。
- 主要提交、取消、更换资源和关闭均可通过键盘到达。
- 错误信息通过字段关联或 alert/status 语义暴露。
- 放弃确认使用独立 alertdialog，焦点不能落到背景 Workspace。
- 关闭后焦点恢复到触发按钮或触发的 Palette 项。
