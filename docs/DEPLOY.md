# 部署文档

## Docker 部署（推荐）

### 前置要求

- Docker 20.10+
- Docker Compose 2.0+

### 快速启动

1. **克隆项目**

```bash
git clone <repository-url>
cd sys
```

2. **配置环境变量**

```bash
cp .env.example .env
# 编辑 .env 文件，修改必要的配置
```

3. **启动服务**

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 查看服务状态
docker-compose ps
```

4. **访问服务**

- 前端: http://localhost:3000
- 后端API: http://localhost:8080
- PostgreSQL: localhost:5432
- Redis: localhost:6379

5. **停止服务**

```bash
# 停止所有服务
docker-compose down

# 停止并删除数据卷
docker-compose down -v
```

### 服务说明

#### PostgreSQL

- **端口**: 5432
- **数据库**: user_system
- **用户**: admin
- **密码**: admin123

#### Redis

- **端口**: 6379
- **密码**: redis123

#### Backend (Golang)

- **端口**: 8080
- **健康检查**: http://localhost:8080/health

#### Frontend (Next.js)

- **端口**: 3000

## 本地开发部署

### 后端开发

1. **安装依赖**

```bash
cd backend
go mod download
```

2. **配置环境变量**

```bash
cp .env.example .env
# 编辑 .env 文件
```

3. **启动数据库**

```bash
# 启动PostgreSQL和Redis
docker-compose up -d postgres redis
```

4. **运行开发服务器**

```bash
go run cmd/server/main.go
```

5. **运行测试**

```bash
go test ./...
```

### 前端开发

1. **安装依赖**

```bash
cd frontend
npm install
```

2. **配置环境变量**

```bash
cp .env.local.example .env.local
# 编辑 .env.local 文件
```

3. **启动开发服务器**

```bash
npm run dev
```

4. **构建生产版本**

```bash
npm run build
npm start
```

## 生产环境部署

### 使用 Docker Compose

1. **修改环境变量**

```bash
# 编辑 .env 文件
vim .env
```

关键配置：
```env
# 生产环境必须修改
JWT_SECRET=your-super-secret-jwt-key-change-in-production
DATABASE_URL=postgres://user:password@host:5432/dbname
GIN_MODE=release
```

2. **使用生产配置启动**

```bash
docker-compose -f docker-compose.prod.yml up -d
```

### 使用 Kubernetes

1. **构建镜像**

```bash
# 构建后端镜像
docker build -t user-system-backend:latest ./backend

# 构建前端镜像
docker build -t user-system-frontend:latest ./frontend
```

2. **部署到Kubernetes**

```bash
kubectl apply -f k8s/
```

### 使用 Nginx 反向代理

配置示例：

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # 前端
    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # 后端API
    location /api {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

## 数据库迁移

### 自动迁移

系统启动时会自动执行数据库迁移。如需手动执行：

```bash
# 进入后端容器
docker-compose exec backend sh

# 运行迁移（在代码中已配置AutoMigrate）
# 或使用golang-migrate工具
```

### 备份数据

```bash
# 备份PostgreSQL
docker-compose exec postgres pg_dump -U admin user_system > backup.sql

# 恢复PostgreSQL
docker-compose exec -T postgres psql -U admin user_system < backup.sql
```

## 监控和日志

### 查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f backend
docker-compose logs -f frontend
```

### 健康检查

```bash
# 检查后端健康状态
curl http://localhost:8080/health

# 检查数据库连接
docker-compose exec postgres pg_isready -U admin

# 检查Redis连接
docker-compose exec redis redis-cli -a redis123 ping
```

## 常见问题

### 端口冲突

如果端口被占用，修改 `docker-compose.yml` 中的端口映射：

```yaml
services:
  backend:
    ports:
      - "8081:8080"  # 修改为其他端口
```

### 数据库连接失败

检查数据库是否启动：

```bash
docker-compose ps
docker-compose logs postgres
```

### 前端无法访问后端

检查CORS配置：

```env
CORS_ORIGINS=http://localhost:3000,http://your-frontend-url
```

## 性能优化

### 数据库优化

1. 创建索引
2. 配置连接池
3. 定期清理日志

### Redis优化

1. 设置最大内存
2. 配置淘汰策略
3. 启用持久化

### 应用优化

1. 启用Gzip压缩
2. 配置静态资源缓存
3. 使用CDN加速前端资源

## 安全建议

1. **修改默认密码**
   - 数据库密码
   - Redis密码
   - JWT密钥

2. **配置HTTPS**
   - 使用Let's Encrypt获取SSL证书
   - 配置Nginx反向代理

3. **限制访问**
   - 配置防火墙规则
   - 限制数据库和Redis的外部访问

4. **定期更新**
   - 及时更新依赖包
   - 关注安全公告

5. **备份策略**
   - 定期备份数据库
   - 备份配置文件
   - 测试恢复流程
