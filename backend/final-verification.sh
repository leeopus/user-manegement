#!/bin/bash

echo "🔧 密码重置功能完整修复和验证"
echo "===================================="
echo ""

# 1. 检查服务状态
echo "1️⃣ 检查服务状态..."
BACKEND_HEALTH=$(curl -s http://localhost:8080/health)
FRONTEND_CHECK=$(curl -s http://localhost:3000 | head -c 100)

if [ "$BACKEND_HEALTH" = '{"status":"ok"}' ]; then
    echo "   ✅ 后端服务正常"
else
    echo "   ❌ 后端服务异常"
    exit 1
fi

if echo "$FRONTEND_CHECK" | grep -q "DOCTYPE"; then
    echo "   ✅ 前端服务正常"
else
    echo "   ❌ 前端服务异常"
    exit 1
fi

# 2. 检查SMTP配置
echo ""
echo "2️⃣ 检查SMTP配置..."
SMTP_LOG=$(tail -50 /Users/lee/my/user-manegement/backend/logs/backend-dev.log | grep "📧.*SMTP" | tail -1)
if [ -n "$SMTP_LOG" ]; then
    echo "   ✅ SMTP服务已启用"
    echo "   $SMTP_LOG"
else
    echo "   ⚠️  SMTP服务可能未启用"
fi

# 3. 测试通用重置页面
echo ""
echo "3️⃣ 测试通用重置页面..."
RESET_PAGE=$(curl -s "http://localhost:3000/reset-password?token=test123")
if echo "$RESET_PAGE" | grep -q "正在跳转到密码重置页面"; then
    echo "   ✅ 通用重置页面正常"
    echo "   🔗 http://localhost:3000/reset-password?token=xxx"
else
    echo "   ⚠️  通用重置页面可能有问题"
fi

# 4. 测试发送重置邮件
echo ""
echo "4️⃣ 发送密码重置邮件..."
RESET_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/password/reset-request \
  -H "Content-Type: application/json" \
  -d '{"email": "1072308180@qq.com"}')

if echo "$RESET_RESPONSE" | grep -q '"success":true'; then
    echo "   ✅ 重置邮件发送成功"
    echo "   📬 收件人: 1072308180@qq.com"
else
    echo "   ❌ 重置邮件发送失败"
    echo "$RESET_RESPONSE"
fi

# 5. 获取测试Token
echo ""
echo "5️⃣ 获取测试Token..."
TEST_TOKEN=$(psql -h 127.0.0.1 -U admin -d user_system -t -c \
    "SELECT token FROM password_reset_tokens WHERE email='1072308180@qq.com' AND used=false ORDER BY created_at DESC LIMIT 1;" 2>/dev/null | tr -d ' ')

if [ -n "$TEST_TOKEN" ]; then
    echo "   ✅ 获取Token成功: ${TEST_TOKEN:0:20}..."
    TEST_URL="http://localhost:3000/reset-password?token=$TEST_TOKEN"
    echo "   🔗 测试链接: $TEST_URL"
else
    echo "   ⚠️  未找到有效Token"
fi

# 6. 测试Token验证API
echo ""
echo "6️⃣ 测试Token验证API..."
if [ -n "$TEST_TOKEN" ]; then
    VALIDATE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/password/validate-token \
      -H "Content-Type: application/json" \
      -d "{\"token\": \"$TEST_TOKEN\"}")

    if echo "$VALIDATE_RESPONSE" | grep -q '"valid":true'; then
        echo "   ✅ Token验证正常"
    else
        echo "   ❌ Token验证失败"
        echo "$VALIDATE_RESPONSE"
    fi
fi

# 7. 显示改进总结
echo ""
echo "7️⃣ 功能改进总结..."
echo "   ✅ URL硬编码问题 → 使用前端配置"
echo "   ✅ 语言路径硬编码 → 自动检测用户语言"
echo "   ✅ 重置页面体验差 → 完整重新设计"
echo "   ✅ 缺少密码强度反馈 → 实时强度显示"
echo "   ✅ 密码匹配提示 → 实时匹配状态"
echo "   ✅ 视觉反馈不足 → 改进UI/UX设计"

echo ""
echo "====================================="
echo "📋 使用方法:"
echo "   1. 检查邮箱: 1072308180@qq.com"
echo "   2. 点击链接: http://localhost:3000/reset-password?token=xxx"
echo "   3. 自动重定向到: /zh/reset-password (中文)"
echo "   4. 或重定向到: /en/reset-password (英文)"
echo "   5. 输入新密码完成重置"
echo ""
echo "🎯 关键改进:"
echo "   • 邮件链接不再包含 /zh/ 或 /en/"
echo "   • 自动检测用户浏览器语言设置"
echo "   • 生产环境只需修改 FRONTEND_URL 配置"
echo "   • 密码强度实时显示（4个安全指标）"
echo "   • 密码匹配实时反馈"
echo "   • 更美观的页面设计"
echo "   • 更好的错误处理"
echo "====================================="