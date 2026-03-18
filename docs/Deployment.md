# LinkStash 部署指南

## 目录

- [环境要求](#环境要求)
- [本地开发部署](#本地开发部署)
- [生产部署](#生产部署)
- [Docker 部署](#docker-部署)
- [Systemd 服务](#systemd-服务)
- [反向代理配置](#反向代理配置)
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
| 磁盘 | ≥100MB（二进制 ~22MB + 数据库按量增长） |
| 内存 | ≥64MB（万级 URL 向量缓存约 20MB） |
| 网络 | 需访问 LLM API（OpenAI / OpenRouter 等） |

> 纯 Go 编译（modernc SQLite），无 CGO 依赖，支持交叉编译。

---

## 本地开发部署

```bash
# 1. 克隆项目
git clone https://github.com/lupguo/linkstash.git
cd linkstash

# 2. 编译
go build -o linkstash-server ./cmd/server/
go build -o linkstash ./cmd/cli/

# 3. 准备配置
cp conf/app_dev.yaml conf/app.yaml
# 编辑 conf/app.yaml，设置 auth.secret_key 和 LLM API Key

# 4. 设置 LLM API Key（如使用 OpenAI）
export OPENAI_API_KEY="sk-xxx"

# 5. 启动
./linkstash-server -conf conf/app.yaml

# 6. 验证
curl http://localhost:8080/health
# 输出: {"status":"ok"}

# 7. 运行冒烟测试
./scripts/smoke_test.sh
```

---

## 生产部署

### 编译生产二进制

```bash
# Linux amd64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o linkstash-server ./cmd/server/
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o linkstash ./cmd/cli/

# Linux arm64（如树莓派、ARM 服务器）
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o linkstash-server ./cmd/server/

# macOS
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o linkstash-server ./cmd/server/
```

### 生产配置文件

创建 `conf/app_prod.yaml`：

```yaml
server:
  host: 127.0.0.1              # 仅监听本地（通过反向代理暴露）
  port: 8080

auth:
  secret_key: "替换为强随机密钥"    # openssl rand -hex 32
  jwt_secret: "替换为强随机密钥"    # openssl rand -hex 32
  jwt_expire_hours: 168          # 7 天

database:
  path: "/var/lib/linkstash/linkstash.db"

llm:
  chat:
    endpoint: "https://api.openai.com/v1/chat/completions"
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4o-mini"
  embedding:
    endpoint: "https://api.openai.com/v1/embeddings"
    api_key: "${OPENAI_API_KEY}"
    model: "text-embedding-3-small"
    dimensions: 512
  prompts:
    url_analysis: |
      分析以下网页内容，返回JSON格式：
      {"title":"标题","keywords":"关键词1,关键词2","description":"50字内摘要","category":"分类","tags":"标签1,标签2"}
      仅返回JSON，不要其他内容。
```

### 目录规划

```bash
sudo mkdir -p /opt/linkstash/bin
sudo mkdir -p /opt/linkstash/conf
sudo mkdir -p /opt/linkstash/web
sudo mkdir -p /var/lib/linkstash          # 数据库目录
sudo mkdir -p /var/log/linkstash          # 日志目录

# 部署文件
sudo cp linkstash-server /opt/linkstash/bin/
sudo cp linkstash /opt/linkstash/bin/
sudo cp conf/app_prod.yaml /opt/linkstash/conf/app.yaml
sudo cp -r web/ /opt/linkstash/web/
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
COPY --from=builder /build/linkstash-server /app/
COPY --from=builder /build/linkstash /app/
COPY --from=builder /build/web/ /app/web/
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
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    restart: unless-stopped

volumes:
  linkstash-data:
```

```bash
# 启动
OPENAI_API_KEY=sk-xxx docker compose up -d

# 查看日志
docker compose logs -f linkstash
```

---

## Systemd 服务

创建 `/etc/systemd/system/linkstash.service`：

```ini
[Unit]
Description=LinkStash URL Resource Manager
After=network.target

[Service]
Type=simple
User=linkstash
Group=linkstash
WorkingDirectory=/opt/linkstash
ExecStart=/opt/linkstash/bin/linkstash-server -conf /opt/linkstash/conf/app.yaml
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

# 安全加固
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/linkstash /var/log/linkstash

# 环境变量
EnvironmentFile=/opt/linkstash/conf/env

[Install]
WantedBy=multi-user.target
```

环境变量文件 `/opt/linkstash/conf/env`：

```bash
OPENAI_API_KEY=sk-xxx
```

```bash
# 创建用户
sudo useradd -r -s /sbin/nologin linkstash
sudo chown -R linkstash:linkstash /var/lib/linkstash /opt/linkstash

# 启动
sudo systemctl daemon-reload
sudo systemctl enable linkstash
sudo systemctl start linkstash

# 查看状态
sudo systemctl status linkstash
sudo journalctl -u linkstash -f
```

---

## 反向代理配置

### Nginx

```nginx
server {
    listen 443 ssl http2;
    server_name linkstash.example.com;

    ssl_certificate     /etc/letsencrypt/live/linkstash.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/linkstash.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 短链重定向单独处理（高优先级）
    location /s/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # 静态资源缓存
    location /static/ {
        proxy_pass http://127.0.0.1:8080;
        expires 7d;
        add_header Cache-Control "public, immutable";
    }
}

# HTTP → HTTPS 重定向
server {
    listen 80;
    server_name linkstash.example.com;
    return 301 https://$host$request_uri;
}
```

### Caddy（更简洁）

```caddyfile
linkstash.example.com {
    reverse_proxy localhost:8080
}
```

---

## 配置详解

### 完整配置项

```yaml
server:
  host: 0.0.0.0                      # 监听地址
  port: 8080                          # 监听端口

auth:
  secret_key: ""                      # [必填] 换取 JWT 的静态密钥
  jwt_secret: ""                      # [必填] JWT 签名密钥
  jwt_expire_hours: 72                # JWT 有效期（小时），默认 72

database:
  path: "./data/linkstash.db"         # SQLite 数据库文件路径

llm:
  chat:
    endpoint: ""                      # [必填] Chat API 端点
    api_key: "${OPENAI_API_KEY}"      # 支持 ${ENV_VAR} 环境变量引用
    model: "gpt-4o-mini"              # Chat 模型
  embedding:
    endpoint: ""                      # [必填] Embedding API 端点
    api_key: "${OPENAI_API_KEY}"
    model: "text-embedding-3-small"   # Embedding 模型
    dimensions: 512                   # 向量维度
  prompts:                            # Prompt 模板（可自定义覆盖）
    url_analysis: "..."
    url_categorize: "..."
```

### 兼容的 LLM 提供商

配置只需替换 `endpoint` + `api_key` + `model`：

| 提供商 | Endpoint | 模型示例 |
|--------|----------|----------|
| OpenAI | `https://api.openai.com/v1/...` | gpt-4o-mini / text-embedding-3-small |
| OpenRouter | `https://openrouter.ai/api/v1/...` | 按需选择 |
| 通义千问 | `https://dashscope.aliyuncs.com/compatible-mode/v1/...` | qwen-turbo / text-embedding-v3 |
| 本地 Ollama | `http://localhost:11434/v1/...` | llama3 / nomic-embed-text |

---

## 数据备份与恢复

### 备份

SQLite 是单文件数据库，备份非常简单：

```bash
# 方式 1：直接复制（需停机或使用 WAL 模式下的 checkpoint）
cp /var/lib/linkstash/linkstash.db /backup/linkstash-$(date +%Y%m%d).db

# 方式 2：使用 sqlite3 热备份（推荐，不停机）
sqlite3 /var/lib/linkstash/linkstash.db ".backup /backup/linkstash-$(date +%Y%m%d).db"
```

### 定时备份 (Cron)

```bash
# 每天凌晨 3 点备份，保留 30 天
0 3 * * * sqlite3 /var/lib/linkstash/linkstash.db ".backup /backup/linkstash-$(date +\%Y\%m\%d).db" && find /backup -name "linkstash-*.db" -mtime +30 -delete
```

### 恢复

```bash
# 停止服务
sudo systemctl stop linkstash

# 恢复数据库
cp /backup/linkstash-20260319.db /var/lib/linkstash/linkstash.db
sudo chown linkstash:linkstash /var/lib/linkstash/linkstash.db

# 启动服务
sudo systemctl start linkstash
```

---

## 监控与运维

### 健康检查

```bash
curl -f http://localhost:8080/health || echo "LinkStash is DOWN"
```

### 日志查看

```bash
# Systemd
sudo journalctl -u linkstash -f --no-pager

# Docker
docker compose logs -f linkstash
```

### 数据库状态

```bash
# 查看数据库大小
ls -lh /var/lib/linkstash/linkstash.db

# 查看各表记录数
sqlite3 /var/lib/linkstash/linkstash.db "
  SELECT 't_urls', COUNT(*) FROM t_urls
  UNION ALL SELECT 't_embeddings', COUNT(*) FROM t_embeddings
  UNION ALL SELECT 't_short_links', COUNT(*) FROM t_short_links
  UNION ALL SELECT 't_visit_records', COUNT(*) FROM t_visit_records
  UNION ALL SELECT 't_llm_logs', COUNT(*) FROM t_llm_logs;
"
```

### 性能参考

| 规模 | 内存占用 | 检索延迟 |
|------|---------|---------|
| 1,000 URLs | ~10MB | <10ms |
| 10,000 URLs | ~30MB | <50ms |
| 50,000 URLs | ~120MB | <200ms |

---

## 故障排查

### 常见问题

**1. 启动报 `disk I/O error`**

```
原因：数据库目录不存在或无写入权限
解决：mkdir -p /var/lib/linkstash && chown linkstash:linkstash /var/lib/linkstash
```

**2. LLM 分析失败（status=failed）**

```bash
# 检查失败的 URL
sqlite3 /var/lib/linkstash/linkstash.db "SELECT id, link, status FROM t_urls WHERE status='failed'"

# 查看 LLM 日志
sqlite3 /var/lib/linkstash/linkstash.db "SELECT url_id, request_type, error_message FROM t_llm_logs WHERE success=0 ORDER BY created_at DESC LIMIT 10"
```

可能原因：API Key 无效、网络不通、模型限流。修复后，重启服务会自动恢复 pending/analyzing 状态的任务。

**3. FTS5 搜索无结果**

确认数据已通过 UPDATE 触发器同步到 FTS 表：

```bash
sqlite3 /var/lib/linkstash/linkstash.db "SELECT * FROM t_urls_fts LIMIT 5"
```

如果 FTS 表为空但 t_urls 有数据，手动重建：

```sql
INSERT INTO t_urls_fts(t_urls_fts) VALUES('rebuild');
```

**4. 短链返回 404**

确认路由 `/s/:code` 已注册（不需鉴权），检查 code 是否存在：

```bash
sqlite3 /var/lib/linkstash/linkstash.db "SELECT code, long_url, expires_at FROM t_short_links WHERE deleted_at IS NULL"
```

**5. Web 页面 500 错误**

确保 `web/` 目录存在且包含模板文件。服务启动时在 `web/` 相对目录下查找模板。
