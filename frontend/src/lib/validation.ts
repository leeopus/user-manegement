// 密码强度等级
export enum PasswordStrength {
  Weak = 0,
  Fair = 1,
  Good = 2,
  Strong = 3,
}

// 验证结果
export interface ValidationResult {
  valid: boolean
  error?: string
}

// 验证用户名
export function validateUsername(username: string): ValidationResult {
  if (!username) {
    return { valid: false, error: "用户名不能为空" }
  }

  if (username.length < 3) {
    return { valid: false, error: "用户名至少3位" }
  }

  if (username.length > 32) {
    return { valid: false, error: "用户名最多32位" }
  }

  // 格式：字母、数字、下划线、连字符，不能以 _ 或 - 开头或结尾
  const pattern = /^[a-zA-Z0-9]([a-zA-Z0-9_-]*[a-zA-Z0-9])?$/
  if (!pattern.test(username)) {
    return { valid: false, error: "用户名只能包含字母、数字、下划线和连字符，且不能以 _ 或 - 开头或结尾" }
  }

  // 检查连续特殊字符
  if (username.includes("--") || username.includes("__")) {
    return { valid: false, error: "用户名不能连续使用 _ 或 -" }
  }

  // 保留用户名列表
  const reserved = [
    "admin", "administrator", "system", "root", "api",
    "www", "mail", "ftp", "localhost", "smtp", "pop",
    "ns1", "ns2", "dns", "host", "webmaster", "support",
    "info", "sales", "marketing", "news", "blog",
  ]

  if (reserved.includes(username.toLowerCase())) {
    return { valid: false, error: "该用户名不可使用" }
  }

  return { valid: true }
}

// 验证邮箱
export function validateEmail(email: string): ValidationResult {
  if (!email) {
    return { valid: false, error: "邮箱不能为空" }
  }

  // 基本邮箱格式
  const pattern = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/
  if (!pattern.test(email)) {
    return { valid: false, error: "邮箱格式不正确" }
  }

  if (email.length > 254) {
    return { valid: false, error: "邮箱地址过长" }
  }

  // 检查域名
  const [, domain] = email.split("@")
  if (domain && domain.length < 3) {
    return { valid: false, error: "邮箱域名不正确" }
  }

  return { valid: true }
}

// 验证密码
export function validatePassword(password: string, username?: string): { strength: PasswordStrength; error?: string } {
  if (!password) {
    return { strength: PasswordStrength.Weak, error: "密码不能为空" }
  }

  // 检查长度
  if (password.length < 8) {
    return { strength: PasswordStrength.Weak, error: "密码至少8位" }
  }

  if (password.length > 64) {
    return { strength: PasswordStrength.Weak, error: "密码最多64位" }
  }

  // 检查字符类型
  let hasUpper = false
  let hasLower = false
  let hasNumber = false
  let hasSpecial = false

  for (const char of password) {
    if (char >= "A" && char <= "Z") hasUpper = true
    else if (char >= "a" && char <= "z") hasLower = true
    else if (char >= "0" && char <= "9") hasNumber = true
    else hasSpecial = true
  }

  // 必须包含小写字母
  if (!hasLower) {
    return { strength: PasswordStrength.Weak, error: "密码必须包含小写字母" }
  }

  // 必须包含数字
  if (!hasNumber) {
    return { strength: PasswordStrength.Weak, error: "密码必须包含数字" }
  }

  let score = 0

  // 长度评分
  if (password.length >= 12) score += 2
  else if (password.length >= 8) score += 1

  // 字符类型评分
  if (hasLower) score += 1
  if (hasUpper) score += 1
  if (hasNumber) score += 1
  if (hasSpecial) score += 2

  // 检查是否包含用户名
  if (username && username.length >= 3) {
    const lowerPassword = password.toLowerCase()
    const lowerUsername = username.toLowerCase()
    if (lowerPassword.includes(lowerUsername)) {
      return { strength: PasswordStrength.Weak, error: "密码不能包含用户名" }
    }
  }

  // 常见弱密码
  const commonPasswords = [
    "password", "12345678", "123456789", "qwerty", "abc123",
    "monkey", "1234567890", "password1", "123123", "qwerty123",
    "password123", "admin123", "welcome1", "login123", "passw0rd",
  ]

  if (commonPasswords.some(common => password.toLowerCase() === common)) {
    return { strength: PasswordStrength.Weak, error: "密码过于简单" }
  }

  // 检查是否全是相同字符
  if (password.length > 0 && password.split("").every(char => char === password[0])) {
    return { strength: PasswordStrength.Weak, error: "密码不能全是相同字符" }
  }

  // 计算强度等级
  let strength: PasswordStrength
  if (score <= 2) strength = PasswordStrength.Weak
  else if (score <= 4) strength = PasswordStrength.Fair
  else if (score <= 6) strength = PasswordStrength.Good
  else strength = PasswordStrength.Strong

  return { strength }
}

// 防抖函数
export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: NodeJS.Timeout | null = null

  return function executedFunction(...args: Parameters<T>) {
    const later = () => {
      timeout = null
      func(...args)
    }

    if (timeout) clearTimeout(timeout)
    timeout = setTimeout(later, wait)
  }
}

// 检查用户名是否可用（API调用）
export async function checkUsernameAvailable(username: string): Promise<boolean> {
  try {
    const response = await fetch(`/api/v1/users/check-username?username=${encodeURIComponent(username)}`)
    const data = await response.json()
    return data.code === 0 && data.data.available
  } catch {
    // 如果请求失败，假设可用
    return true
  }
}

// 检查邮箱是否可用（API调用）
export async function checkEmailAvailable(email: string): Promise<boolean> {
  try {
    const response = await fetch(`/api/v1/users/check-email?email=${encodeURIComponent(email)}`)
    const data = await response.json()
    return data.code === 0 && data.data.available
  } catch {
    // 如果请求失败，假设可用
    return true
  }
}
