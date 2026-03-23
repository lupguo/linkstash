# LinkStash 腾讯云 CVM 部署指南

本文档介绍如何将 LinkStash 部署到腾讯云 CVM（RockLinux / CentOS 8+）服务器，使用 Caddy 作为反向代理并自动配置 HTTPS。

## 环境要求

- **操作系统**：RockLinux 8+ / CentOS 8+ / AlmaLinux 8+
- **域名**：`linkstash.sapaude.tech` 已解析到服务器公网 IP
- **端口**：开放 80（HTTP）和 443（HTTPS）
- **内存**：建议 ≥ 1GB

## 一、安装 LinkStash

### 1.1 创建用户和目录

```bash
# 创建专用用户（无登录 shell）
sudo useradd -r -s /sbin/nologin linkstash

# 创建安装目录
sudo mkdir -p /opt/linkstash/{bin,conf,data,web}
sudo chown -R linkstash:linkstash /opt/linkstash
```

### 1.2 一键安装脚本

从 GitHub Releases 下载最新的 `linux-amd64` 二进制文件：

```bash
# 设置版本号（替换为最新版本）
VERSION="v1.0.0"
REPO="lupguo/linkstash"

# 下载并解压
curl -sSL "https://github.com/${REPO}/releases/download/${VERSION}/linkstash-linux-amd64.tar.gz" \
  | sudo tar -xz -C /opt/linkstash/

# 确保二进制可执行
sudo chmod +x /opt/linkstash/bin/linkstash-server

# 设置权限
sudo chown -R linkstash:linkstash /opt/linkstash
```

### 1.3 目录结构

安装完成后，目录结构如下：

```
/opt/linkstash/
├── bin/linkstash-server    # 主程序二进制
├── conf/app_prod.yaml      # 生产环境配置
├── data/linkstash.db       # SQLite 数据库（运行后自动生成）
└── web/                    # 模板和静态文件
    ├── templates/
    └── static/
```

### 1.4 配置文件

编辑生产配置文件：

```bash
sudo -u linkstash vim /opt/linkstash/conf/app_prod.yaml
```

关键配置项：

```yaml
server:
  host: 127.0.0.1   # 仅本地监听，通过 Caddy 反向代理
  port: 8085         # 避免与其他服务冲突

auth:
  secret_key: "改为你自己的随机密钥"
  jwt_secret: "改为你自己的JWT密钥"
  jwt_expire_hours: 72

database:
  path: "./data/linkstash.db"

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
```

> ⚠️ **安全提示**：请务必修改 `secret_key` 和 `jwt_secret` 为强随机字符串。

### 1.5 环境变量

创建环境变量文件，用于存放敏感信息：

```bash
sudo vim /opt/linkstash/.env
```

内容示例：

```bash
# OpenAI API Key（LLM 分析功能需要）
OPENAI_API_KEY=sk-your-api-key-here

# 可选：代理设置（如果需要）
# HTTP_PROXY=http://127.0.0.1:8118
# HTTPS_PROXY=http://127.0.0.1:8118
```

设置权限（仅 linkstash 用户可读）：

```bash
sudo chown linkstash:linkstash /opt/linkstash/.env
sudo chmod 600 /opt/linkstash/.env
```

## 二、Systemd 服务

### 2.1 创建 Service 文件

```bash
sudo vim /etc/systemd/system/linkstash.service
```

写入以下内容：

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
ReadWritePaths=/opt/linkstash/data

[Install]
WantedBy=multi-user.target
```

### 2.2 启动与管理

```bash
# 重新加载 systemd 配置
sudo systemctl daemon-reload

# 设置开机自启并立即启动
sudo systemctl enable --now linkstash

# 查看服务状态
sudo systemctl status linkstash

# 其他管理命令
sudo systemctl restart linkstash   # 重启
sudo systemctl stop linkstash      # 停止
```

### 2.3 验证服务

```bash
# 检查本地端口是否监听
curl -I http://127.0.0.1:8085

# 查看日志
journalctl -u linkstash -f
```

## 三、Caddy 反向代理 + HTTPS

### 3.1 安装 Caddy

```bash
# 添加 Caddy 官方仓库
sudo dnf install -y 'dnf-command(copr)'
sudo dnf copr enable -y @caddy/caddy
sudo dnf install -y caddy
```

如果使用 CentOS / RockLinux 且 `copr` 不可用，可使用以下方式：

```bash
# 直接安装二进制
sudo curl -sSL "https://caddyserver.com/api/download?os=linux&arch=amd64" -o /usr/bin/caddy
sudo chmod +x /usr/bin/caddy
sudo groupadd --system caddy
sudo useradd --system --gid caddy --create-home --home-dir /var/lib/caddy --shell /usr/sbin/nologin caddy
```

### 3.2 配置 Caddyfile

```bash
sudo vim /etc/caddy/Caddyfile
```

写入以下内容：

```caddyfile
linkstash.sapaude.tech {
    reverse_proxy 127.0.0.1:8085

    # 可选：自定义日志
    log {
        output file /var/log/caddy/linkstash.log
        format json
    }

    # 可选：压缩
    encode gzip zstd
}
```

> 💡 Caddy 会自动申请 Let's Encrypt 证书，并自动处理 HTTP → HTTPS 重定向，无需额外配置。

### 3.3 启动 Caddy

```bash
# 创建日志目录
sudo mkdir -p /var/log/caddy
sudo chown caddy:caddy /var/log/caddy

# 设置开机自启并立即启动
sudo systemctl enable --now caddy

# 查看 Caddy 状态
sudo systemctl status caddy
```

## 四、域名解析

请在腾讯云 DNS 控制台或域名注册商处添加以下 DNS 记录：

| 类型 | 主机记录 | 记录值 | TTL |
|------|----------|--------|-----|
| A | linkstash | `<你的服务器公网 IP>` | 600 |

> 确保域名 `linkstash.sapaude.tech` 已正确解析到服务器 IP，Caddy 才能成功申请 SSL 证书。

## 五、防火墙配置

```bash
# 开放 HTTP 和 HTTPS 端口
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --reload

# 验证
sudo firewall-cmd --list-services
```

> 同时检查腾讯云控制台的安全组规则，确保入站规则开放了 80 和 443 端口。

## 六、验证部署

### 6.1 本地检查

```bash
# 检查 LinkStash 服务
curl -I http://127.0.0.1:8085

# 检查 Caddy 代理
curl -I https://linkstash.sapaude.tech
```

### 6.2 外网访问

在浏览器中访问 `https://linkstash.sapaude.tech`，确认：

- ✅ 页面正常加载
- ✅ HTTPS 证书有效（地址栏显示🔒）
- ✅ 底部 footer 显示 ICP 备案号
- ✅ 各功能正常工作

## 七、日常维护

### 查看日志

```bash
# 实时查看 LinkStash 日志
journalctl -u linkstash -f

# 查看最近 100 条日志
journalctl -u linkstash -n 100

# 查看 Caddy 日志
journalctl -u caddy -f
```

### 备份数据库

```bash
# 手动备份
sudo cp /opt/linkstash/data/linkstash.db /opt/linkstash/data/linkstash.db.bak.$(date +%Y%m%d)

# 建议设置 crontab 定期备份
# 每天凌晨 3 点自动备份
echo "0 3 * * * cp /opt/linkstash/data/linkstash.db /opt/linkstash/data/linkstash.db.bak.\$(date +\%Y\%m\%d)" \
  | sudo crontab -u linkstash -
```

### 更新版本

```bash
# 1. 下载新版本
VERSION="v1.x.x"  # 替换为新版本号
curl -sSL "https://github.com/lupguo/linkstash/releases/download/${VERSION}/linkstash-linux-amd64.tar.gz" \
  | sudo tar -xz -C /opt/linkstash/

# 2. 重启服务
sudo systemctl restart linkstash

# 3. 验证
sudo systemctl status linkstash
curl -I http://127.0.0.1:8085
```

### 常见问题排查

| 问题 | 排查方法 |
|------|----------|
| 服务启动失败 | `journalctl -u linkstash -n 50` 查看错误日志 |
| 端口未监听 | `ss -tlnp \| grep 8085` 检查端口占用 |
| Caddy 证书失败 | 确认域名解析正确，80/443 端口开放 |
| 502 Bad Gateway | 确认 LinkStash 服务正在运行 |
| 数据库锁定 | 检查是否有多个进程访问同一数据库文件 |
