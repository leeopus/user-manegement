package email

import (
	"log"
)

// EmailService 邮件服务接口
type EmailService interface {
	SendPasswordResetEmail(email, resetLink string) error
	SendPasswordChangedNotification(email string) error
}

// developmentEmailService 开发环境邮件服务（控制台输出）
type developmentEmailService struct{}

// NewDevelopmentEmailService 创建开发环境邮件服务
func NewDevelopmentEmailService() EmailService {
	return &developmentEmailService{}
}

func (s *developmentEmailService) SendPasswordResetEmail(email, resetLink string) error {
	log.Println("=====================================")
	log.Println("📧 密码重置邮件（开发环境）")
	log.Println("=====================================")
	log.Printf("收件人: %s", email)
	log.Printf("重置链接: %s", resetLink)
	log.Println("此链接将在 15 分钟后过期")
	log.Println("如果这不是您的操作，请忽略此邮件")
	log.Println("=====================================")
	return nil
}

func (s *developmentEmailService) SendPasswordChangedNotification(email string) error {
	log.Println("=====================================")
	log.Println("📧 密码已更改通知（开发环境）")
	log.Println("=====================================")
	log.Printf("收件人: %s", email)
	log.Println("您的密码已成功更改")
	log.Println("如果这不是您的操作，请立即联系支持")
	log.Println("=====================================")
	return nil
}

// TODO: 生产环境可以添加真实的邮件服务
// 例如使用 SMTP、SendGrid、AWS SES 等
