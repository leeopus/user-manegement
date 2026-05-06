import { describe, it, expect, vi, beforeEach } from 'vitest'
import { APIException } from '../errors'

// Mock CSRF module
vi.mock('../csrf', () => ({
  addCSRFToHeaders: vi.fn().mockResolvedValue({ 'X-CSRF-Token': 'test-csrf', 'Content-Type': 'application/json' }),
}))

describe('APIClient', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('APIException handles register response with message only', async () => {
    const apiErr = {
      code: 'AUTH_REGISTER_EMAIL_EXISTS_400',
      message: 'AUTH_REGISTER_EMAIL_EXISTS',
    }
    const err = APIException.fromAPIError(apiErr)
    expect(err.getHTTPStatus()).toBe(400)
    expect(err.isCategory('AUTH')).toBe(true)
  })

  it('APIException preserves request_id from API error', () => {
    const apiErr = {
      code: 'AUTH_LOGIN_INVALID_CREDENTIALS_401',
      message: 'AUTH_LOGIN_INVALID_CREDENTIALS',
      request_id: 'req_abc123',
    }
    const err = APIException.fromAPIError(apiErr)
    expect(err.code).toBe('AUTH_LOGIN_INVALID_CREDENTIALS_401')
  })

  it('handles non-JSON responses gracefully', () => {
    const err = new Error('NETWORK_ERROR: Server returned non-JSON response (HTTP 502)')
    expect(err.message).toContain('502')
  })

  it('handles request timeout', () => {
    const err = new DOMException('The operation was aborted', 'AbortError')
    expect(err.name).toBe('AbortError')
  })
})
