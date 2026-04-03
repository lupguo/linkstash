# LinkStash 部署指南

## 目录

- [环境要求](#环境要求)
- [本地开发](#本地开发)
- [构建与发布](#构建与发布)
- [生产部署（Linux 服务器）](#生产部署linux-服务器)
- [Docker 部署](#docker-部署)
- [Systemd 服务](#systemd-服务)
- [Caddy 反向代理 + HTTPS](#caddy-反向代理--https)
- [配置详解](#配置详解)
- [数据备份与恢复](#数据备份与恢复)
- [监控与运维](#监控与运维)
- [故障排查](#故障排查)

---

## 环境要求

| 项目 | 要求 |
|------|------|
| Go | 1.21+（仅编译时需要） |
| 操作系统 | Linux / macOS / Windows |
| 磁盘 | ≥100MB（二进制 ~25MB + 数据库按量增长） |
| 内存 | ≥64MB（万级 URL 向量缓存约 20MB） |
| 网络 | 需访问 LLM API（OpenRouter / OpenAI 等） |

> 纯 Go 编译（modernc SQLite），无 CGO 依赖，支持交叉编译。自 v0.4.0 起前端资源已嵌入二进制，下载即用。

---

## 本地开发

```bash
# 1. 克隆并安装依赖
git clone https://github.com/lupguo/linkstash.git
cd linkstash
npm install

# 2. 配置
cp conf/app_example.yaml conf/app_dev.yaml
cp .env.example .env
vim .env                    # 填入 OPENROUTER_API_KEY 等

# 3. 构建并启动
make build                  # 前端 (CSS+JS) + server + CLI
make start                  # 后台启动（端口 8888）

# 4. 验证
curl http://localhost:8888/health

# 5. 其他命令
make stop                   # 停止
make restart                # 重启
make dev-frontend           # 前端 watch 模式（开发用）
make test                   # 运行测试
```

---

## 构建与发布

### 本地发布

```bash
make release-full           # 前端 + 交叉编译全平台二进制
```

产出 `bin/release/` 目录，包含 8 个二进制：

| 二进制 | 平台 |
|--------|------|
| `linkstash-server-linux-amd64` / `arm64` | Linux |
| `linkstash-server-darwin-amd64` / `arm64` | macOS |
| `linkstash-linux-amd64` / `arm64` | Linux CLI |
| `linkstash-darwin-amd64` / `arm64` | macOS CLI |

### CI 发布（GitHub Actions）

推送 semver tag 自动触发：

```bash
git tag v0.7.0 -m "Release description"
git push origin v0.7.0
```

GitHub Actions 自动：构建前端 → 交叉编译 → 创建 Release（含 SHA256 校验）。

### 下载预编译二进制

前往 [Releases](https://github.com/lupguo/linkstash/releases) 下载。

---

## 生产部署（Linux 服务器）

### 一键安装（推荐）

```bash
curl -fsSL https://raw.githubusercontent.com/lupguo/linkstash/main/INSTALL.sh | sudo bash

# 填入 LLM API Key
sudo vim /opt/linkstash/.env

# 启动
sudo systemctl start linkstash
curl -s http://127.0.0.1:8085/health
```

### 手动安装

#### 1. 创建用户和目录

```bash
sudo useradd -r -s /sbin/nologin linkstash
sudo mkdir -p /opt/linkstash/{bin,conf,data,logs}
sudo chown -R linkstash:linkstash /opt/linkstash
```

#### 2. 部署二进制

```bash
VERSION="v0.7.0"
REPO="lupguo/linkstash"

curl -fsSL "https://github.com/${REPO}/releases/download/${VERSION}/linkstash-server-linux-amd64" \
  -o /opt/linkstash/bin/linkstash-server
curl -fsSL "https://github.com/${REPO}/releases/download/${VERSION}/linkstash-linux-amd64" \
  -o /opt/linkstash/bin/linkstash
chmod +x /opt/linkstash/bin/*
```

#### 3. 安装 Chromium（可选，URL 分析用）

```bash
sudo dnf install -y chromium     # RHEL 系
# 或
sudo apt install -y chromium-browser   # Debian/Ubuntu
```

#### 4. 配置

```bash
sudo -u linkstash vim /opt/linkstash/conf/app_prod.yaml
```

```yaml
server:
  host: 127.0.0.1
  port: 8085

auth:
  secret_key: "替换为强随机字符串"         # openssl rand -hex 32
  jwt_secret: "替换为另一个强随机字符串"
  jwt_expire_hours: 72

database:
  path: "./data/linkstash.db"

log:
  level: "info"
  file: "./logs/app.log"
  format: "text"

llm:
  chat:
    provider: "openrouter"
    endpoint: "https://openrouter.ai/api/v1/chat/completions"
    api_key: "${OPENROUTER_API_KEY}"
    model: "minimax/minimax-m2.5"
  embedding:
    provider: "openrouter"
    endpoint: "https://openrouter.ai/api/v1/embeddings"
    api_key: "${OPENROUTER_API_KEY}"
    model: "qwen/qwen3-embedding-8b"
    dimensions: 512

browser:
  enabled: true
  bin_path: "/usr/bin/chromium-browser"
  headless: true
  timeout_sec: 30
```

#### 5. 环境变量

```bash
sudo vim /opt/linkstash/.env
```

```bash
OPENROUTER_API_KEY=sk-or-your-api-key-here
```

```bash
sudo chown linkstash:linkstash /opt/linkstash/.env
sudo chmod 600 /opt/linkstash/.env
```

#### 6. 目录结构

```
/opt/linkstash/
├── bin/
│   ├── linkstash-server
│   └── linkstash
├── conf/
│   └── app_prod.yaml
├── data/
│   ├── linkstash.db          # 自动生成
│   └── linkstash.bleve/      # 全文索引（自动生成）
└── logs/
    └── app.log
```

---

## Docker 部署

### Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o linkstash-server ./cmd/server/ \
 && CGO_ENABLED=0 go build -ldflags="-s -w" -o linkstash ./cmd/cli/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/linkstash-server /build/linkstash /app/
COPY --from=builder /build/conf/app_dev.yaml /app/conf/app.yaml
RUN mkdir -p /app/data
EXPOSE 8080
VOLUME ["/app/data", "/app/conf"]
ENTRYPOINT ["/app/linkstash-server"]
CMD ["-conf", "/app/conf/app.yaml"]
```

### docker-compose.yml

```yaml
version: "3.8"
services:
  linkstash:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - linkstash-data:/app/data
      - ./conf/app_prod.yaml:/app/conf/app.yaml:ro
    environment:
      - OPENROUTER_API_KEY=${OPENROUTER_API_KEY}
    restart: unless-stopped

volumes:
  linkstash-data:
```

```bash
OPENROUTER_API_KEY=sk-xxx docker compose up -d
```

---

## Systemd 服务

创建 `/etc/systemd/system/linkstash.service`：

```ini
[Unit]
Description=LinkStash - Bookmark Management Service
After=network.target

[Service]
Type=simple
User=linkstash
Group=linkstash
WorkingDirectory=/opt/linkstash
ExecStart=/opt/linkstash/bin/linkstash-server -conf /opt/linkstash/conf/app_prod.yaml
EnvironmentFile=/opt/linkstash/.env
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# 安全加固
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/linkstash/data /opt/linkstash/logs

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now linkstash
sudo systemctl status linkstash
```

---

## Caddy 反向代理 + HTTPS

### 安装 Caddy

```bash
# RHEL 系
sudo dnf install -y 'dnf-command(copr)'
sudo dnf copr enable -y @caddy/caddy
sudo dnf install -y caddy

# 或直接下载
sudo curl -fsSL "https://caddyserver.com/api/download?os=linux&arch=amd64" -o /usr/bin/caddy
sudo chmod +x /usr/bin/caddy
```

### 配置 Caddyfile

```caddyfile
your-domain.example.com {
    reverse_proxy 127.0.0.1:8085
    encode gzip zstd
    log {
        output file /var/log/caddy/linkstash.log
        format json
    }
}
```

> Caddy 自动申请 Let's Encrypt 证书，自动 HTTP → HTTPS 重定向。

### 启动

```bash
sudo mkdir -p /var/log/caddy && sudo chown caddy:caddy /var/log/caddy
sudo systemctl enable --now caddy
```

### 防火墙

```bash
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --reload
```

> 云服务器还需在控制台安全组中开放 80/443 端口。

---

## 配置详解

### 完整配置项

```yaml
server:
  host: 0.0.0.0
  port: 8080

auth:
  secret_key: ""                      # [必填] 登录密钥
  jwt_secret: ""                      # [必填] JWT 签名密钥
  jwt_expire_hours: 72

database:
  driver: sqlite                      # sqlite 或 mysql
  sqlite:
    path: "./data/linkstash.db"
  mysql:
    user: root
    password: "${MYSQL_PASSWORD}"
    host: 127.0.0.1
    port: 3306
    dbname: linkstash_db

llm:
  chat:
    endpoint: ""                      # [必填] Chat API
    api_key: "${OPENROUTER_API_KEY}"
    model: "minimax/minimax-m2.5"
  embedding:
    endpoint: ""                      # [必填] Embedding API
    api_key: "${OPENROUTER_API_KEY}"
    model: "qwen/qwen3-embedding-8b"
    dimensions: 512

fetcher:
  strategies: ["http"]                # ["http", "browser"] 需要 Chromium
  http:
    timeout_sec: 15
    max_content: 51200
  browser:
    timeout_sec: 30
    lifecycle: "on-demand"

browser:
  enabled: false
  headless: true
  timeout_sec: 30
```

### 兼容的 LLM 提供商

| 提供商 | Endpoint | 模型示例 |
|--------|----------|----------|
| OpenRouter | `https://openrouter.ai/api/v1/...` | minimax-m2.5 / qwen3-embedding-8b |
| OpenAI | `https://api.openai.com/v1/...` | gpt-4o-mini / text-embedding-3-small |
| 通义千问 | `https://dashscope.aliyuncs.com/compatible-mode/v1/...` | qwen-turbo |
| 本地 Ollama | `http://localhost:11434/v1/...` | llama3 / nomic-embed-text |

---

## 数据备份与恢复

### 备份

```bash
# 热备份（推荐，不停机）
sqlite3 /opt/linkstash/data/linkstash.db ".backup /backup/linkstash-$(date +%Y%m%d).db"

# 定时备份（每天凌晨 3 点，保留 30 天）
0 3 * * * sqlite3 /opt/linkstash/data/linkstash.db ".backup /backup/linkstash-$(date +\%Y\%m\%d).db" && find /backup -name "linkstash-*.db" -mtime +30 -delete
```

### 恢复

```bash
sudo systemctl stop linkstash
cp /backup/linkstash-20260401.db /opt/linkstash/data/linkstash.db
sudo chown linkstash:linkstash /opt/linkstash/data/linkstash.db
sudo systemctl start linkstash
```

---

## 监控与运维

### 健康检查

```bash
curl -f http://localhost:8085/health || echo "LinkStash is DOWN"
```

### 日志

```bash
journalctl -u linkstash -f             # systemd 实时日志
tail -f /opt/linkstash/logs/app.log    # 应用日志
```

### 更新版本

```bash
sudo -u linkstash cp /opt/linkstash/data/linkstash.db /opt/linkstash/data/linkstash.db.bak
VERSION="v0.7.0"
sudo curl -fsSL "https://github.com/lupguo/linkstash/releases/download/${VERSION}/linkstash-server-linux-amd64" \
  -o /opt/linkstash/bin/linkstash-server
sudo chmod +x /opt/linkstash/bin/linkstash-server
sudo systemctl restart linkstash
```

### 性能参考

| 规模 | 内存占用 | 检索延迟 |
|------|---------|---------|
| 1,000 URLs | ~10MB | <10ms |
| 10,000 URLs | ~30MB | <50ms |
| 50,000 URLs | ~120MB | <200ms |

---

## 故障排查

| 问题 | 排查方法 |
|------|----------|
| 启动报 `disk I/O error` | `mkdir -p /opt/linkstash/data && chown linkstash:linkstash /opt/linkstash/data` |
| LLM 分析失败 | 检查 API Key 和网络连通性，`journalctl -u linkstash -n 50` |
| FTS5 搜索无结果 | `sqlite3 linkstash.db "INSERT INTO t_urls_fts(t_urls_fts) VALUES('rebuild');"` |
| 短链 404 | 确认 `/s/:code` 路由（无需鉴权），检查 `t_short_links` 表 |
| Caddy 证书失败 | 确认域名解析正确 + 80/443 端口开放 |
| 502 Bad Gateway | 确认 LinkStash 服务正在运行 |
| 日志写入失败 | 检查 systemd `ReadWritePaths` 包含 logs 目录 |
| 数据库锁定 | 确认只有一个进程访问 SQLite 文件 |
