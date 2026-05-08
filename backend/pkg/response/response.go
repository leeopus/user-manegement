package response

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	apperrors "github.com/user-system/backend/pkg/errors"
)

// Pagination 分页参数
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// ParsePagination 从请求中解析分页参数，返回标准化后的 page、pageSize、offset
func ParsePagination(c *gin.Context) (page, pageSize, offset int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset = (page - 1) * pageSize
	return
}

// ErrorDetail 错误详情
type ErrorDetail struct {
	Code      string                 `json:"code"`                // 错误码
	Message   string                 `json:"message"`             // 翻译键
	RequestID string                 `json:"request_id,omitempty"` // 请求追踪 ID
	Details   map[string]interface{} `json:"details,omitempty"`   // 额外详情
}

// APIResponse 统一响应格式
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorDetail `json:"error,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

// Created 创建成功响应 (201)
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    data,
	})
}

func getRequestID(c *gin.Context) string {
	if id, ok := c.Get("request_id"); ok {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}

// Error 错误响应（使用 AppError）
func Error(c *gin.Context, err error) {
	requestID := getRequestID(c)

	if appErr, ok := apperrors.IsAppError(err); ok {
		status := appErr.HTTPStatus
		if status == 0 {
			status = http.StatusInternalServerError
		}

		c.JSON(status, APIResponse{
			Success: false,
			Error: &ErrorDetail{
				Code:      appErr.Code,
				Message:   appErr.MessageKey,
				RequestID: requestID,
				Details:   appErr.Details,
			},
		})
		return
	}

	c.JSON(http.StatusInternalServerError, APIResponse{
		Success: false,
		Error: &ErrorDetail{
			Code:      apperrors.ErrInternalServer.Code,
			Message:   apperrors.ErrInternalServer.MessageKey,
			RequestID: requestID,
		},
	})
}

// ValidationError 验证错误响应（用于请求参数验证）
func ValidationError(c *gin.Context, message string) {
	// 清理错误信息，避免暴露内部字段名和类型细节
	sanitized := sanitizeValidationMessage(message)

	c.JSON(http.StatusBadRequest, APIResponse{
		Success: false,
		Error: &ErrorDetail{
			Code:    "VALIDATION_ERROR_400",
			Message: "VALIDATION_ERROR",
			Details: map[string]interface{}{
				"message": sanitized,
			},
		},
	})
}

// sanitizeValidationMessage 清理 Gin binding 错误信息，脱敏内部结构体路径、Go 类型名和用户输入值
func sanitizeValidationMessage(msg string) string {
	msg = stripGoFieldPaths(msg)
	// 如果不是已知的 Gin 格式化输出，使用通用提示避免泄露原始输入
	if strings.Contains(msg, "'") && !strings.HasPrefix(msg, "invalid ") && !strings.Contains(msg, "validation failed") {
		msg = "validation failed"
	}
	if len(msg) > 200 {
		msg = msg[:200]
	}
	return msg
}

// stripGoFieldPaths 移除 Go 结构体字段路径
// 将 "Key: 'RegisterRequest.Email' Error:Field validation for 'Email' failed on the 'email' tag"
// 转为 "Email: validation failed on email"
func stripGoFieldPaths(msg string) string {
	const keyPrefix = "Key: '"
	const errorMid = "' Error:Field validation for '"
	const failedOn = "' failed on the '"
	const tagEnd = "' tag"

	keyIdx := strings.Index(msg, keyPrefix)
	if keyIdx == -1 {
		return msg
	}

	// 提取字段路径
	pathStart := keyIdx + len(keyPrefix)
	errorIdx := strings.Index(msg[pathStart:], errorMid)
	if errorIdx == -1 {
		return msg
	}
	fieldPath := msg[pathStart : pathStart+errorIdx]

	// 提取字段名（最后一个 . 之后）
	fieldName := fieldPath
	if dot := strings.LastIndex(fieldPath, "."); dot != -1 {
		fieldName = fieldPath[dot+1:]
	}

	// 提取验证规则
	ruleStart := pathStart + errorIdx + len(errorMid)
	failedIdx := strings.Index(msg[ruleStart:], failedOn)
	if failedIdx == -1 {
		return fieldName + ": validation error"
	}

	tagStart := ruleStart + failedIdx + len(failedOn)
	tagEndIdx := strings.Index(msg[tagStart:], tagEnd)
	if tagEndIdx == -1 {
		return fieldName + ": validation error"
	}

	rule := msg[tagStart : tagStart+tagEndIdx]
	return fieldName + ": validation failed on " + rule
}

// Unauthorized 未授权响应 (401)
func Unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, APIResponse{
		Success: false,
		Error: &ErrorDetail{
			Code:    "UNAUTHORIZED_401",
			Message: "UNAUTHORIZED",
		},
	})
}

// Forbidden 禁止访问响应 (403)
func Forbidden(c *gin.Context) {
	c.JSON(http.StatusForbidden, APIResponse{
		Success: false,
		Error: &ErrorDetail{
			Code:    "FORBIDDEN_403",
			Message: "FORBIDDEN",
		},
	})
}

// NotFound 资源未找到响应 (404)
func NotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, APIResponse{
		Success: false,
		Error: &ErrorDetail{
			Code:    "NOT_FOUND_404",
			Message: "NOT_FOUND",
		},
	})
}
