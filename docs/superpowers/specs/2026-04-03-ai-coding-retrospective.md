# LinkStash AI Coding 项目复盘

- **Date**: 2026-04-03
- **Status**: Approved
- **Framework**: 全生命周期 × KPT 融合

## 项目概览

| 维度 | 数据 |
|------|------|
| **项目** | LinkStash — 个人 URL 管理系统 |
| **周期** | 2026-03-19 → 2026-04-03（15 天） |
| **产出** | 137 commits, ~6000 行 Go + ~1700 行 Preact 前端 |
| **交付物** | 全栈 Web 应用 + CLI 工具 + Alfred Workflow + PopClip 插件 |
| **协作模式** | 人类（需求/决策）+ AI（Claude Code with Superpowers） |
| **核心功能** | URL CRUD、LLM 智能分析、混合搜索、短链、网络分类、一键部署 |

**复盘目标**:
1. 沉淀可复用的 AI Coding 最佳实践
2. 识别效率瓶颈和质量风险点
3. 形成新项目启动检查清单

---

## 阶段一：需求沟通与构思

### ✅ Keep

**1. 设计文档先行，代码后写**

每个功能都有独立的 design spec（共 7 份），再转化为 implementation plan（共 5 份）。这种 **Spec → Plan → Code** 三文档模式效果极佳：设计讨论不被代码细节干扰，实施阶段有明确蓝图可执行。

实测效果：Network Type 功能从 spec 到全部代码提交仅用约 2 小时，得益于实施计划的精细度。

**2. Decision Table 记录决策理由**

每份 spec 都有 Key Decisions 表（决策 / 选择 / 理由），确保回溯时能理解"为什么这样做"。例如 Multi-Database spec 中 "SQLite: FTS5 existing; MySQL: Bleve already in codebase"，选择有据可查。

**3. 明确的 Scope / Out-of-Scope 边界**

每份 spec 都划定不做什么，有效控制范围蔓延。在 AI 协作中尤其重要——AI 倾向于扩展功能范围，明确边界是必要的约束。

### ⚡ Problem / Try

**4. 缺少需求优先级排序机制**

15 天内推进了 14+ 个功能点，节奏很快但无明确优先级标注。可能导致先做了非核心功能。

**→ Try**: 新项目启动时先做 **MVP Feature Map**，标注 P0/P1/P2 优先级，AI 和人类对齐核心路径后再展开。

**5. 用户场景描述可更具体**

Spec 中 Goal 通常是一句话技术描述（如 "Add network_type classification"），缺少用户视角的使用场景。

**→ Try**: 加入 1-2 个 **User Story**（"作为用户，我想看到链接是国内还是海外的，以便选择合适的网络环境打开"），帮助 AI 更准确地理解意图和使用场景。

---

## 阶段二：架构设计与技术选型

### ✅ Keep

**1. DDD + Clean Architecture 从第一天确立**

初始 commit 就包含设计 spec 和清晰分层架构。请求流 `chi router → handler → usecase → repo/service → infra` 贯穿始终，无架构漂移。后期加功能（网络类型、Fetcher 策略链）可精准定位修改位置。

**2. 技术选型追求零依赖 / 低依赖**

- Pure Go SQLite（modernc, 无 CGO）→ 跨平台编译无痛
- Preact 而非 React → 3KB vs 40KB，个人项目的正确取舍
- Tailwind v4 内联 @theme → 无 tailwind.config.js，减少构建链复杂度

**3. Strategy Pattern 解决浏览器获取问题**

Fetcher 策略链（HTTP → Browser fallback）设计优雅，配置驱动，向后兼容。Spec 中明确了 "500MB+ 内存消耗" 的问题根因，先诊断再开方。

### ⚡ Problem / Try

**4. 搜索架构多次调整**

从 Bleve FTS → FTS5 + 向量搜索 → Hybrid Search，搜索方案经历了演化。虽然迭代本身合理，但更深入的初期分析可减少调整次数。

**→ Try**: 对核心功能（搜索、认证等），在 spec 阶段做 **技术方案 Spike**（限时 30 分钟的原型验证）再定方案。

**5. 前端状态管理策略未显式记录**

Signals 用于全局 auth，useState 用于组件——混合策略合理，但没有在 spec 中显式说明选型理由。

**→ Try**: 前端 spec 中加一节 **State Management Strategy**，明确哪些状态全局、哪些局部、为什么。

---

## 阶段三：实施计划与执行

### ✅ Keep

**1. 实施计划精细度极高**

每个 Task 包含：文件列表、逐步 checkbox、完整代码片段、验证命令、预写 commit message。精细到"新开发者可以照搬执行"的程度——这是 AI Coding 的核心最佳实践。

**2. 依赖感知的任务排序**

Config → Entity → Backend → Frontend → CLI，每次都是这个顺序。后端先于前端、数据模型先于业务逻辑——有效减少返工。

**3. 每步都有验证检查点**

`make frontend-js`、`make build-server`、`make test` 穿插在计划中。不是写完所有代码再测，而是增量验证——发现问题早、修复成本低。

**4. Commit 粒度合理**

一个逻辑功能一个 commit（不是一个文件一个），兼顾可读性和 git bisect 友好性。

### ⚡ Problem / Try

**5. 测试覆盖率可以更高**

当前以 smoke test 和 `make test` 为主，缺少针对核心功能的单元测试。Fetcher 策略链、搜索排序逻辑、短链编码等核心逻辑值得专门测试。

**→ Try**: Plan 中为核心逻辑增加 **Test Task**，采用 TDD 方式（写测试 → 实现 → 验证）。

**6. 前端缺少自动化测试**

Preact 组件没有测试，全靠手动验证。对个人项目可以接受，但如果项目规模增长，这是技术债。

**→ Try**: 至少为核心交互（搜索、过滤、无限滚动）写集成测试。

---

## 阶段四：调试与问题解决

### ✅ Keep

**1. 问题根因分析先于修复**

Fetcher 策略链的设计动机是"Browser 500MB+ 内存"，先诊断根因再设计方案。比直接 patch 更持久。

**2. Fix commit 信息清晰**

`fix: add json tags to NetworkTypeOption`、`fix: card clipping`——一句话说明修了什么，便于后续回溯和 changelog 生成。

### ⚡ Problem / Try

**3. 缺少系统化的 Debug 日志策略**

Go 后端用 slog，但从代码量看日志点不多。遇到问题时可能需要临时加 print。

**→ Try**: 在关键路径（LLM 调用、搜索、Fetcher 链）增加结构化日志，预留调试能力。

**4. 错误处理模式可以更统一**

Handler 层的错误返回格式未统一定义。

**→ Try**: 下个项目在架构 spec 中加 **Error Handling Convention** 节，统一定义 error code + message + detail 格式。

---

## 阶段五：部署与运维

### ✅ Keep

**1. 一键部署脚本**

INSTALL.sh 从零到运行只需一条 curl 命令，极大降低部署门槛。

**2. 多种部署方式并存**

Docker + Systemd + Caddy 反向代理，覆盖不同场景需求。

**3. 嵌入式静态资源**

v0.4.0 起前端资源编译后嵌入 Go binary → 单文件部署，运维极简。

**4. Content Hash 缓存击穿**

自动 hash 注入，避免用户看到旧版前端——虽然是小功能但避免了很多部署后的困惑。

### ⚡ Problem / Try

**5. 缺少健康检查和监控端点**

没有 `/health` 或 `/metrics` 端点，线上问题排查不便。

**→ Try**: 初始 spec 中就规划基础运维端点（health、version、metrics）。

**6. 环境配置管理可以更规范**

`.env` + YAML + 环境变量三种方式混用，新成员可能困惑。

**→ Try**: 统一配置加载优先级文档，明确各配置源的覆盖关系。

---

## 阶段六：AI 协作模式专项

### ✅ Keep

**1. Superpowers 工作流高度有效**

Brainstorming → Design Spec → Writing Plans → Executing Plans 四步工作流，减少了"AI 理解偏差导致返工"的风险。实践证明：投入在设计阶段的时间，在实施阶段多倍回报。

**2. CLAUDE.md 作为项目知识库**

包含构建命令、架构图、路由、约定——AI 每次启动都能快速理解项目上下文。特别是 JSON field name 约定（snake_case not GORM uppercase）避免了反复纠正。

**3. 功能分批迭代，不一次做太多**

每次会话聚焦一个功能（UI 美化 / 搜索增强 / 网络分类），避免上下文超载。这是与 AI 协作的关键经验：**一个会话一个功能**。

### ⚡ Problem / Try

**4. 长会话中的上下文丢失**

137 commits / 15 天意味着大量会话，AI 无法跨会话记忆前序决策。CLAUDE.md 部分缓解了这个问题，但无法覆盖所有隐式知识。

**→ Try**: 每个功能完成后，在 CLAUDE.md 中追加 **决策备忘录**（一行说明关键选择和理由）。

**5. AI 生成代码的一致性审查**

AI 可能在不同会话中产生风格略有差异的代码（命名、错误处理模式、注释风格）。

**→ Try**: 在 CLAUDE.md 中补充 **Code Style Guide** 小节（命名约定、错误处理模板、日志格式），为 AI 提供风格约束。

**6. Prompt 精度与效率的权衡**

详细 prompt 获得更精确结果但写 prompt 耗时；模糊 prompt 快但可能返工。

**→ Try**: 建立判断标准——核心功能用 **Spec 驱动**（高精度），小修小补用 **直接指令**（快速）。判断界限：涉及 2+ 文件或新概念引入的改动，走 Spec 流程。

---

## 全局 KPT 汇总

### 🟢 Keep — 继续保持

| # | 实践 | 效果 |
|---|------|------|
| K1 | Spec → Plan → Code 三文档工作流 | 零架构返工，AI 执行准确率极高 |
| K2 | Decision Table 记录决策理由 | 可追溯、可复用、新成员可理解 |
| K3 | DDD + Clean Architecture 从 Day 1 确立 | 后期加功能无痛，修改位置精准 |
| K4 | 实施计划精细到 copy-paste 级别 | AI 执行几乎不偏离预期 |
| K5 | 每步验证检查点 | 问题发现早、修复成本低 |
| K6 | CLAUDE.md 项目知识库 | AI 启动快、理解准、约定不走样 |
| K7 | 功能分批迭代，一次一个 | 避免上下文超载和功能纠缠 |
| K8 | 嵌入式资源 + 一键部署 | 运维极简，部署零摩擦 |
| K9 | 零依赖 / 低依赖技术选型 | 构建简单，跨平台友好 |
| K10 | 问题根因分析先于修复 | 方案持久，不反复 patch |

### 🟡 Problem — 遇到的问题

| # | 问题 | 影响 |
|---|------|------|
| P1 | 缺少 MVP 优先级排序 | 可能先做了非核心功能 |
| P2 | 测试覆盖不足（后端） | 回归风险，重构信心不足 |
| P3 | 搜索方案多次调整 | 返工成本，代码残留 |
| P4 | 前端无自动化测试 | 手动验证耗时，易遗漏 |
| P5 | 跨会话上下文丢失 | 决策可能重复讨论 |
| P6 | 错误处理模式不统一 | 代码一致性差，前端处理复杂 |
| P7 | 缺少运维端点 | 线上问题排查不便 |
| P8 | AI 生成代码风格差异 | 代码库一致性下降 |

### 🔵 Try — 下次改进

| # | 改进项 | 应用时机 |
|---|--------|---------|
| T1 | 启动时做 MVP Feature Map (P0/P1/P2) | 项目启动第一天 |
| T2 | 核心功能 TDD + 单元测试 | 每个 Plan 中加 Test Task |
| T3 | 技术方案 Spike（30 分钟原型验证） | 遇到不确定技术选型时 |
| T4 | CLAUDE.md 追加决策备忘录 | 每个功能完成后 |
| T5 | Code Style Guide 加入 CLAUDE.md | 项目启动时定义 |
| T6 | 基础运维端点在初始 spec 中规划 | 架构设计阶段 |
| T7 | 统一错误处理约定 | 架构 spec 中定义 |
| T8 | 前端状态管理策略显式记录 | 前端 spec 中 |
| T9 | Spec 中加入 User Story | 需求沟通阶段 |
| T10 | 关键路径结构化日志 | 实施阶段 |

---

## 新项目启动检查清单

> 基于 LinkStash 项目 15 天开发经验沉淀

### 🚀 Day 1 — 项目初始化

- [ ] 创建 CLAUDE.md（架构概览、构建命令、Code Style Guide、约定）
- [ ] 做 MVP Feature Map，标注 P0/P1/P2 优先级
- [ ] 确立架构模式（分层 / 模块化），写入初始 design spec
- [ ] 初始 spec 包含：错误处理约定、状态管理策略、运维端点规划
- [ ] 配置基础构建流（Makefile / scripts），确保 `make build` 和 `make test` 可用

### 📐 每个功能 — 三文档流

- [ ] **Design Spec**: Goal + Key Decisions Table + Architecture Changes + Scope/Out-of-Scope
- [ ] Spec 中包含 1-2 个 User Story
- [ ] 核心技术选型做 30 分钟 Spike 验证
- [ ] **Implementation Plan**: 任务列表 + 文件变更 + 代码片段 + 验证命令 + Commit message
- [ ] Plan 中为核心逻辑加 Test Task（TDD）
- [ ] **执行**: 按 Plan 逐步执行，每步验证

### ✅ 每个功能完成后

- [ ] 在 CLAUDE.md 追加决策备忘录（一行：做了什么选择、为什么）
- [ ] 确认 commit 信息遵循 conventional format
- [ ] 关键路径确认有结构化日志

### 🏁 项目收尾

- [ ] 全量测试通过（`make test`）
- [ ] 部署脚本验证（`make smoke-test`）
- [ ] README 更新（截图、使用说明）
- [ ] 运维端点可用（health、version）
- [ ] **做这份复盘** 📋

---

## 核心洞察

> **AI Coding 的最大杠杆点不在编码阶段，而在设计阶段。**

投入在 Spec 和 Plan 上的时间，在实施阶段获得多倍回报：
- AI 执行精确的计划几乎不出错
- AI 执行模糊的指令可能返工多次

LinkStash 项目的成功模式可以概括为一句话：

**"人类负责决策，AI 负责执行，文档是两者的协议。"**
