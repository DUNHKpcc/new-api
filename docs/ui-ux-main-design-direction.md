# 主 UI/UX 设计方向

> 这是从当前前端默认主题提取出的「主设计思路」，不是组件清单，也不是品牌素材复制。适合迁移到后台控制台、运营面板、开发者工具和其他高频工作型产品。

## 一句话定义

**技术工作台的秩序感 + 印刷漫画的视觉记忆点：信息密度高、操作路径短、表面克制，但用黑色墨线、荧光色和硬阴影建立辨识度。**

当前默认视觉主题是 `Comic Ops`。它不是营销型页面，而是一个让用户持续处理任务、查看状态和修改配置的工作空间。

## 核心设计原则

### 1. 先服务工作，再表达风格

- 首屏优先展示任务、状态、数据和下一步操作。
- 页面应支持快速扫描、比较、筛选和批量处理。
- 装饰不能抢过数据；视觉风格通过边框、字体、色块和阴影表达，而不是大面积插画或复杂背景。
- 每个页面只保留一个主要动作，其余动作降级为次要或工具操作。

### 2. 用「指挥台」组织界面

整体结构固定为三层：

1. **顶部命令栏**：品牌、全局导航、搜索、主题和账户等全局工具；始终可见。
2. **左侧工作区导航**：按任务域分组，当前页面有明确的窄标记和背景状态；进入深层设置时切换为上下文导航，而不是无限堆叠菜单。
3. **主工作面**：页面标题、面包屑、主要动作、数据表格或工作面板，内容区域独立滚动。

桌面端强调稳定的空间分区；移动端不压缩桌面布局，而是将侧栏和顶部导航转为抽屉或菜单。

### 3. 视觉语言是「纸张、墨线、荧光标记」

- **底色**：温和的纸张色，不使用刺眼纯白作为唯一层次。
- **文字与主操作**：黑色或近黑色墨色，形成明确的高对比骨架。
- **强调色**：黄色用于注意、选中和关键提示；绿色用于成功、激活和工作区状态。
- **错误色**：只用于危险、失败和不可恢复操作，不参与装饰。
- **辅助色**：蓝色等冷色只服务信息提示和图表，不与黄色、绿色争夺主视觉。
- **背景纹理**：只允许非常克制的点阵网格，用来增加纸面/工程图气质；不使用大面积渐变、光晕或漂浮装饰。

推荐配色关系，而不是固定色值：

```text
paper/background -> warm neutral
ink/primary     -> near black
highlight       -> vivid yellow
positive        -> vivid green
danger          -> restrained red
info            -> cool blue
```

### 4. 平面优先，框架表达层级

- 页面本身保持平面，不把每个区块都做成浮动卡片。
- 卡片只用于一个完整的工作单元，例如统计摘要、表单分组、配置模块或确认内容。
- 卡片使用细小圆角、明显边框和轻微硬阴影；阴影偏移要可感知，但不能像消费类产品一样柔软发散。
- 弹窗、下拉菜单、抽屉等浮层使用更强的边框和偏移阴影，像从工作台上拿起的一张纸。
- 禁止「卡片套卡片」作为默认布局，也不要用玻璃拟态、重模糊和渐变制造层次。

### 5. 字体承担产品性格

- **标题**：使用厚重、略带漫画/技术海报气质的无衬线字体，负责建立权威感和页面节奏。
- **正文与控件**：保持清晰、紧凑、可扫描；中英文和 CJK 都要有稳定 fallback。
- **元数据、表头、代码**：使用等宽字体，强调技术属性和列对齐。
- 标题不依赖夸张字号，依靠字重、墨色和位置建立层级。
- 字距保持自然，不使用装饰性的过度负字距。

### 6. 状态比装饰更重要

所有关键状态都应同时具备：

- 颜色变化；
- 文本或图标说明；
- 可访问状态，例如 `aria-current`、`aria-expanded`、`aria-invalid` 或 `aria-disabled`。

不能只用颜色区分成功、失败、选中或禁用。加载、空数据、错误、无权限和完成状态都要有明确的页面反馈。

## 关键布局基线

这些数值是当前设计的比例参考，迁移到其他项目时可按产品密度微调：

| 项目 | 基线 |
| --- | --- |
| 顶部命令栏 | 约 64px，高度稳定 |
| 展开侧栏 | 约 240px |
| 收起侧栏 | 只保留图标，约 48px |
| 移动侧栏 | 约 280px 抽屉 |
| 默认圆角 | 2px 左右；交互控件可放宽到 4px |
| 页面内容 | 桌面端 12-16px 内边距，移动端优先压缩但不牺牲点击区域 |
| 动效 | 快速反馈 150ms；常规状态变化 250ms；复杂进入 350ms 以内 |

## 交互体验规则

- 顶部导航使用底部线条表达当前页面，不使用大面积胶囊高亮。
- 侧栏使用窄的活动标记和浅色块表达当前位置，不使用过大的圆角选中胶囊。
- 主要按钮采用实心墨色；次要按钮使用描边或低对比底色；危险按钮使用语义红色。
- 图标按钮必须有可访问名称和 tooltip；图标只是加速识别，不能替代必要文本。
- 表格、配置表单和日志页面优先保证列对齐、固定操作区、可滚动和空状态清晰。
- 桌面端可折叠侧栏，且记住用户选择；移动端使用抽屉，不让侧栏永久占用内容宽度。
- 动效只用来说明进入、退出、层级切换和状态变化；支持 `prefers-reduced-motion`，减少动态和模糊。

## 适合迁移的产品气质

适合：

- AI/API 管理后台；
- 开发者控制台；
- 运营和监控面板；
- 资源、渠道、模型、账务等配置型系统；
- 需要长时间使用的内部工具。

不适合直接用于：

- 以情绪和沉浸为主的消费类首页；
- 需要大量柔和渐变和大图叙事的品牌宣传页；
- 以内容阅读为核心、需要宽松排版的编辑型产品。

## 可直接交给其他项目的设计 brief

```text
Build a work-first operational console, not a marketing page.
Use a command-bar + contextual-sidebar + scrollable-workspace layout.
The visual language is technical-comic: warm paper canvas, near-black ink,
one yellow highlight and one green status color, crisp borders, small radii,
and restrained hard-offset shadows. Keep surfaces mostly flat; avoid glass,
large gradients, floating decorative blobs, and excessive rounded cards.
Prioritize scanability, dense tables, short action paths, explicit states,
keyboard access, responsive drawer navigation, and reduced-motion support.
Let heavy display typography establish hierarchy, use clean body text for
controls, and use monospace for metadata, table headers, and code.
```

## 当前代码中的来源

- 主题与颜色关系：`web/src/styles/theme-presets.css` 的 `operator` 主题。
- 基础语义色、圆角和字体 token：`web/src/styles/theme.css`。
- 顶部命令栏与导航状态：`web/src/components/layout/components/app-header-layout.ts`。
- 页面标题、动作区和主工作面：`web/src/components/layout/components/section-page-layout.tsx`。
- 侧栏布局与移动端抽屉：`web/src/components/ui/sidebar.tsx`、`web/src/components/layout/components/app-sidebar.tsx`。
- 动效节奏与 reduced-motion：`web/src/lib/motion.ts`。
