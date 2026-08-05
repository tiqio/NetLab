# Backend Parity Contract: Topology Resource Creation

本功能不新增或修改后端接口。UI 实现必须保持以下现有合同和自动化等价性。

## Catalog Reads

| Purpose | Existing client operation | Required drawer use |
|---|---|---|
| 查询设备模板与版本 | `api.listTemplates()` | 资源目录和表单使用同一目录快照 |
| 查询镜像版本 | `api.listImages()` | 只展示与模板版本兼容、可用且许可已审核的镜像 |

目录在用户提交前必须重新核对。若模板、版本或镜像已经变化，保留其他草稿并要求用户重新选择，不发送过期创建请求。

## Node Creation

**Operation**: `api.createNode(laboratoryId, CreateNodeRequest)`

抽屉必须沿用现有字段：

- `name`
- `kind`
- `template_version_id`
- `image_version_id`
- `interface_count`
- 适用时的 `config.network_interfaces`
- 适用时的 `bootstrap`

成功结果必须使用服务端返回的 Node 和 NodeInterface；UI 不得生成临时权威 ID。

## Network Object Creation

**Operation**: `api.createNetworkObject(laboratoryId, request)`

支持的 kind：

- `pc`
- `bridge`
- `nat_bridge`
- `switch_l2`
- `switch_l3`

配置负载必须继续由现有 PC、L2、L3、Bridge 和 NAT 默认/验证规则产生。

## Error Mapping

- 字段级 problem details 映射到对应抽屉字段。
- 配额、许可、模板不可用、版本冲突和实验室冲突显示为全局错误，同时保留草稿。
- 网络超时或未知失败允许用户修改或重试；不得在画布创建临时资源。
- 重复点击由客户端提交锁阻止；现有服务端幂等与冲突行为保持最后防线。

## UI/API/MCP Equivalence

- UI、直接 HTTP 和 MCP 创建相同类型和配置时，必须得到相同类型的 Node 或 NetworkObject 及相同权威字段。
- Drawer 的打开、滚动、草稿和错误焦点不属于 API/MCP 状态。
- 其他客户端创建的资源可在 Drawer 打开期间通过现有事件流进入画布，但不能覆盖当前 Drawer 草稿。
- UI 成功后必须通过现有 Workspace 刷新或事件收敛到与 API/MCP 查询相同的实验室快照。

## No Contract Changes

- 不新增服务器端草稿。
- 不新增批量创建。
- 不新增数据库迁移。
- 不修改资源生命周期、恢复策略、所有权或删除语义。
