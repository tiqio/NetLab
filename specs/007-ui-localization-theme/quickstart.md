# Quickstart: 全页面中文化与明暗主题验收

## Prerequisites

- 本地已安装 Go、Node.js 和前端依赖。
- 当前功能指针为 `specs/007-ui-localization-theme`。
- 目标机验收前，`10.72.1.7` 已具备可运行的 QEMU、Docker、网络和观测环境。
- 不在源码或验收产物中保存登录凭据、设备密码、抓包内容或专有镜像。

## 1. Specification and contract check

```bash
rg -n "NEEDS CLARIFICATION|\[FEATURE|\[specific" specs/007-ui-localization-theme
```

Expected:

- 没有未解决占位符。
- `contracts/localization-contract.md` 与 `contracts/theme-contract.md` 明确语言边界、主题隔离和会话连续性。

## 2. Focused localization validation

```bash
./scripts/check-ui-localization.sh
cd web
npm test -- --run \
  src/locales \
  src/components/common \
  src/components/shell
```

Expected:

- 没有未分类的用户可见英文。
- 允许保留的英文均来自技术术语豁免表。
- 共享组件、状态、错误和可访问名称测试通过。

## 3. Focused theme validation

```bash
cd web
npm test -- --run \
  src/composables/useThemePreference.test.ts \
  src/components/appearance/ThemeSwitcher.test.ts \
  src/components/charts/EChart.test.ts \
  src/features/topology/TopologyCanvas.a11y.test.ts
npm run build
```

Expected:

- 首次访问跟随设备偏好，无设备信息时使用深色。
- 用户选择在刷新和路由切换后保留。
- 存储失败不阻塞当前会话。
- 图表和拓扑切换主题后数据、位置、缩放和选择保持不变。

## 4. Full local gates

```bash
go test ./...
cd web
npm run format:check
npm run lint
npm test
npm run build
npm run test:e2e -- --list
```

Expected:

- 格式和静态检查无错误。
- 全量单元/合同测试通过。
- 中文化、主题、三视口、键盘和可访问性浏览器场景均可发现。

## 5. Local browser matrix

在 1024×768、1366×768、1920×1080 分别验证：

1. 打开拓扑、模板和自动化页面，检查全部产品文案为中文。
2. 切换“跟随系统 / 浅色 / 深色”，检查页面和已打开浮层同步变化。
3. 刷新和切换实验室，检查主题保持。
4. 使用第二浏览器选择不同主题，检查互不影响。
5. 完成添加资源、连接链路、节点设置、任务搜索、抓包和 Traffic Filter 流程。
6. 仅用键盘操作主题切换、添加抽屉和确认对话框。
7. 检查中文长文本在最小视口无阻塞重叠。

## 6. Candidate build and evidence

从干净里程碑提交构建候选，并记录：

- 源 commit SHA；
- candidate ID、版本、contract digest、binary digest；
- 构建时间；
- SQLite migration/schema state（本功能预期无迁移变化）；
- 本地测试结果和工作区清洁状态。

不得从未提交或直接在目标机修改的源文件构建。

## 7. Target-host validation

部署候选到 `10.72.1.7` 后建立专用验收 Lab：

1. 创建并启动一个 QEMU 节点和一个 Docker 节点。
2. 创建链路并验证权威状态。
3. 打开 Console，会话保持连接。
4. 启动 Capture 或 Traffic Filter 并产生可观察流量。
5. 在浅色、深色、跟随系统之间切换，记录切换前后资源 ID、状态、视口和会话。
6. 在第二浏览器使用不同主题，验证资源共享但主题隔离。
7. 检查拓扑、图表、弹层、任务、错误和中文可访问名称。
8. 删除专用 Lab，确认无遗留节点、链路、命名空间、抓包、过滤会话或任务。

Expected:

- 主题切换不触发资源 mutation，不中断运行或观测会话。
- 目标机仍报告候选版本和健康状态。
- 数据库 schema 版本部署前后一致且完整性检查通过。
- 验收环境清理完成。

## 8. Rollback

若出现主题不可读、页面英文遗漏、会话中断或权威状态变化：

1. 保存失败截图、交互步骤、任务/资源 ID 和候选信息。
2. 回滚到上一已记录候选产物。
3. 验证 SPA、现有实验室、Console 和观测功能恢复。
4. 回到本地工作区修复、补测试、创建新提交并重新部署。
