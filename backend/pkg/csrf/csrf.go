package csrf

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid CSRF token")
	ErrExpiredToken = errors.New("expired CSRF token")
)

// CSRF Manager
type Manager struct {
	tokens map[string]*tokenInfo
	mu     sync.RWMutex
}

type tokenInfo struct {
	createdAt time.Time
	expiresAt time.Time
}

// NewCSRFManager 创建 CSRF 管理器
func NewCSRFManager() *Manager {
	m := &Manager{
		tokens: make(map[string]*tokenInfo),
	}
	// 启动清理 goroutine
	go m.cleanupExpiredTokens()
	return m
}

// GenerateToken 生成 CSRF token
func (m *Manager) GenerateToken() (string, error) {
	// 生成 32 字节随机数
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	// Base64 编码
	token := base64.URLEncoding.EncodeToString(b)

	// 存储 token 信息
	m.mu.Lock()
	m.tokens[token] = &tokenInfo{
		createdAt: time.Now(),
		expiresAt: time.Now().Add(1 * time.Hour), // 1 小时过期
	}
	m.mu.Unlock()

	return token, nil
}

// ValidateToken 验证 CSRF token
func (m *Manager) ValidateToken(token string) error {
	m.mu.RLock()
	info, exists := m.tokens[token]
	m.mu.RUnlock()

	if !exists {
		return ErrInvalidToken
	}

	if time.Now().After(info.expiresAt) {
		m.mu.Lock()
		delete(m.tokens, token)
		m.mu.Unlock()
		return ErrExpiredToken
	}

	return nil
}

// RevokeToken 撤销 token
func (m *Manager) RevokeToken(token string) {
	m.mu.Lock()
	delete(m.tokens, token)
	m.mu.Unlock()
}

// cleanupExpiredTokens 定期清理过期 token
func (m *Manager) cleanupExpiredTokens() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for token, info := range m.tokens {
			if now.After(info.expiresAt) {
				delete(m.tokens, token)
			}
		}
		m.mu.Unlock()
	}
}

// 全局 CSRF 管理器实例
var DefaultManager = NewCSRFManager()
