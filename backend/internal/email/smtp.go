package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// SMTPConfig SMTP配置
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
	SSL      bool
}

// GetSMTPConfig 从环境变量获取SMTP配置
func GetSMTPConfig() *SMTPConfig {
	// 尝试加载.env文件
	_ = godotenv.Load(".env")
	_ = godotenv.Load(".env.smtp") // 备用配置文件

	return &SMTPConfig{
		Host:     getEnv("SMTP_HOST", "smtp.qq.com"),
		Port:     getEnvInt("SMTP_PORT", 465),
		Username: getEnv("SMTP_USER", ""),
		Password: getEnv("SMTP_PASSWORD", ""),
		From:     getEnv("SMTP_FROM", ""),
		FromName: getEnv("SMTP_FROM_NAME", "用户管理系统"),
		SSL:      getEnvBool("SMTP_SSL", true),
	}
}

// smtpEmailService SMTP邮件服务
type smtpEmailService struct {
	config *SMTPConfig
}

// NewSMTPEmailService 创建SMTP邮件服务
func NewSMTPEmailService(config *SMTPConfig) EmailService {
	return &smtpEmailService{config: config}
}

func (s *smtpEmailService) SendPasswordResetEmail(email, resetLink string) error {
	if s.config == nil || s.config.Host == "" {
		// 如果没有配置SMTP，回退到开发环境
		fmt.Println("警告：SMTP未配置，使用开发环境邮件服务")
		return (&developmentEmailService{}).SendPasswordResetEmail(email, resetLink)
	}

	subject := "密码重置请求"
	body := fmt.Sprintf(`尊敬的用户：

您请求了密码重置。请点击以下链接重置您的密码：

%s

此链接将在 15 分钟后过期。

如果这不是您的操作，请忽略此邮件。

祝好，
用户管理系统团队`, resetLink)

	return s.sendEmail(email, subject, body)
}

func (s *smtpEmailService) SendPasswordChangedNotification(email string) error {
	if s.config == nil || s.config.Host == "" {
		fmt.Println("警告：SMTP未配置，使用开发环境邮件服务")
		return (&developmentEmailService{}).SendPasswordChangedNotification(email)
	}

	subject := "密码已更改通知"
	body := `尊敬的用户：

您的账户密码已成功更改。

如果这不是您的操作，请立即联系我们的支持团队。

祝好，
用户管理系统团队`

	return s.sendEmail(email, subject, body)
}

func (s *smtpEmailService) SendEmailVerificationEmail(email, verificationLink string) error {
	if s.config == nil || s.config.Host == "" {
		fmt.Println("警告：SMTP未配置，使用开发环境邮件服务")
		return (&developmentEmailService{}).SendEmailVerificationEmail(email, verificationLink)
	}

	subject := "邮箱验证"
	body := fmt.Sprintf(`尊敬的用户：

请点击以下链接验证您的邮箱地址：

%s

此链接将在 24 小时后过期。

如果这不是您的操作，请忽略此邮件。

祝好，
用户管理系统团队`, verificationLink)

	return s.sendEmail(email, subject, body)
}

func (s *smtpEmailService) sendEmail(to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") {
		return fmt.Errorf("invalid characters in email address")
	}
	if strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("invalid characters in email subject")
	}

	// 构建邮件内容
	message := fmt.Sprintf("From: %s <%s>\r\n", s.config.FromName, s.config.From)
	message += fmt.Sprintf("To: %s\r\n", to)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-version: 1.0;\r\nContent-Type: text/plain; charset=\"UTF-8\";\r\n\r\n"
	message += body

	// SMTP地址
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// 认证信息
	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)

	var err error
	if s.config.SSL {
		// 使用SSL加密连接
		err = s.sendWithSSL(addr, auth, s.config.From, []string{to}, []byte(message))
	} else {
		// 普通连接或TLS
		err = smtp.SendMail(addr, auth, s.config.From, []string{to}, []byte(message))
	}

	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// sendWithSSL 使用SSL发送邮件
func (s *smtpEmailService) sendWithSSL(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// 创建TLS配置
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         s.config.Host,
	}

	// 建立连接
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// 创建SMTP客户端
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// 认证
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	// 设置发件人
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("mail from error: %w", err)
	}

	// 设置收件人
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("rcpt to error: %w", err)
		}
	}

	// 发送邮件内容
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("data error: %w", err)
	}
	defer writer.Close()

	_, err = writer.Write(msg)
	if err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	return nil
}

// 环境变量辅助函数
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var result int
	fmt.Sscanf(value, "%d", &result)
	return result
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1"
}
