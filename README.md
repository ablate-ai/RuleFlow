# RuleFlow - Clash & Stash 规则转换工具

将 Trojan 节点订阅转换为 Clash 和 Stash 配置文件的 Web 服务。

## ✨ 功能特性

- 🔗 **单节点转换** - 支持单个 Trojan 节点链接解析
- 📋 **订阅转换** - 支持批量订阅内容（原始或 Base64 编码）
- 🎯 **多客户端支持** - 支持 Clash 和 Stash 客户端配置生成
- 🌐 **Web 界面** - 简洁美观的在线转换界面
- ⚙️ **自动配置** - 生成完整的代理配置文件
- 📥 **一键下载** - 直接下载生成的 YAML 配置
- 🎯 **规则模板** - 通过 YAML 文件维护多套分流规则
- 🗄️ **数据库支持** - PostgreSQL 持久化订阅配置（可选）
- ⚡ **缓存加速** - Redis 缓存订阅内容，提升性能（可选）
- 🔄 **管理 API** - 完整的订阅管理和缓存管理接口（可选）

## 🚀 快速开始

### 编译

```bash
go build -o ruleflow ./cmd/ruleflow
```

### 运行（基础模式）

```bash
./ruleflow
```

服务将在 `http://localhost:8080` 启动。

### 运行（完整模式 - 带数据库和缓存）

```bash
# 复制环境变量配置文件
cp .env.example .env

# 编辑 .env 文件，配置数据库和 Redis 连接信息
# 然后运行
./ruleflow
```

### 自定义端口

```bash
PORT=3000 ./ruleflow
```

## 📖 使用方法

### 方式 1: 单节点转换

1. 打开浏览器访问 `http://localhost:8080`
2. 点击「单节点转换」标签
3. 粘贴 Trojan 节点链接
4. 点击「转换配置」
5. 下载生成的配置文件

### 方式 2: 订阅转换

1. 点击「订阅转换」标签
2. 粘贴订阅内容（支持 Base64 编码）
3. 点击「转换配置」
4. 下载生成的配置文件

### 方式 3: 数据库模式（新增）

如果启用了数据库和 Redis 支持，可以通过订阅名称访问配置：

```bash
# 创建订阅（需要先执行数据库迁移）
curl -X POST http://localhost:8080/api/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-subscription",
    "url": "https://example.com/trojan-sub",
    "target": "clash",
    "enabled": true
  }'

# 通过订阅名称获取配置
curl "http://localhost:8080/sub/my-subscription?target=clash"
```

## 🔗 Trojan 链接格式

```
trojan://password@server:port?sni=domain&name=节点名称
```

### 参数说明

| 参数 | 说明 | 必填 | 默认值 |
|------|------|------|--------|
| password | Trojan 密码 | ✅ | - |
| server | 服务器地址 | ✅ | - |
| port | 端口 | ❌ | 443 |
| sni | SNI 域名 | ❌ | 服务器地址 |
| name | 节点名称 | ❌ | 服务器地址 |

### 示例

```
# 完整参数
trojan://my-pass@node.example.com:443?sni=example.com&name=香港节点

# 最简参数
trojan://my-pass@node.example.com
```

## 🗄️ 数据库设置（可选）

### PostgreSQL 安装和配置

#### macOS
```bash
brew install postgresql@14
brew services start postgresql@14
createdb ruleflow
```

#### Ubuntu/Debian
```bash
sudo apt-get install postgresql postgresql-contrib
sudo -u postgres createdb ruleflow
```

#### 数据库初始化

```bash
# 推荐：自动加载 .env 后执行
make migrate

# 或者手动执行 SQL
psql ruleflow < migrations/init.sql
psql $DATABASE_URL < migrations/init.sql
```

### Redis 安装和配置

#### macOS
```bash
brew install redis
brew services start redis
```

#### Ubuntu/Debian
```bash
sudo apt-get install redis-server
sudo systemctl start redis
```

## 📡 API 接口

### GET /sub

把订阅地址转换为 Clash 或 Stash YAML 配置（推荐用于客户端订阅）。

**查询参数:**

- `url`（必填）: 原始订阅地址
- `target`（可选）: 目标客户端类型，支持 `clash`（默认）或 `stash`

**示例:**

```bash
# 生成 Clash 配置（默认）
curl "http://localhost:8080/sub?url=https%3A%2F%2Fexample.com%2Fsub"

# 生成 Stash 配置
curl "http://localhost:8080/sub?url=https%3A%2F%2Fexample.com%2Fsub&target=stash"
```

**响应头:**

- `X-Node-Count`: 节点数量
- `X-Rule-Template`: 实际使用的模板名（`clash` 或 `stash`）
- `Content-Disposition`: 文件名（`clash_config.yaml` 或 `stash_config.yaml`）

### GET /sub/{name}（数据库模式）

通过订阅名称获取配置（需要启用数据库和 Redis）。

**查询参数:**

- `target`（可选）: 目标客户端类型，支持 `clash`（默认）或 `stash`

**示例:**

```bash
curl "http://localhost:8080/sub/my-subscription?target=clash"
```

**响应头:**

- `X-Node-Count`: 节点数量
- `X-Rule-Template`: 实际使用的模板名
- `X-Cache`: `HIT` 或 `MISS`（标识是否来自缓存）
- `Content-Disposition`: 文件名

### 订阅管理 API

#### POST /api/subscriptions

创建订阅配置。

**请求体:**

```json
{
  "name": "my-subscription",
  "url": "https://example.com/trojan-sub",
  "target": "clash",
  "enabled": true,
  "description": "我的订阅",
  "tags": ["premium", "hk"]
}
```

**响应:**

```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "my-subscription",
    "url": "https://example.com/trojan-sub",
    "target": "clash",
    "enabled": true,
    "description": "我的订阅",
    "tags": ["premium", "hk"],
    "node_count": 0,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### GET /api/subscriptions

获取所有订阅列表。

**响应:**

```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "name": "my-subscription",
      "url": "https://example.com/trojan-sub",
      "enabled": true,
      "node_count": 10
    }
  ]
}
```

#### GET /api/subscriptions/{name}

获取单个订阅信息。

#### PUT /api/subscriptions/{name}

更新订阅配置。

**请求体:** 同 POST 请求。

#### DELETE /api/subscriptions/{name}

删除订阅配置。

**响应:**

```json
{
  "success": true,
  "data": {
    "message": "订阅已删除"
  }
}
```

### 缓存管理 API

#### POST /api/subscriptions/{name}/refresh

手动刷新订阅缓存。

**示例:**

```bash
curl -X POST "http://localhost:8080/api/subscriptions/my-subscription/refresh?target=clash"
```

#### DELETE /api/cache/{name}

清除指定订阅的缓存。

**示例:**

```bash
curl -X DELETE "http://localhost:8080/api/cache/my-subscription"
```

#### DELETE /api/cache

清除所有缓存。

**示例:**

```bash
curl -X DELETE "http://localhost:8080/api/cache"
```

### 健康检查 API

#### GET /health

检查服务健康状态。

**响应:**

```json
{
  "status": "healthy",
  "database": {
    "status": "healthy",
    "connections": "1/10"
  },
  "redis": {
    "status": "healthy",
    "connected": "true"
  }
}
```

可用模板示例：

- `bcnkd4jv_full`（完整模板，含 `rule-providers`）

### POST /convert

转换 Trojan 节点为 Clash 配置。

**请求体:**

```json
{
  "urls": ["trojan://password@server:443?sni=server.com&name=节点名"]
}
```

或使用订阅内容:

```json
{
  "subscription": "base64编码的订阅内容或原始多行链接"
}
```

**响应:**

```json
{
  "config": "port: 7890\nproxies:\n...",
  "count": 1
}
```

**错误响应:**

```json
{
  "error": "错误信息"
}
```

## ⚙️ 配置说明

### 默认端口

| 端口 | 用途 |
|------|------|
| 7890 | HTTP 代理 |
| 7891 | SOCKS5 代理 |
| 7892 | Redir 代理 |
| 7893 | Mixed 代理 |

### 代理组

- **🚀 节点选择** - 手动选择节点
- **♻️ 自动选择** - URL 测试自动选择

### 分流规则

规则集中维护在 `rules/template.yaml`。

默认文件示例：

```yaml
rule-providers:
  ai_non_ip:
    type: http
    behavior: classical
    format: text
    interval: 43200
    url: https://ruleset.skk.moe/Clash/non_ip/ai.txt
    path: ./sukkaw_ruleset/ai_non_ip.txt
```

程序会直接读取该完整模板文件（支持 `rule-providers`、自定义 `proxy-groups`、`rules`），并把订阅解析出来的节点注入到 `proxies`。
`rules/external/*` 仅作为导入参考。

## 🔧 自定义配置

优先通过模板文件维护规则，不需要改代码：直接编辑 `rules/template.yaml`。

也可以通过环境变量修改模板文件路径：

```bash
RULE_TEMPLATE_FILE=/path/to/template.yaml ./ruleflow
```

## 🎯 Clash vs Stash 配置差异

本工具支持为 Clash 和 Stash 客户端生成优化的配置文件：

### Clash 配置
- 包含代理端口设置（`port`、`socks-port` 等）
- 支持外部控制器（`external-controller`）
- 支持 TUN 模式配置
- 完整的 DNS 和规则集支持

### Stash 配置
- 移除 Clash 特定的端口设置（Stash 使用系统代理）
- 移除外部控制器配置
- 移除 TUN 模式配置（iOS 系统限制）
- 优化 DNS 配置以兼容 Stash
- 保持完整的代理组和规则支持

### 使用建议
- **iOS/macOS 用户**: 使用 `target=stash` 参数生成 Stash 配置
- **其他平台用户**: 使用默认的 Clash 配置或 `target=clash` 参数

## 🐳 Docker 部署

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o ruleflow ./cmd/ruleflow

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/ruleflow .
COPY --from=builder /app/web ./web
COPY --from=builder /app/rules ./rules
EXPOSE 8080
CMD ["./ruleflow"]
```

构建并运行：

```bash
docker build -t ruleflow .
docker run -p 8080:8080 ruleflow
```

## 📦 项目结构

```
RuleFlow/
├── cmd/
│   └── ruleflow/
│       └── main.go             # 程序入口
├── internal/
│   └── app/                    # 核心转换逻辑和页面处理
│       ├── handlers.go
│       ├── models.go
│       ├── subscription.go
│       ├── parser.go
│       └── config_builder.go
├── config/                      # 配置管理
│   └── config.go               # 配置结构和环境变量
├── database/                    # 数据访问层
│   ├── database.go             # 数据库连接管理
│   └── subscription_repo.go    # 订阅仓储
├── cache/                       # 缓存层
│   ├── redis.go                # Redis 客户端
│   └── subscription_cache.go   # 订阅缓存
├── services/                    # 业务逻辑层
│   └── subscription_service.go # 订阅服务
├── api/                         # API 处理器
│   ├── handlers.go             # HTTP 处理器
│   ├── middleware.go           # 中间件
│   └── response.go             # 响应格式化
├── migrations/                  # 数据库初始化脚本
│   └── init.sql
├── web/                         # Web 静态文件
├── rules/                       # 规则模板
├── go.mod                       # Go 模块依赖
├── .env.example                 # 环境变量示例
└── README.md                    # 本文件
```

## 🧪 测试

```bash
GOCACHE=$(pwd)/.cache/go-build go test ./...
```

## 📄 许可证

MIT License
