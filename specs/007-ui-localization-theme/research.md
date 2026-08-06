# Research: 全页面中文化与明暗主题

## Decision 1: 使用固定简体中文资源，而不是引入多语言框架

**Decision**: 新建类型化的 `zh-CN` 文案、术语和状态映射模块，当前仅提供简体中文；不增加新的运行时国际化依赖。

**Rationale**: 当前产品没有语言切换需求，核心问题是消除分散英文和不一致译法。集中资源可以支持扫描、复用和测试，同时避免为单语言引入额外体积和运行复杂度。

**Alternatives considered**:

- 引入完整国际化框架：未来多语言更方便，但当前会增加依赖、键路径迁移和异步加载复杂度。
- 直接在每个组件替换为中文：实施快，但无法持续检查术语一致性，后续容易重新出现英文。

## Decision 2: 产品文案与技术原文分层处理

**Decision**: 产品控制的标题、动作、状态、说明和错误上下文必须中文化；命令、设备输出、资源名、协议、接口名、IP/MAC、镜像标签、厂商型号和错误代码保持原文。

**Rationale**: 技术内容的精确性高于形式一致，盲目翻译可能导致命令不可执行或设备含义改变。中文上下文可以解释影响和下一步，而原文用于排障和自动化对照。

**Alternatives considered**:

- 翻译所有英文：表面一致但会损害技术准确性。
- 仅翻译导航：无法解决任务、诊断和错误路径中的认知负担。

## Decision 3: 建立自动文案扫描门禁

**Decision**: 增加仓库脚本扫描 Vue 模板中的用户可见英文文本、英文 `aria-label`、`title`、`placeholder` 和常见运行时字符串；协议、产品名和技术标识通过受审计豁免表保留。

**Rationale**: 全页面范围大，仅靠人工检查无法防止遗漏和回归。扫描结果必须可定位到文件，并纳入格式/测试门禁。

**Alternatives considered**:

- 仅靠人工清单：容易漏掉错误分支和可访问名称。
- 禁止所有 ASCII 文本：会误报协议、命令、型号和测试标识。

## Decision 4: 主题状态使用三态偏好和二态解析

**Decision**: 用户偏好为 `system | light | dark`，实际应用主题为 `light | dark`。未保存偏好时为 `system`；设备无明确信息时解析为 `dark`。

**Rationale**: 三态模型能区分“跟随设备”和“用户明确选择”，从而正确处理运行中的系统主题变化，同时满足显式白天/黑夜切换。

**Alternatives considered**:

- 仅保存布尔值：无法表达跟随系统，也无法区分默认与手动选择。
- 每个实验室保存主题：主题是客户端显示偏好，与实验室资源无关，会造成不必要切换。

## Decision 5: 主题偏好独立于 Workspace 偏好

**Decision**: 使用全局浏览器键 `netlab.appearance.v1` 保存主题偏好，不放入按实验室保存的 WorkspacePreferences。

**Rationale**: 用户通常希望所有页面和实验室使用相同主题；按实验室存储会造成切换实验室时突然变色，并误导为共享实验室状态。

**Alternatives considered**:

- 复用 WorkspacePreferences：现有键按实验室划分，不符合全局外观语义。
- 服务端持久化：产品无账户体系，且会破坏客户端隔离。

## Decision 6: 使用语义 CSS 变量表达两套主题

**Decision**: 保留 `background/card/popover/foreground/border/ring/success/warning/destructive/info` 等语义变量，为浅色和深色分别赋值；拓扑、图表、粒子和网格增加专用语义变量。

**Rationale**: 组件继续按用途使用颜色，无需在每处判断主题。专用可视化变量可以避免图表和拓扑依赖仅适合深色背景的硬编码颜色。

**Alternatives considered**:

- 对整个页面使用 CSS 滤镜：会破坏图片、终端、图表和状态颜色。
- 在组件内使用主题条件分支：重复逻辑多，容易出现局部漏切换。

## Decision 7: 在应用挂载前解析主题

**Decision**: 页面入口在加载样式和应用前读取已保存偏好及系统外观，将最终 `data-theme` 和 `color-scheme` 应用到根元素；运行时 composable 接管后续变化。

**Rationale**: 如果等待应用挂载后再切换，浅色用户会先看到深色闪烁。入口逻辑必须短小、无依赖、异常安全，并与运行时解析规则共用相同数据合同。

**Alternatives considered**:

- 仅在根组件挂载时切换：实现简单但存在明显闪烁。
- 服务端按 Cookie 输出主题：当前无该需求，且会让本地偏好进入服务端边界。

## Decision 8: 图表主题更新时保留交互状态

**Decision**: EChart 包装层接收解析主题和语义调色板；主题变化时更新主题相关 option，必要时安全重建实例，并保存/恢复 dataZoom、graph roam、选中和当前数据。

**Rationale**: ECharts 初始化主题通常在实例创建时确定，简单修改页面 CSS 不会覆盖 Canvas 内颜色。重建或合并更新必须避免导致拓扑缩放、节点位置和选择丢失。

**Alternatives considered**:

- 只更新背景 CSS：Canvas 内轴线、文字、图例和节点不会同步。
- 每次切换无条件清空重建：可能丢失用户当前视口和交互状态。

## Decision 9: Console 与 VNC 保留客户机内容配色

**Decision**: 仅主题化 Console/VNC 外围标签、工具栏、边框、状态和空态；终端字符颜色与远程桌面画面保持客户机或会话定义。

**Rationale**: 强制改写终端 ANSI 色或 VNC 图像会改变客户机真实显示，并可能降低可读性或排障准确性。

**Alternatives considered**:

- 强制终端跟随页面主题：会覆盖应用输出和用户 shell 配置。
- 对 VNC 使用滤镜：会失真且错误表达设备界面颜色。

## Decision 10: 目标验收必须验证运行会话连续性

**Decision**: 在 `10.72.1.7` 使用运行中的 QEMU、Docker、链路、Console、Capture/Traffic Filter 场景切换主题，比较切换前后资源 ID、任务、会话和拓扑视口。

**Rationale**: 本功能理论上只涉及显示，但全局样式、图表重建和组件重新渲染仍可能意外关闭会话或重置工作区，必须用真实运行场景验证。

**Alternatives considered**:

- 仅做截图对比：无法发现 WebSocket、抓包和运行任务被重建或中断。
- 仅本地模拟 API：无法覆盖目标机真实 Console、QEMU 和网络观测会话。
