# LinkStash 腾讯云 CVM 部署指南

在 RockyLinux 8/9 服务器上部署 LinkStash，使用 Caddy 反向代理 + HTTPS。

## 环境要求

| 项目 | 要求 |
|------|------|
| 操作系统 | RockyLinux 8+ / AlmaLinux 8+ / CentOS Stream 8+ |
| 域名 | `linkstash.sapaude.tech` 已解析到服务器公网 IP |
| 端口 | 80（HTTP）、443（HTTPS）|
| 内存 | ≥ 1GB（Chromium headless 需要额外内存）|
| 依赖 | Chromium（URL 自动分析用）|

## 一、安装 LinkStash

### 1.1 创建用户和目录

```bash
# 创建专用用户
sudo useradd -r -s /sbin/nologin linkstash

# 创建安装目录
sudo mkdir -p /opt/linkstash/{bin,conf,data,logs,web}
sudo chown -R linkstash:linkstash /opt/linkstash
```

### 1.2 部署二进制

**方式 A：一键安装脚本（推荐）**

自动检测 OS/架构，下载最新版本：

```bash
# 安装到 /opt/linkstash/bin（自动获取最新 Release）
curl -fsSL https://raw.githubusercontent.com/lupguo/linkstash/main/scripts/install.sh \
  | sudo bash -s -- --dir /opt/linkstash/bin

# 或指定版本
curl -fsSL https://raw.githubusercontent.com/lupguo/linkstash/main/scripts/install.sh \
  | sudo bash -s -- --dir /opt/linkstash/bin --version v0.2.0
```

**方式 B：手动下载 Release**

```bash
# 自动获取最新版本号
VERSION=$(curl -fsSL https://api.github.com/repos/lupguo/linkstash/releases/latest | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
echo "Latest version: $VERSION"
REPO="lupguo/linkstash"

# 下载 server 和 CLI
sudo curl -fsSL "https://github.com/${REPO}/releases/download/${VERSION}/linkstash-server-linux-amd64" \
  -o /opt/linkstash/bin/linkstash-server
sudo curl -fsSL "https://github.com/${REPO}/releases/download/${VERSION}/linkstash-linux-amd64" \
  -o /opt/linkstash/bin/linkstash
sudo chmod +x /opt/linkstash/bin/linkstash-server /opt/linkstash/bin/linkstash
```

**方式 C：从源码编译**

```bash
# 需要 Go 1.25+、make、esbuild、tailwindcss
git clone https://github.com/lupguo/linkstash.git /tmp/linkstash-build
cd /tmp/linkstash-build
make release

# 复制二进制
sudo cp bin/release/linkstash-server-linux-amd64 /opt/linkstash/bin/linkstash-server
sudo cp bin/release/linkstash-linux-amd64 /opt/linkstash/bin/linkstash

# 复制 web 资源（模板 + 静态文件）
sudo cp -r web/templates web/components web/static /opt/linkstash/web/

# 复制配置模板
sudo cp conf/app_prod.yaml /opt/linkstash/conf/app_prod.yaml
```

### 1.3 安装 Chromium（URL 自动分析依赖）

LinkStash 使用 [rod](https://go-rod.github.io/) 驱动 headless Chromium 抓取网页内容，供 LLM 分析。

```bash
# RockyLinux 8/9
sudo dnf install -y chromium

# 验证
chromium-browser --version
```

如果 `chromium` 包不可用：

```bash
# 使用 EPEL
sudo dnf install -y epel-release
sudo dnf install -y chromium
```

> 在 `app_prod.yaml` 中配置 `browser.bin_path` 指向实际路径：
> ```yaml
> browser:
>   enabled: true
>   bin_path: "/usr/bin/chromium-browser"  # 留空则自动下载
>   headless: true
>   timeout_sec: 30
> ```

### 1.4 目录结构

```
/opt/linkstash/
├── bin/
│   ├── linkstash-server       # 主服务
│   └── linkstash              # CLI 工具
├── conf/
│   └── app_prod.yaml          # 生产配置
├── data/
│   ├── linkstash.db           # SQLite 数据库（自动生成）
│   └── linkstash.bleve/       # Bleve 全文搜索索引（自动生成）
├── logs/
│   └── app.log                # 应用日志
└── web/
    ├── templates/             # Go HTML 模板
    ├── components/            # 共享模板组件
    └── static/                # CSS/JS/图片
```

### 1.5 配置文件

```bash
sudo -u linkstash vim /opt/linkstash/conf/app_prod.yaml
```

```yaml
server:
  host: 127.0.0.1    # 仅本地监听，通过 Caddy 反向代理
  port: 8085

auth:
  secret_key: "替换为强随机字符串"         # 登录密钥
  jwt_secret: "替换为另一个强随机字符串"    # JWT 签名密钥
  jwt_expire_hours: 72

database:
  path: "./data/linkstash.db"

log:
  level: "info"
  file: "./logs/app.log"
  format: "text"

short:
  ttl_options:
    - label: "永久"
      value: ""
    - label: "7 天"
      value: "7d"
    - label: "30 天"
      value: "30d"
    - label: "1 年"
      value: "365d"

# LLM 配置 — 用于 URL 自动分析（标题/摘要/分类/标签）
# 方式 A：OpenRouter（推荐，支持多模型切换）
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
  prompts:
    url_analysis: |
      分析以下网页内容，返回JSON格式：
      {"title":"标题","keywords":"关键词1,关键词2","description":"50字内摘要","category":"分类","tags":"标签1,标签2"}
      category必须从以下选项中选择：技术、设计、产品、商业、科学、生活、工具、资讯、其他
      tags基于内容自由生成，用逗号分隔，2-5个标签。
      仅返回JSON，不要其他内容。

# 方式 B：OpenAI 直连
# llm:
#   chat:
#     endpoint: "https://api.openai.com/v1/chat/completions"
#     api_key: "${OPENAI_API_KEY}"
#     model: "gpt-4o-mini"
#   embedding:
#     endpoint: "https://api.openai.com/v1/embeddings"
#     api_key: "${OPENAI_API_KEY}"
#     model: "text-embedding-3-small"
#     dimensions: 512

categories:
  - "技术"
  - "设计"
  - "产品"
  - "商业"
  - "科学"
  - "生活"
  - "工具"
  - "资讯"
  - "其他"

browser:
  enabled: true
  bin_path: "/usr/bin/chromium-browser"
  headless: true
  timeout_sec: 30

# 可选：代理（服务器无法直接访问外网时）
# proxy:
#   http_proxy: "http://127.0.0.1:8118"
```

> ⚠️ **安全提示**：`secret_key` 和 `jwt_secret` 必须替换为强随机字符串。可用 `openssl rand -hex 32` 生成。

### 1.6 环境变量

```bash
sudo vim /opt/linkstash/.env
```

```bash
# LLM API Key（二选一）
OPENROUTER_API_KEY=sk-or-your-api-key-here
# OPENAI_API_KEY=sk-your-api-key-here

# 可选：代理
# HTTP_PROXY=http://127.0.0.1:8118
# HTTPS_PROXY=http://127.0.0.1:8118
```

```bash
# 仅 linkstash 用户可读
sudo chown linkstash:linkstash /opt/linkstash/.env
sudo chmod 600 /opt/linkstash/.env
```

## 二、Systemd 服务

### 2.1 创建 Service 文件

```bash
sudo vim /etc/systemd/system/linkstash.service
```

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

> 注意 `ReadWritePaths` 包含 `logs` 目录，否则日志写入会被 `ProtectSystem=strict` 阻止。

### 2.2 启动与管理

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now linkstash
sudo systemctl status linkstash
```

### 2.3 验证服务

```bash
# 健康检查
curl -s http://127.0.0.1:8085/health
# 期望输出: {"status":"ok"}

# 查看日志
journalctl -u linkstash -f
```

## 三、Caddy 反向代理 + HTTPS

### 3.1 安装 Caddy

```bash
sudo dnf install -y 'dnf-command(copr)'
sudo dnf copr enable -y @caddy/caddy
sudo dnf install -y caddy
```

如果 `copr` 不可用：

```bash
sudo curl -fsSL "https://caddyserver.com/api/download?os=linux&arch=amd64" -o /usr/bin/caddy
sudo chmod +x /usr/bin/caddy
sudo groupadd --system caddy 2>/dev/null
sudo useradd --system --gid caddy --create-home --home-dir /var/lib/caddy --shell /usr/sbin/nologin caddy 2>/dev/null
```

### 3.2 配置 Caddyfile

```bash
sudo vim /etc/caddy/Caddyfile
```

```caddyfile
linkstash.sapaude.tech {
    reverse_proxy 127.0.0.1:8085

    encode gzip zstd

    log {
        output file /var/log/caddy/linkstash.log
        format json
    }
}
```

> Caddy 自动申请 Let's Encrypt 证书，自动处理 HTTP → HTTPS 重定向。

### 3.3 启动 Caddy

```bash
sudo mkdir -p /var/log/caddy
sudo chown caddy:caddy /var/log/caddy
sudo systemctl enable --now caddy
sudo systemctl status caddy
```

## 四、防火墙 + 安全组

```bash
# firewalld
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --reload
sudo firewall-cmd --list-services
```

> 同时在腾讯云控制台的安全组中开放入站 80 和 443 端口。

## 五、域名解析

在腾讯云 DNS 控制台添加：

| 类型 | 主机记录 | 记录值 | TTL |
|------|----------|--------|-----|
| A | linkstash | `<服务器公网 IP>` | 600 |

## 六、验证部署

```bash
# 本地
curl -s http://127.0.0.1:8085/health

# 外网
curl -I https://linkstash.sapaude.tech
```

浏览器访问 `https://linkstash.sapaude.tech` 确认：
- ✅ HTTPS 证书有效（🔒）
- ✅ 登录页正常显示
- ✅ 登录后首页卡片列表渲染
- ✅ 无限滚动加载正常
- ✅ 搜索功能正常

## 七、日常维护

### 查看日志

```bash
journalctl -u linkstash -f          # 实时日志
journalctl -u linkstash -n 100      # 最近 100 条
journalctl -u caddy -f              # Caddy 日志
tail -f /opt/linkstash/logs/app.log  # 应用日志文件
```

### 备份数据库

```bash
# 手动备份
sudo -u linkstash cp /opt/linkstash/data/linkstash.db \
  /opt/linkstash/data/linkstash.db.bak.$(date +%Y%m%d)

# 定时备份（每天凌晨 3 点）
echo '0 3 * * * cp /opt/linkstash/data/linkstash.db /opt/linkstash/data/linkstash.db.bak.$(date +\%Y\%m\%d)' \
  | sudo crontab -u linkstash -
```

### 更新版本

```bash
# 1. 备份
sudo -u linkstash cp /opt/linkstash/data/linkstash.db /opt/linkstash/data/linkstash.db.bak

# 2. 下载最新版本（自动获取）
VERSION=$(curl -fsSL https://api.github.com/repos/lupguo/linkstash/releases/latest | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
echo "Updating to $VERSION"
sudo curl -fsSL "https://github.com/lupguo/linkstash/releases/download/${VERSION}/linkstash-server-linux-amd64" \
  -o /opt/linkstash/bin/linkstash-server
sudo chmod +x /opt/linkstash/bin/linkstash-server
sudo chown linkstash:linkstash /opt/linkstash/bin/linkstash-server

# 3. 重启
sudo systemctl restart linkstash
sudo systemctl status linkstash
curl -s http://127.0.0.1:8085/health
```

### 常见问题

| 问题 | 排查方法 |
|------|----------|
| 服务启动失败 | `journalctl -u linkstash -n 50` 查看错误 |
| 端口未监听 | `ss -tlnp \| grep 8085` |
| Caddy 证书失败 | 确认域名解析正确 + 80/443 端口开放 |
| 502 Bad Gateway | 确认 LinkStash 服务正在运行 |
| URL 分析失败 | 检查 Chromium 安装 + LLM API Key 配置 |
| 搜索无结果 | 检查 `data/linkstash.bleve/` 索引目录是否存在 |
| 日志写入失败 | 检查 systemd `ReadWritePaths` 是否包含 logs 目录 |
| 数据库锁定 | 确认只有一个进程访问 SQLite 文件 |
