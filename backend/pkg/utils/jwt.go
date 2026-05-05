package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/user-system/backend/internal/config"
)

type Claims struct {
	UserID     uint   `json:"user_id"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	JTI        string `json:"jti"`
	TokenType  string `json:"token_type,omitempty"`
	Scope      string `json:"scope,omitempty"`
	ClientID   string `json:"client_id,omitempty"`
	RememberMe bool   `json:"remember_me,omitempty"`
	jwt.RegisteredClaims
}

// keyEntry 用于支持多 Key 验证（密钥轮换场景）
type keyEntry struct {
	secret []byte
	kid    string
}

// jwtKeyManager 管理签名密钥，支持轮换期间的双 Key 验证
var (
	jwtCurrentKey  keyEntry
	jwtPreviousKey *keyEntry // 轮换后保留旧 key 用于验证
	jwtKeyMu       sync.RWMutex
	jwtKeyInit     sync.Once
)

func initKeys() {
	jwtKeyInit.Do(func() {
		cfg := config.Get()
		secret := []byte(cfg.JWT.Secret)
		jwtCurrentKey = keyEntry{secret: secret, kid: "key-1"}
	})
}

func getCurrentKey() keyEntry {
	jwtKeyInit.Do(func() {
		cfg := config.Get()
		secret := []byte(cfg.JWT.Secret)
		jwtCurrentKey = keyEntry{secret: secret, kid: "key-1"}
	})
	return jwtCurrentKey
}

// RotateJWTSecret 密钥轮换：新 secret 立即用于签名，旧 secret 仍可用于验证
func RotateJWTSecret(newSecret, newKid string) {
	jwtKeyMu.Lock()
	defer jwtKeyMu.Unlock()

	old := getCurrentKey()
	jwtPreviousKey = &keyEntry{
		secret: make([]byte, len(old.secret)),
		kid:    old.kid,
	}
	copy(jwtPreviousKey.secret, old.secret)

	jwtCurrentKey = keyEntry{
		secret: []byte(newSecret),
		kid:    newKid,
	}
	_ = jwtCurrentKey // 确保 keyInit 不再覆盖
}

func generateJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func GenerateToken(userID uint, username, email string) (string, *Claims, error) {
	return generateToken(userID, username, email, time.Hour, "access", false)
}

func GenerateRefreshToken(userID uint, username, email string) (string, *Claims, error) {
	return generateToken(userID, username, email, 720*time.Hour, "refresh", false)
}

func GenerateTokenWithRememberMe(userID uint, username, email string, rememberMe bool) (string, *Claims, error) {
	return generateToken(userID, username, email, time.Hour, "access", rememberMe)
}

func GenerateRefreshTokenWithRememberMe(userID uint, username, email string, rememberMe bool) (string, *Claims, error) {
	return generateToken(userID, username, email, 720*time.Hour, "refresh", rememberMe)
}

func GenerateTokenWithExpiry(userID uint, username, email string, expiry time.Duration) (string, *Claims, error) {
	return generateToken(userID, username, email, expiry, "access", false)
}

func GenerateRefreshTokenWithExpiry(userID uint, username, email string, expiry time.Duration) (string, *Claims, error) {
	return generateToken(userID, username, email, expiry, "refresh", false)
}

func GenerateOAuthToken(userID uint, username, email, scope, clientID string) (string, *Claims, error) {
	return generateOAuthToken(userID, username, email, time.Hour, scope, clientID)
}

func generateToken(userID uint, username, email string, expiry time.Duration, tokenType string, rememberMe bool) (string, *Claims, error) {
	jti, err := generateJTI()
	if err != nil {
		return "", nil, err
	}

	jwtKeyMu.RLock()
	key := getCurrentKey()
	jwtKeyMu.RUnlock()

	claims := Claims{
		UserID:     userID,
		Username:   username,
		Email:      email,
		JTI:        jti,
		TokenType:  tokenType,
		RememberMe: rememberMe,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "user-system",
			Subject:   fmt.Sprintf("%d", userID),
			Audience:  []string{"user-system-api"},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = key.kid
	tokenStr, err := token.SignedString(key.secret)
	if err != nil {
		return "", nil, err
	}
	return tokenStr, &claims, nil
}

func generateOAuthToken(userID uint, username, email string, expiry time.Duration, scope, clientID string) (string, *Claims, error) {
	jti, err := generateJTI()
	if err != nil {
		return "", nil, err
	}

	jwtKeyMu.RLock()
	key := getCurrentKey()
	jwtKeyMu.RUnlock()

	claims := Claims{
		UserID:    userID,
		Username:  username,
		Email:     email,
		JTI:       jti,
		TokenType: "access",
		Scope:     scope,
		ClientID:  clientID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "user-system-oauth",
			Subject:   fmt.Sprintf("%d", userID),
			Audience:  []string{clientID},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = key.kid
	tokenStr, err := token.SignedString(key.secret)
	if err != nil {
		return "", nil, err
	}
	return tokenStr, &claims, nil
}

// resolveSecret 根据 kid 查找对应的签名密钥
func resolveSecret(kid string) []byte {
	jwtKeyMu.RLock()
	defer jwtKeyMu.RUnlock()

	current := getCurrentKey()
	if kid == "" || kid == current.kid {
		return current.secret
	}

	if jwtPreviousKey != nil && kid == jwtPreviousKey.kid {
		return jwtPreviousKey.secret
	}

	return current.secret
}

func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		kid, _ := token.Header["kid"].(string)
		return resolveSecret(kid), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
