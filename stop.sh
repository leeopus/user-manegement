#!/bin/bash

# 停止所有服务

echo "🛑 停止所有服务..."

# 停止 Docker Compose 服务
if [ -f "/root/sys/docker-compose.yml" ]; then
    echo "停止 Docker 服务..."
    cd /root/sys
    docker-compose down 2>/dev/null || true
fi

# 停止后端进程
echo "停止后端进程..."
pkill -f "go run cmd/server/main.go" || true
pkill -f "backend/bin/server" || true

# 停止前端进程
echo "停止前端进程..."
pkill -f "next dev" || true

echo "✅ 所有服务已停止"
