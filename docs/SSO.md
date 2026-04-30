# SSO 单点登录接入文档

## 概述

本系统支持标准的OAuth 2.0和OpenID Connect (OIDC)协议，第三方应用可以通过这些协议实现单点登录。

## 快速开始

### 1. 注册应用

在管理后台创建SSO应用，获取以下信息：

- `client_id`: 客户端ID
- `client_secret`: 客户端密钥
- `redirect_uri`: 回调地址

### 2. 集成流程

#### 步骤1: 引导用户授权

将用户重定向到授权页面：

```
https://your-sso-domain/oauth/authorize?
  client_id={client_id}&
  redirect_uri={redirect_uri}&
  response_type=code&
  state={random_state}&
  scope=openid+email+profile
```

**参数说明:**

| 参数 | 必需 | 说明 |
|------|------|------|
| client_id | 是 | 应用ID |
| redirect_uri | 是 | 授权后重定向的URI，必须与注册时填写的一致 |
| response_type | 是 | 固定值: `code` |
| state | 是 | 随机字符串，用于防止CSRF攻击 |
| scope | 否 | 请求的权限范围，如: `openid email profile` |

#### 步骤2: 用户登录授权

用户在SSO登录页面完成登录和授权。

#### 步骤3: 获取授权码

授权成功后，用户浏览器会被重定向到：

```
{redirect_uri}?code={authorization_code}&state={state}
```

#### 步骤4: 用授权码换取Token

使用POST请求获取access_token：

```http
POST /oauth/token
Content-Type: application/json

{
  "client_id": "your_client_id",
  "client_secret": "your_client_secret",
  "code": "authorization_code_from_step_3",
  "grant_type": "authorization_code"
}
```

**响应:**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 3600
  }
}
```

#### 步骤5: 使用Token访问用户信息

使用access_token获取用户信息：

```http
GET /oauth/userinfo
Authorization: Bearer {access_token}
```

**响应:**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "username": "johndoe",
    "email": "john@example.com",
    "avatar": "https://example.com/avatar.jpg"
  }
}
```

## 完整示例

### 示例应用 (Python/Flask)

```python
from flask import Flask, request, redirect, session, jsonify
import requests

app = Flask(__name__)
app.secret_key = 'your-secret-key'

# SSO配置
SSO_BASE_URL = 'http://localhost:8080'
CLIENT_ID = 'your_client_id'
CLIENT_SECRET = 'your_client_secret'
REDIRECT_URI = 'http://localhost:5000/callback'

@app.route('/')
def home():
    if 'access_token' in session:
        # 已登录，显示用户信息
        userinfo = get_userinfo(session['access_token'])
        return f"Hello, {userinfo['username']}!"
    else:
        # 未登录，引导用户登录
        return f'<a href="/login">Login with SSO</a>'

@app.route('/login')
def login():
    # 生成随机state
    import secrets
    state = secrets.token_urlsafe(16)
    session['oauth_state'] = state

    # 重定向到SSO授权页面
    auth_url = f'{SSO_BASE_URL}/api/v1/oauth/authorize'
    params = {
        'client_id': CLIENT_ID,
        'redirect_uri': REDIRECT_URI,
        'response_type': 'code',
        'state': state,
        'scope': 'openid email profile'
    }
    return redirect(f'{auth_url}?{requests.compat.urlencode(params)}')

@app.route('/callback')
def callback():
    # 验证state
    if request.args.get('state') != session.get('oauth_state'):
        return 'Invalid state', 400

    # 获取授权码
    code = request.args.get('code')

    # 换取access_token
    token_url = f'{SSO_BASE_URL}/api/v1/oauth/token'
    data = {
        'client_id': CLIENT_ID,
        'client_secret': CLIENT_SECRET,
        'code': code,
        'grant_type': 'authorization_code'
    }
    response = requests.post(token_url, json=data)
    token_data = response.json()

    if token_data.get('code') == 0:
        # 保存token
        session['access_token'] = token_data['data']['access_token']
        session['refresh_token'] = token_data['data']['refresh_token']
        return redirect('/')
    else:
        return f"Error: {token_data.get('message')}", 400

def get_userinfo(access_token):
    userinfo_url = f'{SSO_BASE_URL}/api/v1/oauth/userinfo'
    headers = {'Authorization': f'Bearer {access_token}'}
    response = requests.get(userinfo_url, headers=headers)
    return response.json().get('data', {})

if __name__ == '__main__':
    app.run(port=5000)
```

### 示例应用 (Node.js/Express)

```javascript
const express = require('express');
const axios = require('axios');
const crypto = require('crypto');

const app = express();
const PORT = 5000;

// SSO配置
const SSO_BASE_URL = 'http://localhost:8080';
const CLIENT_ID = 'your_client_id';
const CLIENT_SECRET = 'your_client_secret';
const REDIRECT_URI = 'http://localhost:5000/callback';

app.get('/', (req, res) => {
  if (req.session.accessToken) {
    res.send(`Hello, ${req.session.username}!`);
  } else {
    res.send('<a href="/login">Login with SSO</a>');
  }
});

app.get('/login', (req, res) => {
  // 生成随机state
  const state = crypto.randomBytes(16).toString('hex');
  req.session.oauthState = state;

  // 重定向到SSO授权页面
  const authUrl = `${SSO_BASE_URL}/api/v1/oauth/authorize`;
  const params = new URLSearchParams({
    client_id: CLIENT_ID,
    redirect_uri: REDIRECT_URI,
    response_type: 'code',
    state: state,
    scope: 'openid email profile'
  });

  res.redirect(`${authUrl}?${params.toString()}`);
});

app.get('/callback', async (req, res) => {
  // 验证state
  if (req.query.state !== req.session.oauthState) {
    return res.status(400).send('Invalid state');
  }

  // 获取授权码
  const code = req.query.code;

  try {
    // 换取access_token
    const tokenUrl = `${SSO_BASE_URL}/api/v1/oauth/token`;
    const tokenResponse = await axios.post(tokenUrl, {
      client_id: CLIENT_ID,
      client_secret: CLIENT_SECRET,
      code: code,
      grant_type: 'authorization_code'
    });

    if (tokenResponse.data.code === 0) {
      // 保存token
      req.session.accessToken = tokenResponse.data.data.access_token;
      req.session.refreshToken = tokenResponse.data.data.refresh_token;

      // 获取用户信息
      const userinfoUrl = `${SSO_BASE_URL}/api/v1/oauth/userinfo`;
      const userinfoResponse = await axios.get(userinfoUrl, {
        headers: {
          Authorization: `Bearer ${req.session.accessToken}`
        }
      });

      req.session.username = userinfoResponse.data.data.username;
      res.redirect('/');
    } else {
      res.status(400).send(`Error: ${tokenResponse.data.message}`);
    }
  } catch (error) {
    res.status(500).send(`Error: ${error.message}`);
  }
});

app.listen(PORT, () => {
  console.log(`Server running on http://localhost:${PORT}`);
});
```

## Token刷新

Access_token有效期为1小时，过期后可以使用refresh_token获取新的access_token：

```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "your_refresh_token"
}
```

**响应:**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user": {...},
    "access_token": "new_access_token"
  }
}
```

## 安全建议

1. **state参数**: 始终使用随机state参数防止CSRF攻击
2. **HTTPS**: 生产环境必须使用HTTPS
3. **密钥保护**: 妥善保管client_secret，不要泄露
4. **Token存储**: 安全存储access_token和refresh_token
5. **Token过期**: 定期检查token是否过期
6. **回调验证**: 验证redirect_uri是否匹配

## 错误处理

| 错误码 | 说明 | 处理方式 |
|--------|------|----------|
| 400 | 请求参数错误 | 检查参数是否正确 |
| 401 | 未授权 | 重新引导用户登录 |
| 403 | 禁止访问 | 检查应用权限 |
| 404 | 资源不存在 | 检查URL是否正确 |

## 常见问题

### Q: 如何获取client_id和client_secret?

A: 登录管理后台，进入"SSO应用管理"，创建新应用后即可获得。

### Q: 支持哪些scope?

A: 目前支持的scope包括：
- `openid`: OpenID Connect基础信息
- `email`: 邮箱地址
- `profile`: 用户基本信息

### Q: Token有效期是多久?

A: Access_token有效期为1小时，refresh_token有效期为30天。

### Q: 如何撤销Token?

A: 调用登出接口或在管理后台删除应用。

## 技术支持

如有问题，请联系技术支持团队或提交Issue。
