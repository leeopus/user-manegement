package errors

import (
	"fmt"
	"net/http"
)

// AppError 应用错误类型
type AppError struct {
	Code       string                 // 错误码，如 "AUTH_LOGIN_INVALID_CREDENTIALS_401"
	MessageKey string                 // 前端翻译键，如 "AUTH_LOGIN_INVALID_CREDENTIALS"
	Details    map[string]interface{} // 额外错误详情
	HTTPStatus int                    // HTTP 状态码
}

func (e *AppError) Error() string {
	if e.Details != nil && len(e.Details) > 0 {
		return fmt.Sprintf("%s: %v", e.Code, e.Details)
	}
	return e.Code
}

// New 创建新的应用错误
func New(code, messageKey string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		MessageKey: messageKey,
		HTTPStatus: httpStatus,
	}
}

// WithDetails 创建一个带有详情的新错误实例（不修改原对象，避免并发竞态）
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	return &AppError{
		Code:       e.Code,
		MessageKey: e.MessageKey,
		HTTPStatus: e.HTTPStatus,
		Details:    details,
	}
}

// =====================================================
// 认证相关错误 (AUTH)
// =====================================================

var (
	// 登录错误
	ErrInvalidCredentials = New(
		"AUTH_LOGIN_INVALID_CREDENTIALS_401",
		"AUTH_LOGIN_INVALID_CREDENTIALS",
		http.StatusUnauthorized,
	)

	ErrAccountLocked = New(
		"AUTH_ACCOUNT_LOCKED_423",
		"AUTH_ACCOUNT_LOCKED",
		423, // Locked
	)

	ErrAccountNotActive = New(
		"AUTH_LOGIN_ACCOUNT_NOT_ACTIVE_403",
		"AUTH_LOGIN_ACCOUNT_NOT_ACTIVE",
		http.StatusForbidden,
	)

	// 注册错误
	ErrEmailAlreadyExists = New(
		"AUTH_REGISTER_EMAIL_EXISTS_400",
		"AUTH_REGISTER_EMAIL_EXISTS",
		http.StatusBadRequest,
	)

	ErrUsernameAlreadyExists = New(
		"AUTH_REGISTER_USERNAME_EXISTS_400",
		"AUTH_REGISTER_USERNAME_EXISTS",
		http.StatusBadRequest,
	)

	ErrPasswordTooWeak = New(
		"AUTH_REGISTER_PASSWORD_WEAK_400",
		"AUTH_REGISTER_PASSWORD_WEAK",
		http.StatusBadRequest,
	)

	ErrDisposableEmail = New(
		"AUTH_REGISTER_DISPOSABLE_EMAIL_400",
		"AUTH_REGISTER_DISPOSABLE_EMAIL",
		http.StatusBadRequest,
	)

	// Token 错误
	ErrInvalidRefreshToken = New(
		"AUTH_TOKEN_INVALID_REFRESH_401",
		"AUTH_TOKEN_INVALID_REFRESH",
		http.StatusUnauthorized,
	)

	ErrInvalidAccessToken = New(
		"AUTH_TOKEN_INVALID_ACCESS_401",
		"AUTH_TOKEN_INVALID_ACCESS",
		http.StatusUnauthorized,
	)

	ErrTokenExpired = New(
		"AUTH_TOKEN_EXPIRED_401",
		"AUTH_TOKEN_EXPIRED",
		http.StatusUnauthorized,
	)
)

// =====================================================
// 验证相关错误 (VALIDATION)
// =====================================================

var (
	// 用户名验证
	ErrUsernameRequired = New(
		"VALIDATION_USERNAME_REQUIRED_400",
		"VALIDATION_USERNAME_REQUIRED",
		http.StatusBadRequest,
	)

	ErrUsernameTooShort = New(
		"VALIDATION_USERNAME_TOO_SHORT_400",
		"VALIDATION_USERNAME_TOO_SHORT",
		http.StatusBadRequest,
	).WithDetails(map[string]interface{}{
		"min": 3,
	})

	ErrUsernameTooLong = New(
		"VALIDATION_USERNAME_TOO_LONG_400",
		"VALIDATION_USERNAME_TOO_LONG",
		http.StatusBadRequest,
	).WithDetails(map[string]interface{}{
		"max": 32,
	})

	ErrUsernameInvalidPattern = New(
		"VALIDATION_USERNAME_INVALID_PATTERN_400",
		"VALIDATION_USERNAME_INVALID_PATTERN",
		http.StatusBadRequest,
	)

	// 邮箱验证
	ErrEmailRequired = New(
		"VALIDATION_EMAIL_REQUIRED_400",
		"VALIDATION_EMAIL_REQUIRED",
		http.StatusBadRequest,
	)

	ErrEmailInvalid = New(
		"VALIDATION_EMAIL_INVALID_400",
		"VALIDATION_EMAIL_INVALID",
		http.StatusBadRequest,
	)

	// 密码验证
	ErrPasswordRequired = New(
		"VALIDATION_PASSWORD_REQUIRED_400",
		"VALIDATION_PASSWORD_REQUIRED",
		http.StatusBadRequest,
	)

	ErrPasswordTooShort = New(
		"VALIDATION_PASSWORD_TOO_SHORT_400",
		"VALIDATION_PASSWORD_TOO_SHORT",
		http.StatusBadRequest,
	).WithDetails(map[string]interface{}{
		"min": 8,
	})

	ErrPasswordTooLong = New(
		"VALIDATION_PASSWORD_TOO_LONG_400",
		"VALIDATION_PASSWORD_TOO_LONG",
		http.StatusBadRequest,
	).WithDetails(map[string]interface{}{
		"max": 64,
	})

	ErrPasswordContainsUsername = New(
		"VALIDATION_PASSWORD_CONTAINS_USERNAME_400",
		"VALIDATION_PASSWORD_CONTAINS_USERNAME",
		http.StatusBadRequest,
	)

	ErrPasswordAllSameChars = New(
		"VALIDATION_PASSWORD_ALL_SAME_CHARS_400",
		"VALIDATION_PASSWORD_ALL_SAME_CHARS",
		http.StatusBadRequest,
	)
)

// =====================================================
// 密码重置相关错误 (PASSWORD_RESET)
// =====================================================

var (
	ErrInvalidResetToken = New(
		"AUTH_RESET_INVALID_TOKEN_400",
		"AUTH_RESET_INVALID_TOKEN",
		http.StatusBadRequest,
	)

	ErrResetTokenExpired = New(
		"AUTH_RESET_TOKEN_EXPIRED_400",
		"AUTH_RESET_TOKEN_EXPIRED",
		http.StatusBadRequest,
	)

	ErrResetTokenAlreadyUsed = New(
		"AUTH_RESET_TOKEN_USED_400",
		"AUTH_RESET_TOKEN_USED",
		http.StatusBadRequest,
	)
)

// =====================================================
// 用户相关错误 (USER)
// =====================================================

var (
	ErrUserNotFound = New(
		"USER_NOT_FOUND_404",
		"USER_NOT_FOUND",
		http.StatusNotFound,
	)

	ErrUserAlreadyExists = New(
		"USER_ALREADY_EXISTS_409",
		"USER_ALREADY_EXISTS",
		http.StatusConflict,
	)
)

// =====================================================
// OAuth 相关错误 (OAUTH)
// =====================================================

var (
	ErrOAuthInvalidClient = New(
		"OAUTH_INVALID_CLIENT_401",
		"OAUTH_INVALID_CLIENT",
		http.StatusUnauthorized,
	)

	ErrOAuthInvalidClientSecret = New(
		"OAUTH_INVALID_CLIENT_SECRET_401",
		"OAUTH_INVALID_CLIENT_SECRET",
		http.StatusUnauthorized,
	)

	ErrOAuthInvalidCode = New(
		"OAUTH_INVALID_CODE_400",
		"OAUTH_INVALID_CODE",
		http.StatusBadRequest,
	)

	ErrOAuthInvalidRedirectURI = New(
		"OAUTH_INVALID_REDIRECT_URI_400",
		"OAUTH_INVALID_REDIRECT_URI",
		http.StatusBadRequest,
	)
)

// =====================================================
// 通用错误 (COMMON)
// =====================================================

var (
	ErrInternalServer = New(
		"INTERNAL_SERVER_ERROR_500",
		"INTERNAL_SERVER_ERROR",
		http.StatusInternalServerError,
	)

	ErrDatabase = New(
		"DATABASE_ERROR_500",
		"DATABASE_ERROR",
		http.StatusInternalServerError,
	)

	ErrNetwork = New(
		"NETWORK_ERROR_503",
		"NETWORK_ERROR",
		http.StatusServiceUnavailable,
	)
)

// IsAppError 判断是否为应用错误
func IsAppError(err error) (*AppError, bool) {
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}
