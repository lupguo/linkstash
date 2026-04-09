# LinkStash 项目 AI Coding 复盘报告

> 复盘时间：2026-04-03 | 项目周期：~3周（2026.03.24 - 2026.04.03）

## 项目概览

| 维度 | 数据 |
|------|------|
| 开发周期 | ~3周集中冲刺（2026.03.24 - 2026.04.03） |
| 总提交数 | 135 commits |
| AI协作率 | 92.6%（125/135 commits由Claude协助） |
| 技术栈 | Go + Preact SPA + Tailwind v4 + SQLite/MySQL |
| 架构模式 | DDD + Clean Architecture + Wire DI |

---

## ✅ 一、做得好的（值得沉淀为标准实践）

### 1. 架构设计先行 — "Spec → Plan → Code"工作流

**具体表现：** 项目中有`docs/superpowers/specs/`和`docs/superpowers/plans/`目录，每个重大特性都是先写设计文档，再写实施计划，最后编码。

**Git证据：**
```
docs: add network type feature design spec     (12:52)
docs: add network type feature implementation plan  (12:55)
feat: add network type classification...        (13:xx - 14:xx, 12 commits)
Merge feature/homepage-ux-improvements          (14:24)
```

**经验沉淀：**
> 🏆 **黄金法则：** 对任何超过2小时的特性，先用AI帮忙写Spec和Plan，再动手编码。这避免了"边想边写"导致的返工，也让AI有明确上下文执行实现。

---

### 2. DDD + Clean Architecture 从第一天就坚持

**具体表现：** `handler → application → domain/services → domain/repos → infra` 分层清晰，依赖方向始终向内。Google Wire 编译期DI确保依赖关系透明。

**经验沉淀：**
> 🏆 **起步就搭好骨架。** AI非常擅长在已有清晰架构的项目中工作——它能理解分层约束，不会把数据库代码写到handler里。第一天投入的架构时间，后面每次对话都在回收红利。

---

### 3. 高质量的 CLAUDE.md 项目说明

**具体表现：** CLAUDE.md包含构建命令、架构图、API路由、JSON字段命名约定、前端模式等关键信息，396行非常全面。

**经验沉淀：**
> 🏆 **CLAUDE.md 是AI的"新人入职手册"。** 每次新对话，AI都从这里获取项目上下文。好的CLAUDE.md直接决定了AI输出质量的下限。建议项目初期就维护，每个里程碑更新一次。

---

### 4. Feature Branch + 语义化提交

**具体表现：** 使用`feature/ui-refactor-tailwind-v4`、`feature/fetcher-strategy-chain`等特性分支，提交消息遵循`feat/fix/docs/refactor/chore`语义化格式。

**经验沉淀：**
> 🏆 **让AI帮忙写commit message时，给它看最近几条提交作为风格参考。** 135条提交保持了一致的语义化格式，这说明在对话中建立了良好的约定传递。

---

### 5. Makefile 全自动化构建

**具体表现：** 15+个构建目标覆盖了开发（`dev-frontend`）、测试（`smoke-test`）、部署（`release-full`）全流程。一条`make build`搞定前后端。

**经验沉淀：**
> 🏆 **Makefile是AI的"手"。** AI不能记住复杂的构建步骤，但它能`make xxx`。每增加一个构建步骤，都封装成Makefile target，并在CLAUDE.md中记录。

---

### 6. 前端架构选型精准

**具体表现：** Preact（3KB）+ Signals（极简状态管理）+ esbuild（极速构建）+ Tailwind v4（设计token系统），个人项目的最佳平衡。

**经验沉淀：**
> 🏆 **个人项目/小团队：Preact > React。** 体积小、构建快、API兼容。配合signals做状态管理，避免了Redux的复杂度。AI对这套栈的理解和输出质量很高。

---

## ⚠️ 二、做得中规中矩，可以优化的

### 1. 前端框架迁移了两次（HTMX → Alpine.js → Preact）

**证据：** Git历史显示经历了 HTMX服务端渲染 → Alpine.js → Preact SPA 的演进。

**问题：** 每次迁移都是重写，前期的UI工作基本作废。

**改进建议：**
> ⚡ **在第一次对话中就让AI帮做技术选型对比。** 比如："我要做一个URL管理工具，需要搜索、列表、详情页，单人使用，请对比HTMX/Alpine/Preact/React的适合度"。用5分钟对话省掉两次重写。

---

### 2. Usecase层偏薄，有些过度设计

**具体表现：** `URLUsecase`大部分方法只是透传到Service层，没有真正的编排逻辑。对个人项目来说，handler → service → repo 三层可能就够了。

**改进建议：**
> ⚡ **DDD的层数要匹配项目复杂度。** 个人项目可以handler直接调service。当业务编排变复杂时（如一个操作要协调多个service），再抽出usecase层。AI倾向生成"标准架构"，需要人为判断是否需要这么多层。

---

### 3. 搜索功能复杂度较高，但缺少用户反馈闭环

**具体表现：** 实现了关键词(FTS5) + 语义(embedding) + 混合搜索三种模式，架构精良，但对于个人URL管理来说可能过早优化。

**改进建议：**
> ⚡ **先实现最简方案（FTS5关键词搜索），收集真实使用数据后再决定是否需要语义搜索。** 和AI协作时容易"技术兴奋"，把时间花在了锦上添花而非核心体验上。

---

### 4. API没有版本化和DTO层

**具体表现：** Handler直接使用`entity.URL`作为请求/响应对象，API契约与数据库模型耦合。

**改进建议：**
> ⚡ **至少对外API加一层Response DTO。** 这样数据库字段变更不会直接破坏客户端。让AI生成DTO + 转换函数成本很低。

---

### 5. 单次对话做太多事

**证据：** 有些天一天35个commit，说明单次对话中完成了大量特性。

**改进建议：**
> ⚡ **一次对话聚焦一个特性。** AI的上下文窗口有限，做太多事会导致后面的改动质量下降。建议：一个Feature Branch = 一次Claude对话 = 一个明确的目标。

---

## ❌ 三、做得不好的（必须改进）

### 1. 零测试覆盖 — 最大的债务

**具体表现：** 整个项目**没有一个** `*_test.go` 文件，前端也没有测试框架。`package.json`的test脚本是 `echo "Error: no test specified" && exit 1`。

**风险：**
- 无法安全重构（而项目已经重构了多次）
- 回归Bug无法自动检测
- `make test` 存在但等于空跑

**改进建议：**
> 🔴 **TDD或至少"Code-then-Test"。** 每写完一个handler/service，立即让AI生成对应测试。最低要求：
> - 所有handler的HTTP测试（表驱动测试）
> - 核心service的单元测试
> - smoke-test脚本保持可用
>
> **具体做法：** 在CLAUDE.md中加一条规则："每个新增的handler函数和service方法，必须同时提交对应的_test.go文件"。

---

### 2. 不安全的类型断言 — 服务器可被崩溃

**具体代码：**
```go
// app/handler/url_handler.go
existing.Title = v.(string)           // 客户端发错类型 → PANIC
existing.ManualWeight = v.(float64)   // 客户端发错类型 → PANIC
existing.VisitCount = int(v.(float64)) // 客户端发错类型 → PANIC
```

**风险：** 任何恶意或格式错误的JSON请求都能直接崩溃服务器。

**改进建议：**
> 🔴 **用结构化的请求DTO替代 `map[string]interface{}`。** 或至少用 `v, ok := v.(string)` 安全断言。这是让AI编码时容易忽略的边界情况——在Prompt中明确要求"所有类型断言必须使用comma-ok模式"。

---

### 3. 静默错误吞噬

**具体代码：**
```go
_ = h.usecase.UpdateURL(url)      // 更新失败？不知道
_ = w.llmLogRepo.Create(chatLog)  // 日志写失败？不知道
```

**风险：** 线上问题排查时完全没有线索。

**改进建议：**
> 🔴 **在CLAUDE.md中加规则："禁止使用 `_ = someFunc()` 忽略错误，至少用slog.Warn记录"。** AI生成代码时经常用`_`忽略"不重要"的错误，但这是Go的反模式。

---

### 4. 并发竞态条件

**具体表现：** URL创建时，先同步设置icon，然后起goroutine异步拉favicon覆盖同一字段：
```go
url.Icon = defaultIcons[rand.Intn(len(defaultIcons))]
_ = h.usecase.UpdateURL(url)  // 设置默认icon
go func() {
    favicon := fetchFavicon(link)
    u.Favicon = favicon
    h.usecase.UpdateURL(u)     // 异步覆盖，可能和其他更新冲突
}()
```

**改进建议：**
> 🔴 **异步操作应通过Worker队列处理，而非裸goroutine。** 项目已有WorkerService，favicon获取应该也走队列。提示AI时说"请使用项目已有的异步worker模式，不要直接go func"。

---

### 5. 输入验证缺失

**具体表现：** URL格式、标题长度、描述长度、分类白名单——都没有服务端校验。

**改进建议：**
> 🔴 **加一个validation中间件或在handler层统一校验。** 和AI对话时，在Spec中明确写出每个字段的校验规则。

---

### 6. 数据库错误检测用字符串匹配

**具体代码：**
```go
if strings.Contains(err.Error(), "UNIQUE constraint") ||
   strings.Contains(err.Error(), "Duplicate entry") {
```

**风险：** SQLite和MySQL的错误消息不同，这种匹配脆弱且不可测试。

**改进建议：**
> 🔴 **定义领域错误类型（如`ErrDuplicateURL`），在Repository层翻译数据库错误。** 这符合Clean Architecture的原则——handler层不应该知道底层是SQLite还是MySQL。

---

## 🧠 四、AI Coding 方法论经验（跨项目通用）

### 经验1：CLAUDE.md 是投入产出比最高的文档
投入30分钟写好CLAUDE.md，后续每次对话都省5-10分钟上下文交代时间。按135次对话算，节省了**11-22小时**。

### 经验2："Spec → Plan → Code"三步法适合中大特性
小修改直接做；超过1小时的特性走三步法。这个项目做到了，继续保持。

### 经验3：AI容易"过度架构"，人需要把关
AI倾向生成"教科书式"的完整架构（比如薄的Usecase层）。人的判断是：当前复杂度需要几层？不要让架构超前于需求太多。

### 经验4：AI的"快乐路径偏见"
AI写代码时倾向优先处理正常流程，对错误处理、边界情况、并发安全的关注不够。**需要在Prompt中显式要求：** "请同时处理所有错误路径和边界情况"。

### 经验5：技术选型应在第一次对话完成
这个项目的前端框架迁移了两次，代价是几天的重写。应在项目初始化对话中就完成技术选型决策。

### 经验6：测试是AI协作的"安全网"
没有测试的项目，每次让AI修改都是在走钢丝。有了测试，AI改完代码可以立即验证，大幅提升迭代速度和信心。

---

## 📋 五、推荐写入CLAUDE.md的新规则

```markdown
## Coding Standards (AI协作守则)

1. **错误处理**: 禁止 `_ = someFunc()`，所有错误必须处理或用slog记录
2. **类型安全**: 所有类型断言必须使用comma-ok模式 `v, ok := x.(Type)`
3. **测试**: 每个新handler/service方法必须有对应_test.go
4. **异步**: 不要裸用go func{}，使用项目WorkerService队列
5. **输入验证**: 所有API入参必须在handler层校验
6. **数据库错误**: 在repo层翻译为领域错误类型，上层不做字符串匹配
7. **单次对话**: 一次对话聚焦一个Feature，避免上下文超载
```

---

## 🎯 六、总结评分

| 维度 | 评分 | 说明 |
|------|------|------|
| 架构设计 | ⭐⭐⭐⭐⭐ | DDD+Clean Arch执行到位 |
| AI协作流程 | ⭐⭐⭐⭐ | Spec→Plan→Code做得好，但缺测试闭环 |
| 代码质量 | ⭐⭐⭐ | 架构好但细节（错误处理/类型安全）有债 |
| 测试覆盖 | ⭐ | 零测试，最大短板 |
| 构建工具 | ⭐⭐⭐⭐⭐ | Makefile+esbuild+Tailwind CLI 全自动 |
| 技术选型 | ⭐⭐⭐ | 最终选型好，但经历了两次迁移 |
| 文档 | ⭐⭐⭐⭐ | CLAUDE.md优秀，但缺API文档 |

**一句话总结：** 这个项目证明了AI Coding在架构设计和快速实现方面的巨大价值（3周完成一个全功能产品），但也暴露了AI协作的典型陷阱——**快乐路径偏见、缺少测试安全网、容易过度架构**。下个项目第一天就应该做的三件事：**① 写CLAUDE.md ② 建测试框架 ③ 完成技术选型**。
