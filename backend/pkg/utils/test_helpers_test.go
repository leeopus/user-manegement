package utils

import (
	"os"
	"sync"
	"testing"

	"github.com/user-system/backend/internal/config"
)

func setupTestConfig(t *testing.T) {
	t.Helper()

	// Reset the sync.Once so config can be reloaded
	jwtKeyInit = sync.Once{}
	jwtCurrentKey = keyEntry{}
	jwtPreviousKey = nil

	// Set a valid JWT secret for testing
	os.Setenv("JWT_SECRET", "test-secret-that-is-at-least-32-bytes-long-for-testing")
	defer os.Unsetenv("JWT_SECRET")

	config.AppConfig = &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-that-is-at-least-32-bytes-long-for-testing",
		},
		Security: config.SecurityConfig{
			AccessTokenMaxTTLMin: 15,
			RefreshTokenTTLDays: 30,
		},
	}
	// Re-init keys since config is now set
	jwtKeyInit = sync.Once{}
	jwtCurrentKey = keyEntry{}
	jwtPreviousKey = nil

	// Force key initialization
	_ = getCurrentKey()
}
