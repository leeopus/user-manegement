# API 文档

## 基础信息

- **Base URL**: `http://localhost:8080`
- **API Prefix**: `/api/v1`
- **认证方式**: Bearer Token (JWT)

## 认证接口

### 用户注册

```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "string",
  "email": "user@example.com",
  "password": "string"
}
```

**响应:**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user": {
      "id": 1,
      "username": "string",
      "email": "user@example.com",
      "status": "active"
    }
  }
}
```

### 用户登录

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "string"
}
```

**响应:**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user": {
      "id": 1,
      "username": "string",
      "email": "user@example.com"
    },
    "access_token": "string",
    "refresh_token": "string"
  }
}
```

### 获取当前用户信息

```http
GET /api/v1/auth/me
Authorization: Bearer {access_token}
```

### 刷新Token

```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "string"
}
```

### 登出

```http
POST /api/v1/auth/logout
Authorization: Bearer {access_token}
```

## 用户管理接口

### 获取用户列表

```http
GET /api/v1/users?page=1&page_size=10
Authorization: Bearer {access_token}
```

**响应:**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "users": [...],
    "total": 100,
    "page": 1,
    "page_size": 10
  }
}
```

### 获取用户详情

```http
GET /api/v1/users/{id}
Authorization: Bearer {access_token}
```

### 创建用户

```http
POST /api/v1/users
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "username": "string",
  "email": "user@example.com",
  "password": "string"
}
```

### 更新用户

```http
PUT /api/v1/users/{id}
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "username": "string",
  "email": "user@example.com"
}
```

### 删除用户

```http
DELETE /api/v1/users/{id}
Authorization: Bearer {access_token}
```

## 角色管理接口

### 获取角色列表

```http
GET /api/v1/roles?page=1&page_size=10
Authorization: Bearer {access_token}
```

### 创建角色

```http
POST /api/v1/roles
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "name": "string",
  "code": "string",
  "description": "string"
}
```

### 更新角色

```http
PUT /api/v1/roles/{id}
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "name": "string",
  "code": "string",
  "description": "string"
}
```

### 删除角色

```http
DELETE /api/v1/roles/{id}
Authorization: Bearer {access_token}
```

## 权限管理接口

### 获取权限列表

```http
GET /api/v1/permissions?page=1&page_size=10
Authorization: Bearer {access_token}
```

### 创建权限

```http
POST /api/v1/permissions
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "name": "string",
  "code": "string",
  "resource": "string",
  "action": "string",
  "description": "string"
}
```

## SSO/OAuth接口

### 授权

```http
POST /api/v1/oauth/authorize
Content-Type: application/json

{
  "client_id": "string",
  "redirect_uri": "string"
}
```

### 获取Token

```http
POST /api/v1/oauth/token
Content-Type: application/json

{
  "client_id": "string",
  "client_secret": "string",
  "code": "string"
}
```

**响应:**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "string",
    "refresh_token": "string",
    "token_type": "Bearer"
  }
}
```

### 获取用户信息

```http
GET /api/v1/oauth/userinfo
Authorization: Bearer {access_token}
```

### SSO应用管理

#### 获取应用列表

```http
GET /api/v1/oauth/applications?page=1&page_size=10
Authorization: Bearer {access_token}
```

#### 创建应用

```http
POST /api/v1/oauth/applications
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "name": "string",
  "client_secret": "string",
  "redirect_uris": "string"
}
```

## 错误码

| Code | 说明 |
|------|------|
| 0 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 403 | 禁止访问 |
| 404 | 资源不存在 |
| 500 | 服务器错误 |

## 响应格式

所有API响应遵循以下格式：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```
