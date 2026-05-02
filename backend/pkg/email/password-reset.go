package email

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// PasswordResetToken 密码重置令牌
type PasswordResetToken struct {
	Token     string
	Email     string
	ExpiresAt time.Time
}

// GenerateResetToken 生成密码重置令牌
func GenerateResetToken() (string, error) {
	// 生成 32 字节随机数
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	// Base64 URL 编码
	return base64.URLEncoding.EncodeToString(b), nil
}

// SendPasswordResetEmail 发送密码重置邮件
func SendPasswordResetEmail(email, token string) error {
	// TODO: 集成真实的邮件服务（SMTP）
	// 这里使用模拟实现

	resetLink := fmt.Sprintf("http://localhost:3000/reset-password?token=%s", token)

	// 模拟发送邮件
	fmt.Printf(`
=====================================
密码重置邮件
=====================================
收件人: %s

重置链接: %s

此链接将在 1 小时后过期。

如果您没有请求密码重置，请忽略此邮件。
=====================================
`, email, resetLink)

	return nil
}

// SendPasswordChangedNotification 发送密码已更改通知
func SendPasswordChangedNotification(email string) error {
	// TODO: 集成真实的邮件服务
	fmt.Printf(`
=====================================
密码已更改通知
=====================================
收件人: %s

您的账户密码已成功更改。

如果这不是您本人操作，请立即联系支持团队。
=====================================
`, email)

	return nil
}
