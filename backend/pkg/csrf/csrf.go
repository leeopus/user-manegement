package csrf

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/user-system/backend/internal/config"
)

const (
	csrfTokenPrefix      = "csrf:tokens:"
	csrfSessionPrefix    = "csrf:session:"
	csrfSessionMaxTokens = 5
	csrfOpTimeout        = 2 * time.Second
)

func getCSRFTokenTTL() time.Duration {
	cfg := config.Get()
	if cfg != nil && cfg.Security.CSRFTokenTTLMin > 0 {
		return time.Duration(cfg.Security.CSRFTokenTTLMin) * time.Minute
	}
	return 30 * time.Minute
}

var (
	ErrInvalidToken = errors.New("invalid CSRF token")
	ErrExpiredToken = errors.New("expired CSRF token")
)

type Manager struct {
	redis *redis.Client
}

func NewCSRFManager(redisClient *redis.Client) *Manager {
	return &Manager{redis: redisClient}
}

func (m *Manager) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), csrfOpTimeout)
}

func tokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// incrAndCapSessionTokens 原子性递增并限制 session token 计数（Lua 脚本避免竞态）
var incrAndCapSessionTokens = redis.NewScript(`
	local key = KEYS[1]
	local max_tokens = tonumber(ARGV[1])
	local ttl = tonumber(ARGV[2])

	local count = redis.call("INCR", key)
	redis.call("EXPIRE", key, ttl)

	if count > max_tokens then
		redis.call("SET", key, tostring(max_tokens), "EX", ttl)
	end

	return count
`)
func (m *Manager) GenerateToken(sessionID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	token := base64.URLEncoding.EncodeToString(b)
	hash := tokenHash(token)

	if m.redis != nil {
		ctx, cancel := m.ctx()
		defer cancel()

		key := fmt.Sprintf("%s%s", csrfTokenPrefix, hash)
		if err := m.redis.Set(ctx, key, sessionID, getCSRFTokenTTL()).Err(); err != nil {
			return "", err
		}

		sessionKey := fmt.Sprintf("%s%s", csrfSessionPrefix, sessionID)
		ctx2, cancel2 := m.ctx()
		defer cancel2()
		incrAndCapSessionTokens.Run(ctx2, m.redis, []string{sessionKey}, csrfSessionMaxTokens, int(getCSRFTokenTTL().Seconds()))
	}

	return token, nil
}

// ValidateToken 验证 CSRF token
func (m *Manager) ValidateToken(token, sessionID string) error {
	if token == "" {
		return ErrInvalidToken
	}

	if m.redis == nil {
		return ErrInvalidToken
	}

	ctx, cancel := m.ctx()
	defer cancel()

	key := fmt.Sprintf("%s%s", csrfTokenPrefix, tokenHash(token))

	storedSession, err := m.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrInvalidToken
		}
		return err
	}

	if storedSession != sessionID {
		return ErrInvalidToken
	}

	return nil
}

// RevokeToken 撤销 token
func (m *Manager) RevokeToken(token string) {
	if m.redis == nil || token == "" {
		return
	}
	ctx, cancel := m.ctx()
	defer cancel()

	key := fmt.Sprintf("%s%s", csrfTokenPrefix, tokenHash(token))
	m.redis.Del(ctx, key)
}
