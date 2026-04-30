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

### 开发模式

所有服务本地运行，支持热重载：

```bash
./dev.sh
```

- 前端: http://localhost:3000
- 后端: http://localhost:8080
- 支持 Next.js 热重载
- 后端使用 `go run` 直接运行

**停止开发模式**:
- 按 `Ctrl+C` 停止前端
- `pkill -f "go run cmd/server/main.go"` 停止后端
- 或使用: `./stop.sh`

### 生产模式

Docker Compose 部署所有服务：

```bash
./production.sh
```

- 前端: http://106.15.3.98:3000
- 后端: http://106.15.3.98:8080
- 所有服务运行在 Docker 容器中
- 自动配置健康检查和重启策略

**停止生产模式**:
```bash
docker-compose down
# 或
./stop.sh
```

### 手动控制 Docker

```bash
# 查看日志
docker-compose logs -f

# 重启服务
docker-compose restart

# 进入容器
docker-compose exec backend sh
docker-compose exec postgres psql -U admin -d user_system
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
