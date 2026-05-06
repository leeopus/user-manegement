import { describe, it, expect } from 'vitest'
import {
  validateEmail,
  validateUsername,
  validatePassword,
  PasswordStrength,
} from '../validation'

describe('validateEmail', () => {
  it('accepts valid emails', () => {
    expect(validateEmail('user@example.com').valid).toBe(true)
    expect(validateEmail('user.name@example.com').valid).toBe(true)
    expect(validateEmail('user+tag@example.com').valid).toBe(true)
  })

  it('rejects empty email', () => {
    expect(validateEmail('').valid).toBe(false)
  })

  it('rejects email without @', () => {
    expect(validateEmail('no-at-sign').valid).toBe(false)
  })

  it('rejects email with short domain', () => {
    expect(validateEmail('user@a').valid).toBe(false)
  })

  it('rejects overly long email', () => {
    const long = 'a'.repeat(250) + '@x.com'
    expect(validateEmail(long).valid).toBe(false)
  })
})

describe('validateUsername', () => {
  it('accepts valid usernames', () => {
    expect(validateUsername('john').valid).toBe(true)
    expect(validateUsername('john_doe').valid).toBe(true)
    expect(validateUsername('user123').valid).toBe(true)
  })

  it('rejects too short username', () => {
    expect(validateUsername('ab').valid).toBe(false)
  })

  it('rejects reserved usernames', () => {
    expect(validateUsername('admin').valid).toBe(false)
    expect(validateUsername('system').valid).toBe(false)
  })

  it('rejects consecutive special chars', () => {
    expect(validateUsername('john--doe').valid).toBe(false)
    expect(validateUsername('john__doe').valid).toBe(false)
  })

  it('rejects username starting with special char', () => {
    expect(validateUsername('_john').valid).toBe(false)
    expect(validateUsername('-john').valid).toBe(false)
  })
})

describe('validatePassword', () => {
  it('rejects empty password', () => {
    expect(validatePassword('').error).toBeDefined()
  })

  it('rejects password shorter than 8 chars', () => {
    const result = validatePassword('Ab1')
    expect(result.error).toBeDefined()
    expect(result.strength).toBe(PasswordStrength.Weak)
  })

  it('rejects password longer than 64 chars', () => {
    const result = validatePassword('A'.repeat(65) + '1a')
    expect(result.error).toBeDefined()
  })

  it('rejects common weak passwords', () => {
    expect(validatePassword('password').error).toBeDefined()
    expect(validatePassword('12345678').error).toBeDefined()
    expect(validatePassword('qwerty').error).toBeDefined()
  })

  it('rejects all same characters', () => {
    expect(validatePassword('AAAAAAAAAAA').error).toBeDefined()
  })

  it('accepts long passphrases without mixed case (NIST)', () => {
    const result = validatePassword('correcthorsebatterystaple')
    expect(result.error).toBeUndefined()
  })

  it('scores strong passwords highly', () => {
    const result = validatePassword('MyStr0ng!Passw0rd#2024')
    expect(result.error).toBeUndefined()
    expect(result.strength).toBeGreaterThanOrEqual(PasswordStrength.Good)
  })
})
