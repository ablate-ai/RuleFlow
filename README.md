# RuleFlow

将代理订阅转换为 Clash Meta、Stash、Surge 客户端配置文件的自托管 Web 服务。

支持多协议解析、规则模板注入、链式代理、节点过滤，提供 Web 管理控制台和 REST API。

---

## ✨ 功能特性

- **多客户端支持** — 生成 Clash Meta（YAML）、Stash（YAML）、Surge（INI）配置
- **多协议解析** — Trojan、VMess、VLESS、Shadowsocks、Hysteria2、TUIC v5
- **规则模板** — 上传自定义 YAML/INI 模板，自动注入节点
- **节点过滤与分组** — 在模板中用正则表达式过滤节点进入特定代理组
- **链式代理** — 模板扩展字段 `dialer-proxy`，自动生成中转落地配置
- **Web 管理控制台** — 可视化管理订阅、节点、模板、配置策略
- **REST API** — 完整 CRUD 接口，可程序化集成
- **Redis 缓存** — 缓存生成的配置文件，加速订阅分发（可选）
- **定时同步** — 自动拉取和刷新订阅源（可选）
- **鉴权保护** — `ADMIN_PASSWORD` 保护控制台和 API（可选）

---

## 🚀 快速开始

### 方式一：直接运行（无数据库）

```bash
git clone <repo-url>
cd RuleFlow
go build -o ruleflow .
./ruleflow
# 访问 http://localhost:8080
```

在 `rules/template.yaml` 中维护规则模板，重启服务生效。

### 方式二：完整模式（含数据库 + Redis）

```bash
# 1. 复制环境变量配置
cp .env.example .env
# 编辑 .env，填写数据库、Redis 连接信息和管理密码

# 2. 初始化数据库
make migrate

# 3. 启动服务
make run
```

### 方式三：Docker

```bash
docker build -t ruleflow .

# 基础运行
docker run -p 8080:8080 ruleflow

# 带数据库和鉴权
docker run -p 8080:8080 \
  -e ADMIN_PASSWORD=your-password \
  -e DATABASE_URL=postgresql://user:pass@host:5432/ruleflow \
  -e REDIS_ADDR=redis:6379 \
  ruleflow
```

**docker-compose 示例：**

```yaml
services:
  ruleflow:
    image: ruleflow
    ports:
      - "8080:8080"
    environment:
      ADMIN_PASSWORD: your-password
      DATABASE_URL: postgresql://ruleflow:password@postgres:5432/ruleflow?sslmode=disable
      REDIS_ADDR: redis:6379
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: ruleflow
      POSTGRES_USER: ruleflow
      POSTGRES_PASSWORD: password
    volumes:
      - pgdata:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine

volumes:
  pgdata:
```

---

## 📖 使用流程

### 基础用法

1. **添加订阅源**：在控制台「订阅源」页面填入订阅 URL
2. **同步节点**：点击同步按钮拉取节点列表
3. **选择或上传模板**：使用内置模板或在「规则模板」页面上传自定义模板
4. **创建配置策略**：绑定订阅源 + 模板 + 目标客户端，生成专属订阅链接
5. **在客户端中使用**：将生成的 `/config?token=xxx` 链接填入客户端

### 订阅链接格式

```
http://your-server:8080/config?token=<token>
```

Surge 客户端会自动识别响应中的 `#!MANAGED-CONFIG` 头，支持远程更新。

---

## 🔧 规则模板

### Clash Meta / Stash 模板（YAML）

在 `proxy-groups` 中支持两个扩展字段，生成配置时自动处理并从输出中删除：

#### `filter` — 节点过滤

```yaml
proxy-groups:
  - name: 🇸🇬 新加坡
    type: url-test
    filter: "SG|新加坡|Singapore"    # 正则，仅匹配的节点进入该组
    proxies: ["__NODES__"]
    url: http://cp.cloudflare.com/generate_204
    interval: 300
```

#### `exclude-filter` — 排除节点

```yaml
  - name: 🇸🇬 新加坡（直连）
    type: url-test
    filter: "SG|新加坡"
    exclude-filter: "IPLC|BGP|中转"  # 在 filter 结果中再排除
    proxies: ["__NODES__"]
```

#### `dialer-proxy` — 链式代理（中转落地）

```yaml
  - name: 🇺🇸 美国 via 新加坡
    type: select
    filter: "US|美国"
    dialer-proxy: "SG|新加坡"        # 正则匹配第一个新加坡节点作为中转
    proxies: ["__NODES__"]
```

生成效果：

```yaml
proxies:
  - name: 🇸🇬 SG-01
    type: trojan
    ...                              # 中转节点，无 dialer-proxy
  - name: 🇺🇸 US-01
    type: vmess
    dialer-proxy: 🇸🇬 SG-01         # 自动注入
    ...
```

### Surge 模板（INI）

```ini
[Proxy]
__NODES__

[Proxy Group]
🇸🇬 SG = url-test, __NODES__, filter=SG|新加坡, url=http://cp.cloudflare.com/generate_204, interval=300
🤖 AI = select, __NODES__, filter=US|美国, exclude-filter=IPLC|BGP, dialer-proxy=SG|新加坡

[Rule]
RULE-SET,https://ruleset.skk.moe/Clash/non_ip/ai.txt,🤖 AI
FINAL,🇸🇬 SG
```

生成后 `filter=`、`exclude-filter=`、`dialer-proxy=` 均不保留；`dialer-proxy` 会被翻译为节点行的 `underlying-proxy=` 参数。

> **注意**：`dialer-proxy` 只对包含实际节点的组有效，不能在「组套组」的外层组上使用。

---

## 📡 API 参考

设置 `ADMIN_PASSWORD` 后，`/api/*` 接口需登录后方可访问（Cookie session）。

| 资源 | 方法 | 路径 |
|------|------|------|
| 订阅源 | CRUD | `/api/subscriptions` |
| 订阅同步 | POST | `/api/subscriptions/{id}/sync` |
| 节点 | CRUD + 批量导入 | `/api/nodes` |
| 规则模板 | CRUD | `/api/templates` |
| 配置策略 | CRUD | `/api/config-policies` |
| 清除配置缓存 | DELETE | `/api/config-policies/{id}/cache` |
| 生成配置 | GET | `/config?token=xxx` |
| 健康检查 | GET | `/health` |

---

## ⚙️ 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `PORT` | HTTP 服务端口 | `8080` |
| `ADMIN_PASSWORD` | 控制台登录密码；为空则不启用鉴权 | 空 |
| `DATABASE_URL` | PostgreSQL 连接串 | `postgresql://ruleflow:password@localhost:5432/ruleflow?sslmode=disable` |
| `REDIS_ADDR` | Redis 地址 | `localhost:6379` |
| `REDIS_PASSWORD` | Redis 密码 | 空 |
| `REDIS_DB` | Redis 数据库编号 | `0` |
| `CACHE_TTL_SECONDS` | 配置缓存有效期（秒） | `3600` |
| `SURGE_MANAGED_CONFIG_BASE_URL` | Surge `#!MANAGED-CONFIG` 的外部地址，例如 `https://sub.example.com`；为空时自动从请求头推断 | 空 |

---

## 🗄️ 数据库初始化

```bash
# 推荐（自动读取 .env）
make migrate

# 手动执行
psql $DATABASE_URL < migrations/init.sql
```

新版本升级时执行 `migrations/` 目录下对应日期的增量迁移文件。

---

## 📦 项目结构

```
RuleFlow/
├── main.go                          # 入口，路由注册
├── internal/app/                    # 核心逻辑
│   ├── parser.go                    # 多协议节点 URL 解析
│   ├── config_builder.go            # Clash Meta / Stash 配置生成
│   ├── surge_builder.go             # Surge INI 配置生成
│   ├── models.go                    # 数据模型
│   ├── subscription.go              # 订阅拉取
│   └── country_emoji.go             # 节点名称地区 emoji
├── api/                             # HTTP 处理层
│   ├── handlers.go                  # REST API 处理器
│   ├── middleware.go                # 鉴权、CORS
│   └── surge_managed_config.go      # Surge #!MANAGED-CONFIG 支持
├── services/                        # 业务逻辑层
├── database/                        # 数据访问层（PostgreSQL）
├── cache/                           # 缓存层（Redis）
├── config/                          # 环境变量加载
├── web/                             # Web 控制台（HTML/React）
├── rules/                           # 内置规则模板
│   └── template.yaml
├── migrations/                      # 数据库迁移脚本
├── Dockerfile
├── Makefile
└── .env.example
```

---

## 🧪 测试

```bash
make test
# 或
GOCACHE=$(pwd)/.cache/go-build go test ./...
```

---

## 📄 许可证

MIT License
