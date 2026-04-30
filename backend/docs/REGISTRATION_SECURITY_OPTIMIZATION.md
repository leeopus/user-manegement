# 用户注册系统安全与体验优化方案

## 📋 当前状态分析

### 现有功能
- ✅ 基本的注册流程（用户名、邮箱、密码）
- ✅ 密码确认匹配验证
- ✅ 邮箱/用户名唯一性检查
- ✅ 密码 BCrypt 加密
- ✅ 审计日志记录

### 存在问题
- ❌ 密码强度要求太弱（仅6位）
- ❌ 无防止恶意注册机制
- ❌ 无验证码
- ❌ 无邮箱验证
- ❌ 用户体验不友好（无实时反馈、密码强度提示）

---

## 🔐 一、密码强度验证（必须）

### 主流密码标准

**方案 A：严格模式（推荐金融/政务系统）**
```
- 长度：8-32 位
- 必须包含：大写字母、小写字母、数字、特殊字符
- 不能包含：用户名、邮箱前缀
- 不能使用：常见弱密码（password123、12345678等）
```

**方案 B：平衡模式（推荐互联网应用）**
```
- 长度：8-64 位
- 至少包含：字母 + 数字
- 推荐包含：特殊字符（加分项）
- 不能包含：用户名
```

**方案 C：宽松模式（内部系统）**
```
- 长度：6-128 位
- 不能全是：纯数字或纯字母
```

### 密码强度评分系统

```go
评分规则：
- 长度 < 8: 0分（拒绝）
- 长度 8-11: +1分
- 长度 12+: +2分
- 包含小写: +1分
- 包含大写: +1分
- 包含数字: +1分
- 包含特殊字符: +2分
- 包含用户名: -5分（拒绝）

结果：
0-2分: 弱（显示红色，建议改进）
3-4分: 中等（显示黄色，可接受）
5-7分: 强（显示绿色，优秀）
```

---

## 🛡️ 二、防止恶意注册（必须）

### 1. IP 限流（关键）

**实现方式：**
```go
// 使用 Redis 实现滑动窗口限流
// 规则：同一 IP 1小时内最多注册 3 个账号
// 使用 mem：register:limit:{ip} = {count, timestamp}
```

**配置参数：**
```yaml
rate_limit:
  register:
    max_attempts: 3          # 最大尝试次数
    window_seconds: 3600     # 时间窗口（1小时）
    block_duration: 86400    # 封禁时长（24小时）
```

### 2. 验证码（推荐）

**图形验证码：**
- 适用于：公开注册的互联网应用
- 工具：Go 的 `github.com/mojocn/base64Captcha`
- 特点：防止机器人批量注册

**邮箱验证码：**
- 适用于：需要邮箱验证的系统
- 特点：双重验证（邮箱所有权 + 防机器人）
- 有效期：10分钟
- 限流：同一邮箱 1天内最多发送 3 次

**短信验证码：**
- 适用于：移动端应用
- 成本：需要短信服务商
- 特点：最安全，但有成本

### 3. 邮箱验证（推荐）

**流程：**
```
1. 用户注册 → 状态为 "pending"
2. 发送验证邮件（包含验证链接）
3. 用户点击链接 → 状态变为 "active"
4. 未验证用户无法登录
5. 验证邮件有效期：24小时
```

**邮件模板：**
```html
Subject: 验证你的邮箱 - User System

Hi {username},

感谢注册！请点击下方链接验证邮箱：
{verify_url}

此链接24小时内有效。

如果没有注册此账号，请忽略此邮件。
```

### 4. 设备指纹（可选）

**采集信息：**
- IP 地址
- User-Agent
- 浏览器指纹
- 时区
- 语言

**规则：**
- 同一设备 24小时内只能注册 1 个账号
- 异常设备（如无 User-Agent）直接拒绝

---

## 📧 三、用户名与邮箱验证（必须）

### 用户名规则

```go
规则：
- 长度：3-32 字符
- 允许字符：a-z, A-Z, 0-9, _, -
- 不能以 - 或 _ 开头或结尾
- 不能连续出现 -- 或 __
- 保留用户名：admin, system, root, api 等
```

### 邮箱规则

```go
验证：
1. 标准邮箱格式验证（RFC 5322 简化版）
2. 域名必须有 MX 记录（可选）
3. 禁止一次性邮箱（temp-mail, guerrillamail 等）
4. 同一邮箱只能注册一次（已有）
```

### 一次性邮箱黑名单

```
黑名单域名示例：
- temp-mail.com
- guerrillamail.com
- 10minutemail.com
- mailinator.com
- ...（更新维护）
```

---

## 🎨 四、用户体验优化（推荐）

### 前端实时验证

**密码强度指示器：**
```
密码输入框下方显示强度条：
□□□□□ 极弱（红色）
■□□□□ 弱（橙色）
■■□□□ 中等（黄色）
■■■□□ 强（浅绿）
■■■■■ 很强（深绿）
```

**实时反馈：**
```javascript
输入时立即检查：
✓ 用户名可用性检查（防抖500ms）
✓ 邮箱格式验证
✓ 密码强度评分
✓ 密码确认匹配
```

**友好的错误提示：**
```
❌ "Invalid input"
✅ "密码至少8位，包含字母和数字"

❌ "Email already exists"
✅ "该邮箱已注册，<a href='/login'>立即登录</a>"
```

### 注册流程优化

**分步注册（可选）：**
```
步骤 1: 输入邮箱和密码
步骤 2: 设置用户名（可选）
步骤 3: 验证邮箱
```

**注册成功后：**
```
选项 A：自动登录并跳转到 Dashboard
选项 B：显示成功页面，引导去邮箱验证
选项 C：跳转到登录页（当前做法）
```

**推荐：选项 A + 邮箱验证提示**

---

## 🔍 五、安全记录与监控（推荐）

### 记录关键信息

```go
type AuditLog struct {
    UserID      uint
    Action      string  // "register_attempt", "register_success"
    Resource    string  // "user"
    IP          string  // 记录IP
    UserAgent   string  // 记录设备
    Details     string  // 失败原因等
    Status      string  // "success", "failed", "blocked"
    CreatedAt   time.Time
}
```

### 异常检测

**监控指标：**
- 同一 IP 短时间内多次注册失败
- 同一设备多次更换邮箱注册
- 注册来源异常（如无 Referer）
- 批量相同密码注册

**自动响应：**
- 超过阈值：自动封禁 IP 24小时
- 发送告警：通知管理员

---

## 📱 六、其他安全建议

### 1. HTTPS（必须）
生产环境必须使用 HTTPS，防止中间人攻击

### 2. CSRF Token
注册表单添加 CSRF Token

### 3. Content Security Policy (CSP)
防止 XSS 攻击

### 4. SQL 注入防护
使用参数化查询（GORM 已支持）

### 5. 敏感信息屏蔽
错误响应不泄露系统信息：
```
❌ "Error connecting to database"
✅ "Registration failed, please try again later"
```

---

## 🚀 七、实施优先级

### P0 - 立即实施（安全关键）
1. ✅ 密码强度验证（8位，字母+数字）
2. ✅ 用户名格式验证
3. ✅ IP 限流（Redis）
4. ✅ 记录注册 IP 和 User-Agent

### P1 - 本周实施（重要）
1. ✅ 密码强度前端指示器
2. ✅ 实时表单验证
3. ✅ 图形验证码
4. ✅ 邮箱验证流程

### P2 - 下阶段（优化）
1. ⭕ 邮箱 MX 记录验证
2. ⭕ 一次性邮箱黑名单
3. ⭕ 设备指纹识别
4. ⭕ 异常注册监控告警

### P3 - 可选（高级功能）
1. ⭕ 手机号注册 + 短信验证
2. ⭕ 第三方登录（OAuth）
3. ⭕ 邀请码注册（私有部署）
4. ⭕ KYC 实名认证

---

## 📊 八、配置示例

### 后端配置
```yaml
# config/security.yaml
password_policy:
  min_length: 8
  max_length: 64
  require_uppercase: false
  require_lowercase: true
  require_number: true
  require_special_char: false
  forbid_username: true
  common_passwords_file: "/config/common_passwords.txt"

rate_limit:
  register:
    max_attempts: 3
    window_seconds: 3600
    block_duration: 86400

username_rules:
  min_length: 3
  max_length: 32
  pattern: "^[a-zA-Z0-9]([a-zA-Z0-9_-]*[a-zA-Z0-9])?$"
  reserved: ["admin", "system", "root", "api", "www", "mail"]

email_validation:
  check_mx_record: false
  disposable_blacklist: "/config/disposable_emails.txt"
```

---

## 🎯 九、验证清单

完成优化后，测试以下场景：

### 功能测试
- [ ] 弱密码被拒绝（12345678）
- [ ] 包含用户名的密码被拒绝
- [ ] 用户名不符合格式被拒绝
- [ ] IP 限流生效（第4次注册被拒绝）
- [ ] 验证码正确才能注册
- [ ] 邮箱验证邮件正常发送
- [ ] 验证链接正确激活账号

### 安全测试
- [ ] SQL 注入尝试被防御
- [ ] XSS 尝试被转义
- [ ] 批量注册被限流阻止
- [ ] 异常 IP 被自动封禁
- [ ] 一次性邮箱被拒绝

### 用户体验测试
- [ ] 密码强度实时显示
- [ ] 错误提示清晰友好
- [ ] 表单验证即时反馈
- [ ] 注册流程顺畅
- [ ] 成功后引导清晰

---

**建议：优先实施 P0 和 P1 级别的优化，既保证安全又提升体验。**
