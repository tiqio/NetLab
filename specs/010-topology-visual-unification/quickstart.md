# Quickstart: 拓扑连接视觉与资源放置统一验收

## 1. Preconditions

- 在 `/home/dd/netlab` 的功能分支 `010-topology-visual-unification` 工作。
- 本地先完成测试和里程碑提交；不得直接在 `10.72.1.7` 修改源码。
- 目标机具备现有 QEMU、Docker、netns、Bridge、NAT、抓包与 Traffic Filter 运行条件。
- 不下载、提交或记录专有镜像、凭据和抓包内容。

## 2. Focused Local Verification

```bash
cd /home/dd/netlab

go test ./internal/domain/... \
  ./internal/app/command/... \
  ./internal/store/sqlite/... \
  ./internal/api/http/... \
  ./internal/api/mcp/...

cd web
npm test -- --run \
  src/features/topology/topologyVisualSemantics.test.ts \
  src/features/topology/linkPresentation.test.ts \
  src/features/topology/topologyLayout.test.ts \
  src/features/topology/topologyPlacementBatch.test.ts \
  src/features/topology/TopologyCanvas.test.ts \
  src/features/topology/TopologyWorkspace.test.ts
```

新增 placement allocator 测试必须覆盖：首选位置可用、碰撞避让、20 资源连续创建、边界、候选耗尽、revision 冲突、幂等重试、并发创建、事务回滚和资源删除级联。

## 3. Contract and Static Gates

```bash
cd /home/dd/netlab

# 使用仓库既有 OpenAPI/MCP 合同校验或生成命令；确认 feature delta 已同步到正式合同。
go test ./internal/api/... ./internal/app/...

cd web
npm run format:check
npm run lint
npm run build
npm test
```

确认没有未决澄清标记、模板占位、硬编码连接状态颜色或只存在于 UI 的创建参数。

## 4. Reference Laboratory

创建专用实验室 `visual-unification-acceptance-<candidate>`，至少包含：

- 1 台 Ubuntu QEMU；
- 1 个 BusyBox Docker；
- 1 个 Ubuntu Docker；
- 1 个 PC；
- 1 个普通 Bridge；
- 1 个 Internet NAT；
- 1 个 Lightweight L2；
- 1 个 Lightweight L3。

创建以下连接：

1. QEMU ↔ Docker 节点直连；
2. Docker ↔ Bridge；
3. QEMU ↔ NAT；
4. PC ↔ L2；
5. L2 ↔ L3 网络对象直连；
6. L3 ↔ Bridge；
7. 同一对资源之间至少三条不同接口的平行连接。

记录每条连接的用户可读名称、实际状态和语义 marker，不记录包内容。

## 5. Unified State Matrix

依次观察或构造 connected、transition、failed、disconnecting 和 unknown/stopped 状态：

- 三类连接在相同状态下基础颜色、线型和状态文字一致；
- 失败状态即使被选中或有 Traffic Filter observation 仍可识别为失败；
- NAT 等差异只表现为辅助 marker，不改变成功颜色；
- 删除/恢复连接后图例数量与线路状态同步；
- 线路 tooltip/检查器显示 `资源:端口 ↔ 资源:端口`，不以长 ID 为主标题。

## 6. Parallel Link and Zoom Test

在 50%、100%、200% 缩放下验证三条平行连接：

- 每条线路路径独立可见；
- 可分别点击、右键、删除和选择抓包；
- Traffic Filter 只高亮命中的接口线路；
- 切换主题、底部 Console/Capture/Traffic Filter 和右侧检查器后不重合、不回弹；
- 拖动任一端点后曲线稳定，松手不向旧位置回跳。

## 7. Dynamic Legend Test

1. 无 NAT/共享域语义时确认左下角不显示无关特殊图例。
2. 创建 NAT attachment 后确认出现中文 NAT 上联说明和正确数量。
3. 创建共享 Bridge/L2 多接入后确认仅在设计规则需要时显示共享广播域说明。
4. 点击/键盘聚焦图例项，确认只突出对应连接。
5. 删除最后一条对应连接，确认图例项随共享快照消失。
6. 在浅色/深色下确认样例与实际线路使用相同 token，状态和语义均不只靠颜色。

## 8. Twenty-Resource Placement Test

将画布停在已有资源较密集的区域，连续创建 20 个混合资源：QEMU、Docker、PC、Bridge、NAT、L2、L3。

验收：

- 每个创建响应包含 `placement_assignment`；
- 新资源主体、名称、端口和连接加号不与已有资源重叠；
- 已有 placement 的 `x/y/revision` 除实验室 revision 外不被修改；
- 被避让的创建返回 `adjusted: true` 和 `collision_avoided`；
- 新资源 2 秒内可见，或出现可用的“定位新资源”动作；
- 刷新浏览器后所有坐标保持一致。

可用自动化几何断言将每个 placement 映射为规范足迹矩形，并确认任意两矩形（含 clearance）不相交。

## 9. Concurrent Client and API/MCP Test

1. 同时打开两个浏览器客户端并记录相同 laboratory revision。
2. 两端在相同画布中心各创建资源，同时通过 API 或 MCP 再创建资源。
3. 对 revision 冲突按正式客户端策略刷新并重试，不复用不同请求的 idempotency key。
4. 重复 10 组。

验收：

- 每组资源 ID 和权威位置不同；
- 等价幂等重试返回同一资源和 placement；
- 不等价同 key 请求返回结构化冲突；
- 所有客户端在 2 秒内收敛到同一资源集合与坐标；
- 无资源先出现后永久停留在 `(0,0)` 或客户端预测中心。

## 10. Failure, Cancellation and Cleanup

执行 20 次循环：无效 intent、revision 冲突、创建取消、候选耗尽模拟、创建后删除、实验室删除。

验收：

- 失败事务不留下节点、网络对象、placement、预留或孤立 outbox 事件；
- 删除资源同步删除 placement 和相关连接；
- 删除实验室后 placement、连接和运行时资源清理率为 100%；
- 相同区域可再次创建，不存在永久占位。

## 11. Traffic Filter Decay

在 node link、attachment 和 object link 上分别产生 ICMP、TCP、UDP 流量：

- 粒子方向与实际发起端一致；
- 平行的非活动接口线路不被点亮；
- 动态切换过滤表达式时旧会话正常停止，不占用 capture slots；
- 停止流量和会话后，粒子立即停止，方向提示在约定衰减时间后消失；
- 基础连接状态、语义 marker 和选择状态正确恢复。

## 12. Recovery and Existing Coordinate Preservation

1. 升级前导出三个已有实验室全部 placement 和连接端点摘要。
2. 部署候选后打开实验室，不触发拖动或自动整理。
3. 比较升级前后坐标和端点，变化数必须为 0。
4. 创建新资源并记录位置，重启 NetLab 服务后重新查询。
5. 打开两个客户端确认连接视觉、动态图例和 placement 一致。

服务重启不得触发全图重新布局；Traffic Filter 临时动画无需跨重启保留，但基础连接状态必须恢复。

## 13. Full Frontend Acceptance

```bash
cd /home/dd/netlab/web
npm run test:acceptance-unit
npm run test:e2e:local
```

目标机候选部署后运行：

```bash
cd /home/dd/netlab/web
NETLAB_BASE_URL=http://10.72.1.7 npm run test:e2e:target
```

若目标实际端口由部署配置决定，使用现有 acceptance profile 的正式变量，不把地址或凭据写入源码。

## 14. Milestone Commits and Deployment Record

每个里程碑完成后记录：

```text
Milestone:
Commit SHA:
Focused tests:
Result:
```

全部本地门禁通过且 worktree 干净后构建候选，记录：

```text
Candidate ID:
Commit SHA:
Artifact digest:
Contract version:
Build time:
Deployment time:
Target validation result:
Previous artifact / rollback command:
```

仅将该制品部署到 `10.72.1.7`。若出现状态误导、旧坐标变化、初始重叠、多客户端不一致或清理失败，立即恢复上一已验证制品，并在本地修复、测试、提交后生成新候选。
