# Go-Vue-Admin API

基于 Go (Gin) 的轻量级 RBAC 后台管理 API 服务

## 📋 项目介绍

这是一个纯后端 API 服务，提供完整的 RBAC（基于角色的访问控制）权限管理功能，包括用户管理、角色管理、菜单管理等基础模块。

### 功能特性

- ✅ RESTful API 设计
- ✅ JWT 认证与授权（支持自动刷新）
- ✅ Token 黑名单（登出即失效）
- ✅ RBAC 权限管理（用户/角色/菜单）
- ✅ 限流保护（登录/API 双限流）
- ✅ 密码加密存储（bcrypt）
- ✅ Swagger API 文档
- ✅ CORS 跨域支持
- ✅ 操作日志记录（支持开关控制）
- ✅ 登录日志审计（支持开关控制）
- ✅ Casbin RBAC 权限控制（API 级别）
- ✅ 系统设置管理
- ✅ 登录验证码（纯 Go 生成，内存存储，5 分钟 TTL）
- ✅ 账户锁定（连续 5 次失败锁定 30 分钟）
- ✅ 自定义参数校验（phone / password / status 标签）
- ✅ 日志按日轮转（自动切割，防止单文件过大）
- ✅ 统一错误码体系（1000~6999 分段语义化）

## 🏗️ 技术栈

- **语言**: Go 1.24+
- **框架**: Gin
- **ORM**: GORM
- **数据库**: MySQL 5.7+
- **文档**: Swagger
- **认证**: JWT

## 📁 项目结构

项目采用清晰的分层架构，职责分离：

```
.
├── api/v1/              # API 接口层（HTTP 处理）
│   ├── system_user.go   # 用户管理接口
│   ├── system_role.go   # 角色管理接口
│   └── user.go          # 前台用户接口
├── services/v1/         # 业务逻辑层（核心业务）
│   ├── system_user.go   # 用户业务逻辑
│   ├── system_role.go   # 角色业务逻辑
│   └── user.go          # 前台用户业务逻辑
├── models/              # 数据模型层（结构体定义）
│   ├── system_user.go   # 用户模型
│   ├── system_role.go   # 角色模型（包含菜单）
│   ├── system_log.go    # 日志模型
│   └── res/             # 响应模型
├── router/v1/           # 路由层
├── middleware/          # 中间件
│   ├── jwt_auth.go      # JWT认证
│   └── cors.go          # 跨域处理
├── docs/                # Swagger 文档
├── util/                # 工具函数
│   └── captcha.go       # 验证码生成与校验（内存 TTL 存储）
├── global/              # 全局变量
├── conf/                # 配置加载
├── core/                # 核心初始化
│   ├── validator.go     # 自定义 Gin 校验器（phone / password / status）
│   └── logrus.go        # 日志按日轮转
├── flag/                # 命令行工具
├── setting.yaml         # 配置文件
├── Makefile             # 快捷命令
└── main.go              # 入口文件
```

### 架构分层说明

```
┌────────────────────────────────────────┐
│  api/v1/                               │
│  • 接收 HTTP 请求                       │
│  • 参数校验、绑定                        │
│  • 调用 Service 层                      │
│  • 返回 JSON 响应                       │
├────────────────────────────────────────┤
│  services/v1/                          │
│  • 业务逻辑实现                          │
│  • 数据库操作（ORM）                     │
│  • 事务管理                             │
│  • 返回业务错误或数据                     │
├────────────────────────────────────────┤
│  models/                               │
│  • 数据结构定义                          │
│  • 表名配置                             │
│  • 请求/响应 DTO                        │
└────────────────────────────────────────┘
```

**分层原则**：

- **API 层**：只处理 HTTP 相关，不直接操作数据库
- **Service 层**：处理业务逻辑，封装数据库操作
- **Model 层**：定义数据结构，保持简洁

## 📚 API 文档

项目集成了 Swagger API 文档，启动服务后可以访问：

```
http://localhost:8080/swagger/index.html
```

### Swagger 文档说明

Swagger 文档提供了以下功能：

- **在线测试**: 可直接在浏览器中测试 API 接口
- **认证支持**: 支持 JWT Token 认证（点击 Authorize 按钮输入 Bearer token）
- **参数说明**: 详细的请求/响应参数说明
- **示例代码**: 提供请求示例和响应示例

### 生成 Swagger 文档

```bash
# 安装 swag 工具
go install github.com/swaggo/swag/cmd/swag@latest

# 生成文档
make swagger
# 或
swag init -g main.go -o ./docs
```

### API 接口分类

| 分类 | 接口路径                | 说明         |
| ---- | ----------------------- | ------------ |
| 认证 | `/api/v1/system/login`  | 用户登录（需验证码） |
| 认证 | `/api/v1/system/logout` | 用户登出     |
| 认证 | `/api/v1/system/captcha` | 获取验证码图片 |
| 认证 | `/api/v1/system/refresh-token` | 刷新 JWT Token |
| 用户 | `/api/v1/system/user/*` | 用户管理     |
| 角色 | `/api/v1/system/role/*` | 角色管理     |
| 菜单 | `/api/v1/system/menu/*` | 菜单管理     |
| 路由 | `/api/v1/system/routes` | 获取动态路由 |
| 操作日志 | `/api/v1/system/log/operation/*` | 操作日志管理 |
| 登录日志 | `/api/v1/system/log/login/*` | 登录日志管理 |
| 系统设置 | `/api/v1/system/settings` | 日志开关控制 |

## 🚀 快速开始

### 环境要求

- Go 1.24+
- MySQL 5.7+

### 1. 克隆项目

```bash
git clone https://github.com/Liukers/go-vue-admin-api
cd go-vue-admin-api
```

### 2. 数据库初始化

创建 MySQL 数据库：

```sql
CREATE DATABASE go-vue-admin DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 3. 配置

修改配置文件 `setting.yaml`：

```yaml
system:
  addr: 8080 # 服务端口
  mode: debug # 运行模式: debug/release
  trust-proxy: false # 是否信任反向代理的 X-Forwarded-For（生产环境有 Nginx/Traefik 时设为 true）

mysql:
  path: localhost
  port: 3306
  config: "charset=utf8mb4&parseTime=True&loc=Local"
  db-name: go-vue-admin
  username: root
  password: "your-password"
  max-idle-conns: 10
  max-open-conns: 100

jwt:
  signing-key: your-secret-key # JWT签名密钥（生产环境必须修改）
  expires-time: 24 # token过期时间（小时）
  buffer-time: 1 # 缓冲时间（小时）

# 跨域配置（CORS）
# 开发环境：allow-all: true 允许所有来源
# 生产环境：设置 allow-all: false，在 whitelist 中配置前端域名
cors:
  allow-all: true
  whitelist:
    - allow-origin: "http://your-frontend-domain.com"
      allow-headers: "content-type, authorization, x-requested-with"
      allow-methods: "GET, POST, PUT, DELETE, OPTIONS"
      allow-credentials: true
```

### 4. 运行

```bash
# 下载依赖
go mod tidy

# 初始化数据库（创建表和基础数据）
go run main.go -db

# 启动服务
go run main.go

# 或编译运行
go build -o go-vue-admin-api
./go-vue-admin-api
```

服务默认运行在 `http://localhost:8080`

> **注意**：首次启动若看到 `You trusted all proxies` 警告是正常的，生产环境（`mode: release`）会自动配置安全代理。

### 5. 访问 Swagger 文档

打开浏览器访问：`http://localhost:8080/swagger/index.html`

默认账号：

- 用户名：`admin`
- 密码：`admin123`

## 🔌 前端配套（可选）

本项目可作为独立 API 服务部署，也可以配合前端使用：

### 方式一：配合配套前端（推荐）

我们提供了配套的 Vue3 管理界面：

| 仓库 | GitHub | Gitee |
|------|--------|-------|
| **go-vue-admin-ui** | [GitHub](https://github.com/Liukers/go-vue-admin-ui) | [Gitee](https://gitee.com/liukers/go-vue-admin-ui) |

**快速启动完整项目**：

```bash
# 1. 克隆后端
git clone https://github.com/Liukers/go-vue-admin-api.git
# 或 git clone https://gitee.com/liukers/go-vue-admin-api.git
cd go-vue-admin-api
go run main.go -db  # 初始化数据库
go run main.go      # 启动后端 (端口 8080)

# 2. 克隆前端（新终端）
git clone https://github.com/Liukers/go-vue-admin-ui.git
# 或 git clone https://gitee.com/liukers/go-vue-admin-ui.git
cd go-vue-admin-ui
pnpm install
pnpm dev  # 启动前端 (端口 8848)
```

### 方式二：独立部署 API

本项目提供标准 RESTful API，可独立部署供任意客户端使用：

```bash
# 部署 API 服务
go build -o go-vue-admin-api
./go-vue-admin-api

# API 文档地址
http://localhost:8080/swagger/index.html
```

**支持的客户端**：
- Vue/React/Angular 等前端框架
- 移动端 App（iOS/Android）
- 小程序（微信/支付宝等）
- 第三方系统对接

### API 配置说明

- **Base URL**：`http://localhost:8080/api/v1`
- **认证方式**：JWT Token（Header: `Authorization: Bearer {token}`）
- **跨域支持**：已配置 CORS，支持浏览器端调用
- **文档地址**：`http://localhost:8080/swagger/index.html`

## 🛠️ 开发指南

### 如何新增 API 接口

参考项目中已有的**用户管理**实现：

**1. 创建数据模型** (`models/`)

参考 `models/system_user.go`：
- 定义结构体，添加 `gorm` 标签
- 实现 `TableName()` 方法
- 定义请求/响应 DTO

**2. 创建 Service 层** (`services/v1/`)

参考 `services/v1/system_user.go`：
- 实现业务逻辑
- 使用 `global.DB` 操作数据库
- 返回业务错误或数据

**3. 创建 API 接口层** (`api/v1/`)

参考 `api/v1/system_user.go`：
- 接收 HTTP 请求
- 参数校验、绑定
- 调用 Service 层
- 返回 JSON 响应

**4. 注册路由** (`router/v1/`)

参考 `router/v1/system.go`：
- 创建路由组
- 使用 `middleware.JWTAuth()` 保护需要认证的路由
- 注册到入口 `router/v1/enter.go`

**5. 生成 Swagger 文档**

```bash
make swagger
# 或
swag init -g main.go -o ./docs
```

## 🔐 认证说明

### JWT Token

登录接口返回的 token 需要在后续请求的 Header 中携带：

```
Authorization: Bearer {your-jwt-token}
```

### 权限控制

系统采用 **RBAC + Casbin** 模型：

```
用户 → 角色 → 菜单权限 → Casbin 策略 → API 访问控制
```

- **JWT 认证**: `middleware.JWTAuth()` 验证用户身份
- **Casbin 授权**: `middleware.CasbinAuth()` 验证 API 访问权限
- **自动同步**: 角色菜单权限变更时，Casbin 策略自动更新
- **权限存储**: 策略存储在 MySQL `casbin_rule` 表中

### 系统设置

支持动态开关控制：

| 设置项 | 说明 | 影响 |
|--------|------|------|
| 操作日志记录 | 开启/关闭操作日志 | 控制日志记录和菜单显示 |
| 登录日志记录 | 开启/关闭登录日志 | 控制日志记录和菜单显示 |

设置变更后立即生效，无需重启服务。

## 📝 常用命令

### Makefile 快捷命令

```bash
# 构建项目
make build

# 运行项目
make run

# 生成 swagger 文档
make swagger

# 初始化数据库
make db

# 重置数据库
make reset-db

# 重置管理员密码
make reset-pwd

# 查看所有命令
make help
```

### Go 命令

```bash
# 初始化数据库
go run main.go -db

# 重置数据库
go run main.go -reset-db

# 重置管理员密码
go run main.go -reset-pwd

# 查看帮助
go run main.go -h

# 编译
go build -o go-vue-admin-api

# 运行
./go-vue-admin-api
```

## 📦 部署

### 生产环境准备

部署前必须修改以下配置：

1. **修改 JWT 密钥**（`setting.yaml`）

   ```yaml
   jwt:
     signing-key: your-strong-secret-key-here # 生产环境必须使用强密钥
   ```

2. **配置反向代理信任**（`setting.yaml`）

   如果使用 Nginx/Traefik 等反向代理，设为 `true` 以正确获取客户端真实 IP（影响限流和日志）：

   ```yaml
   system:
     trust-proxy: true
   ```

   **直连暴露模式请保持 `false`**，否则客户端可伪造 IP 绕过登录限流。

3. **修改数据库密码**

   ```yaml
   mysql:
     password: "your-strong-password"
   ```

4. **切换运行模式**

   ```yaml
   system:
     mode: release # 改为 release 模式
   ```

5. **配置 CORS 白名单**
   ```yaml
   cors:
     allow-all: false # 关闭允许所有
     whitelist:
       - allow-origin: "https://your-frontend-domain.com"
         allow-headers: "content-type, authorization, x-requested-with"
         allow-methods: "GET, POST, PUT, DELETE, OPTIONS"
         allow-credentials: true
   ```

### 编译部署

```bash
# 编译（Linux）
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o go-vue-admin-api

# 复制到服务器
cp go-vue-admin-api setting.yaml /opt/go-admin/

# 运行
./go-vue-admin-api
```

### Docker 部署

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o go-vue-admin-api main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/go-vue-admin-api .
COPY --from=builder /app/setting.yaml .
EXPOSE 8080
CMD ["./go-vue-admin-api"]
```

## 🙏 致谢

### 依赖项目

| 项目        | 说明         | 地址                              |
| ----------- | ------------ | --------------------------------- |
| **Gin**     | Go Web 框架  | https://github.com/gin-gonic/gin  |
| **GORM**    | Go ORM 库    | https://github.com/go-gorm/gorm   |
| **Swagger** | API 文档工具 | https://github.com/swaggo/swag    |
| **JWT-Go**  | JWT 认证库   | https://github.com/golang-jwt/jwt |

## 📄 许可证

MIT License

## 🔗 相关仓库

| 平台 | 地址 |
|------|------|
| **GitHub** | https://github.com/Liukers/go-vue-admin-api |
| **Gitee** | https://gitee.com/liukers/go-vue-admin-api |

**配套前端**：
- GitHub: https://github.com/Liukers/go-vue-admin-ui
- Gitee: https://gitee.com/liukers/go-vue-admin-ui

## 💬 问题反馈

- GitHub Issues: https://github.com/Liukers/go-vue-admin-api/issues
- Gitee Issues: https://gitee.com/liukers/go-vue-admin-api/issues

---

**如果觉得项目有帮助，请给个 Star ⭐️ 支持一下！**
