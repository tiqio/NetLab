# 前端中文化与主题

NetLab 产品界面固定使用简体中文；命令、设备输出、IP/MAC、接口名、协议缩写、镜像、厂商型号、错误代码和资源 ID 保持原文。

全局“外观主题”控件提供“跟随系统 / 浅色 / 深色”。偏好仅保存在当前浏览器的 `netlab.appearance.v1`，不会进入实验室状态、HTTP、MCP、事件或 SQLite。首次访问跟随系统，无法判断时使用深色。

主题通过 `data-theme` 和语义 CSS 变量应用。切换不会刷新页面、重建拓扑或关闭 Console、VNC、Capture、Traffic Filter。Console ANSI 内容和 VNC 客户机画面保持原始配色。

新增产品文案应进入 `web/src/locales/`；运行 `scripts/check-ui-localization.sh` 检查高频英文可访问名称。技术术语豁免集中维护于 `web/src/locales/terminology.ts`。
