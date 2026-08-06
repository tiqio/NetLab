# Data Model: 全页面视觉与中文化治理

本功能不新增数据库表、API 资源或服务器权威状态。以下模型用于前端布局规则、测试夹具、审计记录和验收证据；运行时可使用类型化对象或 JSON 文件表达。

## 1. VisualAuditScenario（视觉审计场景）

表示一个可重复的页面状态组合。

| Field | Type | Required | Rules |
|---|---|---:|---|
| `id` | string | yes | 在本功能内唯一、稳定、可用于测试标题 |
| `surface` | enum | yes | `topology`、`inspector`、`context-menu`、`drawer`、`dialog`、`tasks`、`console`、`capture`、`traffic-filter`、`templates`、`automation` |
| `route` | string | yes | 现有 SPA 路由，不包含凭据或动态秘密 |
| `theme` | enum | yes | `light` 或 `dark` |
| `viewport` | object | yes | `width >= 1024`、`height >= 768` |
| `display_scale` | number | yes | 基线为 `1.0`，关键流程额外使用 `1.25` |
| `density` | enum | yes | `empty`、`normal`、`dense` |
| `interaction_states` | string[] | yes | 可含 hover、focus、selected、disabled、loading、running、failed、menu-open 等 |
| `fixture_ref` | string | yes | 引用测试夹具或验收 Lab，不内嵌敏感数据 |
| `expected_regions` | string[] | yes | 必须可见且可操作的布局区域 ID |

### Relationships

- 一个场景包含多个 `LayoutRegion`。
- 一个场景可产生零个或多个 `VisualFinding`。
- 一个场景通过一个或多个 `AcceptanceEvidence` 证明结果。

## 2. LayoutRegion（布局区域）

表示具有明确边界、层级和交互职责的视觉区域。

| Field | Type | Required | Rules |
|---|---|---:|---|
| `id` | string | yes | 场景内唯一 |
| `role` | enum | yes | `identity`、`status`、`graphic`、`metric`、`action`、`port`、`connector`、`label`、`scroll-content`、`overlay` |
| `owner` | string | yes | 页面、组件或拓扑资源的稳定身份 |
| `priority` | enum | yes | `critical`、`primary`、`secondary`、`decorative` |
| `bounds` | rectangle | yes | 相对当前视口的 `x`、`y`、`width`、`height` |
| `z_layer` | integer | yes | 用于验证覆盖关系，不作为业务状态 |
| `interactive` | boolean | yes | 是否必须具有独立命中区域 |
| `may_overlap` | string[] | yes | 允许覆盖的目标区域 ID；默认空数组 |
| `overflow_policy` | enum | yes | `wrap`、`truncate`、`scroll`、`collapse-secondary`、`none` |

### Validation Rules

- `critical` 和 `primary` 区域不得被未列入 `may_overlap` 的区域遮挡。
- 两个 `interactive=true` 区域的有效点击面积不得相交。
- `overlay` 只有在可关闭、焦点受控且不遮挡当前任务目标时才允许覆盖。
- `scroll-content` 必须能通过鼠标、触控板和键盘到达末尾。

## 3. VisualFinding（视觉碰撞发现）

表示审计中发现的布局、可读性或交互问题。

| Field | Type | Required | Rules |
|---|---|---:|---|
| `id` | string | yes | 唯一 |
| `scenario_id` | string | yes | 引用 `VisualAuditScenario.id` |
| `kind` | enum | yes | `overlap`、`clipping`、`overflow`、`low-contrast`、`hidden-focus`、`hit-target-conflict`、`layout-shift`、`untranslated-text`、`terminology-mismatch` |
| `severity` | enum | yes | `blocking`、`serious`、`moderate`、`cosmetic` |
| `region_ids` | string[] | yes | 至少一个受影响区域 |
| `reproduction` | string[] | yes | 可重复步骤，不包含秘密 |
| `observed` | string | yes | 用户可观察现象 |
| `expected` | string | yes | 对应合同或规格期望 |
| `status` | enum | yes | 见状态转换 |
| `waiver_reason` | string | conditional | 仅 `waived` 时允许，必须说明有意覆盖 |
| `evidence_ids` | string[] | yes | 修复前后或豁免证据 |

### State Transitions

```text
discovered -> confirmed -> fixing -> fixed -> verified
                         \-> waived
fixed -> reopened -> fixing
```

- `blocking` 或 `serious` 发现不得以纯装饰理由豁免。
- `waived` 仅适用于满足有意覆盖合同的设计，并需人工审阅。

## 4. ReadabilityState（可读状态）

描述文字、背景、图标、边框、焦点和状态线索在特定主题中的结果。

| Field | Type | Required | Rules |
|---|---|---:|---|
| `foreground_token` | string | yes | 使用语义主题标识，不记录临时颜色名称 |
| `background_token` | string | yes | 使用语义主题标识 |
| `contrast_result` | enum | yes | `pass`、`fail`、`not-applicable` |
| `non_color_cue` | string | conditional | 状态使用颜色时必须提供文字、图标、边框或形态线索 |
| `focus_visible` | boolean | yes | 交互控件聚焦时必须为 true |
| `disabled_readable` | boolean | yes | 禁用文本仍需可辨，但不得看起来可操作 |

## 5. LocalizationAuditItem（中文化审计项）

表示一个可见或辅助技术可读的文案位置。

| Field | Type | Required | Rules |
|---|---|---:|---|
| `id` | string | yes | 唯一且稳定 |
| `surface` | string | yes | 所在页面或功能域 |
| `location_kind` | enum | yes | `visible-text`、`label`、`placeholder`、`tooltip`、`aria-name`、`empty-state`、`status`、`error`、`chart-text` |
| `source_text` | string | yes | 当前观察到的文本 |
| `target_text` | string | conditional | 产品控制文案必须提供中文目标 |
| `classification` | enum | yes | `product-copy`、`technical-term`、`user-data`、`raw-diagnostic` |
| `allow_english` | boolean | yes | 仅后三类可为 true |
| `terminology_key` | string | optional | 引用核心术语表 |
| `result` | enum | yes | `pending`、`pass`、`fail`、`waived` |

### Validation Rules

- `product-copy` 不得设置 `allow_english=true`。
- `raw-diagnostic` 必须与中文摘要和处理建议同时展示。
- 可见文案和对应 `aria-name` 的含义必须一致。

## 6. TerminologyEntry（界面术语条目）

| Field | Type | Required | Rules |
|---|---|---:|---|
| `key` | string | yes | 稳定键 |
| `preferred_zh_cn` | string | yes | 产品统一译法 |
| `source_terms` | string[] | yes | 需要识别的英文或旧译法 |
| `preserve_source` | boolean | yes | 技术名称是否与中文并列保留 |
| `contexts` | string[] | yes | 导航、按钮、状态、帮助、错误等 |

### Required Core Entries

| Key | Preferred Chinese | Source Terms |
|---|---|---|
| `workspace.console` | 终端 | Console, Terminal workspace |
| `workspace.capture` | 抓包 | Capture |
| `workspace.trafficFilter` | 流量过滤 | Traffic Filter |
| `workspace.tasks` | 任务 | Tasks |
| `workspace.inspector` | 检查器 | Inspector |
| `workspace.diagnostics` | 诊断 | Diagnostics |

## 7. AcceptanceEvidence（验收证据）

| Field | Type | Required | Rules |
|---|---|---:|---|
| `id` | string | yes | 唯一 |
| `scenario_id` | string | yes | 引用审计场景 |
| `candidate_id` | string | yes | 可追溯到干净提交和构建产物 |
| `commit_sha` | string | yes | 完整或可唯一识别的 Git SHA |
| `result` | enum | yes | `passed`、`failed`、`waived` |
| `checks` | object[] | yes | 碰撞、溢出、对比度、中文化、键盘和行为连续性结果 |
| `artifacts` | object[] | yes | 脱敏截图、跟踪或报告引用，不嵌入二进制 |
| `cleanup` | object | conditional | 目标机验收必须包含前后资源摘要和残留数 |
| `created_at` | timestamp | yes | UTC 时间 |

### Sensitive Data Rules

- 不保存密码、bootstrap secret、终端输入输出全文或数据包内容。
- 截图必须避开凭据字段和敏感终端区域，或在保存前脱敏。
- 目标机资源清单只记录身份、类型、所有权和清理结果。
