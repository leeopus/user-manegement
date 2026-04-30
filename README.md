# 用户管理系统

一个轻量级、高性能的用户管理系统，支持万级用户规模，具备完整的认证授权和SSO单点登录功能。

## 技术栈

### 后端
- **Golang** + **Gin** - 高性能Web框架
- **GORM** - ORM库
- **PostgreSQL** - 主数据库
- **Redis** - 缓存与会话存储
- **JWT** - 身份认证

### 前端
- **Next.js 15** - React框架
- **TypeScript** - 类型安全
- **shadcn/ui** - UI组件库
- **Tailwind CSS** - 样式框架

### 部署
- **Docker** + **Docker Compose** - 容器化部署

## 核心功能

- ✅ 用户注册/登录（邮箱、手机号）
- ✅ JWT认证 + Token刷新
- ✅ RBAC权限控制（角色、权限）
- ✅ SSO单点登录（OAuth2/OIDC）
- ✅ 用户管理（CRUD）
- ✅ 审计日志
- ✅ 第三方应用接入

## 快速开始

### 前置要求

- Docker & Docker Compose
- Go 1.21+ (本地开发)
- Node.js 20+ (本地开发)

### 使用Docker启动

```bash
# 克隆项目
git clone <repository-url>
cd sys

# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 访问前端
open http://localhost:3000

# 访问后端API
curl http://localhost:8080/health
```

### 本地开发

#### 后端开发

```bash
cd backend

# 安装依赖
go mod download

# 运行开发服务器
go run cmd/server/main.go

# 运行测试
go test ./...
```

#### 前端开发

```bash
cd frontend

# 安装依赖
pnpm install

# 运行开发服务器
pnpm dev

# 构建生产版本
pnpm build
```

## 项目结构

```
sys/
├── backend/           # Golang后端
│   ├── cmd/          # 入口文件
│   ├── internal/     # 内部包
│   │   ├── config/   # 配置
│   │   ├── models/   # 数据模型
│   │   ├── handler/  # HTTP处理器
│   │   ├── service/  # 业务逻辑
│   │   ├── repository/ # 数据访问
│   │   └── middleware/ # 中间件
│   └── pkg/          # 公共包
├── frontend/         # Next.js前端
│   ├── app/         # App Router页面
│   ├── components/  # React组件
│   ├── lib/         # 工具库
│   └── public/      # 静态资源
├── docker/          # Docker配置
├── docs/            # 文档
└── docker-compose.yml
```

## 环境变量

复制 `.env.example` 到 `.env` 并配置：

```bash
cp .env.example .env
```

### 必需配置

- `DATABASE_URL` - PostgreSQL连接字符串
- `REDIS_URL` - Redis连接字符串
- `JWT_SECRET` - JWT密钥
- `FRONTEND_URL` - 前端URL

## API文档

详见 [API.md](docs/API.md)

## SSO接入文档

详见 [SSO.md](docs/SSO.md)

## 部署文档

详见 [DEPLOY.md](docs/DEPLOY.md)

## 贡献

欢迎提交Issue和Pull Request！

## 许可证

MIT License
