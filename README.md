# 🔗 ShortLinkGo

<div align="center">

**短链接服务 · Self-hosted** — 使用 Go + Vue 编写，单二进制 + SQLite 即可运行

[![Release](https://img.shields.io/github/v/release/xiaoman1221/ShortLinkGo?sort=semver&label=%E7%89%88%E6%9C%AC&color=18181b)](https://github.com/xiaoman1221/ShortLinkGo/releases/latest)
[![Build](https://github.com/xiaoman1221/ShortLinkGo/actions/workflows/release.yml/badge.svg?label=%E6%9E%84%E5%BB%BA)](https://github.com/xiaoman1221/ShortLinkGo/actions/workflows/release.yml)
[![Docker](https://img.shields.io/docker/v/xiaoman1221/shortlinkgo?sort=semver&label=Docker&color=2496ED&logo=docker&logoColor=white)](https://hub.docker.com/r/xiaoman1221/shortlinkgo)
![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)
![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vuedotjs&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-嵌入式-003B57?logo=sqlite&logoColor=white)

</div>

---

短链接服务 - 使用 Go 语言编写的短链接系统。

## 功能特性

- **多页面管理后台** - 统计 / 链接管理 / 个人中心 / 系统管理；右上角头像下拉进入个人中心
- **统计与访问地图** - 链接总数、累计访问等概览；按 IP 解析的访客分布地图；近 14 天访问趋势；访问最多的 Top 链接
- **短链生成与自定义** - 自动生成唯一短码（去除易混淆字符）；支持自定义短码（2-32 位字母数字下划线连字符），系统保留路由关键字自动拦截
- **链接审核与角色范围** - 普通用户仅见本人链接且新链接默认「待审核」；管理员/超级管理员可见全部链接并标注创建者，可审核（通过/停用/待审核）
- **账号体系与角色** - 首个注册用户为超级管理员（UID=1）；角色 super/admin/vip/user，支持注册 / 登录 / 邮箱找回密码 / 个人资料 / 修改密码 / 头像
- **API Token** - 个人中心创建令牌，可用 `Authorization: Bearer <token>` 增删改查本人短链接（与 JWT 等效）
- **系统管理** - 仅管理员/超级管理员可访问；用户管理（超级管理员可改角色、封禁）；网站基本信息（名称/简介/站点地址自动检测/菜单栏 Logo）；SMTP 配置与测试邮件；GeoIP 数据库（网页开关、下载源配置、启动自动下载、每小时检查更新、热重载）
- **QQ 登录** - 官方 QQ 互联 OAuth2 登录（系统管理配置 App ID/App Key，回调自动拼接）；QQ 新用户自动注册，可再设置密码/绑定邮箱
- **SMTP 邮件** - 支持 465 隐式 TLS 与 25/587 STARTTLS，用于发送找回密码邮件
- **访问留痕与 GeoIP** - 每次访问记录 IP、国家/城市、UA、来源；真实 IP 自动探测转发头（smart/always/direct/cidr 四种模式网页可切）；GeoLite2 数据库在后台「访问地图」一键启用，自动下载并每小时更新；解析不出的 IP 自动调用公共查询 API（ip-api/ipinfo/ipwho.is/自定义可切换）兜底并缓存入库
- **接口文档** - `/docs` 提供接入文档（含 API Token
- 用法）、OpenAPI 规范与 Swagger UI

## 技术栈

**后端**
- Go 1.27 + Gin
- GORM + SQLite
- JWT 认证

**前端**
- Vue 3 + Vite
- Pinia 状态管理
- Vue Router
- 自研设计系统（极简单色 · 现代编辑排印）

## 快速开始

### Docker 部署（推荐）

```bash
docker compose up -d --build
```

访问 http://localhost:8080

> 从旧版本升级：上传目录已从 `./uploads` 迁移至 `./data/uploads`（容器内为数据卷 `/app/data/uploads`，随数据库一同持久化）。宿主机部署请手动移动旧目录：`mkdir -p data && mv uploads data/uploads`。

#### 使用已发布的镜像

每个 `vX.Y.Z` 版本会自动构建多架构镜像（amd64/arm64）发布到 Docker Hub，tag 包含 `latest` 与对应版本号：

```bash
docker run -d -p 8080:8080 -v shortlinkgo-data:/app/data \
  -e JWT_KEY=请替换为高强度随机字符串 \
  xiaoman1221/shortlinkgo:latest
```

也可以从 [Releases](https://github.com/xiaoman1221/ShortLinkGo/releases) 下载对应平台的二进制压缩包（已包含前端资源与文档），解压后直接运行。

> 数据默认保存在 Docker 命名卷 `shortlinkgo-data`（挂载到容器 `/app/data`）中。
> 建议在部署前设置环境变量 `JWT_KEY`（持久化随机密钥）与 `HOST`（站点对外地址，
> 用于拼装短链接），示例：

```bash
JWT_KEY=your-strong-random-secret HOST=https://s.example.com docker compose up -d
```

#### 中国大陆网络环境构建

Dockerfile 默认已使用大陆可直连的镜像源（`goproxy.cn`、`sum.golang.google.cn`、
`npmmirror`），无需额外配置。海外构建需切回官方源：

```bash
NPM_REGISTRY=https://registry.npmjs.org \
GOPROXY=https://proxy.golang.org,direct \
GOSUMDB=sum.golang.org \
docker compose build
```

若无法从 Docker Hub 拉取基础镜像，可覆盖基础镜像（阿里云镜像仓库）或为 Docker
守护进程配置 registry-mirror：

```bash
BASE_NODE_IMAGE=registry.cn-hangzhou.aliyuncs.com/library/node:20-alpine \
BASE_GO_IMAGE=registry.cn-hangzhou.aliyuncs.com/library/golang:1.27-alpine \
BASE_RUNTIME_IMAGE=registry.cn-hangzhou.aliyuncs.com/library/alpine:3.20 \
docker compose build
```

### 手动部署

**环境要求**

- Go 1.27+
- Node.js 18+
- npm

**配置**

复制 `.env.example` 为 `.env` 并修改配置：

```bash
cp .env.example .env
```

| 变量 | 说明 | 默认值  |
|------|------|---------|
| `PORT` | 服务端口 | 8080    |
| `GIN_MODE` | Gin模式（debug/release） | release |
| `DB_PATH` | 数据库文件路径 | data.db |
| `JWT_KEY` | JWT密钥 | -       |
| `JWT_EXPIRE` | JWT有效期（秒） | 86400   |
| `SMTP_HOST` `SMTP_PORT` `SMTP_USER` `SMTP_PASS` `SMTP_FROM` | 首次启动写入系统设置的 SMTP 配置（也可在后台“系统管理”配置） | - |
| `SITE_NAME` `SITE_DESC` | 首次启动写入的站点名称/简介 | - |
| `GEO_DB_PATH` | MaxMind GeoLite2-City.mmdb 路径（访问地图地理解析，可选） | - |

**构建运行**

```bash
# 一键构建（前端+后端）
make
# 或
bash build.sh

# 运行
./shortlinkgo
```

### 账号体系

系统不再内置默认账号：

- **第一个注册的用户自动成为超级管理员（UID=1）**
- 角色：超级管理员（super）/ 管理员（admin）/ VIP / 用户（user）
- 超级管理员可设置用户角色、封禁/解封；管理员可查看用户与配置系统
- 支持注册 / 登录 / 邮箱找回密码（需配置 SMTP）

## API 接口

| 模块 | 路径 | 说明 |
|------|------|------|
| 短码跳转 | `GET /{code}` | 302 跳转；不存在/停用/过期/审核中返回 404 |
| 链接管理 | `/api/links` | 增删改查、分页、自定义短码；管理员视角含创建者与审核（`POST /:id/review`） |
| 统计 | `/api/stats/*` | summary / trend / top / geo（访客地理分布） |
| 认证 | `/api/auth` | 注册、登录、QQ 登录（`/qq`、`/qq/callback`）、资料、改密、找回密码、头像上传 |
| API 令牌 | `/api/tokens` | 令牌创建/列表/删除，用于以本人身份调用链接接口 |
| 用户管理 | `/api/admin/users` | 用户列表（管理员）；角色与封禁（仅超级管理员） |
| 系统设置 | `/api/settings` | 网站信息 + SMTP 配置、Logo 上传、测试邮件（仅管理员） |
| 站点信息 | `GET /api/site` | 站点名称/Logo/简介（公开，供前端菜单栏展示） |
| 健康检查 | `/api/health` | 服务存活检测 |
| 接口文档 | `/docs` | 接入文档（含 API Token 用法）、`openapi.yaml`、Swagger UI |

### 使用示例

创建短链接：

```bash
curl -X POST http://localhost:8080/api/links \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/very/long/path","remark":"示例"}'
```

返回：

```json
{"code":0,"msg":"创建成功","data":{"id":1,"code":"aB3xYz","url":"https://example.com/very/long/path","short_url":"http://localhost:8080/aB3xYz","remark":"示例","status":1,"visit_count":0}}
```

访问跳转：

```bash
curl -I http://localhost:8080/aB3xYz   # 302 Location: https://example.com/very/long/path
```

## 项目结构

```
ShortLinkGo/
├── docs/            # 接口文档资源（index.html / openapi.yaml / swagger.html）
├── data/            # 运行时数据（数据库 / 上传文件 / GeoIP 库，运行时生成，已忽略）
├── server/          # 后端（Go）
│   ├── api/         #   HTTP 层：路由、处理器、鉴权中间件（JWT/API Token、角色门禁）
│   ├── app/         #   应用装配：配置加载、数据库初始化与迁移
│   ├── services/    #   业务逻辑与数据模型
│   ├── utils/       #   工具（响应/密码/JWT/邮件/IP 地理解析，可选 GeoLite2 mmdb）
│   └── main.go      #   入口
├── web/             # Vue前端
│   ├── src/
│   └── dist/        # 前端构建产物
├── .dockerignore
├── .env.example     # 环境变量模板
├── Dockerfile       # 多阶段构建（前端 + 后端）
├── docker-compose.yml
├── go.mod
└── main.go
```

## License

MIT
