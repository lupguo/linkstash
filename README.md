# LinkStash — 个人 URL 资源管理器

LinkStash 是一款面向个人的 URL 资源管理工具，支持 URL 收集、LLM 智能分析、关键词/语义混合检索、短链生成，通过 Web 界面、CLI 工具和 PopClip 插件三种方式交互。

## ✨ 核心功能

| 功能 | 说明 |
|------|------|
| **URL 管理** | 添加、编辑、删除、分页浏览，支持分类 / 标签 / 热度排序 |
| **LLM 智能分析** | 添加 URL 后异步抓取页面，LLM 自动提取标题、关键词、摘要、分类、标签 |
| **混合检索** | FTS5 关键词检索 + 512 维向量语义检索 + 加权混合检索 |
| **短链服务** | SHA256+Base62 短码生成，302 重定向，支持 TTL 过期（410 Gone） |
| **Terminal 风格 Web UI** | 暗黑极客主题，htmx 无刷新交互，移动端适配 |
| **CLI 工具** | `linkstash add / list / search / short / info` 全命令行操作 |
| **PopClip 插件** | macOS 上选中 URL 一键保存 |

## 🏗️ 技术栈

```
Go 1.21+ · GORM · SQLite (modernc 纯 Go) · chi · htmx · Alpine.js · Tailwind CSS · JWT · cobra
```

## 📁 项目结构

```
linkstash/
├── cmd/
│   ├── server/main.go            # 服务端入口
│   └── cli/                      # CLI 工具 (cobra)
│       ├── main.go
│       └── cmd/                  # add, list, search, short, info 子命令
├── app/
│   ├── handler/                  # HTTP Handler（API + Web 页面）
│   ├── middleware/               # JWT 鉴权中间件
│   ├── application/              # 应用层：用例编排
│   ├── domain/
│   │   ├── entity/               # 领域实体（5 张表）
│   │   ├── services/             # 领域服务
│   │   └── repos/                # 仓储接口
│   └── infra/                    # 基础设施层
│       ├── db/                   # GORM 仓储实现 + DB 初始化
│       ├── llm/                  # OpenAI 兼容 LLM 客户端
│       ├── config/               # YAML 配置加载
│       └── search/               # FTS5 + 向量检索
├── web/
│   ├── templates/                # Go HTML 模板
│   ├── static/                   # 静态资源
│   └── components/               # htmx partial 模板
├── popclip/                      # PopClip 插件
├── configs/app_dev.yaml          # 示例配置
├── scripts/smoke_test.sh         # 冒烟测试脚本
├── go.mod
└── go.sum
```

**调用链**：`handler → application → domain service → repo (interface) ← infra (实现)`

## 🚀 快速开始

### 1. 构建

```bash
# 编译服务端和 CLI
go build -o linkstash-server ./cmd/server/
go build -o linkstash ./cmd/cli/
```

### 2. 配置

复制并编辑配置文件：

```bash
cp configs/app_dev.yaml configs/app.yaml
```

关键配置项：

```yaml
auth:
  secret_key: "your-secret-key"       # 用于换取 JWT 的静态密钥
  jwt_secret: "your-jwt-secret"       # JWT 签名密钥（务必修改）

database:
  path: "./data/linkstash.db"         # SQLite 数据库路径

llm:
  chat:
    endpoint: "https://api.openai.com/v1/chat/completions"
    api_key: "${OPENAI_API_KEY}"      # 支持环境变量引用
    model: "gpt-4o-mini"
  embedding:
    endpoint: "https://api.openai.com/v1/embeddings"
    api_key: "${OPENAI_API_KEY}"
    model: "text-embedding-3-small"
    dimensions: 512
```

### 3. 启动服务

```bash
./linkstash-server -conf configs/app.yaml
```

服务默认监听 `0.0.0.0:8080`。

### 4. 获取 JWT Token

```bash
curl -X POST http://localhost:8080/api/auth/token \
  -H "Content-Type: application/json" \
  -d '{"secret_key":"your-secret-key"}'
```

### 5. 使用 CLI

```bash
export LINKSTASH_SERVER=http://localhost:8080
export LINKSTASH_TOKEN=<your-jwt-token>

linkstash add https://github.com
linkstash list
linkstash search "GitHub" --type keyword
linkstash short https://example.com/long-path --ttl 7d
linkstash info 1
```

## 📡 REST API

### 鉴权

```
POST /api/auth/token               # secret_key 换 JWT
```

### URL 管理

```
POST   /api/urls                   # 添加 URL（触发异步 LLM 分析）
GET    /api/urls                   # 列表（?page=1&size=20&sort=time&category=&tags=）
GET    /api/urls/:id               # 详情
PUT    /api/urls/:id               # 更新（支持 partial update）
DELETE /api/urls/:id               # 软删除
POST   /api/urls/:id/visit         # 记录访问
```

### 检索

```
GET    /api/search?q=<query>&type=keyword|semantic|hybrid&page=1&size=20
```

### 短链

```
POST   /api/short-links            # 创建短链（{"long_url":"...", "ttl":"7d"}）
GET    /api/short-links            # 短链列表
DELETE /api/short-links/:id        # 删除
GET    /s/:code                    # 302 重定向（无需鉴权）
```

### Web 页面

```
GET    /                           # URL 列表
GET    /login                      # 登录页
GET    /urls/:id                   # 详情页
GET    /search                     # 搜索页
GET    /short                      # 短链管理
```

## 🧪 测试

```bash
# 启动服务后运行冒烟测试（34 项）
./scripts/smoke_test.sh
```

测试覆盖：JWT 鉴权、URL CRUD、FTS5 搜索、短链创建/重定向/过期、Web 页面、CLI 全命令。

## 📐 数据模型

| 表名 | 说明 |
|------|------|
| `t_urls` | URL 资源（link, title, keywords, description, category, tags, status, weight, visits） |
| `t_embeddings` | 512 维向量（BLOB 存储，启动时加载到内存） |
| `t_short_links` | 短链（code, long_url, expires_at, click_count） |
| `t_visit_records` | 访问记录（url_id / short_id, ip, user_agent） |
| `t_llm_logs` | LLM 请求日志（request_type, tokens, latency, success） |
| `t_urls_fts` | FTS5 虚拟表（title, keywords, description 全文索引） |

## 🔑 鉴权说明

单用户系统，无注册登录流程：

1. 配置文件设置 `auth.secret_key`
2. `POST /api/auth/token` 用 secret_key 换取 JWT（有效期可配置，默认 72h）
3. API 调用：`Authorization: Bearer <jwt>`
4. Web 页面：JWT 存入 HttpOnly Cookie `linkstash_token`
5. 中间件统一校验：优先 Bearer → 降级 Cookie → 401

## 📎 PopClip 插件

安装 `popclip/LinkStash.popclipext`，在 PopClip 设置中配置环境变量：

```
LINKSTASH_SERVER=http://localhost:8080
LINKSTASH_TOKEN=your-jwt-token
```

选中 URL 文本后点击 LinkStash 图标即可一键保存。

## License

MIT
