#!/bin/bash

# 停止开发环境服务 (macOS)

echo "🛑 停止开发环境..."

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

stopped_any=false

# 停止后端进程（通过端口8080）
BACKEND_PID=$(lsof -ti:8080 2>/dev/null || true)
if [ -n "$BACKEND_PID" ]; then
    echo "📦 停止后端服务 (PID: $BACKEND_PID)..."
    kill -9 $BACKEND_PID 2>/dev/null || true
    echo "   ✅ 后端已停止"
    stopped_any=true
else
    echo "   ⚠️  后端进程不存在"
fi

# 清理所有Go相关进程
if pgrep -f "go run cmd/server/main.go" > /dev/null; then
    echo "📦 停止Go开发进程..."
    pkill -9 -f "go run cmd/server/main.go" 2>/dev/null || true
    echo "   ✅ Go进程已停止"
    stopped_any=true
fi

# 清理可能的后台Go进程
if pgrep -f "cmd/server/main.go" > /dev/null; then
    echo "📦 停止后台服务器进程..."
    pkill -9 -f "cmd/server/main.go" 2>/dev/null || true
    echo "   ✅ 后台服务器进程已停止"
    stopped_any=true
fi

# 停止前端进程（通过端口3000和3001）
FRONTEND_PID_3000=$(lsof -ti:3000 2>/dev/null || true)
FRONTEND_PID_3001=$(lsof -ti:3001 2>/dev/null || true)

if [ -n "$FRONTEND_PID_3000" ]; then
    echo "🎨 停止前端服务 (PID: $FRONTEND_PID_3000)..."
    kill -9 $FRONTEND_PID_3000 2>/dev/null || true
    echo "   ✅ 前端(3000)已停止"
    stopped_any=true
fi

if [ -n "$FRONTEND_PID_3001" ]; then
    echo "🎨 停止前端服务 (PID: $FRONTEND_PID_3001)..."
    kill -9 $FRONTEND_PID_3001 2>/dev/null || true
    echo "   ✅ 前端(3001)已停止"
    stopped_any=true
fi

# 清理npm/node进程
if pgrep -f "npm run dev" > /dev/null; then
    echo "🎨 停止npm开发进程..."
    pkill -9 -f "npm run dev" 2>/dev/null || true
    echo "   ✅ npm进程已停止"
    stopped_any=true
fi

# 清理Next.js进程
if pgrep -f "next dev" > /dev/null; then
    echo "🎨 停止Next.js进程..."
    pkill -9 -f "next dev" 2>/dev/null || true
    echo "   ✅ Next.js进程已停止"
    stopped_any=true
fi

# 清理PID文件
if [ -f "backend/logs/backend.pid" ]; then
    rm backend/logs/backend.pid 2>/dev/null || true
fi

if [ "$stopped_any" = true ]; then
    echo "   🧹 已清理所有相关进程"
else
    echo "   ℹ️  没有运行的开发进程"
fi

echo ""
echo "✅ 开发环境已停止"