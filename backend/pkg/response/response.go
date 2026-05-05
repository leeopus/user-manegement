package response

import (
	"net/http"
	"strconv"

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
	Code    string                 `json:"code"`              // 错误码
	Message string                 `json:"message"`           // 翻译键
	Details map[string]interface{} `json:"details,omitempty"` // 额外详情
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

// Error 错误响应（使用 AppError）
func Error(c *gin.Context, err error) {
	// 检查是否为应用错误
	if appErr, ok := apperrors.IsAppError(err); ok {
		status := appErr.HTTPStatus
		if status == 0 {
			status = http.StatusInternalServerError
		}

		c.JSON(status, APIResponse{
			Success: false,
			Error: &ErrorDetail{
				Code:    appErr.Code,
				Message: appErr.MessageKey,
				Details: appErr.Details,
			},
		})
		return
	}

	// 未知错误，返回通用错误
	c.JSON(http.StatusInternalServerError, APIResponse{
		Success: false,
		Error: &ErrorDetail{
			Code:    apperrors.ErrInternalServer.Code,
			Message: apperrors.ErrInternalServer.MessageKey,
		},
	})
}

// ValidationError 验证错误响应（用于请求参数验证）
func ValidationError(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, APIResponse{
		Success: false,
		Error: &ErrorDetail{
			Code:    "VALIDATION_ERROR_400",
			Message: "VALIDATION_ERROR",
			Details: map[string]interface{}{
				"message": message,
			},
		},
	})
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
