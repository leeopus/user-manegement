// 密码强度等级
export enum PasswordStrength {
  Weak = 0,
  Fair = 1,
  Good = 2,
  Strong = 3,
}

// 验证结果（error 为 i18n 翻译 key，如 "validation.username.required"）
export interface ValidationResult {
  valid: boolean
  error?: string
}

// 验证用户名
export function validateUsername(username: string): ValidationResult {
  if (!username) {
    return { valid: false, error: "validation.username.required" }
  }

  if (username.length < 3) {
    return { valid: false, error: "validation.username.minLength" }
  }

  if (username.length > 32) {
    return { valid: false, error: "validation.username.maxLength" }
  }

  const pattern = /^[a-zA-Z0-9]([a-zA-Z0-9_-]*[a-zA-Z0-9])?$/
  if (!pattern.test(username)) {
    return { valid: false, error: "validation.username.pattern" }
  }

  if (username.includes("--") || username.includes("__")) {
    return { valid: false, error: "validation.username.consecutive" }
  }

  const reserved = [
    "admin", "administrator", "system", "root", "api",
    "www", "mail", "ftp", "localhost", "smtp", "pop",
    "ns1", "ns2", "dns", "host", "webmaster", "support",
    "info", "sales", "marketing", "news", "blog",
  ]

  if (reserved.includes(username.toLowerCase())) {
    return { valid: false, error: "validation.username.reserved" }
  }

  return { valid: true }
}

// 验证邮箱
export function validateEmail(email: string): ValidationResult {
  if (!email) {
    return { valid: false, error: "validation.email.required" }
  }

  const pattern = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/
  if (!pattern.test(email)) {
    return { valid: false, error: "validation.email.invalid" }
  }

  if (email.length > 254) {
    return { valid: false, error: "validation.email.tooLong" }
  }

  const [, domain] = email.split("@")
  if (domain && domain.length < 3) {
    return { valid: false, error: "validation.email.invalid" }
  }

  return { valid: true }
}

// 验证密码
export function validatePassword(password: string, _username?: string): { strength: PasswordStrength; error?: string } {
  if (!password) {
    return { strength: PasswordStrength.Weak, error: "validation.password.required" }
  }

  if (password.length < 8) {
    return { strength: PasswordStrength.Weak, error: "validation.password.minLength" }
  }

  if (password.length > 64) {
    return { strength: PasswordStrength.Weak, error: "validation.password.maxLength" }
  }

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

  let score = 0

  if (password.length >= 16) score += 4
  else if (password.length >= 12) score += 3
  else if (password.length >= 8) score += 1

  if (hasLower) score += 1
  if (hasUpper) score += 1
  if (hasNumber) score += 1
  if (hasSpecial) score += 1

  const commonPasswords = [
    "password", "12345678", "123456789", "qwerty", "abc123",
    "monkey", "1234567890", "password1", "123123", "qwerty123",
    "password123", "admin123", "welcome1", "login123", "passw0rd",
  ]

  if (commonPasswords.some(common => password.toLowerCase() === common)) {
    return { strength: PasswordStrength.Weak, error: "validation.password.tooWeak" }
  }

  if (password.length > 0 && password.split("").every(char => char === password[0])) {
    return { strength: PasswordStrength.Weak, error: "validation.password.sameChars" }
  }

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
    return true
  }
}
