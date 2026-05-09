#!/bin/bash

# macOS 开发模式启动脚本
# 所有服务本地运行，支持热重载

set -e

echo "🚀 启动开发环境 (macOS)..."

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 检测操作系统
OS="$(uname -s)"
if [[ "$OS" != "Darwin" ]]; then
    echo "❌ 此脚本仅适用于 macOS，请使用原版 dev.sh"
    exit 1
fi

# 检查 Homebrew
if ! command -v brew &> /dev/null; then
    echo "❌ 未找到 Homebrew，请先安装: https://brew.sh/"
    exit 1
fi

# 检查 PostgreSQL
echo "🔍 检查 PostgreSQL..."
if ! command -v psql &> /dev/null; then
    echo "❌ PostgreSQL 未安装"
    echo "   请运行: brew install postgresql@14"
    echo "   或: brew install postgresql"
    exit 1
fi

# 检查 PostgreSQL 服务状态
if ! brew services list | grep postgresql | grep -q started; then
    echo "⚠️  PostgreSQL 未运行，正在启动..."
    brew services start postgresql
    sleep 2
fi

# 检查 Redis
echo "🔍 检查 Redis..."
if ! command -v redis-cli &> /dev/null; then
    echo "❌ Redis 未安装"
    echo "   请运行: brew install redis"
    exit 1
fi

# 检查 Redis 服务状态
if ! brew services list | grep redis | grep -q started; then
    echo "⚠️  Redis 未运行，正在启动..."
    brew services start redis
    sleep 2
fi

# 检查 Go
echo "🔍 检查 Go..."
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安装"
    echo "   请运行: brew install go"
    exit 1
fi

# 检查 Node.js
echo "🔍 检查 Node.js..."
if ! command -v node &> /dev/null; then
    echo "❌ Node.js 未安装"
    echo "   请运行: brew install node"
    exit 1
fi

# 检查环境变量文件
if [[ ! -f ".env" ]]; then
    echo "⚠️  未找到 .env 文件，正在创建..."
    cp .env.example .env
    echo "   ✅ 已创建 .env 文件，请根据需要修改配置"
fi

# 清理旧进程：杀 PID 文件记录的进程 + 清理端口残留
echo "🧹 清理旧进程..."
cleanup_port() {
    local port=$1
    local pids=$(lsof -ti :$port -sTCP:LISTEN 2>/dev/null || true)
    if [ -n "$pids" ]; then
        echo "   清理端口 :$port 上的残留进程..."
        echo "$pids" | xargs kill -9 2>/dev/null || true
    fi
}

# 先尝试优雅停止 PID 文件记录的进程
for pidfile in backend/logs/backend.pid; do
    if [ -f "$pidfile" ]; then
        pid=$(cat "$pidfile")
        if kill -0 "$pid" 2>/dev/null; then
            kill -- -"$pid" 2>/dev/null || kill "$pid" 2>/dev/null || true
        fi
        rm -f "$pidfile"
    fi
done

# pkill go run 相关进程
pkill -f "go run cmd/server/main.go" 2>/dev/null || true

sleep 1

# 强制清理端口残留（go run 的子进程可能存活）
cleanup_port 8080
cleanup_port 3000

# 启动后端（后台）
echo "📦 启动后端服务..."
cd "$SCRIPT_DIR/backend"

# 创建日志目录
mkdir -p logs

# 复制 .env 文件到后端目录（如果不存在）
if [[ ! -f ".env" ]]; then
    if [[ -f "$SCRIPT_DIR/.env" ]]; then
        cp "$SCRIPT_DIR/.env" .env
        echo "   ✅ 已复制 .env 文件到后端目录"
    else
        echo "   ⚠️  未找到 .env 文件"
    fi
fi

# 启动后端
nohup go run cmd/server/main.go > logs/backend-dev.log 2>&1 &
BACKEND_PID=$!
echo "   后端 PID: $BACKEND_PID"

# 保存 PID 到文件
echo $BACKEND_PID > logs/backend.pid

# 等待后端启动
echo "   等待后端启动..."
sleep 5

# 检查后端是否启动成功
if ! kill -0 $BACKEND_PID 2>/dev/null; then
    echo "❌ 后端启动失败，请查看日志: logs/backend-dev.log"
    tail -n 20 logs/backend-dev.log
    exit 1
fi

# 检查后端健康状态
echo "   检查后端健康状态..."
for i in {1..10}; do
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo "   ✅ 后端启动成功"
        break
    fi
    if [[ $i -eq 10 ]]; then
        echo "   ⚠️  后端可能未完全启动，请检查日志"
    fi
    sleep 1
done

# 启动前端（前台，支持热重载）
echo "🎨 启动前端服务..."
cd "$SCRIPT_DIR/frontend"
export BACKEND_URL=http://localhost:8080

# 检查 node_modules
if [[ ! -d "node_modules" ]]; then
    echo "   安装前端依赖..."
    npm install
fi

# 添加 node_modules/.bin 到 PATH（确保 npm scripts 能找到本地二进制文件）
export PATH="$PWD/node_modules/.bin:$PATH"

echo ""
echo "✅ 开发环境启动完成！"
echo ""
echo "   前端地址: http://localhost:3000"
echo "   后端地址: http://localhost:8080"
echo "   后端日志: $SCRIPT_DIR/backend/logs/backend-dev.log"
echo ""
echo "   按 Ctrl+C 停止前端服务"
echo "   后端 PID: $BACKEND_PID (会自动停止)"
echo ""
echo "   如需单独停止后端，请运行: ./stop-dev.sh"
echo ""

# 设置 trap，确保脚本退出时清理
trap 'echo ""; echo "🛑 正在停止服务..."; kill $BACKEND_PID 2>/dev/null || true; echo "   ✅ 后端已停止"; exit 0' INT TERM

# 启动前端
npm run dev