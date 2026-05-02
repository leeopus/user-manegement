#!/bin/bash

echo "🔍 系统架构深度审核报告"
echo "===================================="
echo ""

# 1. 检查数据库连接和事务处理
echo "1️⃣ 数据库层审核..."
echo "   🔍 检查数据库连接池配置..."
grep -r "sql.Open\|SetMaxIdleConns\|SetMaxOpenConns" /Users/lee/my/user-manegement/backend --include="*.go" | head -3

echo ""
echo "   🔍 检查事务处理..."
grep -r "db.Begin\|Transaction\|Commit\|Rollback" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l
echo "   找到 $(grep -r "db.Begin\|Transaction" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l | tr -d ' ') 个事务处理"

echo ""
echo "   🔍 检查SQL注入风险..."
grep -r "fmt.Sprintf.*SELECT\|fmt.Sprintf.*UPDATE\|fmt.Sprintf.*DELETE" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l
echo "   发现 $(grep -r "fmt.Sprintf.*SQL" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l | tr -d ' ') 处潜在的SQL注入风险"

# 2. 检查认证和授权
echo ""
echo "2️⃣ 认证授权审核..."
echo "   🔍 检查JWT密钥配置..."
grep -r "JWT_SECRET\|jwt.*secret" /Users/lee/my/user-manegement/backend/.env 2>/dev/null || echo "   ⚠️  未找到JWT配置"

echo ""
echo "   🔍 检查token刷新机制..."
grep -r "RefreshToken\|refresh.*token" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l
echo "   找到 $(grep -r "RefreshToken" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l | tr -d ' ') 个token刷新相关代码"

echo ""
echo "   🔍 检查密码存储..."
grep -r "bcrypt\|password.*hash\|HashPassword" /Users/lee/my/user-manegement/backend --include="*.go" | head -3

# 3. 检查错误处理
echo ""
echo "3️⃣ 错误处理审核..."
echo "   🔍 检查全局错误处理..."
grep -r "panic\|recover" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l
echo "   发现 $(grep -r "panic\|recover" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l | tr -d ' ') 处panic/recover代码"

echo ""
echo "   🔍 检查错误日志..."
grep -r "log.*Error\|fmt.Printf.*error" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l
echo "   发现 $(grep -r "log.*Error\|fmt.Printf.*error" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l | tr -d ' ') 处错误日志"

# 4. 检查并发和竞态条件
echo ""
echo "4️⃣ 并发安全审核..."
echo "   🔍 检查互斥锁和并发控制..."
grep -r "sync.Mutex\|sync.RWMutex\|channel\|goroutine" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l
echo "   发现 $(grep -r "sync.Mutex\|sync.RWMutex\|channel" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l | tr -d ' ') 处并发控制代码"

# 5. 检查敏感信息泄露
echo ""
echo "5️⃣ 信息安全审核..."
echo "   🔍 检查敏感信息日志..."
grep -r "password\|token\|secret.*log\|log.*password" /Users/lee/my/user-manegement/backend --include="*.go" | grep -v "PasswordHash\|PasswordReset\|reset.*password" | head -3

echo ""
echo "   🔍 检查环境变量使用..."
grep -r "os.Getenv" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l
echo "   发现 $(grep -r "os.Getenv" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l | tr -d ' ') 处环境变量使用"

# 6. 检查前端安全性
echo ""
echo "6️⃣ 前端安全审核..."
echo "   🔍 检查XSS防护..."
grep -r "dangerouslySetInnerHTML\|innerHTML" /Users/lee/my/user-manegement/frontend/src --include="*.tsx" --include="*.ts" | wc -l
echo "   发现 $(grep -r "dangerouslySetInnerHTML\|innerHTML" /Users/lee/my/user-manegement/frontend/src --include="*.tsx" --include="*.ts" | wc -l | tr -d ' ') 处潜在的XSS风险"

echo ""
echo "   🔍 检查localStorage使用..."
grep -r "localStorage" /Users/lee/my/user-manegement/frontend/src --include="*.tsx" --include="*.ts" | wc -l
echo "   发现 $(grep -r "localStorage" /Users/lee/my/user-manegement/frontend/src --include="*.tsx" --include="*.ts" | wc -l | tr -d ' ') 处localStorage使用"

# 7. 检查API设计
echo ""
echo "7️⃣ API设计审核..."
echo "   🔍 检查API版本控制..."
grep -r "/api/v[0-9]" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l
echo "   发现 $(grep -r "/api/v[0-9]" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l | tr -d ' ') 个版本化API端点"

echo ""
echo "   🔍 检查输入验证..."
grep -r "binding:\"required\"\|binding:\"min\|binding:\"max" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l
echo "   发现 $(grep -r "binding:\"required" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l | tr -d ' ') 处输入验证"

# 8. 检查性能问题
echo ""
echo "8️⃣ 性能优化审核..."
echo "   🔍 检查数据库查询优化..."
grep -r "Preload\|Joins\|Select.*preload" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l
echo "   发现 $(grep -r "Preload" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l | tr -d ' ') 处预加载查询"

echo ""
echo "   🔍 检查缓存机制..."
grep -r "cache\|Cache\|redis.*Set\|redis.*Get" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l
echo "   发现 $(grep -r "redis.*Set\|redis.*Get" /Users/lee/my/user-manegement/backend --include="*.go" | wc -l | tr -d ' ') 处Redis操作"

echo ""
echo "====================================="
echo "📊 审核完成，开始深度分析..."
echo "====================================="