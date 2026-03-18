# LinkStash - URL Resource Manager Design Spec

## Context

个人在浏览网站、X、Blog过程中发现的优质URL难以有效管理和回溯。LinkStash是一款个人URL资源管理器，支持URL收集、LLM智能分析、关键词/语义检索、短链生成，通过Web界面、PopClip插件和CLI三种方式交互。

## 技术选型

| 组件 | 选择 | 理由 |
|------|------|------|
| 语言 | Go | 用户要求 |
| ORM | GORM (`gorm.io/gorm`) | 用户选择，AutoMigrate方便 |
| SQLite | `gorm.io/driver/sqlite`（modernc纯Go底层） | 无CGO依赖，交叉编译无障碍 |
| HTTP框架 | `go-chi/chi/v5` | 轻量、兼容stdlib、中间件生态好 |
| 前端 | Go `html/template` + htmx + Alpine.js | 服务端渲染，单二进制部署 |
| CSS | Tailwind CSS (CDN) | 快速实现Terminal/Hacker极客风格 |
| JWT | `golang-jwt/jwt/v5` | 标准JWT库 |
| CLI | `spf13/cobra` | Go CLI标准库 |
| 配置 | `gopkg.in/yaml.v3` | YAML解析 |
| LLM | 自封装HTTP客户端（OpenAI兼容协议） | 直接调用，无额外依赖 |
| 向量检索 | 应用层计算（BLOB存储+内存余弦相似度，512维） | 万级数据够用，零额外依赖 |
| 关键词检索 | SQLite FTS5 全文索引 | 原生支持，性能优秀 |

**注意**：FTS5虚拟表需要手动raw SQL创建，GORM AutoMigrate不支持。需在DB初始化时验证modernc sqlite驱动已包含FTS5支持。

## 项目结构（DDD分层）

```
linkstash/
├── cmd/
│   ├── server/main.go           # 服务端入口
│   └── cli/main.go              # CLI工具入口
├── app/
│   ├── handler/                  # 接口层：HTTP Handler
│   │   ├── url_handler.go
│   │   ├── search_handler.go
│   │   ├── shorturl_handler.go
│   │   ├── auth_handler.go
│   │   └── web_handler.go       # 服务端渲染页面
│   ├── middleware/               # 中间件
│   │   ├── auth.go              # JWT鉴权（Bearer Token + Cookie双模式）
│   │   └── logging.go
│   ├── application/              # 应用层：用例编排（薄层）
│   │   ├── url_usecase.go       # 添加URL、获取详情、列表
│   │   ├── search_usecase.go    # 搜索编排（关键词+向量合并）
│   │   ├── shorturl_usecase.go  # 短链创建与访问
│   │   └── analysis_usecase.go  # LLM异步分析编排
│   ├── domain/
│   │   ├── entity/              # 领域实体
│   │   │   ├── url.go
│   │   │   ├── short_link.go
│   │   │   ├── visit_record.go
│   │   │   ├── embedding.go
│   │   │   └── llm_log.go
│   │   ├── services/            # 领域服务（单一职责）
│   │   │   ├── url_service.go
│   │   │   ├── search_service.go
│   │   │   ├── shorturl_service.go
│   │   │   ├── visit_service.go
│   │   │   └── worker_service.go
│   │   └── repos/               # 仓储接口定义
│   │       ├── url_repo.go
│   │       ├── shorturl_repo.go
│   │       ├── visit_repo.go
│   │       ├── embedding_repo.go
│   │       └── llm_log_repo.go
│   └── infra/                   # 基础设施层（接口实现）
│       ├── db/
│       │   ├── gorm.go          # DB初始化、AutoMigrate、FTS5 DDL
│       │   ├── url_repo_impl.go
│       │   ├── shorturl_repo_impl.go
│       │   ├── visit_repo_impl.go
│       │   ├── embedding_repo_impl.go
│       │   └── llm_log_repo_impl.go
│       ├── llm/
│       │   ├── client.go        # OpenAI兼容协议统一客户端
│       │   └── embedding.go     # Embedding向量生成
│       ├── config/
│       │   └── config.go        # YAML配置加载
│       └── search/
│           ├── keyword.go       # FTS5关键词检索
│           └── vector.go        # 应用层向量相似度计算
├── web/
│   ├── templates/               # Go HTML模板
│   ├── static/                  # CSS/JS（htmx, Alpine.js）
│   └── components/              # htmx partial模板
├── popclip/
│   └── LinkStash.popclipext/    # PopClip插件
├── configs/
│   └── app_dev.yaml             # 示例配置
├── go.mod
└── go.sum
```

**调用链**：`handler → application (用例编排) → domain service → repo (interface) ← infra (实现)`

## 鉴权方案

**单用户系统**，不做用户注册/登录。鉴权流程：

1. YAML配置文件中设置 `auth.secret_key`（静态密钥）
2. `POST /api/auth/token` — 客户端发送 `{"secret_key": "xxx"}`，服务端验证后返回JWT
3. **API调用**：`Authorization: Bearer <jwt>` 头部
4. **Web页面**：登录后将JWT写入HttpOnly Cookie `linkstash_token`，中间件从Cookie中提取JWT验证
5. **统一中间件**：`auth.go` 优先检查Bearer Token，不存在则检查Cookie，都没有返回401

## 数据模型

所有表名使用 `t_` 前缀，所有实体内嵌 `gorm.Model`（含ID/CreatedAt/UpdatedAt/DeletedAt软删除）。

### t_urls - URL资源表

```go
type URL struct {
    gorm.Model
    Link         string     `gorm:"uniqueIndex;not null"`   // 原始URL
    Title        string                                      // LLM分析的标题
    Keywords     string                                      // 关键字（逗号分隔）
    Description  string                                      // 内容描述
    Category     string     `gorm:"index"`                   // 分类（自由形式，LLM建议但不强制）
    Tags         string                                      // 标签（逗号分隔）
    Status       string     `gorm:"default:pending"`         // pending|analyzing|ready|failed
    AutoWeight   float64    `gorm:"default:0"`               // 自动热度（每次访问+1）
    ManualWeight float64    `gorm:"default:0"`               // 手动权重（用户可调节）
    LastVisitAt  *time.Time                                  // 最后访问时间
    VisitCount   int        `gorm:"default:0"`               // 总访问次数
}
func (URL) TableName() string { return "t_urls" }
```

**热度排序算法**：`score = (auto_weight + manual_weight) * e^(-0.05 * days_since_last_visit)`

### t_embeddings - 向量表

```go
type Embedding struct {
    gorm.Model
    URLID  uint   `gorm:"uniqueIndex;not null"`
    Vector []byte // 512维float32向量序列化为[]byte（BLOB），约2KB/条
}
func (Embedding) TableName() string { return "t_embeddings" }
```

**内存占用**：万级URL × 512维 × 4字节 ≈ 20MB，启动时加载到内存缓存。新增embedding时同步更新缓存。

### t_short_links - 短链表

```go
type ShortLink struct {
    gorm.Model
    Code       string     `gorm:"uniqueIndex;size:16"`  // 短码（Base62，6字符）
    LongURL    string     `gorm:"not null"`             // 任意长链接（独立于t_urls）
    ExpiresAt  *time.Time                               // 过期时间，nil=永不过期
    ClickCount int        `gorm:"default:0"`
}
func (ShortLink) TableName() string { return "t_short_links" }
```

**短链独立于URL管理**：可以为任意URL生成短链，不要求该URL存在于t_urls中。

### t_visit_records - 访问记录表

```go
type VisitRecord struct {
    gorm.Model
    URLID     uint   `gorm:"index"`     // 关联URL（URL访问时填写，短链访问时为0）
    ShortID   uint   `gorm:"index"`     // 关联短链（短链访问时填写，URL访问时为0）
    IP        string
    UserAgent string
}
func (VisitRecord) TableName() string { return "t_visit_records" }
```

**两种访问场景独立使用**：
- URL访问：`POST /api/urls/:id/visit` → 创建VisitRecord(URLID=id, ShortID=0) + 更新URL的auto_weight/visit_count/last_visit_at
- 短链访问：`GET /s/:code` → 创建VisitRecord(URLID=0, ShortID=id) + 更新ShortLink的click_count

### t_llm_logs - LLM请求日志表

```go
type LLMLog struct {
    gorm.Model
    URLID         uint    `gorm:"index"`
    RequestType   string  `gorm:"index"`    // chat | embedding
    Model         string                     // 模型名称
    PromptKey     string                     // 使用的prompt配置key
    InputContent  string  `gorm:"type:text"` // 请求内容
    OutputContent string  `gorm:"type:text"` // 响应内容
    InputTokens   int
    OutputTokens  int
    TotalTokens   int
    LatencyMs     int64                      // 请求耗时（毫秒）
    TokensPerSec  float64                    // 输出速度（tokens/s）
    StatusCode    int                        // HTTP状态码
    ErrorMessage  string
    Success       bool    `gorm:"index"`
}
func (LLMLog) TableName() string { return "t_llm_logs" }
```

## YAML 配置

```yaml
server:
  host: 0.0.0.0
  port: 8080

auth:
  secret_key: "your-secret-key-here"    # 静态密钥，用于换取JWT

database:
  path: "./data/linkstash.db"

llm:
  chat:
    endpoint: "https://api.openai.com/v1/chat/completions"
    api_key: "${OPENAI_API_KEY}"         # 支持环境变量引用
    model: "gpt-4o-mini"                 # 速度优先
  embedding:
    endpoint: "https://api.openai.com/v1/embeddings"
    api_key: "${OPENAI_API_KEY}"
    model: "text-embedding-3-small"
    dimensions: 512                       # 降维到512
  prompts:                                # Prompt配置（map结构，可自定义覆盖）
    url_analysis: |
      分析以下网页内容，返回JSON格式：
      {"title":"标题","keywords":"关键词1,关键词2","description":"50字内摘要","category":"分类","tags":"标签1,标签2"}
      仅返回JSON，不要其他内容。
    url_categorize: |
      根据以下URL和标题，判断最合适的分类（建议从以下选项中选择，也可自定义）：
      技术、设计、产品、商业、科学、生活、工具、资讯、其他
```

## REST API

### 标准错误响应格式

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid URL format"
  }
}
```

错误码约定：`VALIDATION_ERROR`, `NOT_FOUND`, `UNAUTHORIZED`, `INTERNAL_ERROR`, `EXPIRED`

### 鉴权
```
POST   /api/auth/token               # secret_key换JWT
```

### URL资源管理
```
POST   /api/urls                     # 添加URL（触发异步LLM分析）
GET    /api/urls                     # 列表（分页+排序+过滤）
GET    /api/urls/:id                 # 详情
PUT    /api/urls/:id                 # 更新（含手动调节manual_weight）
DELETE /api/urls/:id                 # 软删除
POST   /api/urls/:id/visit           # 记录访问（更新热度）
```

### 检索
```
GET    /api/search?q=&type=keyword|semantic|hybrid
       &page=1&size=20&category=&tags=&sort=time|weight
```

### 短链
```
POST   /api/short-links              # 创建短链
GET    /api/short-links              # 短链列表
DELETE /api/short-links/:id          # 删除短链
GET    /s/:code                      # 302重定向（无需鉴权）
```

### Web页面（服务端渲染）
```
GET    /                             # 首页（URL列表）
GET    /urls/:id                     # URL详情页
GET    /search                       # 搜索页
GET    /short                        # 短链管理页
GET    /login                        # 登录页
```

**分页参数**：`page`(默认1), `size`(20/50/100), `sort`(time/weight,默认time倒序), `category`, `tags`

**鉴权**：API使用JWT Bearer Token，Web页面使用JWT存Cookie（HttpOnly），短链重定向无需鉴权。统一中间件优先检查Bearer，其次Cookie。

## LLM 集成

### 统一客户端

自封装OpenAI兼容协议HTTP客户端，支持chat/completions和embeddings两个端点。所有LLM提供商（OpenAI/OpenRouter/QWen）都兼容OpenAI协议，通过配置不同的endpoint+api_key+model即可切换，无需provider抽象。

### 异步分析流程

```
用户提交URL → handler → url_usecase.AddURL()
  ├─ url_service.Save(status=pending)  → 立即返回201
  └─ worker_service.Enqueue(urlID)     → 推入channel

Worker goroutine（1个worker，buffered channel容量100）：
  ← channel 接收 urlID
  ├─ 1. 抓取URL页面内容（net/http，超时10s）
  ├─ 2. LLM分析：发送内容 → ChatAPI → JSON{title,keywords,desc,category,tags}
  ├─ 3. Embedding生成：内容摘要 → EmbeddingAPI → 512维向量
  ├─ 4. 更新URL记录 + 保存Embedding（status=ready）
  ├─ 5. 记录LLMLog（chat和embedding各一条）
  └─ 失败处理：
     - status=failed，ErrorMessage记录原因
     - 自动重试：最多3次，指数退避（1s, 2s, 4s）
     - 服务重启恢复：启动时查询status=pending|analyzing的记录，重新入队
```

## 检索方案

- **关键词检索**：SQLite FTS5全文索引
  - 虚拟表：`CREATE VIRTUAL TABLE t_urls_fts USING fts5(title, keywords, description, content=t_urls, content_rowid=id)`
  - 通过触发器保持FTS与t_urls同步（INSERT/UPDATE/DELETE）
  - 查询：`SELECT ... FROM t_urls_fts WHERE t_urls_fts MATCH ? ORDER BY rank`
- **语义检索**：启动时加载所有embedding到内存map[uint][]float32，计算余弦相似度，返回TopN
- **混合检索**：
  - 关键词和语义分别检索各取Top50
  - 分数归一化到0-1范围（关键词用BM25 rank归一化，语义用cosine similarity原值）
  - 综合分数 = 0.5 * keyword_score + 0.5 * semantic_score
  - 合并去重后取TopN返回

## 短链服务

- **生成算法**：取`SHA256(longURL + timestamp + random_bytes)`的前6字节转为uint48，再 `mod 62^6`，用Base62编码为6字符短码
- **碰撞处理**：DB唯一索引，碰撞时重新生成（更换random_bytes），最多重试3次
- **有效期**：`expires_at`字段，访问时检查是否过期，过期返回410 Gone
- **重定向**：`GET /s/:code` → 查DB → 未过期302重定向 → 异步goroutine记录访问+更新click_count

## Web UI风格

**Terminal/Hacker极客风格**：
- 暗黑背景（#0a0e17）+ 绿色终端字体（#00ff41）
- 等宽字体（JetBrains Mono / Fira Code）
- 模拟终端界面元素（边框、光标、命令行提示符）
- 色彩搭配：绿色(#00ff41)主色、红色(#ff6b6b)警告、青色(#4ecdc4)链接、灰色(#888)次要
- Tailwind CSS实现，做好Wap端响应式适配（移动端简化边框装饰，保持暗色调）

## PopClip 插件

```
popclip/LinkStash.popclipext/
├── Config.plist (或 Config.json)
└── action.sh
```
Shell action通过curl调用服务端 `POST /api/urls` API。用户在PopClip设置中配置 `LINKSTASH_SERVER`（服务地址）和 `LINKSTASH_TOKEN`（JWT Token）。

## CLI 工具

```bash
# 环境变量
export LINKSTASH_SERVER=http://localhost:8080
export LINKSTASH_TOKEN=your-jwt-token

# 命令
linkstash add <url>                    # 添加URL
linkstash list [--page 1 --size 20]    # 列表
linkstash search <query> [--type hybrid] # 搜索
linkstash short <url> [--ttl 7d]       # 生成短链
linkstash info <id>                    # URL详情
```
使用cobra构建，通过HTTP调用服务端REST API。

## 实现阶段

- **Phase 1**：项目骨架 + 配置加载 + DB初始化(含FTS5) + 数据模型 + JWT鉴权 + URL CRUD API
- **Phase 2**：LLM客户端 + 异步分析Worker + LLM日志 + 重启恢复
- **Phase 3**：FTS5关键词检索 + 向量语义检索(512维) + 混合检索API
- **Phase 4**：Web界面（Terminal/Hacker风格，htmx交互，含URL详情页）
- **Phase 5**：短链服务
- **Phase 6**：CLI工具
- **Phase 7**：PopClip插件
