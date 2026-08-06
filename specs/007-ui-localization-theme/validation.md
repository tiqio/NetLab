# Validation: 全页面中文化与明暗主题

## 最终里程碑

- 验收日期：`2026-08-06`。
- 最终应用源码提交：`df7b5a6a8b61ffdfdb94032b16b71f2c5255a0c8`。
- 关键里程碑：`706d5b2`、`48c1a16`、`af4043a`、`dab287b`、`c9ee441`、`daaeba9`、`c9ab67f`、`df7b5a6`。
- 最终版本：`0.1.0-ui-zh-theme`。
- 最终候选：`ui-zh-theme-df7b5a6a8b61`。
- 合同摘要：`sha256:0790bbfb6ae93aa78862622cfdad9980fe56de911909c8520f5bf6b2756a4ea4`。
- 嵌入式 binary digest：`sha256:bc730a2be162cddd35f4cdeeba02b469a10bbe3f2d20a187ad1db2a94c3a8171`。
- 安装产物摘要：`sha256:9b3a72d6457c1e84208795f1056a9ac47096cf52bb44e8f75dd2b16ed3f81deb`。
- 构建时间：`2026-08-06T04:44:13Z`。
- 数据库迁移：无。

## 本地验证

- `scripts/check-ui-localization.sh`：PASS，产品文本、属性和运行时消息均已分类。
- `npm run format:check`：PASS。
- `npm run lint`：PASS，无错误；仓库既有 Vue 样式规则仅产生 warning。
- `npm test`：PASS，`69` 个测试文件、`263` 个测试全部通过。
- `npm run build`：PASS，Vue 类型检查、Vite 构建和嵌入式 SPA 资源生成成功。
- Playwright target 配置枚举：PASS，共发现 `43` 个文件中的 `114` 项测试。
- `go test ./...`：PASS，包含 HTTP、MCP、应用命令、QEMU、Docker、capture、SQLite、contract、integration、recovery 和 security 包。
- 中文、主题、拓扑语义色、ECharts 合并、FormField/Select 稳定字段、键盘与 axe 测试均包含在上述结果中。

## 目标机部署

- 目标：`10.72.1.7:18082`，单机 authoritative 实例。
- 仅部署由干净提交生成的二进制、release 配置与 template-readiness 清单；未在目标机修改源码。
- 最终替换前备份：`/var/lib/netlab/rollback/20260806044513/`。
- `GET /healthz`：`{"status":"ok"}`。
- `GET /api/v1/capabilities` 返回的 version、candidate ID、binary digest、contract digest 和 built-at 与最终候选一致。
- `/usr/local/bin/netlabd` 实际 SHA-256 与安装产物摘要一致。

## 目标机 UI 验收

- 输出目录：`/tmp/netlab-ui-target-release`。
- 结果：PASS，`10/10`。
- 覆盖浅色/深色主题以及 `1024×768`、`1366×768`、`1920×1080` 三个视口。
- 覆盖中文主路由、中文长文本无页面级横向溢出、键盘焦点转移、刷新与路由切换主题连续性、两个浏览器上下文主题隔离。
- axe serious/critical 门禁全部通过。
- 目标机历史任务数据触发了两项本地空基线未暴露的问题并已修复：浅色失败状态对比度不足，以及模板版本横向滚动区不可键盘聚焦。

## 目标机联合运行时验收

- 输出目录：`/tmp/netlab-ui-target-runtime-release`。
- 结果：PASS，`5/5`，总耗时约 `48.1s`。
- QEMU：Ubuntu QEMU 创建、启动、Serial/VNC 会话切换、重连与关闭通过。
- Docker：BusyBox/Ubuntu Docker 创建、启动和真实控制台路径通过。
- 共享状态：浏览器 HTTP 与 MCP 创建的资源在另一浏览器中可见，刷新后不存在陈旧状态复活。
- 链路：真实链路创建、运行中重连和拓扑状态收敛通过。
- Capture：接口抓包启动、刷新、元数据、实时流地址、停止和清理通过。
- Traffic Filter：创建、启动、观测刷新、拓扑路径高亮、停止和清理通过。
- Console、Capture 与 Traffic Filter 在标签切换后保持服务端会话和权威状态。

## 稳定字段与可访问性修复

- `FormField`、`Input` 和 `Select` 通过稳定 control ID 建立 label/control 关联。
- 添加资源抽屉使用 `data-field` 定位名称、模板、版本、镜像和接口数量。
- 实验室创建名称使用 `data-field="name"`，避免中文文案变化破坏自动化。
- 删除了 `FormField` 标签中意外渲染的 `>` 文本，并增加组件回归测试。
- 目标运行时 page objects 同时兼容当前中文名称与必要的历史英文名称。

## 合同与安全

- 未改变后端领域模型、应用命令、HTTP API、MCP、事件、SQLite schema 或机器可读状态字段。
- 主题偏好继续仅保存在浏览器本地 `netlab.appearance.v1`，不同客户端完全隔离。
- 未提交凭据、专有镜像、bootstrap secret、packet capture 或设备输出。
- 最终目标机验收清理后 `GET /api/v1/labs` 返回 `[]`，验收 evidence 显示 `0` 个剩余资源且 baseline restored 为 `true`。

## 回滚证据

- 本功能周期已使用上一记录候选执行回滚演练：旧候选恢复后服务 active 且 `/healthz` 通过，随后重新部署中文主题候选并再次通过健康检查。
- 每次最终候选替换前均保留二进制、配置和 template-readiness 的目标机回滚副本。
