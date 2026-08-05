# Data Model: 拓扑添加抽屉

本功能不新增服务端持久化实体。以下模型描述当前浏览器中的抽屉会话，以及它与现有权威资源之间的关系。

## 1. Add Drawer Session

代表当前浏览器内唯一的添加抽屉实例。

| Field | Type | Description |
|---|---|---|
| `open` | boolean | 抽屉是否可见 |
| `laboratoryId` | string | 创建目标实验室；必须与当前活动实验室一致 |
| `phase` | enum | `closed`、`selecting`、`editing`、`submitting`、`failed` |
| `selection` | Resource Selection or null | 当前选择的资源类型、模板和推荐版本 |
| `draft` | Resource Create Draft or null | 当前未提交配置 |
| `initialSignature` | string | 初始化后的规范化草稿签名，用于判断脏状态 |
| `fieldErrors` | map | 字段路径到中文错误信息的映射 |
| `globalError` | string or null | 无法映射到字段的结构化失败 |
| `staleMessage` | string or null | 模板或镜像目录变化提示 |
| `scrollTop` | number | 当前抽屉 body 的浏览器本地滚动位置 |
| `triggerFocus` | element reference or null | 关闭后应恢复焦点的触发控件 |
| `inspectorSnapshot` | Inspector Snapshot | 打开前 Inspector 的本地显示状态 |

### Validation rules

- 同一 Workspace 同时只能有一个 open session。
- `editing`、`submitting` 和 `failed` 必须具有 selection 与 draft。
- `submitting` 期间不接受第二次 submit，也不允许无确认切换实验室。
- `laboratoryId` 与当前活动实验室不一致时不得提交；保留草稿并显示冲突。
- `dirty` 由当前规范化草稿签名与 `initialSignature` 比较得出，不单独持久化。

## 2. Resource Selection

描述用户将要创建的资源类别。

| Field | Type | Description |
|---|---|---|
| `kind` | enum | `qemu`、`docker`、`pc`、`switch_l2`、`switch_l3` |
| `networkObjectKind` | enum or null | `pc`、`bridge`、`nat_bridge`、`switch_l2`、`switch_l3` |
| `name` | string | 面向用户的默认资源名称 |
| `description` | string | 类型用途和运行方式说明 |
| `templateId` | string or null | QEMU/Docker 设备模板 |
| `templateVersionId` | string or null | 已启用的模板版本 |
| `imageVersionId` | string or null | 与模板版本兼容的可用镜像 |
| `availability` | enum | `available`、`loading`、`unavailable`、`stale` |

### Validation rules

- QEMU/Docker 必须选择已启用模板版本。
- 模板需要镜像时，镜像必须运行时类型一致、许可已审核、状态可用且包含在兼容列表中。
- 网络对象不携带模板或镜像标识。

## 3. Resource Create Draft

统一草稿由共享字段与类型专属配置组成。

### Shared fields

| Field | Type | Description |
|---|---|---|
| `name` | string | 资源名称 |
| `interfaceCount` | integer | 节点初始接口数量 |
| `templateVersionId` | string or null | 节点模板版本 |
| `imageVersionId` | string or null | 节点镜像版本 |

### Node network fields

| Field | Type | Description |
|---|---|---|
| `ipv4Mode` | enum | `none`、`static`、`dhcpv4` |
| `ipv4Address` | string | 静态 IPv4 CIDR |
| `ipv6Mode` | enum | `none`、`static`、`slaac`、`dhcpv6` |
| `ipv6Address` | string | 静态 IPv6 CIDR |
| `routes` | Route Draft[] | IPv4/IPv6 静态路由 |

### Bootstrap fields

| Field | Type | Description |
|---|---|---|
| `cloudUsername` | string | 支持的 Ubuntu 类模板初始用户名 |
| `cloudPassword` | string | 创建时使用的节点级 bootstrap 密码；不得写入日志或版本库 |

### Network object fields

- PC 使用现有 PC 配置结构。
- Lightweight L2 使用端口、PVID、tagged VLAN 和 VLAN filtering 配置。
- Lightweight L3 使用接口地址、静态路由和 IPv4/IPv6 转发配置。
- Bridge 与 NAT Bridge 使用现有默认配置和 Inspector 可修改字段。

### Validation rules

- 名称去除首尾空白后必须非空。
- `interfaceCount` 必须位于模板和平台允许范围内。
- 静态地址模式必须提供对应族的合法 CIDR。
- 路由 destination 必须是合法 CIDR；gateway 和 metric 遵循现有合同。
- 类型切换只保留语义兼容的共享字段；会丢弃其他字段前必须确认。

## 4. Inspector Snapshot

记录抽屉打开前的浏览器本地 Workspace 状态。

| Field | Type | Description |
|---|---|---|
| `collapsed` | boolean | Inspector 是否折叠 |
| `size` | number | Inspector 宽度 |
| `selectedResourceIds` | string[] | 打开前选中资源 |
| `focusedResourceId` | string or null | 键盘焦点资源 |
| `triggerKind` | enum | toolbar、palette、keyboard、command |

关闭抽屉只恢复显示和焦点上下文；其他客户端事件导致资源已不存在时，选择状态必须通过现有清理规则收敛，不能恢复无效资源。

## 5. Authoritative Creation Result

沿用现有服务端返回：

- 节点创建：Node 加初始 NodeInterface 列表。
- 网络对象创建：NetworkObject。
- 后续实验室快照或事件提供位置、版本和实际状态。

抽屉不得把草稿或临时卡片当作 Authoritative Creation Result。

## State Transitions

```text
closed
  ├─ open without preselection ──> selecting
  └─ open with palette selection ─> editing

selecting
  ├─ choose available resource ───> editing
  └─ close ───────────────────────> closed

editing
  ├─ valid submit ────────────────> submitting
  ├─ choose/change resource ──────> editing (confirm when dirty)
  ├─ validation failure ──────────> editing
  └─ close ───────────────────────> closed (confirm when dirty)

submitting
  ├─ authoritative success ───────> closed
  ├─ structured/retryable failure > failed
  └─ laboratory conflict ─────────> failed

failed
  ├─ edit or retry ───────────────> editing/submitting
  └─ confirmed discard ───────────> closed
```

刷新页面会销毁 selecting/editing/failed 草稿；已成功创建的资源从权威实验室快照恢复。
