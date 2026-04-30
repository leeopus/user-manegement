#!/bin/bash

# 生产模式启动脚本
# 使用 Docker Compose 部署所有服务

set -e

COMPOSE_FILE="/root/sys/docker-compose.yml"

echo "🚀 启动生产环境..."

# 检查 docker-compose.yml 是否存在
if [ ! -f "$COMPOSE_FILE" ]; then
    echo "❌ 错误: $COMPOSE_FILE 不存在"
    exit 1
fi

# 检查 .env 文件
if [ ! -f "/root/sys/.env.production" ]; then
    echo "⚠️  未找到 .env.production，创建默认配置..."
    cat > /root/sys/.env.production << EOF
# PostgreSQL
POSTGRES_PASSWORD=admin123

# Redis
REDIS_PASSWORD=redis123

# 后端配置
DATABASE_URL=postgres://admin:admin123@postgres:5432/user_system?sslmode=disable
REDIS_URL=redis://:redis123@redis:6379/0
JWT_SECRET=wXINUMfv7qfm5lthN9L3ch7SdgwlzJraxS7LAqjRUzI=
JWT_EXPIRATION=1h
REFRESH_TOKEN_EXPIRATION=720h
SERVER_PORT=8080
GIN_MODE=release
FRONTEND_URL=http://106.15.3.98:3000
CORS_ORIGINS=http://106.15.3.98:3000,http://106.15.3.98

# 前端配置
NEXT_PUBLIC_API_URL=http://106.15.3.98:8080
EOF
fi

# 停止旧容器
echo "🛑 停止旧容器..."
cd /root/sys
docker-compose -f "$COMPOSE_FILE" down 2>/dev/null || true

# 构建镜像
echo "🔨 构建 Docker 镜像..."
docker-compose -f "$COMPOSE_FILE" build

# 启动服务
echo "🚀 启动服务..."
docker-compose -f "$COMPOSE_FILE" up -d

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 10

# 检查服务状态
echo ""
echo "📊 服务状态:"
docker-compose -f "$COMPOSE_FILE" ps

echo ""
echo "✅ 生产环境启动完成！"
echo ""
echo "   前端地址: http://106.15.3.98:3000"
echo "   后端地址: http://106.15.3.98:8080"
echo ""
echo "   查看日志: docker-compose -f $COMPOSE_FILE logs -f"
echo "   停止服务: docker-compose -f $COMPOSE_FILE down"
echo ""
