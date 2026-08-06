# Localization Contract

## Purpose

定义当前简体中文界面的覆盖边界、核心术语、技术原文豁免和错误呈现方式，避免用户流程中出现未经批准的英文句子或不一致译法。

## Default Language

- 产品界面默认并唯一支持简体中文；本功能不增加语言切换器。
- 产品控制的标题、导航、标签、按钮、菜单、说明、状态、占位文字、工具提示、空状态、错误摘要、图表文字和无障碍名称使用中文。

## Core Terminology

| Source | Required UI Term |
|---|---|
| Console / Terminal workspace | 终端 |
| Capture | 抓包 |
| Traffic Filter | 流量过滤 |
| Tasks | 任务 |
| Inspector | 检查器 |
| Diagnostics | 诊断 |
| Settings | 设置 |
| Start / Stop / Refresh | 启动 / 停止 / 刷新 |
| Running / Stopped / Failed | 运行中 / 已停止 / 失败 |
| Reconnect | 重新连接 |
| Close / Cancel / Remove / Delete | 关闭 / 取消 / 移除 / 删除 |

同一概念在导航、菜单、检查器、空状态、错误和测试断言中使用同一译法。

## Allowed Source-Language Content

下列内容允许保留原文，但其周围解释和操作必须中文化：

- QEMU、Docker、Wireshark、VNC、SSH、Telnet、TCP、UDP、ICMP、IPv4、IPv6、DHCP、SLAAC、VLAN、NAT、pcap、pcapng；
- 设备模板、厂商型号、镜像名称与版本；
- 接口名称、IP/MAC、CIDR、资源 ID；
- 命令、路径、配置片段、设备输出；
- 用户输入的实验室和节点名称；
- 第三方原始错误和错误代码，但必须按错误合同配套中文信息。

## Empty-State Contract

空状态必须回答三个问题：

1. 当前缺少什么；
2. 用户从哪里开始；
3. 可执行的下一步是什么。

例如终端空状态必须以中文说明“尚未打开终端会话，请右键节点并选择‘终端’”。抓包空状态必须说明先选择节点/链路及接口，再开始抓包。

## Error Contract

已知错误采用以下层级：

1. **中文摘要**：发生了什么；
2. **中文建议**：用户可以执行的下一步；
3. **原始详情**：可展开查看的第三方原文、状态码或路径。

Wireshark 集成至少覆盖：辅助程序无响应、浏览器无法连接、允许来源不匹配、未找到 Wireshark、启动失败、抓包流不可用。

未知服务端状态显示为“未知状态：`raw-value`”，不得只显示原始英文，也不得丢弃原值。

## Accessibility Contract

- `aria-label`、隐藏标题、关闭按钮和图标按钮名称使用中文。
- 可见标签与辅助名称含义一致。
- 技术缩写可保留，但必须让操作目的可被中文读屏用户理解。

## Scan Contract

中文化扫描至少检查：

- Vue 模板文本节点；
- `title`、`aria-label`、`placeholder`、label、description 和 hint；
- 常见状态、空状态、异常分支和动态拼接字符串；
- 页面对象和测试断言中的旧英文产品标签。

扫描白名单必须是明确、最小且可审阅的技术术语集合。不得使用整文件忽略来掩盖产品文案遗漏。
