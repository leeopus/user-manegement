#!/bin/bash

echo "=== 密码重置功能完整验证 ==="
echo ""

# 1. 测试发送重置邮件
echo "📧 1. 发送密码重置邮件..."
curl -s -X POST http://localhost:8080/api/v1/auth/password/reset-request \
  -H "Content-Type: application/json" \
  -d '{"email": "1072308180@qq.com"}' | jq .

echo ""
echo "⏳ 请检查邮箱 1072308180@qq.com 获取重置链接"
echo "📋 复制链接格式: http://localhost:3000/zh/reset-password?token=xxxxx"
echo ""

# 等待用户提供链接
read -p "粘贴从邮箱复制的重置链接 (或按Enter使用数据库中的最新token): " USER_LINK

if [ -z "$USER_LINK" ]; then
    echo "🔍 使用数据库中的最新token..."
    TOKEN=$(psql -h 127.0.0.1 -U admin -d user_system -t -c "SELECT token FROM password_reset_tokens WHERE email='1072308180@qq.com' AND used=false ORDER BY created_at DESC LIMIT 1;" 2>/dev/null | tr -d ' ')
    USER_LINK="http://localhost:3000/zh/reset-password?token=$TOKEN"
fi

echo "🔗 使用链接: $USER_LINK"
echo ""

# 从链接中提取token
TOKEN=$(echo "$USER_LINK" | grep -o 'token=[^&]*' | cut -d'=' -f2)

if [ -z "$TOKEN" ]; then
    echo "❌ 无法从链接中提取token"
    exit 1
fi

echo "✅ Token: ${TOKEN:0:20}..."
echo ""

# 2. 验证token
echo "🔐 2. 验证重置token..."
VALID_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/password/validate-token \
  -H "Content-Type: application/json" \
  -d "{\"token\": \"$TOKEN\"}")

echo "$VALID_RESPONSE" | jq .

if echo "$VALID_RESPONSE" | grep -q '"valid":true'; then
    echo "✅ Token验证成功"
else
    echo "❌ Token验证失败"
    exit 1
fi

echo ""

# 3. 重置密码
echo "🔑 3. 重置密码..."
RESET_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/password/reset \
  -H "Content-Type: application/json" \
  -d "{\"token\": \"$TOKEN\", \"new_password\": \"NewPassword123!\"}")

echo "$RESET_RESPONSE" | jq .

if echo "$RESET_RESPONSE" | grep -q '"success":true'; then
    echo "✅ 密码重置成功"
    echo ""
    echo "🎉 完整流程验证成功！"
    echo "📝 新密码: NewPassword123!"
    echo "🔗 登录页面: http://localhost:3000/zh/login"
else
    echo "❌ 密码重置失败"
    exit 1
fi

echo ""
echo "=== 验证完成 ==="