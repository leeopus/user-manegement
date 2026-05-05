package utils

import (
	"crypto/rand"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

// HashSecret 使用 bcrypt 对 secret 进行哈希（用于 OAuth client secret 等）
func HashSecret(secret string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifySecret 验证 secret 是否匹配 bcrypt 哈希
func VerifySecret(secret, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret)) == nil
}

// GenerateRandomString 生成指定长度的随机十六进制字符串
func GenerateRandomString(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
