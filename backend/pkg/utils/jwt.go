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
	jwtPreviousKey *keyEntry
	jwtKeyMu       sync.RWMutex
	jwtKeyInit     sync.Once
)

// initJWTKey 确保密钥只初始化一次，使用写锁保护
func initJWTKey() {
	jwtKeyMu.Lock()
	defer jwtKeyMu.Unlock()

	jwtKeyInit.Do(func() {
		cfg := config.Get()
		secret := []byte(cfg.JWT.Secret)
		jwtCurrentKey = keyEntry{secret: secret, kid: "key-1"}
	})
}

func getCurrentKey() keyEntry {
	initJWTKey()

	jwtKeyMu.RLock()
	defer jwtKeyMu.RUnlock()
	return jwtCurrentKey
}

// RotateJWTSecret 密钥轮换：新 secret 立即用于签名，旧 secret 仍可用于验证
func RotateJWTSecret(newSecret, newKid string) {
	initJWTKey()

	jwtKeyMu.Lock()
	defer jwtKeyMu.Unlock()

	old := jwtCurrentKey
	jwtPreviousKey = &keyEntry{
		secret: make([]byte, len(old.secret)),
		kid:    old.kid,
	}
	copy(jwtPreviousKey.secret, old.secret)

	jwtCurrentKey = keyEntry{
		secret: []byte(newSecret),
		kid:    newKid,
	}
}

func generateJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func getAccessTokenTTL() time.Duration {
	cfg := config.Get()
	if cfg != nil && cfg.Security.AccessTokenMaxTTLMin > 0 {
		return time.Duration(cfg.Security.AccessTokenMaxTTLMin) * time.Minute
	}
	return 15 * time.Minute
}

func getRefreshTokenTTL() time.Duration {
	cfg := config.Get()
	if cfg != nil {
		return cfg.GetRefreshTokenTTL()
	}
	return 30 * 24 * time.Hour
}

func GenerateToken(userID uint, username, email string) (string, *Claims, error) {
	return generateToken(userID, username, email, getAccessTokenTTL(), "access", false)
}

func GenerateRefreshToken(userID uint, username, email string) (string, *Claims, error) {
	return generateToken(userID, username, email, getRefreshTokenTTL(), "refresh", false)
}

func GenerateTokenWithRememberMe(userID uint, username, email string, rememberMe bool) (string, *Claims, error) {
	return generateToken(userID, username, email, getAccessTokenTTL(), "access", rememberMe)
}

func GenerateRefreshTokenWithRememberMe(userID uint, username, email string, rememberMe bool) (string, *Claims, error) {
	return generateToken(userID, username, email, getRefreshTokenTTL(), "refresh", rememberMe)
}

func GenerateTokenWithExpiry(userID uint, username, email string, expiry time.Duration) (string, *Claims, error) {
	return generateToken(userID, username, email, expiry, "access", false)
}

func GenerateRefreshTokenWithExpiry(userID uint, username, email string, expiry time.Duration) (string, *Claims, error) {
	return generateToken(userID, username, email, expiry, "refresh", false)
}

func GenerateOAuthToken(userID uint, username, email, scope, clientID string) (string, *Claims, error) {
	return generateOAuthToken(userID, username, email, getAccessTokenTTL(), scope, clientID)
}

func generateToken(userID uint, username, email string, expiry time.Duration, tokenType string, rememberMe bool) (string, *Claims, error) {
	jti, err := generateJTI()
	if err != nil {
		return "", nil, err
	}

	key := getCurrentKey()

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

	key := getCurrentKey()

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
	initJWTKey()

	jwtKeyMu.RLock()
	defer jwtKeyMu.RUnlock()

	if kid == "" || kid == jwtCurrentKey.kid {
		return jwtCurrentKey.secret
	}

	if jwtPreviousKey != nil && kid == jwtPreviousKey.kid {
		return jwtPreviousKey.secret
	}

	return jwtCurrentKey.secret
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
