#!/bin/bash

echo "🧹 项目文件清理报告"
echo "===================================="
echo ""

# 统计当前项目状态
echo "📊 当前项目状态分析..."
echo ""

BACKEND_SIZE=$(du -sh /Users/lee/my/user-manegement/backend 2>/dev/null | cut -f1)
FRONTEND_SIZE=$(du -sh /Users/lee/my/user-manegement/frontend 2>/dev/null | cut -f1)
NODE_MODULES_SIZE=$(du -sh /Users/lee/my/user-manegement/frontend/node_modules 2>/dev/null | cut -f1)
NEXT_SIZE=$(du -sh /Users/lee/my/user-manegement/frontend/.next 2>/dev/null | cut -f1)

echo "📁 项目大小统计:"
echo "   后端代码: ${BACKEND_SIZE:-N/A}"
echo "   前端代码: ${FRONTEND_SIZE:-N/A}"
echo "   node_modules: ${NODE_MODULES_SIZE:-N/A}"
echo "   .next缓存: ${NEXT_SIZE:-N/A}"
echo ""

echo "🔍 发现的可清理文件分类:"
echo ""

# 1. 日志文件
echo "1️⃣ 日志文件 (可安全删除)"
LOG_FILES=$(find /Users/lee/my/user-manegement -name "*.log" 2>/dev/null | wc -l | tr -d ' ')
echo "   发现 $LOG_FILES 个日志文件"
find /Users/lee/my/user-manegement -name "*.log" 2>/dev/null | head -5
echo ""

# 2. 测试脚本
echo "2️⃣ 测试脚本 (可删除)"
TEST_SCRIPTS=$(find /Users/lee/my/user-manegement -name "test-*.sh" -o -name "*test*.sh" 2>/dev/null | wc -l | tr -d ' ')
echo "   发现 $TEST_SCRIPTS 个测试脚本"
find /Users/lee/my/user-manegement -name "test-*.sh" -o -name "*test*.sh" 2>/dev/null | head -5
echo ""

# 3. 临时文档
echo "3️⃣ 临时审计文档 (建议删除)"
TEMP_DOCS=$(find /Users/lee/my/user-manegement -name "*_COMPLETE.md" -o -name "*_AUDIT.md" -o -name "*_SUMMARY.md" -o -name "IMPROVEMENT*.md" -o -name "TRANSLATION_*.md" -o -name "P0_*.md" -o -name "FINAL_*.md" 2>/dev/null | wc -l | tr -d ' ')
echo "   发现 $TEMP_DOCS 个临时文档"
echo "   建议: 保留README.md和ARCHITECTURE_REVIEW.md，删除其他"
echo ""

# 4. 重复配置
echo "4️⃣ 重复的环境配置"
DUPLICATE_ENV=$(find /Users/lee/my/user-manegement -name ".env.*" ! -name ".env" ! -name ".env.example" 2>/dev/null | wc -l | tr -d ' ')
echo "   发现 $DUPLICATE_ENV 个重复配置文件"
find /Users/lee/my/user-manegement -name ".env.*" ! -name ".env" ! -name ".env.example" 2>/dev/null | head -5
echo ""

# 5. 开发脚本
echo "5️⃣ 开发脚本 (可优化)"
DEV_SCRIPTS=$(find /Users/lee/my/user-manegement -name "*.sh" 2>/dev/null | wc -l | tr -d ' ')
echo "   发现 $DEV_SCRIPTS 个脚本文件"
echo "   建议: 只保留必要的，删除重复的"
echo ""

echo "====================================="
echo "📋 建议删除的文件列表:"
echo ""
echo "🗑️  临时审计文档 (可安全删除):"
TEMP_DOC_FILES=$(find /Users/lee/my/user-manegement -maxdepth 1 -name "*_COMPLETE.md" -o -name "*_AUDIT.md" -o -name "*_SUMMARY.md" -o -name "IMPROVEMENT*.md" -o -name "TRANSLATION_*.md" -o -name "P0_*.md" -o -name "FINAL_*.md" -o -name "PASSWORD_RESET_*.md" -o -name "I18N_*.md" -o -name "LOGIN_*.md" -o -name "CODE_*.md" -o -name "QUICK_*.md" -o -name "DEV_*.md" -o -name "SCALABILITY_*.md" 2>/dev/null)

for file in $TEMP_DOC_FILES; do
    if [ -f "$file" ]; then
        echo "   - $(basename $file)"
    fi
done
echo ""

echo "🧪 测试脚本 (可删除):"
TEST_FILES=$(find /Users/lee/my/user-manegement -maxdepth 1 -name "test-*.sh" 2>/dev/null)
for file in $TEST_FILES; do
    echo "   - $(basename $file)"
done
echo ""

echo "🗑️ 重复配置文件 (可删除):"
ENV_FILES=$(find /Users/lee/my/user-manegement -maxdepth 1 -name ".env.*" ! -name ".env" ! -name ".env.example" 2>/dev/null)
for file in $ENV_FILES; do
    echo "   - $(basename $file)"
done
echo ""

echo "📝 开发脚本 (建议保留主要脚本):"
ALL_SCRIPTS=$(find /Users/lee/my/user-manegement -maxdepth 1 -name "*.sh" 2>/dev/null)
for file in $ALL_SCRIPTS; do
    basename_file=$(basename $file)
    if [[ "$basename_file" == "stop-dev.sh" || "$basename_file" == "dev-macos.sh" ]]; then
        echo "   ✅ 保留: $basename_file"
    else
        echo "   ⚠️  可删除: $basename_file"
    fi
done
echo ""

echo "====================================="
echo "🚀 开始清理..."
echo ""

# 停止服务
echo "🛑 停止运行的服务..."
./stop-dev.sh > /dev/null 2>&1
echo "   ✅ 服务已停止"
echo ""

# 清理日志文件
echo "📝 清理日志文件..."
find /Users/lee/my/user-manegement -name "*.log" -delete 2>/dev/null
echo "   ✅ 已删除日志文件"
echo ""

# 清理临时文档
echo "📄 清理临时审计文档..."
DELETE_COUNT=0
for file in $TEMP_DOC_FILES; do
    if [ -f "$file" ]; then
        rm "$file"
        DELETE_COUNT=$((DELETE_COUNT + 1))
    fi
done
echo "   ✅ 已删除 $DELETE_COUNT 个临时文档"
echo ""

# 清理测试脚本
echo "🧪 清理测试脚本..."
TEST_DELETE_COUNT=0
for file in $TEST_FILES; do
    if [ -f "$file" ]; then
        rm "$file"
        TEST_DELETE_COUNT=$((TEST_DELETE_COUNT + 1))
    fi
done
echo "   ✅ 已删除 $TEST_DELETE_COUNT 个测试脚本"
echo ""

# 清理重复配置
echo "⚙️  清理重复配置..."
ENV_DELETE_COUNT=0
for file in $ENV_FILES; do
    if [ -f "$file" ]; then
        rm "$file"
        ENV_DELETE_COUNT=$((ENV_DELETE_COUNT + 1))
    fi
done
echo "   ✅ 已删除 $ENV_DELETE_COUNT 个重复配置"
echo ""

# 清理临时编译文件
echo "🔧 清理临时编译文件..."
rm -rf /Users/lee/my/user-manegement/backend/cmd/test-smtp 2>/dev/null
echo "   ✅ 已清理编译临时文件"
echo ""

echo "====================================="
echo "✅ 清理完成！"
echo ""
echo "📊 清理总结:"
echo "   🗑️  临时审计文档: $DELETE_COUNT 个"
echo "   🧪 测试脚本: $TEST_DELETE_COUNT 个"
echo "   ⚙️ 重复配置: $ENV_DELETE_COUNT 个"
echo "   📝 日志文件: 已清理"
echo ""
echo "🔥 保留的重要文件:"
echo "   ✅ README.md - 项目说明"
echo "   ✅ ARCHITECTURE_REVIEW.md - 架构审核报告"
echo "   ✅ .env - 环境配置"
echo "   ✅ .env.example - 配置示例"
echo "   ✅ stop-dev.sh - 停止服务脚本"
echo "   ✅ dev-macos.sh - 启动服务脚本"
echo ""
echo "📦 项目结构已优化，更加清晰整洁！"
echo "====================================="