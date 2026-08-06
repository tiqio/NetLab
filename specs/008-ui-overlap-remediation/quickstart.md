# Quickstart Validation: 全页面视觉与中文化治理

本指南验证 [spec.md](./spec.md)、[visual-layout-contract.md](./contracts/visual-layout-contract.md)、[localization-contract.md](./contracts/localization-contract.md) 和 [acceptance-evidence.schema.json](./contracts/acceptance-evidence.schema.json)。所有实现先在本地完成、测试并形成里程碑提交，再部署到 `10.72.1.7`。

## 1. Prerequisites

- 本地工作树位于仓库根目录，Node.js 与 Go 版本满足项目要求。
- 已安装前端依赖和 Playwright 浏览器。
- 不在仓库中放置密码、专有镜像、终端敏感输出或数据包内容。
- 目标机验收前记录已有 Lab 身份和数量；只允许清理本功能创建的验收 Lab。

```bash
make bootstrap
cd web && npx playwright install chromium
```

## 2. Validate Planning Contracts

```bash
jq empty specs/008-ui-overlap-remediation/contracts/acceptance-evidence.schema.json
git diff --check
rg -n 'NEEDS'' CLARIFICATION|template placeholder' \
  specs/008-ui-overlap-remediation
```

Expected:

- JSON Schema 可以解析。
- 没有空白错误和未解决占位符。
- 计划明确说明本功能不改变 API、MCP、数据库或运行时状态。

## 3. Localization Gate

```bash
./scripts/check-ui-localization.sh
cd web && npm test -- \
  src/locales \
  src/features/diagnostics \
  src/features/tasks \
  src/features/templates \
  src/features/topology
```

Expected:

- `Right-click a node and choose Terminal`、Tasks、Console、Capture、Reconnect、Close、Select、Remove、running、stopped 等通用产品文案不再以英文出现。
- QEMU、Docker、Wireshark、VNC、SSH、TCP、pcap、接口名、命令和用户数据仍可保留原文。
- Wireshark 等错误分支先显示中文摘要和处理建议，并可访问原始详情。
- 可见标签、`aria-label`、隐藏标题和工具提示使用一致中文术语。

## 4. Shared Component and Theme Validation

```bash
cd web
npm test -- \
  src/components/common \
  src/components/ui \
  src/components/charts \
  src/components/shell \
  src/styles
npm run lint
npm run format:check
npm run build
```

Expected:

- Dialog、Sheet、Dropdown、Tooltip、表单、空状态、错误和状态组件在浅色/深色下均可读。
- 禁用、危险、选中、失败和焦点状态具有非颜色线索。
- 图表在容器变化和主题切换后重排，标签与指标不覆盖绘图区。

## 5. Topology Geometry Validation

运行拓扑相关组件和浏览器测试：

```bash
cd web
npm test -- \
  src/features/topology \
  src/features/analytics
cd ..
make build
NETLAB_ACCEPTANCE_PROFILE=local \
  NETLAB_ACCEPTANCE_RUN_ID=ui-overlap-local \
  ./acceptance/frontend-acceptance.sh
```

至少覆盖：

1. 各创建一个具有 1、2、4、8、16 个接口的代表性节点。
2. 验证每个端口、标签和连接入口均可辨、可点击，鼠标经过不改变端口坐标。
3. 拖动节点并执行缩放、平移、适配、重置，确认节点内部元素不漂移。
4. 在两个节点之间创建两条或更多平行链路，确认路径、标签和端点可区分。
5. 运行 Traffic Filter，确认只有承载匹配流量的链路显示方向和粒子；流量停止后粒子停止且残留装饰按规则消退。

Expected:

- 交互目标不相交，透明装饰不截获点击。
- 页面无非预期横向滚动。
- 节点拖动和流量动画不引起整个拓扑收缩、跳动或漂移。

## 6. Inspector, Menu, and Workspace Matrix

对每个主题和视口执行：

| Theme | Viewport |
|---|---|
| light | 1024×768 |
| dark | 1024×768 |
| light | 1366×768 |
| dark | 1366×768 |
| light | 1920×1080 |
| dark | 1920×1080 |

在每个组合中验证：

1. 打开 QEMU、Docker、PC、Bridge、NAT Bridge、Lightweight L2/L3 检查器。
2. 同时显示长名称、失败状态、资源图、vCPU、CPU 配额、内存、上下行和多个按钮。
3. 在视口四角打开节点、链路、网络对象和画布右键菜单。
4. 展开并切换任务、终端、抓包、流量过滤和诊断工作区。
5. 重复主题切换、面板展开/收起、路由切换和刷新各 20 次。

Expected:

- 检查器标题、状态、图表、指标和按钮不重叠。
- 菜单完整留在视口内，文字、图标和状态清晰，键盘焦点可见。
- 底部标签不会消失，持久终端或抓包状态不因切换而被意外销毁。
- 无白字白底、深字深底或浅色禁用文字不可辨现象。

## 7. 125% Display-Scale Manual Check

在 Chromium 浏览器将页面缩放设置为 125%，使用 1024×768 或更大的窗口，完成：

1. 创建节点；
2. 连接接口；
3. 打开检查器；
4. 使用节点和链路右键菜单；
5. 打开终端；
6. 选择接口并开始抓包。

Expected: 六个流程均可完成，无控件被遮挡、误点击相邻目标或需要 Reset 才能恢复。

## 8. Full Local Quality Gates

每个里程碑运行适用的聚焦测试；候选构建前运行完整门禁：

```bash
make lint
make test
make test-contract
make test-web
make test-security
make test-frontend-artifacts
make test-compliance
make build
make test-e2e-local
```

若本地环境不具备特权网络能力，记录跳过原因，并在目标机补跑：

```bash
NETLAB_PRIVILEGED=1 make test-integration
NETLAB_PRIVILEGED=1 make test-recovery
NETLAB_PRIVILEGED=1 CYCLES=100 make test-leaks
```

## 9. Milestone Commit and Candidate Record

每个里程碑通过后创建聚焦提交。部署候选前确保工作树干净并记录来源：

```bash
git status --short
git rev-parse HEAD
git log -1 --oneline

export CANDIDATE_ID="ui-overlap-$(date -u +%Y%m%dT%H%M%SZ)"
export VERSION="008-ui-overlap-remediation"
make build
sha256sum bin/netlabd
```

Expected:

- 部署来源是已提交的干净 SHA。
- 保存 candidate ID、commit SHA、二进制摘要、合同摘要、构建时间和无数据库迁移说明。

## 10. Deploy and Validate on `10.72.1.7`

使用现有部署流程复制已构建候选，不在目标机编辑源码。部署前记录已有 Lab 基线，然后运行目标浏览器验收：

```bash
NETLAB_ACCEPTANCE_PROFILE=target-host \
NETLAB_ACCEPTANCE_BASE_URL=http://10.72.1.7 \
NETLAB_ACCEPTANCE_RUN_ID="$CANDIDATE_ID" \
./acceptance/frontend-acceptance.sh
```

若实际服务使用不同端口，将 `NETLAB_ACCEPTANCE_BASE_URL` 设置为对应的已部署地址。

目标 Lab 至少包含：

- Ubuntu QEMU 和一个其他 QEMU 模板；
- BusyBox Docker 与 Ubuntu Docker；
- PC、Bridge、NAT Bridge、Lightweight L2、Lightweight L3；
- 1/2/4/8/16 接口代表节点；
- 平行链路、运行状态、停止状态、失败状态；
- 活动终端、抓包和 Traffic Filter 会话。

Expected:

- 双主题、三视口和关键 125% 流程满足视觉合同。
- 不存在未经批准的英文产品文案。
- 两个浏览器客户端可以使用不同主题和面板状态，同时看到相同服务器权威资源状态。
- 终端、抓包、流量过滤和运行中链路操作未因布局修复中断。

## 11. Evidence and Cleanup

验收结果按 [acceptance-evidence.schema.json](./contracts/acceptance-evidence.schema.json) 输出。截图必须脱敏，不包含密码、终端敏感输出或包内容。

验收完成后：

1. 删除本功能创建的专用 Lab。
2. 确认其 QEMU、容器、netns、接口、链路、抓包、Traffic Filter 和辅助进程均已清理。
3. 对比已有用户 Lab 身份和数量，必须与基线一致。
4. 验证 `cleanup.owned_residual_count == 0` 且前后资源摘要符合预期。

若任一检查失败，回滚到上一个已记录候选；修复必须返回本地工作树，增加测试并形成新提交后重新部署。
