import { describe, it, expect } from 'vitest'
import {
  APIException,
  isAuthError,
  isValidationError,
  isUnauthorized,
  isNetworkError,
} from '../errors'

describe('APIException', () => {
  it('extracts HTTP status from error code', () => {
    const err = new APIException(
      'AUTH_LOGIN_INVALID_CREDENTIALS_401',
      'AUTH_LOGIN_INVALID_CREDENTIALS',
    )
    expect(err.getHTTPStatus()).toBe(401)
  })

  it('extracts error category', () => {
    const err = new APIException('VALIDATION_ERROR_400', 'VALIDATION_ERROR')
    expect(err.getCategory()).toBe('VALIDATION')
  })

  it('checks error code match', () => {
    const err = new APIException('AUTH_TOKEN_EXPIRED_401', 'AUTH_TOKEN_EXPIRED')
    expect(err.isErrorCode('AUTH_TOKEN_EXPIRED_401')).toBe(true)
    expect(err.isErrorCode('OTHER_CODE')).toBe(false)
  })

  it('checks category prefix', () => {
    const err = new APIException('AUTH_LOGIN_FAILED_401', 'AUTH_LOGIN_FAILED')
    expect(err.isCategory('AUTH')).toBe(true)
    expect(err.isCategory('USER')).toBe(false)
  })

  it('creates from APIError', () => {
    const apiErr = {
      code: 'USER_NOT_FOUND_404',
      message: 'USER_NOT_FOUND',
      details: { id: 123 },
    }
    const err = APIException.fromAPIError(apiErr)
    expect(err.code).toBe('USER_NOT_FOUND_404')
    expect(err.getHTTPStatus()).toBe(404)
    expect(err.details).toEqual({ id: 123 })
  })

  it('returns 500 for non-numeric status codes', () => {
    const err = new APIException('SOME_ERROR', 'SOME_ERROR')
    expect(err.getHTTPStatus()).toBe(500)
  })
})

describe('isAuthError', () => {
  it('returns true for AUTH category errors', () => {
    const err = new APIException('_AUTH_ERROR_401', 'AUTH_ERROR')
    // Note: category is extracted from first part before _
    const err2 = new APIException('AUTH_LOGIN_401', 'AUTH_LOGIN')
    expect(isAuthError(err2)).toBe(true)
  })

  it('returns false for non-AUTH errors', () => {
    const err = new APIException('VALIDATION_ERROR_400', 'VALIDATION_ERROR')
    expect(isAuthError(err)).toBe(false)
  })

  it('returns false for non-APIException errors', () => {
    expect(isAuthError(new Error('generic'))).toBe(false)
  })
})

describe('isValidationError', () => {
  it('returns true for VALIDATION category', () => {
    const err = new APIException('VALIDATION_FAILED_400', 'VALIDATION_FAILED')
    expect(isValidationError(err)).toBe(true)
  })
})

describe('isUnauthorized', () => {
  it('returns true for 401 errors', () => {
    const err = new APIException('AUTH_UNAUTHORIZED_401', 'AUTH_UNAUTHORIZED')
    expect(isUnauthorized(err)).toBe(true)
  })

  it('returns false for non-401 errors', () => {
    const err = new APIException('AUTH_FORBIDDEN_403', 'AUTH_FORBIDDEN')
    expect(isUnauthorized(err)).toBe(false)
  })
})

describe('isNetworkError', () => {
  it('detects network errors from Error message', () => {
    expect(isNetworkError(new Error('Failed to fetch data'))).toBe(true)
    expect(isNetworkError(new Error('NetworkError at fetch'))).toBe(true)
    expect(isNetworkError(new Error('Something else'))).toBe(false)
  })
})
