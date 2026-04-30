# 系统架构说明

## 🎯 系统定位

这是一个 **SSO 单点登录系统**，主要功能是：
- 为普通用户提供统一的身份认证
- 为第三方应用提供 OAuth 2.0/OIDC 接入
- 为管理员提供用户和应用管理功能

## 👥 用户角色

### 1. 普通用户

**使用场景**：需要登录到第三方应用的用户

**访问路径**：
```
注册 → /register
登录 → /login
登录成功 → /profile (用户中心)
```

**用户中心功能**：
- 查看个人信息
- 查看可访问的应用列表
- 点击应用图标，使用 SSO 登录到第三方应用
- 可选：进入管理后台（如果有权限）

**典型流程**：
1. 用户在 `/login` 登录
2. 跳转到 `/profile` (用户中心)
3. 选择要访问的应用
4. 系统生成授权码，重定向到第三方应用
5. 第三方应用使用授权码换取 access_token
6. 用户成功登录到第三方应用

### 2. 管理员

**使用场景**：管理用户、角色、权限、SSO 应用

**访问路径**：
```
登录 → /login
登录成功 → /profile (用户中心)
点击"管理后台" → /dashboard
```

**管理后台功能**：
- `/dashboard/users` - 用户管理（增删改查）
- `/dashboard/roles` - 角色管理
- `/dashboard/permissions` - 权限管理
- `/dashboard/applications` - SSO 应用管理
- `/dashboard/audit-logs` - 审计日志

## 🔄 认证流程

### 普通用户登录流程

```
1. 访问 /login
2. 输入邮箱和密码
3. 后端验证成功，返回 JWT
4. 前端存储 token
5. 重定向到 /profile
6. 显示可访问的应用列表
7. 用户选择应用
8. OAuth 2.0 授权流程
9. 重定向到第三方应用
```

### 管理员访问后台流程

```
1. 访问 /login
2. 输入管理员账号密码
3. 后端验证成功，返回 JWT
4. 前端存储 token
5. 重定向到 /profile
6. 点击"管理后台"按钮
7. 后端验证管理员权限
8. 进入 /dashboard
```

## 🏗️ 页面结构

```
/                          → 自动跳转到 /login
/login                     → 普通用户/管理员登录页
/register                  → 普通用户注册页
/profile                   → 用户中心（所有用户登录后的首页）
  └─ 显示个人信息
  └─ 显示可访问的应用列表
  └─ 提供管理后台入口（如果有权限）
/dashboard                 → 管理后台（需要管理员权限）
  └─ /dashboard/users      → 用户管理
  └─ /dashboard/roles      → 角色管理
  └─ /dashboard/permissions → 权限管理
  └─ /dashboard/applications → SSO 应用管理
  └─ /dashboard/audit-logs  → 审计日志
```

## 🚀 第三方应用接入流程

### 1. 管理员注册应用

```
1. 登录到管理后台
2. 进入"SSO 应用管理"
3. 填写应用信息：
   - 应用名称
   - 回调地址 (redirect_uri)
   - 权限范围 (scopes)
4. 系统生成 client_id 和 client_secret
```

### 2. 第三方应用集成 OAuth 2.0

```
步骤 1: 用户点击"使用 SSO 登录"
步骤 2: 重定向到授权端点
  GET /oauth/authorize?
    client_id=xxx&
    response_type=code&
    redirect_uri=https://app.com/callback&
    scope=user.profile

步骤 3: 用户确认授权

步骤 4: 系统生成授权码，重定向回应用
  https://app.com/callback?code=AUTHORIZATION_CODE

步骤 5: 应用使用授权码换取 token
  POST /oauth/token
  {
    "grant_type": "authorization_code",
    "code": "AUTHORIZATION_CODE",
    "client_id": "xxx",
    "client_secret": "xxx",
    "redirect_uri": "https://app.com/callback"
  }

步骤 6: 返回 access_token
  {
    "access_token": "xxx",
    "refresh_token": "xxx",
    "expires_in": 3600
  }

步骤 7: 应用使用 access_token 获取用户信息
  GET /oauth/userinfo
  Authorization: Bearer xxx

步骤 8: 返回用户信息
  {
    "id": 123,
    "username": "user",
    "email": "user@example.com"
  }
```

## 🔐 权限控制

### 前端路由守卫
- `/profile` - 需要登录
- `/dashboard/*` - 需要登录 + 管理员权限

### 后端 API 权限
- `/api/v1/auth/*` - 公开接口
- `/api/v1/users` - 需要管理员权限
- `/api/v1/roles` - 需要管理员权限
- `/api/v1/oauth/*` - 需要 client 认证

## 📊 数据库表关系

```
users (用户表)
  ↓ 1:N
user_roles (用户角色关联)
  ↓ N:1
roles (角色表)
  ↓ 1:N
role_permissions (角色权限关联)
  ↓ N:1
permissions (权限表)

oauth_applications (SSO应用表)
oauth_tokens (OAuth令牌表)
audit_logs (审计日志表)
```

## 🎨 UI 设计原则

### 用户中心 (/profile)
- **简洁**：只显示必要信息
- **导向**：突出应用列表，引导用户访问应用
- **入口**：提供管理后台入口（小按钮，不突兀）

### 管理后台 (/dashboard)
- **专业**：表格、筛选、分页
- **高效**：批量操作、快捷操作
- **安全**：权限提示、操作确认

## 📝 待实现功能

### 高优先级
- [ ] OAuth 2.0 授权端点
- [ ] OAuth 2.0 Token 端点
- [ ] 用户信息端点
- [ ] 应用列表展示
- [ ] 管理员权限验证

### 中优先级
- [ ] 用户授权确认页面
- [ ] 应用授权管理（用户可以撤销授权）
- [ ] 用户头像上传
- [ ] 密码修改功能

### 低优先级
- [ ] 手机号注册/登录
- [ ] 第三方登录（微信、GitHub 等）
- [ ] 多因素认证（MFA）
- [ ] 密码找回功能
