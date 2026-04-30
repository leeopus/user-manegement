# 部署指南

## 标准启动方式

### 📦 开发模式

所有服务本地运行，支持热重载，适合日常开发。

```bash
./dev.sh
```

**服务**:
- PostgreSQL: 本地服务 (systemctl)
- Redis: 本地服务 (systemctl)
- 后端: `go run` 直接运行
- 前端: `npm run dev` 支持热重载

**访问地址**:
- 前端: http://localhost:3000
- 后端: http://localhost:8080

**停止**:
```bash
./stop.sh
# 或手动停止
pkill -f "go run cmd/server/main.go"
pkill -f "next dev"
```

---

### 🚀 生产模式

Docker Compose 部署所有服务，适合生产环境。

```bash
./production.sh
```

**服务**:
- PostgreSQL: Docker 容器 (postgres:15-bookworm)
- Redis: Docker 容器 (redis:7-bookworm)
- 后端: Docker 容器
- 前端: Docker 容器

**访问地址**:
- 前端: http://106.15.3.98:3000
- 后端: http://106.15.3.98:8080

**停止**:
```bash
docker-compose down
# 或
./stop.sh
```

**查看日志**:
```bash
docker-compose logs -f          # 所有服务
docker-compose logs -f backend  # 仅后端
docker-compose logs -f frontend # 仅前端
```

---

## 环境配置

### 开发环境配置

**后端** (`backend/.env`):
```bash
DATABASE_URL=host=localhost port=5432 user=admin password=admin123 dbname=user_system sslmode=disable
REDIS_URL=redis://@localhost:6379/0
JWT_SECRET=dev-secret-key
GIN_MODE=debug
```

**前端** (`frontend/.env.local`):
```bash
NEXT_PUBLIC_API_URL=  # 空值，使用 Next.js rewrites
```

### 生产环境配置

**环境文件** (`.env.production`):
```bash
# 数据库密码
POSTGRES_PASSWORD=your_secure_password

# Redis 密码
REDIS_PASSWORD=your_secure_password

# JWT 密钥（必须修改）
JWT_SECRET=your_super_secret_key_change_this

# 前端 URL
FRONTEND_URL=http://106.15.3.98:3000

# CORS 允许的来源
CORS_ORIGINS=http://106.15.3.98:3000,http://106.15.3.98

# 前端 API 地址
NEXT_PUBLIC_API_URL=http://106.15.3.98:8080
```

---

## Docker 镜像说明

### PostgreSQL

| 镜像 | 大小 | 特点 |
|------|------|------|
| `postgres:15-bookworm` | ~390MB | Debian 12，推荐生产使用 |
| `postgres:15-alpine` | ~230MB | Alpine，节省资源 |

### Redis

| 镜像 | 大小 | 特点 |
|------|------|------|
| `redis:7-bookworm` | ~132MB | Debian 12，推荐生产使用 |
| `redis:7-alpine` | ~32MB | Alpine，节省资源 |

当前使用 **bookworm** 版本，更稳定、兼容性更好。

---

## 常见问题

### 1. 开发模式启动失败

**问题**: PostgreSQL/Redis 未运行

**解决**:
```bash
sudo systemctl start postgresql
sudo systemctl start redis
```

### 2. 生产模式容器无法启动

**问题**: 端口被占用

**解决**:
```bash
# 查看占用端口的进程
sudo lsof -i :3000
sudo lsof -i :8080
sudo lsof -i :5432
sudo lsof -i :6379

# 停止旧的服务
docker-compose down
./stop.sh
```

### 3. 前端无法连接后端

**开发模式**:
- 检查 `next.config.ts` 中的 rewrites 配置
- 确保后端正在运行

**生产模式**:
- 检查 `NEXT_PUBLIC_API_URL` 环境变量
- 确保使用容器名称或外部 IP

### 4. 数据库连接失败

**开发模式**:
```bash
# 测试连接
psql -h localhost -U admin -d user_system

# 检查服务状态
sudo systemctl status postgresql
```

**生产模式**:
```bash
# 进入容器测试
docker-compose exec postgres psql -U admin -d user_system

# 查看日志
docker-compose logs postgres
```

---

## 维护命令

```bash
# 重新构建镜像
docker-compose build --no-cache

# 查看资源使用
docker stats

# 清理未使用的镜像和容器
docker system prune -a

# 数据库备份
docker-compose exec postgres pg_dump -U admin user_system > backup.sql

# 数据库恢复
docker-compose exec -T postgres psql -U admin user_system < backup.sql
```
