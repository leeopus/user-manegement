package utils

import (
	"errors"
	"math/rand"
	"regexp"
	"strings"
	"unicode"
)

// 密码强度等级
type PasswordStrength int

const (
	PasswordWeak   PasswordStrength = 0
	PasswordFair   PasswordStrength = 1
	PasswordGood   PasswordStrength = 2
	PasswordStrong PasswordStrength = 3
)

// 密码策略配置
type PasswordPolicy struct {
	MinLength       int
	MaxLength       int
	RequireUpper    bool
	RequireLower    bool
	RequireNumber   bool
	RequireSpecial  bool
	ForbidUsername  bool
}

// 默认密码策略（平衡模式 - 降低门槛）
var DefaultPasswordPolicy = PasswordPolicy{
	MinLength:      8,
	MaxLength:      64,
	RequireUpper:   false,
	RequireLower:   false,
	RequireNumber:  false,
	RequireSpecial: false,
	ForbidUsername: false,
}

// 常见弱密码列表（示例）
var CommonPasswords = []string{
	"password", "12345678", "123456789", "qwerty", "abc123",
	"monkey", "1234567890", "password1", "123123", "qwerty123",
	"password123", "admin123", "welcome1", "login123", "passw0rd",
}

// ValidatePassword 验证密码强度和规则（平衡模式）
func ValidatePassword(password, username string) (PasswordStrength, error) {
	policy := DefaultPasswordPolicy

	// 检查长度（硬性要求）
	if len(password) < policy.MinLength {
		return PasswordWeak, errors.New("密码至少8位")
	}
	if len(password) > policy.MaxLength {
		return PasswordWeak, errors.New("密码最多64位")
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
		score      int
	)

	// 检查字符类型
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	// 根据长度评分（更重视长度）
	if len(password) >= 16 {
		score += 4
	} else if len(password) >= 12 {
		score += 3
	} else if len(password) >= 8 {
		score += 1
	}

	// 根据字符类型评分（鼓励但不强制）
	if hasLower {
		score += 1
	}
	if hasUpper {
		score += 1
	}
	if hasNumber {
		score += 1
	}
	if hasSpecial {
		score += 1
	}

	// 只检查常见弱密码和明显安全问题
	for _, common := range CommonPasswords {
		if strings.EqualFold(password, common) {
			return PasswordWeak, errors.New("密码过于简单，请使用更复杂的密码")
		}
	}

	// 检查是否全是相同字符
	if allSame(password) {
		return PasswordWeak, errors.New("密码不能全是相同字符")
	}

	// 计算强度等级（更宽松）
	var strength PasswordStrength
	if score <= 2 {
		strength = PasswordWeak
	} else if score <= 4 {
		strength = PasswordFair
	} else if score <= 6 {
		strength = PasswordGood
	} else {
		strength = PasswordStrong
	}

	return strength, nil
}

// allSame 检查字符串是否全部由相同字符组成
func allSame(s string) bool {
	if len(s) == 0 {
		return false
	}
	first := s[0]
	for i := 1; i < len(s); i++ {
		if s[i] != first {
			return false
		}
	}
	return true
}

// 用户名规则
type UsernamePolicy struct {
	MinLength   int
	MaxLength   int
	Pattern     *regexp.Regexp
	Reserved    []string
}

// 默认用户名策略
var DefaultUsernamePolicy = UsernamePolicy{
	MinLength: 3,
	MaxLength: 32,
	Pattern:   regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9_-]*[a-zA-Z0-9])?$`),
	Reserved: []string{
		"admin", "administrator", "system", "root", "api",
		"www", "mail", "ftp", "localhost", "smtp", "pop",
		"ns1", "ns2", "dns", "host", "webmaster", "support",
		"info", "sales", "marketing", "news", "blog",
		"forum", "community", "help", "docs", "assets",
		"static", "cdn", "media", "images", "img",
	},
}

// ValidateUsername 验证用户名
func ValidateUsername(username string) error {
	policy := DefaultUsernamePolicy

	// 检查长度
	if len(username) < policy.MinLength {
		return errors.New("用户名至少3位")
	}
	if len(username) > policy.MaxLength {
		return errors.New("用户名最多32位")
	}

	// 检查格式
	if !policy.Pattern.MatchString(username) {
		return errors.New("用户名只能包含字母、数字、下划线和连字符，且不能以 _ 或 - 开头或结尾")
	}

	// 检查保留用户名
	lowerUsername := strings.ToLower(username)
	for _, reserved := range policy.Reserved {
		if lowerUsername == reserved {
			return errors.New("该用户名不可使用")
		}
	}

	// 检查连续特殊字符
	if strings.Contains(username, "--") || strings.Contains(username, "__") {
		return errors.New("用户名不能连续使用 _ 或 -")
	}

	return nil
}

// ValidateEmail 验证邮箱格式（RFC 5322 简化版）
func ValidateEmail(email string) error {
	if email == "" {
		return errors.New("邮箱不能为空")
	}

	// 基本格式验证
	pattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !pattern.MatchString(email) {
		return errors.New("邮箱格式不正确")
	}

	// 检查长度
	if len(email) > 254 {
		return errors.New("邮箱地址过长")
	}

	// 检查本地部分和域名部分
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return errors.New("邮箱格式不正确")
	}

	local := parts[0]
	domain := parts[1]

	if len(local) == 0 || len(local) > 64 {
		return errors.New("邮箱格式不正确")
	}

	if len(domain) < 3 || len(domain) > 255 {
		return errors.New("邮箱域名不正确")
	}

	// 检查域名中是否有点
	if !strings.Contains(domain, ".") {
		return errors.New("邮箱域名不正确")
	}

	return nil
}

// IsDisposableEmail 检查是否为一次性邮箱
func IsDisposableEmail(email string) bool {
	domain := strings.ToLower(strings.Split(email, "@")[1])

	// 一次性邮箱黑名单（示例）
	disposableDomains := []string{
		"temp-mail.com", "guerrillamail.com", "10minutemail.com",
		"mailinator.com", "throwaway.email", "fakeinbox.com",
	}

	for _, disposable := range disposableDomains {
		if strings.HasSuffix(domain, disposable) {
			return true
		}
	}

	return false
}

// GetPasswordStrengthText 获取密码强度文本
func GetPasswordStrengthText(strength PasswordStrength) string {
	switch strength {
	case PasswordWeak:
		return "弱"
	case PasswordFair:
		return "中等"
	case PasswordGood:
		return "强"
	case PasswordStrong:
		return "很强"
	default:
		return "未知"
	}
}

// GetPasswordStrengthColor 获取密码强度颜色
func GetPasswordStrengthColor(strength PasswordStrength) string {
	switch strength {
	case PasswordWeak:
		return "red"
	case PasswordFair:
		return "yellow"
	case PasswordGood:
		return "green"
	case PasswordStrong:
		return "emerald"
	default:
		return "gray"
	}
}

// GenerateUsernameFromEmail 从email自动生成username
func GenerateUsernameFromEmail(email string) string {
	// 提取@之前的部分
	parts := strings.Split(email, "@")
	if len(parts) == 0 {
		return "user"
	}

	localPart := parts[0]

	// 处理Gmail风格的+号（user+tag@gmail.com -> user）
	if plusIndex := strings.Index(localPart, "+"); plusIndex > 0 {
		localPart = localPart[:plusIndex]
	}

	// 移除点号（user.name@gmail.com -> username）
	localPart = strings.ReplaceAll(localPart, ".", "")

	// 转换为小写
	localPart = strings.ToLower(localPart)

	// 如果为空或太短，使用默认值
	if len(localPart) < 3 {
		return "user"
	}

	// 截断过长的用户名（保留前20个字符）
	if len(localPart) > 20 {
		localPart = localPart[:20]
	}

	// 确保符合用户名格式（字母开头）
	if len(localPart) > 0 && !isAlpha(rune(localPart[0])) {
		localPart = "user_" + localPart
	}

	return localPart
}

// RandomSuffix 生成指定长度的随机数字后缀
func RandomSuffix(length int) string {
	const digits = "0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = digits[rand.Intn(len(digits))]
	}
	return string(result)
}

// isAlpha 检查字符是否为字母
func isAlpha(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
