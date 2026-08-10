# Quickstart: 统一拓扑端口连接体验

## 1. Prerequisites

- 在 `/home/dd/netlab` 的本地 Git worktree 实施和测试。
- 使用 Go 1.26.x、Node.js/npm 与仓库锁定的前端依赖。
- 本地非特权测试不得要求访问 `10.72.1.7`。
- 特权运行时与最终候选验证仅在已记录的干净提交和制品上执行。
- 不向仓库添加镜像、凭据、bootstrap secret 或 packet capture。

## 2. Design Contracts

- HTTP delta: `specs/009-unified-link-interaction/contracts/openapi.yaml`
- MCP delta: `specs/009-unified-link-interaction/contracts/mcp-tools.json`
- UI interaction: `specs/009-unified-link-interaction/contracts/ui-contract.md`
- Entities and transitions: `specs/009-unified-link-interaction/data-model.md`
- 视觉结果必须继续满足 `specs/010-topology-visual-unification/`。

## 3. Focused Backend Validation

在实现过程中先运行最窄测试，再扩大范围：

```bash
cd /home/dd/netlab
go test ./internal/domain/... \
  ./internal/app/command/... \
  ./internal/store/sqlite/... \
  ./internal/api/http/... \
  ./internal/api/mcp/...
```

必须覆盖：

- 四种允许的端点组合与所有非法、自连、跨实验室组合；
- 节点接口和对象命名端口的跨 backing kind 唯一占用；
- revision conflict、endpoint occupied、capacity 和 idempotency conflict；
- 创建事务在连接、两端预留、task、audit/outbox 或 revision 任一步失败时全部回滚；
- 删除、取消、部分 runtime 失败和服务恢复后的预留清理；
- 旧专用 HTTP/MCP mutation 与新统一命令结果等价；
- 新建轻量 L2/L3 缺省四端口，已有/导入对象不自动扩口。

## 4. Focused Frontend Validation

```bash
cd /home/dd/netlab/web
npm test -- \
  src/features/topology/topologyConnectionController.test.ts \
  src/features/topology/topologyInteractionController.test.ts \
  src/features/topology/TopologyCanvas.test.ts \
  src/features/topology/TopologyCanvas.a11y.test.ts \
  src/features/topology/TopologyWorkspace.test.ts \
  src/features/nodes/lightweightSwitchConfig.test.ts
```

组件和控制器断言：

- 端口拖拽预览始终锚定源端口；
- 加号、单击端口、拖拽和键盘进入同一草稿状态机；
- 一个目标端口自动选择，多个端口打开 chooser；
- 节点到节点、节点到轻量对象、对象到对象、节点到 Bridge/NAT 使用同一提交 API；
- occupied、incompatible 和 ambiguous 目标有非颜色反馈；
- Escape、空白、同源、窗口失焦、文档隐藏和实验室切换均取消草稿；
- 一次手势不会同时移动节点、框选、平移或缩放；
- 新建轻量 L2/L3 编辑器显示并允许修改 `eth0`–`eth3`。

## 5. HTTP Contract Scenarios

### Node Interface to Node Interface

```bash
curl -fsS -X POST \
  -H 'Content-Type: application/json' \
  -H 'If-Match: <laboratory-revision>' \
  -H 'Idempotency-Key: <unique-key>' \
  -H 'X-NetLab-Entry-Point: http' \
  http://127.0.0.1:8080/api/v1/labs/<lab-id>/connections \
  -d '{
    "source":{"kind":"node_interface","resource_id":"<node-a>","port_id":"<if-a>"},
    "target":{"kind":"node_interface","resource_id":"<node-b>","port_id":"<if-b>"}
  }'
```

Expected: `202`, one pending unified connection, one durable task, and a new laboratory revision. Repeating the equivalent request with the same key returns the same connection/task.

### Node Interface to Lightweight L2 Port

```bash
curl -fsS -X POST \
  -H 'Content-Type: application/json' \
  -H 'If-Match: <laboratory-revision>' \
  -H 'Idempotency-Key: <unique-key>' \
  http://127.0.0.1:8080/api/v1/labs/<lab-id>/connections \
  -d '{
    "source":{"kind":"node_interface","resource_id":"<node>","port_id":"<if-id>"},
    "target":{"kind":"network_object_port","resource_id":"<switch>","port_name":"eth1"},
    "config":{"pvid":10,"tagged_vlans":[20,30]}
  }'
```

Expected: backing kind is server-derived as `network_attachment`; the caller does not choose it.

### Conflict

Run two requests concurrently for the same concrete endpoint with different idempotency keys.

Expected: at most one request is accepted; the other returns `409 endpoint_occupied` or `revision_conflict`, commits no connection/task reservation owned by that failed request, and all clients converge after refresh.

## 6. MCP Contract Scenarios

Validate `netlab.topology_connections.create`, `.list`, `.get`, and `.delete` with the schemas in `contracts/mcp-tools.json`.

Required parity checks:

- the same endpoint pair produces the same backing connection and task through HTTP and MCP;
- equivalent retry returns the same result;
- conflicting reuse of a key returns `idempotency_conflict`;
- delete returns a durable task and releases endpoint reservations only after authoritative deletion;
- compatibility MCP tools route through the same command and structured errors.

## 7. Local Playwright Journeys

Add or extend journeys for:

```text
tests/e2e/journeys/unifiedPortConnection.spec.ts
tests/e2e/journeys/unifiedConnectionConcurrency.spec.ts
tests/e2e/journeys/lightweightFourPorts.spec.ts
tests/e2e/journeys/unifiedConnectionRecovery.spec.ts
tests/e2e/journeys/unifiedPlusConnection.spec.ts
tests/e2e/journeys/topologyVisualRecognition.spec.ts
tests/e2e/matrices/laboratoryNavigationInputMatrix.spec.ts
```

Run:

```bash
cd /home/dd/netlab/web
npm run test:acceptance-unit
NO_PROXY=127.0.0.1,localhost no_proxy=127.0.0.1,localhost npm run test:e2e:local
```

Minimum UI matrix:

1. QEMU ↔ Docker by direct port drag.
2. Docker ↔ lightweight L2 by drag to a named port and by resource-body chooser.
3. PC ↔ lightweight L3 by plus entry and keyboard-only flow.
4. lightweight L2 ↔ lightweight L3 by object-port drag.
5. QEMU/Docker ↔ Bridge and NAT by logical access target.
6. Three parallel connections remain independently selectable at 50%, 100%, and 200% zoom.
7. Capture and Traffic Filter work for each backing kind without highlighting inactive parallel links.
8. Escape, background click, same-source click, blur and laboratory switch leave no preview or mutation.
9. Fifty drag gestures cause zero unintended placement or viewport changes.

## 8. Four-Port Compatibility

Create a new lightweight L2 and L3 through SPA, HTTP, and MCP with omitted port config.

Expected:

- each new resource has exactly `eth0`, `eth1`, `eth2`, `eth3`;
- all four ports are visible and independently connectable;
- the user can alter the draft before creation;
- an existing one-port object remains one-port after upgrade, service restart, export/import and edit.

## 9. Failure, Cancellation and Cleanup

Perform 20 cycles covering invalid endpoint, occupied endpoint, revision conflict, duplicate submission, task cancellation, runtime partial failure, delete and reconnect.

Expected:

- no ghost line or permanently occupied endpoint;
- failed transactions leave no backing record, reservation, orphan task result or outbox event;
- partial runtime resources owned by failed operations are removed;
- successful deletion releases both endpoints for immediate reuse;
- audit entries record endpoint/config summaries and cleanup state without packet or terminal payloads.

## 10. Recovery

1. Create all supported connection combinations while resources are running.
2. Record connection IDs, endpoints, observed states, placements and endpoint reservations.
3. Restart the NetLab service using the local acceptance restart hook.
4. Compare the authoritative topology after recovery.
5. Delete the test laboratory and verify zero owned connection/runtime/capture resources remain.

Expected: connections and port occupancy recover exactly; 010 placements and visual routes do not globally relayout; only transient drafts and Traffic Filter animation are discarded.

## 11. Full Local Gates

```bash
cd /home/dd/netlab
gofmt -w <changed-go-files>
go test ./...

cd /home/dd/netlab/web
npm run format:check
npm run lint
npm run build
npm test
npm run test:acceptance-unit
npm run test:e2e:local
```

Run the remaining contract, recovery, privileged integration and leak checks required by `specs/001-network-simulator-platform/quickstart.md`. Record any hardware-only skip with its reason and target-host command.

## 12. Candidate and Target Validation

After all local gates pass:

1. Create focused milestone commits and ensure the final worktree is clean.
2. Build the deployment artifact from the identified commit.
3. Record commit SHA, artifact digest, migration state, contract version, build time and previous rollback artifact.
4. Deploy only that artifact to `10.72.1.7`; do not edit source on the host.
5. Run the target-host acceptance profile with mixed QEMU, Docker, PC, L2, L3, Bridge and NAT resources.

```bash
cd /home/dd/netlab
NETLAB_ACCEPTANCE_PROFILE=target-host \
NETLAB_ACCEPTANCE_BASE_URL=http://10.72.1.7:8088 \
NETLAB_ACCEPTANCE_OUTPUT_DIR=test-results/acceptance/009-target \
NO_PROXY=10.72.1.7,127.0.0.1,localhost \
no_proxy=10.72.1.7,127.0.0.1,localhost \
  ./acceptance/frontend-acceptance.sh
```

Run privileged restart and leak checks from the deployed candidate checkout/artifact directory on the
target host, using the service-local port:

```bash
NETLAB_BASE_URL=http://127.0.0.1:8088 ./acceptance/t225-service-restart.sh
NETLAB_PRIVILEGED=1 CYCLES=20 go test ./tests/integration/... -run Leak -count=1
```

Store command output and Playwright evidence under a candidate-specific directory. Record the service
unit status, `/healthz`, `/api/v1/version`, applied migration through
`0014_network_attachment_revision.sql`, artifact SHA-256, contract digest, deployment time, previous
artifact path, and cleanup counts in `validation/deployment.md` and `validation/target-acceptance.md`.

Target acceptance must include two clients plus HTTP/MCP contention, 50 drag gestures, 20 create/delete/reconnect cycles, live running-resource mutations, capture, Wireshark, Traffic Filter, service restart and laboratory deletion cleanup.

If any authority, runtime, visual, recovery or leak assertion fails, restore the previous recorded artifact and return to local development for a new tested commit and candidate.

Rollback replaces the service binary with the previously recorded artifact, runs `systemctl daemon-reload`
only if the unit changed, restarts `netlab.service`, waits for `/healthz`, verifies the reported release
identity, and records the failed candidate plus rollback result. Never patch source on the target host.
