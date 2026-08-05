# Quickstart Validation: 拓扑添加抽屉

## 1. Prerequisites

- 本地工作区位于仓库根目录且依赖已经安装。
- 可启动本地 NetLab 服务，或可以访问目标机 `10.72.1.7:18082`。
- 本地测试不得包含商业镜像、真实密码、抓包文件或其他秘密。
- 开始部署前工作区必须干净，且三个实现里程碑均已独立提交。

## 2. Shared Sheet Validation

```bash
cd web
npx vitest run \
  src/components/ui/sheet/Sheet.test.ts \
  src/components/shell/LaboratoryShell.test.ts
```

验证：

- left、right、bottom 方向保持兼容。
- header/footer 固定，body 独立滚动。
- Escape、遮罩和关闭按钮经过相同关闭请求。
- dirty 关闭先出现放弃确认。
- 打开时焦点进入抽屉，关闭后返回触发控件。

## 3. Drawer Component Validation

```bash
cd web
npx vitest run \
  src/features/topology/TopologyResourceCatalog.test.ts \
  src/features/topology/topologyResourceDraft.test.ts \
  src/features/topology/CreateTopologyResourceDrawer.test.ts \
  src/features/topology/TopologyWorkspace.test.ts
```

验证：

- QEMU、Docker、PC、Bridge、NAT Bridge、Lightweight L2 和 L3 均能生成正确创建请求。
- 模板和镜像只展示兼容、可用、许可已审核的版本。
- 类型切换确认、长表单草稿保留、字段错误定位、重复提交锁和服务端失败恢复均通过。
- 成功后由 Workspace 刷新并选中新资源，不产生临时幽灵节点。

## 4. Frontend Quality Gates

```bash
cd web
npm run lint
npm run format:check
npm test
npm run build
```

如果仓库全量测试出现与本功能无关的已知失败，必须记录具体测试、错误和独立复现结果；相关 Sheet、Drawer、Workspace 和创建合同测试不得跳过。

## 5. Local Browser Validation

启动本地服务后运行：

```bash
cd web
NODE_PATH="$PWD/node_modules" \
NETLAB_ACCEPTANCE_PROFILE=local \
npx playwright test \
  tests/e2e/journeys/topologyAddDrawer.spec.ts \
  tests/e2e/matrices/formAndDialogMatrix.spec.ts \
  tests/e2e/frontend_responsive_keyboard.spec.ts
```

至少验证以下场景：

1. 从工具栏打开空白资源选择抽屉。
2. 从左侧 DevicePalette 选择 Ubuntu 或 BusyBox，直接进入编辑步骤。
3. 在 1024×768 视口滚动最长表单，确认页面和拓扑画布不滚动，header/footer 始终可见。
4. 填写内容后按 Escape，选择“继续编辑”，确认草稿不丢失。
5. 触发字段错误和模拟结构化服务端失败，确认定位与保留草稿。
6. 快速双击提交，只产生一个创建请求和一个资源。
7. 成功后抽屉关闭、新资源被选中，画布 viewport 不变。

## 6. Milestone Commits

每个里程碑通过对应测试后提交：

1. Sheet 基础能力与回归测试。
2. 资源目录、草稿模型和创建抽屉。
3. Workspace 集成、E2E 和验收文档。

记录最终干净提交：

```bash
git status --short
git rev-parse HEAD
```

`git status --short` 必须为空后才能构建部署候选。

## 7. Build and Deploy Candidate

按照项目正式构建和安装流程生成带版本、候选标识、合同摘要和构建时间的产物，并记录二进制摘要。禁止在 `10.72.1.7` 直接编辑源文件。

部署完成后验证：

```bash
curl --fail http://10.72.1.7:18082/api/v1/capabilities | jq .
```

并确认目标服务发布信息与本地提交、候选标识和产物摘要一致。

## 8. Target-Host Browser Acceptance

在 `http://10.72.1.7:18082` 新建专用实验室，例如 `Topology Add Drawer Acceptance`。

依次通过抽屉创建：

1. 一个具有可用合法镜像的 Ubuntu QEMU。
2. 一个 BusyBox Docker。
3. 一个 PC。
4. 一个 Lightweight L2 或 NAT Bridge。

每次创建验证：

- 表单在抽屉 body 内滚动。
- 提交按钮保持可见且提交期间锁定。
- 成功后资源无需刷新页面即出现在画布并被选中。
- Inspector、底部 Console/Tasks 和画布 viewport 没有异常重置。
- 第二个浏览器能看到成功资源，但看不到第一个浏览器的抽屉草稿和滚动状态。
- 删除资源后画布、任务和权威快照均完成清理。

## 9. Failure and Rollback

至少验证一次模板/镜像过期或后端结构化失败：抽屉保留草稿、显示下一步且画布无幽灵资源。

若目标验收失败：

1. 记录候选标识、提交 SHA、失败步骤、截图或非敏感日志。
2. 使用正式安装器回滚到上一记录产物。
3. 验证 SPA 可加载、已有实验室可打开、创建 API 仍可用。
4. 回到本地工作区修复、测试、创建新提交和新候选，不在目标机打补丁。
