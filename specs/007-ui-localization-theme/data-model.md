# Data Model: 全页面中文化与明暗主题

本功能不新增服务端持久化实体。以下模型描述浏览器本地主题状态、中文文案资源和验证产物；实验室及运行资源继续使用现有服务端模型。

## 1. Theme Preference Record

表示当前浏览器保存的外观选择。

| Field | Type | Description |
|---|---|---|
| `schemaVersion` | integer | 当前为 `1`，用于安全升级本地记录 |
| `preference` | enum | `system`、`light`、`dark` |
| `explicit` | boolean | 用户是否明确选择；首次默认跟随系统时为 `false` |

### Validation rules

- 未知、损坏或版本不支持的记录按 `system`、`explicit=false` 处理。
- 保存失败不得阻止当前页面主题切换。
- 记录不包含实验室 ID、用户 ID、资源 ID或任何运行状态。
- 浏览器存储键固定为 `netlab.appearance.v1`，不得写入服务端、API、MCP 或实验室导出。

## 2. Resolved Theme State

代表当前实际应用到界面的主题。

| Field | Type | Description |
|---|---|---|
| `preference` | Theme Preference | 当前三态偏好 |
| `resolved` | enum | `light` 或 `dark` |
| `systemDark` | boolean | 当前设备是否偏好深色 |
| `storageAvailable` | boolean | 本地持久化当前是否可用 |
| `source` | enum | `system-default`、`user-choice`、`fallback` |

### State transitions

```text
startup
  ├── valid saved light/dark → resolved saved theme / user-choice
  ├── valid saved system → resolve current device preference
  ├── no saved record → system / explicit=false
  └── invalid or inaccessible storage → system fallback, current-session only

system preference changes
  ├── preference=system → update resolved theme
  └── preference=light|dark → no visible change

user selects theme
  ├── system → resolve current device preference and listen for changes
  ├── light → resolve light and ignore device changes
  └── dark → resolve dark and ignore device changes
```

### Invariants

- `resolved` 始终为 `light` 或 `dark`。
- 主题变化仅改变根元素外观属性和显示变量。
- 主题变化不得修改路由、实验室、资源选择、任务、拓扑、会话或表单草稿。

## 3. Chinese UI Text Catalog

产品自有中文文案的类型化集合。

| Field | Type | Description |
|---|---|---|
| `key` | stable string | 按领域和用途组织，例如 `tasks.actions.cancel` |
| `text` | string | 简体中文产品文案 |
| `parameters` | string[] | 运行时插入的资源名、数量、时间或错误原因 |
| `domain` | enum | shell、laboratory、topology、node、task、console、capture、traffic、template、automation、common |
| `kind` | enum | title、label、action、status、help、error、confirmation、empty、loading、accessibility |

### Validation rules

- 同一 key 不得重复或在运行时改变含义。
- 参数必须显式命名，不能依赖参数顺序猜测含义。
- 产品动作和高风险确认不得退化为只有“确定/取消”的无对象文本。
- 固定中文语言不进入浏览器主题记录，也不产生服务端状态。

## 4. Terminology Entry

用于保证核心概念译法一致，并声明允许保留原文的技术术语。

| Field | Type | Description |
|---|---|---|
| `source` | string | 原英文概念或内部机器值 |
| `preferredChinese` | string | 标准中文展示名 |
| `retainedToken` | string optional | 必须保留的协议缩写、产品名或设备词 |
| `notes` | string | 使用范围和禁止的冲突译法 |

### Required terminology groups

- 生命周期：期望状态、实际状态、启动中、运行中、停止中、已停止、失败。
- 拓扑：节点、接口、链路、网络对象、连接、断开、重新接线。
- 观测：Console、VNC、抓包、Traffic Filter、流量方向、会话、数据包。
- 自动化：任务、进度、取消、重试、冲突、幂等、修订版本。
- 保留原文：QEMU、Docker、QMP、QGA、VNC、Telnet、SSH、DHCP、SLAAC、TCP、UDP、ICMP、IP、MAC、VLAN、NAT、MCP、Wireshark。

## 5. Theme Semantic Palette

两种主题共享的用途级视觉语义。

| Token group | Required uses |
|---|---|
| Surface | page background、card、popover、muted、overlay |
| Text | primary、secondary、muted、inverse、code |
| Interaction | primary、secondary、accent、focus ring、disabled |
| State | success、running、warning、failure、offline、selected |
| Topology | canvas、grid、node、network object、link、route handle、selection box、traffic particle、direction arrow |
| Chart | background、axis、grid、legend、tooltip、series palette、empty state |

### Validation rules

- 每个 token 在浅色和深色主题中都必须有值。
- 关键状态必须同时具有非颜色表达方式。
- 组件不得为主题化表面写入仅适合单一主题的固定前景色。
- Console ANSI 内容和 VNC 图像不属于主题调色板。

## 6. Localization Coverage Finding

文案扫描门禁发现的潜在遗漏。

| Field | Type | Description |
|---|---|---|
| `file` | path | 发现位置 |
| `line` | integer | 近似行号 |
| `kind` | enum | visible-text、aria、title、placeholder、runtime-message |
| `text` | string | 发现的英文候选文本 |
| `classification` | enum | translate、technical-retain、false-positive |
| `allowlistReason` | string optional | 保留原文时的审计理由 |

### Validation rules

- 未分类 finding 使文案门禁失败。
- allowlist 必须精确到术语或模式，不允许整个目录无条件跳过。
- 新增英文产品文案必须先进入中文资源或获得明确技术术语豁免。

## Relationships

```text
Theme Preference Record ──resolves──> Resolved Theme State
Resolved Theme State ──selects──> Theme Semantic Palette
Chinese UI Text Catalog ──uses──> Terminology Entry
Localization Coverage Finding ──validated against──> Chinese UI Text Catalog + Terminology Entry
```
