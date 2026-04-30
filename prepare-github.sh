#!/bin/bash

# 准备提交到 GitHub 的脚本

set -e

echo "🔍 检查项目状态..."
echo ""

# 检查是否有不应该提交的文件被追踪
echo "1️⃣  检查是否有编译结果被追踪..."
if git ls-files | grep -qE "backend/bin|frontend/.next|frontend/node_modules"; then
  echo "⚠️  发现编译结果被追踪，正在移除..."
  git ls-files | grep -E "backend/bin|frontend/.next|frontend/node_modules" | xargs git rm -r --cached 2>/dev/null || true
  echo "✅ 已移除编译结果的追踪"
else
  echo "✅ 没有编译结果被追踪"
fi

echo ""
echo "2️⃣  检查 .gitignore 文件..."
if [ ! -f .gitignore ]; then
  echo "❌ .gitignore 文件不存在"
  exit 1
else
  echo "✅ .gitignore 文件存在"
fi

echo ""
echo "3️⃣  检查环境变量文件..."
if git ls-files | grep -q "^\.env$" || git ls-files | grep -q "backend/\.env$"; then
  echo "⚠️  警告：.env 文件被追踪，建议从 git 中移除"
  echo "    运行: git rm --cached .env backend/.env"
else
  echo "✅ 没有敏感环境变量文件被追踪"
fi

echo ""
echo "4️⃣  添加所有新文件..."
git add .

echo ""
echo "5️⃣  查看将要提交的更改..."
git status --short

echo ""
echo "📊 统计信息："
echo "   新增文件: $(git status --short | grep "^A" | wc -l)"
echo "   修改文件: $(git status --short | grep "^M" | wc -l)"
echo "   删除文件: $(git status --short | grep "^D" | wc -l)"

echo ""
echo "✅ 准备完成！"
echo ""
echo "下一步操作："
echo "1. 查看将要提交的更改: git status"
echo "2. 如果确认无误，提交代码:"
echo "   git commit -m 'feat: 完善用户管理系统'"
echo "3. 推送到 GitHub:"
echo "   git remote add origin https://github.com/your-username/repo.git"
echo "   git push -u origin master"
