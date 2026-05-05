package utils

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
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

// 默认密码策略（安全模式 - 要求字符多样性）
var DefaultPasswordPolicy = PasswordPolicy{
	MinLength:      8,
	MaxLength:      64,
	RequireUpper:   true,
	RequireLower:   true,
	RequireNumber:  true,
	RequireSpecial: false,
	ForbidUsername: true,
}

// 邮箱验证正则（预编译，避免每次调用重新编译）
var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

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
		return PasswordWeak, errors.New("validation.password.minLength")
	}
	if len(password) > policy.MaxLength {
		return PasswordWeak, errors.New("validation.password.maxLength")
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
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasNumber = true
		default:
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
			return PasswordWeak, errors.New("validation.password.tooWeak")
		}
	}

	// 检查是否全是相同字符
	if allSame(password) {
		return PasswordWeak, errors.New("validation.password.sameChars")
	}

	// 强制字符类型要求（根据策略）
	if policy.RequireUpper && !hasUpper {
		return PasswordWeak, errors.New("validation.password.requireUpper")
	}
	if policy.RequireLower && !hasLower {
		return PasswordWeak, errors.New("validation.password.requireLower")
	}
	if policy.RequireNumber && !hasNumber {
		return PasswordWeak, errors.New("validation.password.requireNumber")
	}
	if policy.RequireSpecial && !hasSpecial {
		return PasswordWeak, errors.New("validation.password.requireSpecial")
	}
	if policy.ForbidUsername && username != "" && strings.Contains(strings.ToLower(password), strings.ToLower(username)) {
		return PasswordWeak, errors.New("validation.password.containsUsername")
	}

	// 计算强度等级
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
		return errors.New("validation.username.minLength")
	}
	if len(username) > policy.MaxLength {
		return errors.New("validation.username.maxLength")
	}

	// 检查格式
	if !policy.Pattern.MatchString(username) {
		return errors.New("validation.username.pattern")
	}

	// 检查保留用户名
	lowerUsername := strings.ToLower(username)
	for _, reserved := range policy.Reserved {
		if lowerUsername == reserved {
			return errors.New("validation.username.reserved")
		}
	}

	// 检查连续特殊字符
	if strings.Contains(username, "--") || strings.Contains(username, "__") {
		return errors.New("validation.username.consecutive")
	}

	return nil
}

// ValidateEmail 验证邮箱格式（RFC 5322 简化版）
func ValidateEmail(email string) error {
	if email == "" {
		return errors.New("validation.email.required")
	}

	// 基本格式验证（使用预编译正则）
	if !emailPattern.MatchString(email) {
		return errors.New("validation.email.invalid")
	}

	// 检查长度
	if len(email) > 254 {
		return errors.New("validation.email.tooLong")
	}

	// 检查本地部分和域名部分
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return errors.New("validation.email.invalid")
	}

	local := parts[0]
	domain := parts[1]

	if len(local) == 0 || len(local) > 64 {
		return errors.New("validation.email.invalid")
	}

	if len(domain) < 3 || len(domain) > 255 {
		return errors.New("validation.email.invalid")
	}

	// 检查域名中是否有点
	if !strings.Contains(domain, ".") {
		return errors.New("validation.email.invalid")
	}

	return nil
}

// IsDisposableEmail 检查是否为一次性邮箱
func IsDisposableEmail(email string) bool {
	domain := strings.ToLower(strings.Split(email, "@")[1])

	for _, d := range disposableDomains {
		if domain == d || strings.HasSuffix(domain, "."+d) {
			return true
		}
	}

	return false
}

// disposableDomains 常见一次性邮箱域名黑名单
var disposableDomains = []string{
	// 10minutemail 系列
	"10minutemail.com", "10minutemail.net", "10minutemail.info",
	// guerrillamail 系列
	"guerrillamail.com", "guerrillamail.info", "guerrillamail.net",
	"guerrillamail.org", "guerrillamailblock.com", "grr.la",
	"guerrillamail.de", "guerrillamailblock.com",
	// tempmail 系列
	"temp-mail.com", "temp-mail.org", "tempmail.com", "tempmail.org",
	"tempmail.net", "tempmailaddress.com", "tempmailo.com",
	// mailinator 系列
	"mailinator.com", "mailinator.net", "mailinator.org",
	"manybrain.com", "msgos.com", "notmailinator.com",
	// mailnesia
	"mailnesia.com", "mailnesia.net",
	// yopmail 系列
	"yopmail.com", "yopmail.fr", "yopmail.net", "yopmail.org",
	"yopmail.com", "yopmail.gq", "yopmail.ml",
	// trashmail 系列
	"trashmail.com", "trashmail.net", "trashmail.org",
	"trashmail.de", "trash-mail.com", "trashmail.ws",
	// throwaway
	"throwaway.email", "throwawaymail.com", "throwaway.emailaddress.org",
	// 其他常见
	"fakeinbox.com", "maildrop.cc", "maildrop.xyz",
	"dispostable.com", "mailcatch.com", "mailexpire.com",
	"mytemp.email", "mytempemail.com", "tempinbox.com",
	"mailscrap.com", "mailinater.com", "messagebeamer.de",
	"recyclemail.dk", "sharklasers.com", "spamavert.com",
	"uggsrock.com", "mailnesia.com", "getairmail.com",
	" guerrillamail.biz", "harakirimail.com", "meltmail.com",
	"mintemail.com", "safetymail.info", "safetypost.de",
	"spamfree24.org", "squizzy.de", "uggsrock.com",
	"mailnull.com", "nomail.xl.cx", "throwam.com",
	"tempail.com", "tempr.email", "disposableemailaddresses.emailmiser.com",
	"mailmoat.com", "nomail2me.com", "throwawayemailaddress.com",
	"ephemail.net", "emailisvalid.com", "mailexpire.com",
	"jetable.org", "jetable.com", "mailforspam.com",
	"s0ny.net", "safetymail.info", "filzmail.com",
	"incognitomail.org", "incognitomail.com", "incognitomail.net",
	"mailblocks.com", "mailme.lv", "mailshell.com",
	"meltmail.com", "safetypost.de", "smapfree24.org",
	"spamavert.com", "spammotel.com", "trashymail.com",
	"mailexpire.com", "mintemail.com", "quickinbox.com",
	"receiveee.com", "tmail.ws", "tempinbox.com",
	"trashmail.io", "mohmal.com", "burpcollaborator.net",
	"guerrillamailblock.com", "pokemail.net", "sharklasers.com",
	"spam4.me", "trbvm.com", "txen.de", "yert.ye",
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

// RandomSuffix 生成指定长度的随机数字后缀（使用 crypto/rand）
func RandomSuffix(length int) (string, error) {
	const digits = "0123456789"
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", fmt.Errorf("crypto/rand failed: %w", err)
		}
		result[i] = digits[n.Int64()]
	}
	return string(result), nil
}

// isAlpha 检查字符是否为字母
func isAlpha(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// SanitizeHTML 对字符串进行 HTML 实体转义，防止 XSS
func SanitizeHTML(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 16)
	for _, c := range s {
		switch c {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&#x27;")
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}

// SanitizeInput 对用户输入进行消毒：去除首尾空白、控制字符，转义 HTML
func SanitizeInput(s string) string {
	// 去除首尾空白
	s = strings.TrimSpace(s)
	// 移除控制字符（保留换行和制表符）
	var b strings.Builder
	b.Grow(len(s))
	for _, c := range s {
		if c == '\n' || c == '\t' || c == '\r' {
			continue
		}
		if c < 32 {
			continue
		}
		b.WriteRune(c)
	}
	return b.String()
}
