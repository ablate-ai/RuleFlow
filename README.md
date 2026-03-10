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
go build -o ruleflow .
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

# 编辑 .env 文件，配置管理密码、数据库和 Redis 连接信息
# 然后运行
./ruleflow
```

如果设置了 `ADMIN_PASSWORD`，访问 `/web/*` 控制台页面时会先跳转到 `/login` 登录页，`/api/*` 在未登录时会返回 `401`。

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

当配置了 `ADMIN_PASSWORD` 后，除 `/sub`、`/sub/{name}`、`/config`、`/health` 外，控制台相关接口需要先登录：

- `/web/*` 未登录时跳转到 `/login`
- `/api/*` 未登录时返回 `401`

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

#### GET /api/subscriptions/{id}

获取单个订阅信息。

#### PUT /api/subscriptions/{id}

更新订阅配置。

**请求体:** 同 POST 请求。

#### DELETE /api/subscriptions/{id}

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

## ⚙️ 配置说明

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `PORT` | HTTP 服务端口 | `8080` |
| `ADMIN_PASSWORD` | Web 控制台和管理 API 登录密码；为空时不启用鉴权 | 空 |
| `DATABASE_URL` | PostgreSQL 连接串 | `postgresql://ruleflow:password@localhost:5432/ruleflow?sslmode=disable` |
| `REDIS_ADDR` | Redis 地址 | `localhost:6379` |
| `REDIS_PASSWORD` | Redis 密码 | 空 |
| `REDIS_DB` | Redis DB 编号 | `0` |
| `CACHE_TTL_SECONDS` | 订阅缓存 TTL | `3600` |

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

### 模板扩展字段

`proxy-groups` 支持两个服务端专属扩展字段，生成配置时会被自动处理并从输出中删除：

#### `filter` — 节点过滤

按正则表达式筛选进入该组的节点，只有名称匹配的节点会被放入 `__NODES__` 展开位置。

```yaml
proxy-groups:
  - name: 🇸🇬 SG
    type: url-test
    filter: "SG|新加坡|Singapore"   # 只选名称含 SG / 新加坡 / Singapore 的节点
    proxies: ["__NODES__"]
    url: http://cp.cloudflare.com/generate_204
    interval: 300
```

#### `dialer-proxy` — 链式代理（落地中转）

按正则表达式在所有节点中找到**第一个**匹配的节点作为中转，自动为该组内的节点 `proxies` 条目注入 `dialer-proxy` 字段，实现流量 A → 中转 → 目标的链式路由。

> Clash/Stash 会输出 `dialer-proxy`；Surge 会自动翻译为节点级 `underlying-proxy`。

```yaml
proxy-groups:
  - name: "🇺🇸 US via SG"
    type: select
    filter: "US|美国"              # 该组只包含美国节点
    dialer-proxy: "SG|新加坡"      # 这些节点的流量先经过第一个匹配的新加坡节点
    proxies: ["__NODES__"]
```

生成的 Clash 配置效果：

```yaml
proxies:
  - name: 🇸🇬 SG-01          # 中转节点本身，无 dialer-proxy
    type: trojan
    ...
  - name: 🇺🇸 US-01
    type: vmess
    dialer-proxy: 🇸🇬 SG-01   # 自动注入
    ...
proxy-groups:
  - name: "🇺🇸 US via SG"
    type: select               # dialer-proxy 字段已删除
    proxies: [🇺🇸 US-01, ...]
```

两个字段可以同时使用，也可以单独使用，对没有这两个字段的组完全向后兼容。

### Surge 模板写法

Surge 的链式代理最终落在 `[Proxy]` 节点行的 `underlying-proxy=` 参数上，但模板里仍然可以继续使用统一的扩展字段 `dialer-proxy=`。生成器会先根据 `[Proxy Group]` 规则选中节点，再把中转信息回写到 `[Proxy]` 展开的节点行。

推荐写法 1：配合 `__NODES__` 和 `filter=`

```ini
[Proxy]
__NODES__

[Proxy Group]
🤖 AI = select, __NODES__, filter=US|美国, dialer-proxy=SG|新加坡
🎬 Stream = select, 🤖 AI, 🇭🇰 HK, DIRECT
```

推荐写法 1b：同时使用 `filter=` 和 `exclude-filter=`

`exclude-filter=` 在 `filter=` 圈出候选节点后，再将匹配的节点踢掉，适合排除 IPLC / BGP / 中转等特定类型：

```ini
[Proxy Group]
🇸🇬 SG = url-test, __NODES__, url=http://www.gstatic.com/generate_204, interval=1200, filter=新加坡|SG, exclude-filter=IPLC|BGP|中转, dialer-proxy=SG|新加坡
```

执行顺序：先 `filter=` 保留匹配节点 → 再 `exclude-filter=` 踢掉不想要的 → 最终节点列表展开到 `__NODES__`。生成结果中 `filter=` 和 `exclude-filter=` 均不保留。

推荐写法 2：直接给显式节点列表挂链

```ini
[Proxy]
__NODES__

[Proxy Group]
🤖 AI = select, us.lax.dmit, us.hnl.qqpw, url=http://www.gstatic.com/generate_204, interval=1200, dialer-proxy=SG|新加坡
```

生成后的 Surge `[Proxy]` 片段会类似这样：

```ini
[Proxy]
🇸🇬 SG Relay = trojan, sg.example.com, 443, password=xxx, sni=sg.example.com
🇺🇸 us.lax.dmit = trojan, us1.example.com, 443, password=yyy, sni=us1.example.com, underlying-proxy=🇸🇬 SG Relay
🇺🇸 us.hnl.qqpw = trojan, us2.example.com, 443, password=zzz, sni=us2.example.com, underlying-proxy=🇸🇬 SG Relay
```

注意：

- `dialer-proxy=` 应写在包含真实节点的 `[Proxy Group]` 行上，不能指望外层“组套组”自动向下传播。
- 例如 `🎬 Stream = select, 🤖 AI, 🇺🇸 US, dialer-proxy=SG|新加坡` 这种写法不会直接修改任何节点，因为这行成员主要是组名，不是实际节点。
- 生成后的 Surge 配置里不会保留 `dialer-proxy=`，只会留下节点级 `underlying-proxy=`。

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
RUN go build -ldflags="-s -w" -o ruleflow .

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
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

# 启用控制台登录鉴权
docker run -p 8080:8080 -e ADMIN_PASSWORD=change-me ruleflow
```

## 📦 项目结构

```
RuleFlow/
├── main.go                     # 程序入口
├── internal/
│   └── app/                    # 核心转换逻辑和页面处理
│       ├── handlers.go
│       ├── models.go
│       ├── subscription.go
│       ├── parser.go
│       └── config_builder.go
├── config/                      # 配置管理
│   └── config.go               # 配置结构和环境变量
├── database/                   # 数据访问层
│   ├── database.go             # 数据库连接管理
│   ├── config_policy_repo.go   # 配置策略仓储
│   ├── node_repo.go            # 节点仓储
│   ├── subscription_repo.go    # 订阅仓储
│   └── template_repo.go        # 模板仓储
├── cache/                      # 缓存层
│   ├── redis.go                # Redis 客户端
│   └── subscription_cache.go   # 订阅缓存
├── services/                   # 业务逻辑层
│   ├── config_policy_service.go
│   ├── node_service.go
│   ├── subscription_service.go
│   ├── subscription_sync_service.go
│   └── template_service.go
├── api/                        # API 处理器
│   ├── handlers.go             # HTTP 处理器
│   ├── middleware.go           # 中间件
│   └── response.go             # 响应格式化
├── migrations/                 # 数据库初始化脚本
│   └── init.sql
├── web/                        # Web 静态文件
│   ├── index.html
│   ├── login.html
│   ├── subscriptions.html
│   ├── nodes.html
│   ├── templates.html
│   └── configs.html
├── rules/                      # 规则模板
│   └── template.yaml
├── Makefile
├── Dockerfile
├── .drone.yml
├── go.mod
├── .env.example
└── README.md
```

## 🧪 测试

```bash
GOCACHE=$(pwd)/.cache/go-build go test ./...
```

## 📄 许可证

MIT License
