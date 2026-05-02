#!/bin/bash

# 开发模式启动脚本
# 所有服务本地运行，支持热重载

set -e

echo "🚀 启动开发环境..."

# 检查 PostgreSQL
if ! systemctl is-active --quiet postgresql; then
    echo "⚠️  PostgreSQL 未运行，正在启动..."
    sudo systemctl start postgresql
fi

# 检查 Redis
if ! systemctl is-active --quiet redis; then
    echo "⚠️  Redis 未运行，正在启动..."
    sudo systemctl start redis
fi

# 启动后端（后台）
echo "📦 启动后端服务..."
cd /root/user-management/backend
go run cmd/server/main.go > /tmp/backend-dev.log 2>&1 &
BACKEND_PID=$!
echo "   后端 PID: $BACKEND_PID"

# 等待后端启动
sleep 3

# 启动前端（前台，支持热重载）
echo "🎨 启动前端服务..."
cd /root/user-management/frontend
echo ""
echo "✅ 开发环境启动完成！"
echo ""
echo "   前端地址: http://localhost:3000"
echo "   后端地址: http://localhost:8080"
echo ""
echo "   按 Ctrl+C 停止前端服务"
echo "   后端 PID: $BACKEND_PID (需要手动 kill)"
echo ""

npm run dev
