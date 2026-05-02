#!/bin/bash

# 检查并安装 macOS 开发环境依赖

set -e

echo "🔍 检查开发环境依赖..."
echo ""

# 检查 Homebrew
echo "1️⃣  检查 Homebrew..."
if ! command -v brew &> /dev/null; then
    echo "   ❌ Homebrew 未安装"
    echo "   请访问 https://brew.sh/ 安装 Homebrew"
    echo "   安装命令: /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
    exit 1
else
    echo "   ✅ Homebrew 已安装"
fi

# 检查 Go
echo ""
echo "2️⃣  检查 Go..."
if ! command -v go &> /dev/null; then
    echo "   ❌ Go 未安装"
    echo "   正在安装 Go..."
    brew install go
    echo "   ✅ Go 已安装"
else
    GO_VERSION=$(go version | awk '{print $3}')
    echo "   ✅ Go 已安装 ($GO_VERSION)"
fi

# 检查 Node.js
echo ""
echo "3️⃣  检查 Node.js..."
if ! command -v node &> /dev/null; then
    echo "   ❌ Node.js 未安装"
    echo "   正在安装 Node.js..."
    brew install node
    echo "   ✅ Node.js 已安装"
else
    NODE_VERSION=$(node -v)
    echo "   ✅ Node.js 已安装 ($NODE_VERSION)"
fi

# 检查 PostgreSQL
echo ""
echo "4️⃣  检查 PostgreSQL..."
if ! command -v psql &> /dev/null; then
    echo "   ❌ PostgreSQL 未安装"
    echo "   正在安装 PostgreSQL..."
    brew install postgresql@14
    echo "   ✅ PostgreSQL 已安装"
    echo "   正在启动 PostgreSQL 服务..."
    brew services start postgresql@14
    sleep 3
else
    PSQL_VERSION=$(psql --version | awk '{print $3}')
    echo "   ✅ PostgreSQL 已安装 ($PSQL_VERSION)"

    # 检查服务状态
    if brew services list | grep postgresql | grep -q started; then
        echo "   ✅ PostgreSQL 服务正在运行"
    else
        echo "   ⚠️  PostgreSQL 服务未运行，正在启动..."
        brew services start postgresql
        sleep 2
    fi
fi

# 检查 Redis
echo ""
echo "5️⃣  检查 Redis..."
if ! command -v redis-cli &> /dev/null; then
    echo "   ❌ Redis 未安装"
    echo "   正在安装 Redis..."
    brew install redis
    echo "   ✅ Redis 已安装"
    echo "   正在启动 Redis 服务..."
    brew services start redis
    sleep 3
else
    REDIS_VERSION=$(redis-cli --version | awk '{print $2}')
    echo "   ✅ Redis 已安装 ($REDIS_VERSION)"

    # 检查服务状态
    if brew services list | grep redis | grep -q started; then
        echo "   ✅ Redis 服务正在运行"
    else
        echo "   ⚠️  Redis 服务未运行，正在启动..."
        brew services start redis
        sleep 2
    fi
fi

# 检查环境变量文件
echo ""
echo "6️⃣  检查环境配置..."
if [[ ! -f ".env" ]]; then
    echo "   ⚠️  未找到 .env 文件，正在创建..."
    cp .env.example .env
    echo "   ✅ 已创建 .env 文件"
    echo "   📝 请根据需要修改 .env 文件中的配置"
else
    echo "   ✅ .env 文件已存在"
fi

# 检查后端依赖
echo ""
echo "7️⃣  检查后端依赖..."
if [[ -f "backend/go.mod" ]]; then
    cd backend
    if ! go list -m all &> /dev/null; then
        echo "   正在安装 Go 依赖..."
        go mod download
        echo "   ✅ Go 依赖已安装"
    else
        echo "   ✅ Go 依赖已安装"
    fi
    cd ..
else
    echo "   ⚠️  未找到 backend/go.mod"
fi

# 检查前端依赖
echo ""
echo "8️⃣  检查前端依赖..."
if [[ -f "frontend/package.json" ]]; then
    cd frontend
    if [[ ! -d "node_modules" ]]; then
        echo "   正在安装 npm 依赖..."
        npm install
        echo "   ✅ npm 依赖已安装"
    else
        echo "   ✅ npm 依赖已安装"
    fi
    cd ..
else
    echo "   ⚠️  未找到 frontend/package.json"
fi

# 创建必要的目录
echo ""
echo "9️⃣  创建必要目录..."
mkdir -p backend/logs
echo "   ✅ 目录已创建"

# 设置脚本权限
echo ""
echo "🔟 设置脚本权限..."
chmod +x dev-macos.sh
chmod +x stop-dev.sh
chmod +x check-deps.sh
echo "   ✅ 权限已设置"

echo ""
echo "🎉 所有依赖检查完成！"
echo ""
echo "现在可以运行以下命令启动开发环境："
echo "   ./dev-macos.sh"
echo ""
echo "如需停止开发环境，运行："
echo "   ./stop-dev.sh"
echo ""