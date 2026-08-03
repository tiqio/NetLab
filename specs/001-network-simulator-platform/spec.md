# Feature Specification: NetLab Network Simulator Platform

**Feature Branch**: `001-network-simulator-platform`

**Created**: 2026-07-23

**Status**: Draft

**Input**: User description: "构建一个比 EVE-NG 更适合团队需求的单机网络模拟器，支持有限且明确的虚拟机、容器、网络节点、在线诊断和自动化编排能力。"

## Clarifications

### Session 2026-07-23

- Q: 服务重启时如何处理运行中的节点？ → A: 节点继续运行；服务重启后自动发现、接管并恢复控制。
- Q: 无登录认证时，管理服务默认监听范围是什么？ → A: 默认监听全部网卡，由宿主机防火墙或网络边界负责限制访问。
- Q: 整台宿主机重启后，原先运行中的节点如何恢复？ → A: 每个实验室可配置恢复策略，默认自动恢复重启前的运行状态。
- Q: 实验室导出包应包含哪些内容？ → A: 包含拓扑、模板版本引用和脱敏节点配置；排除镜像、秘密、初始化敏感资料和抓包。
- Q: 首版验收的单实验室参考规模是多少？ → A: 10 个节点，其中最多 4 个 QEMU 节点。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Build and Operate a Shared Topology (Priority: P1)

实验人员在浏览器中创建实验室，放置设备和网络节点，连接接口，启动设备，并与其他浏览器
或自动化客户端同时观察和操作同一份拓扑。任何客户端的会话都不会使其他客户端失效，
所有客户端看到的节点、链路和任务状态保持一致。

**Why this priority**: 共享且可靠的拓扑操作是产品替代现有平台的首要价值，也是其他能力的基础。

**Independent Test**: 两个浏览器会话和一个自动化客户端同时打开同一实验室，分别创建、
启动、连接和停止节点；所有客户端均在规定时间内看到相同最终状态且会话持续有效。

**Acceptance Scenarios**:

1. **Given** 一个空实验室被多个客户端同时打开，**When** 任一客户端添加节点或链路，
   **Then** 其他客户端无需重新登录或刷新身份即可看到同一资源及状态。
2. **Given** 一个包含运行中节点的实验室，**When** 浏览器与自动化客户端同时提交冲突修改，
   **Then** 系统明确接受一个修改并将另一个标记为冲突或安全重试，不产生静默覆盖。
3. **Given** 节点正在启动、停止或失败恢复，**When** 新客户端打开实验室，**Then** 客户端
   获得服务器确认的当前状态以及未完成任务，而不是仅依赖本地页面状态。
4. **Given** 一个已保存实验室包含运行中节点，**When** 控制服务重启，**Then** 节点不因控制
   服务重启而停止，服务自动发现并接管现有实例，恢复控制后向所有客户端显示一致状态。
5. **Given** 实验室使用默认恢复策略且宿主机发生重启，**When** 平台重新可用，**Then** 系统
   自动重建该实验室在重启前运行的节点，并报告每个节点的恢复进度和最终结果。
6. **Given** 一个实验室包含节点配置、镜像引用、初始化秘密和抓包，**When** 用户导出实验室，
   **Then** 导出包保留拓扑、模板版本引用和脱敏配置，但不包含镜像、秘密或抓包数据。

---

### User Story 2 - Use Versioned Device Templates (Priority: P1)

实验人员从设备模板创建节点，并为每个节点选择可用镜像版本。首期虚拟机模板包括
FancyWAN、Ubuntu、FortiGate 和 VyOS，容器模板包括 BusyBox 和 Ubuntu。模板统一描述设备
能力，使镜像升级不改变用户创建拓扑的方式。

**Why this priority**: 模板和镜像版本管理决定实验是否可重复，也是用户最常使用的创建入口。

**Independent Test**: 为每个首期模板登记至少两个允许的版本，创建节点并验证模板能力、
选中版本、资源默认值、接口和控制台选项正确保存与展示。

**Acceptance Scenarios**:

1. **Given** 一个模板存在多个可用镜像版本，**When** 用户创建节点并选择版本，**Then** 节点
   固定引用该版本，模板默认版本后续变化不会悄悄改变已有节点。
2. **Given** 镜像缺失、校验不一致或未记录使用许可，**When** 用户尝试创建或启动节点，
   **Then** 系统阻止操作并说明需要修复的镜像问题。
3. **Given** Ubuntu、VyOS 或 FancyWAN 模板声明支持自动预配置，**When** 用户提供初始化资料
   并启动节点，**Then** 节点在首次启动时获得该资料且资料不会泄露给其他实验室。
4. **Given** 模板不支持某项能力，**When** 用户查看或调用该能力，**Then** 系统明确标记为
   不支持，而不是提交必然失败的任务。

---

### User Story 3 - Automate Topologies through API and MCP (Priority: P1)

自动化程序或大语言模型可以发现模板和镜像、创建实验室、编排拓扑、启动节点、执行节点
操作并读取任务结果。自动化和浏览器使用相同资源与状态，不需要专用账户，也不会影响浏览器
会话。

**Why this priority**: 自动化编排是现有平台的关键痛点，也是产品区别于传统手工模拟器的核心。

**Independent Test**: 仅使用自动化接口创建一套包含虚拟机、容器、网桥和 NAT 网桥的拓扑，
启动节点、变更链路、采集流量并删除实验室；浏览器全程观察到相同状态。

**Acceptance Scenarios**:

1. **Given** 客户端不知道系统当前资源，**When** 客户端查询能力，**Then** 返回可用模板、
   镜像版本、节点能力、操作参数和限制的结构化描述。
2. **Given** 一个创建或启动请求因超时被客户端重试，**When** 使用相同幂等标识再次提交，
   **Then** 系统返回原任务或原结果，不创建重复资源。
3. **Given** 一个长时间运行的操作，**When** 客户端查询或取消任务，**Then** 获得明确的进度、
   当前状态、结果、错误和取消结果。
4. **Given** 自动化客户端启动抓包，**When** 客户端查询结果，**Then** 返回抓包会话元数据、
   有界摘要和可流式读取或下载的制品引用，而不是无界二进制响应。

---

### User Story 4 - Diagnose Traffic Live (Priority: P2)

实验人员在节点运行期间打开 Telnet 或 VNC 控制台，选择任意链路或接口进行在线抓包，并用
数据包特征过滤器查看匹配流量经过哪些连接。用户能够把抓包流交给 Wireshark 实时分析，
也能保存有界抓包制品供后续下载。

**Why this priority**: 网络实验的价值来自快速观察和定位转发行为，而不只是启动设备。

**Independent Test**: 在三跳拓扑中生成已知流量，启动实时抓包和特征过滤，验证控制台可用、
Wireshark 能持续读取匹配数据，且界面正确标识流量经过的链路。

**Acceptance Scenarios**:

1. **Given** 一个运行中节点声明支持 Telnet 或 VNC，**When** 用户打开控制台，**Then** 用户
   能交互并在断开后重新连接，不改变节点生命周期。
2. **Given** 一条承载流量的链路，**When** 用户开始在线抓包，**Then** 用户可在规定时间内
   看到新数据包，并可停止、取消或下载抓包结果。
3. **Given** 用户配置源/目的地址、协议、端口或组合特征，**When** 匹配流量穿过拓扑，
   **Then** 系统实时标识匹配流量经过的接口和链路，并显示匹配计数与时间范围。
4. **Given** 抓包达到时长、容量或保留限制，**When** 会话结束，**Then** 系统明确标记正常
   完成或截断原因，并按保留策略清理制品。

---

### User Story 5 - Change Running Nodes Safely (Priority: P2)

高级用户可以在不停止节点的情况下修改接线，为运行中的虚拟机热添加或删除网卡，选择网卡
类型，把任意客户机端口映射到宿主机，执行客户机命令，并为节点设置 CPU 时间、内存和其他
资源限制。

**Why this priority**: 在线变更和受控执行支持故障演练、ZTP、外部 SSH 接入及资源可预测性。

**Independent Test**: 启动一个支持热插拔与客户机命令的节点，在线增加网卡并接线、建立端口
映射、执行命令、验证 CPU 时间限制，然后撤销全部资源且节点保持运行。

**Acceptance Scenarios**:

1. **Given** 两个运行中节点，**When** 用户连接或断开它们的接口，**Then** 操作无需停止节点，
   且最终接线状态对所有客户端一致可见。
2. **Given** 一个支持热插拔的虚拟机，**When** 用户添加或删除网卡，**Then** 系统报告每一步
   结果；失败时保留或恢复到可解释的一致状态。
3. **Given** 用户为客户机端口配置宿主机映射，**When** 映射可用，**Then** 外部工具可访问
   目标服务；端口冲突时系统拒绝创建并指出冲突资源。
4. **Given** 一个可执行客户机命令的节点，**When** 用户提交命令，**Then** 系统返回任务状态、
   退出状态和受限输出，并记录超时、取消或不可用原因。
5. **Given** 一个配置两个虚拟 CPU 但限额等同一个宿主机核心时间的节点，**When** 节点产生
   持续计算负载，**Then** 节点保留两个虚拟 CPU，同时实际 CPU 时间不超过所配限额的容差。

---

### User Story 6 - Add Lightweight Network and PC Nodes (Priority: P3)

实验人员可以创建轻量二层交换机、三层交换机和 PC 节点，用于不需要完整虚拟机的基础网络
实验。PC 节点支持静态 IPv4/IPv6、DHCPv4、DHCPv6 和 IPv6 SLAAC，并可与虚拟机、容器、
网桥和 NAT 网桥共同接线。

**Why this priority**: 轻量节点降低资源消耗并补全常见接入、路由和地址分配验证场景。

**Independent Test**: 创建双栈 PC、二层交换机、三层交换机、DHCP 服务和 NAT 出口组成的拓扑，
验证各地址获取方式、二层转发、三层转发和外部连通性。

**Acceptance Scenarios**:

1. **Given** 一个二层交换机节点连接多个端点，**When** 端点交换同一广播域流量，**Then**
   交换机按二层行为转发且流量可被抓包和过滤观察。
2. **Given** 一个三层交换机节点配置多个网络，**When** 端点发送跨网段流量，**Then** 流量
   按已配置的三层规则转发。
3. **Given** PC 节点选择 DHCPv4、DHCPv6 或 SLAAC，**When** 对应服务可达，**Then** PC 获得
   地址、默认路由和可用的诊断状态；服务不可达时显示超时而不无限等待。
4. **Given** 一个 NAT 网桥连接内部节点，**When** 内部节点访问允许的外部目标，**Then**
   能正常通信，删除实验室后相关转换和转发规则被清理。

### Edge Cases

- 服务在节点启动、网卡热插拔、抓包或删除资源的中间阶段重启。
- 虚拟机、容器或网络进程在系统未发出停止请求时退出。
- 两个客户端同时修改同一节点、接口、端口映射或模板版本。
- 用户删除仍有运行节点、活动抓包、开放控制台或未完成任务的实验室。
- 镜像文件被移动、损坏、重复登记、校验不一致或缺少许可证说明。
- 宿主机端口、接口名、网络地址、MAC 地址或资源标识发生冲突。
- 网卡热添加部分成功，但客户机未识别设备或后续接线失败。
- 客户机代理不存在、失联、返回超量输出或命令超过执行时限。
- 抓包没有匹配数据、产生数据过快、客户端过慢、连接中断或制品空间不足。
- 流量过滤规则无效、范围过宽或匹配包跨越环路并重复出现。
- DHCPv4、DHCPv6、SLAAC 或 NAT 的上游服务不可用或配置互相冲突。
- 宿主机资源不足以满足启动请求，或 CPU 时间限额低于可执行下限。
- 删除操作只清理了部分虚拟机、容器、命名空间、网桥、端口映射或抓包进程。

## Requirements *(mandatory)*

### Functional Requirements

#### Shared Labs and Topology State

- **FR-001**: System MUST provide shared laboratories containing nodes, interfaces, links, placement,
  configuration references, and lifecycle state.
- **FR-002**: System MUST maintain one server-authoritative state for every laboratory and expose the
  same durable and runtime state to browser, HTTP automation, and MCP clients.
- **FR-003**: System MUST allow multiple clients to remain connected and operate concurrently without
  invalidating another client's session.
- **FR-004**: System MUST detect conflicting concurrent mutations and return an explicit accepted,
  rejected, or retryable result without silent data loss.
- **FR-005**: System MUST persist topology desired state and durable operation history across service
  restarts.
- **FR-006**: System MUST leave managed node instances running during a control-service restart, then
  automatically discover, validate ownership of, adopt, and resume control of those instances while
  reconciling persisted desired state with actual process, network, capture, and port-mapping state.
  After a full host restart, each laboratory MUST follow its configurable recovery policy, which
  defaults to recreating the nodes that were running immediately before the restart.
- **FR-007**: System MUST model node operations with explicit requested, transitional, running,
  stopped, failed, and deleting outcomes, including timestamps and actionable errors.
- **FR-008**: System MUST support creating, renaming, duplicating, exporting, importing, and deleting
  laboratories without requiring account ownership or per-user visibility partitions. Export packages
  MUST contain topology, template-version references, and redacted non-secret node configuration, and
  MUST exclude device images, credentials, bootstrap secrets, and packet captures. Import MUST validate
  the package before creating resources and report missing template or image versions without silently
  substituting a different version.

#### Node and Network Types

- **FR-009**: System MUST support QEMU virtual-machine nodes and Docker container nodes.
- **FR-010**: System MUST support standard bridges and NAT bridges as topology network objects.
- **FR-011**: System MUST support lightweight PC, layer-2 switch, and layer-3 switch nodes based on
  isolated network contexts.
- **FR-012**: System MUST allow links to connect compatible interfaces across virtual-machine,
  container, PC, switch, bridge, and NAT-bridge nodes.
- **FR-013**: System MUST support connecting and disconnecting links while affected nodes are running.
- **FR-014**: System MUST expose link and interface operational status separately from desired wiring.
- **FR-015**: System MUST clean up every node-owned interface, bridge attachment, route, forwarding
  rule, address, and helper process after deletion or failed creation.

#### Templates and Images

- **FR-016**: System MUST provide QEMU template families for FancyWAN, Ubuntu, FortiGate, and VyOS.
- **FR-017**: System MUST provide Docker template families for BusyBox and Ubuntu.
- **FR-018**: System MUST allow each template family to offer multiple selectable image versions.
- **FR-019**: System MUST keep each created node pinned to the selected image version until explicitly
  changed through a supported migration or replacement action.
- **FR-020**: Each template version MUST declare image identity, checksum, source/provenance, format,
  license or entitlement notes, supported consoles, interface limits, default resources, supported
  network adapter types, bootstrap methods, guest-command capability, and hot-plug capability.
- **FR-021**: System MUST reject unavailable, checksum-mismatched, incompatible, or license-unreviewed
  images before starting a node.
- **FR-022**: System MUST NOT automatically obtain proprietary appliance images from unofficial
  repositories; operators supply images they are entitled to use.
- **FR-023**: System MUST allow template metadata and image versions to change without requiring
  topology users to learn a different node-creation workflow.

#### Bootstrap, Console, and Guest Operations

- **FR-024**: System MUST support attaching cloud-init seed material to compatible QEMU nodes for
  Ubuntu, VyOS, and FancyWAN automated preconfiguration.
- **FR-025**: System MUST isolate bootstrap material by laboratory and node and MUST prevent secrets
  from appearing in logs, exports, task summaries, or other laboratories.
- **FR-026**: System MUST provide Telnet and VNC online console sessions for templates that declare
  the corresponding console capability.
- **FR-027**: System MUST allow console sessions to reconnect without changing node lifecycle state.
- **FR-028**: System MUST execute arbitrary commands through an available QEMU guest agent and return
  task state, exit status, bounded standard output, bounded error output, timeout, and cancellation.
- **FR-029**: System MUST clearly report when a guest agent is absent, not ready, or disconnected.
- **FR-030**: System MUST retain an audit record for privileged guest-command requests without
  recording supplied secrets or unrestricted command output.

#### Live Virtual-Machine Operations and Resources

- **FR-031**: System MUST support hot-adding and hot-removing QEMU network adapters through the
  running virtual machine's control interface when the selected template supports it.
- **FR-032**: System MUST make multi-step hot-plug outcomes observable and MUST reconcile or roll back
  partial failures to a documented consistent state.
- **FR-033**: System MUST allow users to select a supported network adapter driver for each QEMU
  interface.
- **FR-034**: System MUST map any valid, unoccupied host TCP or UDP port to a declared guest address
  and port, including mappings used for SSH and ZTP workflows.
- **FR-035**: System MUST detect host-port conflicts before activation and MUST remove mappings after
  node, mapping, or laboratory deletion.
- **FR-036**: System MUST support independent virtual CPU count and CPU-time quota settings, including
  assigning two virtual CPUs while limiting total CPU time to approximately one host core.
- **FR-037**: System MUST enforce declared memory, storage, CPU-time, interface, and process limits and
  expose both configured limits and observed runtime status.
- **FR-038**: System MUST reject node starts that cannot be safely satisfied by available host
  resources and provide an actionable reason. The first-release acceptance baseline MUST support a
  single laboratory with at least 10 total nodes, including up to 4 simultaneously running QEMU nodes.

#### Addressing and Lightweight Nodes

- **FR-039**: PC nodes MUST support static IPv4 and IPv6 addresses, routes, and DNS settings.
- **FR-040**: PC nodes MUST support DHCPv4, DHCPv6, and IPv6 SLAAC independently or in compatible
  combinations.
- **FR-041**: PC nodes MUST expose acquired addresses, routes, lease or advertisement status, and
  diagnostic errors.
- **FR-042**: Layer-2 switch nodes MUST provide configurable port membership and observable forwarding
  across one or more broadcast domains.
- **FR-043**: Layer-3 switch nodes MUST provide configurable interfaces, addresses, routes, and
  forwarding state sufficient for multi-network laboratory scenarios.
- **FR-044**: NAT bridges MUST expose their internal network, address allocation, external attachment,
  translation status, and cleanup status.

#### Capture, Traffic Filters, and Artifacts

- **FR-045**: System MUST start and stop live packet captures on selectable links or interfaces without
  stopping connected nodes.
- **FR-046**: System MUST provide a live capture stream usable by Wireshark-compatible workflows and a
  downloadable capture artifact when retention is requested.
- **FR-047**: Every capture session MUST expose status, source, start/end time, byte and packet counts,
  active filter, retention deadline, completion reason, truncation state, and error state.
- **FR-048**: Automation and MCP capture results MUST return structured metadata, a bounded packet
  summary, and a retrievable stream or artifact reference instead of embedding unbounded capture data.
- **FR-049**: System MUST support traffic filters based on source/destination addresses, protocol,
  source/destination ports, and logical combinations of those fields.
- **FR-050**: System MUST identify the interfaces and links traversed by packets matching an active
  traffic filter and expose match counts and observation times.
- **FR-051**: System MUST enforce configurable capture duration, size, concurrency, and retention
  limits and clearly distinguish completed, cancelled, failed, and truncated sessions.
- **FR-052**: System MUST remove expired capture artifacts and abandoned capture processes without
  affecting running nodes.

#### Automation and Operational Safety

- **FR-053**: System MUST expose versioned HTTP operations for every durable resource and state-changing
  browser action.
- **FR-054**: System MUST expose typed MCP tools for topology discovery, laboratory orchestration, node
  lifecycle, link changes, guest commands, console discovery, captures, traffic filters, and task status.
- **FR-055**: System MUST return a task identifier for long-running operations and expose task status,
  progress, result, error, timestamps, and cancellation state.
- **FR-056**: Retriable create and mutation operations MUST accept an idempotency identifier or have
  inherently idempotent behavior.
- **FR-057**: System MUST publish state changes so connected clients can converge without repeatedly
  reloading the complete laboratory.
- **FR-058**: System MUST provide structured, actionable errors that identify the failed resource,
  lifecycle phase, retryability, and any cleanup still in progress.
- **FR-059**: System MUST provide operational audit records for topology mutations, lifecycle actions,
  guest commands, port mappings, captures, and traffic filters.
- **FR-060**: System MUST operate without application-level account/password login and without
  account-scoped topology or runtime state.
- **FR-061**: System MUST listen on all host network interfaces by default, allow administrators to
  restrict management exposure to selected interfaces, and prominently warn that host firewall or
  trusted-network controls are required because the application provides no login authentication.
- **FR-062**: System MUST NOT store deployment credentials, proprietary images, bootstrap secrets, or
  captured user traffic in source-controlled project assets.

### Key Entities

- **Laboratory**: Shared topology workspace containing nodes, links, captures, filters, revisions, and
  desired-state metadata, including its host-restart recovery policy.
- **Laboratory Export**: Portable, versioned representation of topology, template-version references,
  and redacted non-secret configuration, excluding images, credentials, bootstrap secrets, and captures.
- **Node**: Runtime participant with a type, template version, resource limits, lifecycle state,
  interfaces, console capabilities, and runtime ownership information.
- **Device Template**: Named family describing supported capabilities, defaults, constraints, and
  selectable image versions.
- **Image Version**: Immutable image reference with version, checksum, provenance, format, availability,
  compatibility, and license/entitlement status.
- **Interface**: Node connection point with adapter type, desired attachment, operational state, and
  optional addressing information.
- **Link**: Desired and observed connection between compatible interfaces, including live-change state.
- **Network Object**: Bridge, NAT bridge, PC, or switching/routing node with network-specific settings
  and owned host resources.
- **Port Mapping**: Host protocol/address/port binding to a guest endpoint with ownership and conflict
  status.
- **Bootstrap Profile**: Node-scoped initialization data and non-secret metadata with lifecycle and
  redaction rules.
- **Operation Task**: Durable asynchronous action record containing intent, progress, result, error,
  cancellation, idempotency, timestamps, and affected resources.
- **Console Session**: Reconnectable Telnet or VNC access descriptor associated with a node.
- **Capture Session**: Live or completed packet observation with source, filter, limits, counters,
  retention, stream reference, artifact reference, and completion state.
- **Traffic Filter**: Packet-match definition and its observed traversal results across interfaces and
  links.
- **Audit Event**: Redacted record of a privileged or state-changing action and its outcome.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A first-time operator can create, wire, start, inspect, stop, and delete a four-node mixed
  topology in no more than 10 minutes using only the browser.
- **SC-002**: An automation client can create and fully remove the same reference topology without
  browser interaction, and 100% of its durable actions are visible in an already-open browser session.
- **SC-003**: In concurrent-client tests, 100% of accepted topology mutations converge to the same final
  state in all connected clients within 3 seconds, with no session invalidation.
- **SC-004**: Users can connect or disconnect a running-node link and see a confirmed final result within
  10 seconds without stopping either node in at least 95% of valid attempts.
- **SC-005**: A live packet capture begins presenting newly observed matching traffic within 5 seconds
  of a valid start request, and every ended session reports whether it completed, failed, was cancelled,
  or was truncated.
- **SC-006**: In a known three-hop traffic test, the traffic-filter view identifies every traversed test
  link with no false path omissions across 100 consecutive matching flows.
- **SC-007**: After an unplanned service restart during each supported lifecycle transition, the system
  reaches an accurate, client-visible reconciled state within 60 seconds on the acceptance host.
- **SC-008**: After 100 create/start/stop/delete cycles of the reference topology, no topology-owned
  process, interface, bridge attachment, namespace, port mapping, capture process, or retained artifact
  remains beyond its configured cleanup window.
- **SC-009**: Every retry test using the same idempotency identifier creates at most one durable resource
  and returns one consistent operation outcome.
- **SC-010**: All supported templates can be instantiated from a selected image version, and 100% of
  invalid or checksum-mismatched images are rejected before node startup.
- **SC-011**: A reference dual-stack PC topology successfully demonstrates static IPv4/IPv6, DHCPv4,
  DHCPv6, SLAAC, layer-2 forwarding, layer-3 forwarding, and NAT connectivity in one repeatable test run.
- **SC-012**: At least 90% of evaluators complete the primary topology-building and live-capture tasks
  without assistance on their first attempt, and no critical usability blocker remains open for release.
- **SC-013**: After a full acceptance-host restart, 100% of laboratories using automatic recovery
  recreate every node that was running before shutdown or report a specific terminal failure within
  5 minutes; laboratories with recovery disabled start no nodes automatically.
- **SC-014**: Exporting and re-importing each reference laboratory preserves 100% of its topology,
  template-version references, and non-secret configuration while automated inspection finds no image,
  credential, bootstrap secret, or packet-capture payload in the export package.
- **SC-015**: On the declared acceptance host, one laboratory containing 10 nodes, including 4 running
  QEMU nodes, can complete topology viewing, lifecycle operations, live link changes, console access,
  and packet capture without resource-limit failures or loss of shared-state convergence.

## Assumptions

- The first release runs on one trusted Linux laboratory server; multi-host clustering and migration
  between servers are outside scope.
- The application listens on all host network interfaces by default and has no account/password login,
  user ownership, role model, or per-user state partition. Deployments rely on host firewall or
  trusted-network controls to prevent unauthorized access.
- Operators legally acquire and manually provide commercial appliance images. The product distributes
  metadata and validation behavior, not proprietary images.
- FancyWAN, Ubuntu, and VyOS images used for acceptance expose bootstrap behavior compatible with the
  supplied initialization material; unsupported image versions are marked accordingly.
- Guest command execution and virtual-machine network hot-plug are available only when the selected
  image and template declare the required guest/control capability.
- Capture artifacts use bounded, configurable retention. The default policy may be chosen during
  planning but must be visible to users and automation clients.
- Capacity targets are validated on a declared acceptance-host configuration using a documented
  10-node reference topology containing up to 4 QEMU nodes; different image workloads may require
  different resource allocations.
- Each laboratory has a host-restart recovery policy. Automatic restoration of the pre-restart running
  set is enabled by default and can be disabled for laboratories that must remain stopped after reboot.
- Cisco device emulation, exhaustive EVE-NG template compatibility, multi-node clusters, and
  application-level authentication are explicitly outside scope.
