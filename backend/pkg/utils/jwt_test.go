package utils

import (
	"testing"
	"time"
)

func TestGenerateAndParseToken(t *testing.T) {
	setupTestConfig(t)

	userID := uint(1)
	username := "testuser"
	email := "test@example.com"

	token, claims, err := GenerateToken(userID, username, email)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if token == "" {
		t.Fatal("GenerateToken() returned empty token")
	}

	if claims.UserID != userID {
		t.Errorf("claims.UserID = %d, want %d", claims.UserID, userID)
	}

	if claims.TokenType != "access" {
		t.Errorf("claims.TokenType = %s, want access", claims.TokenType)
	}

	// Parse the token back
	parsedClaims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}

	if parsedClaims.UserID != userID {
		t.Errorf("parsedClaims.UserID = %d, want %d", parsedClaims.UserID, userID)
	}

	if parsedClaims.Username != username {
		t.Errorf("parsedClaims.Username = %s, want %s", parsedClaims.Username, username)
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	setupTestConfig(t)

	userID := uint(42)
	_, claims, err := GenerateRefreshToken(userID, "user", "user@example.com")
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	if claims.TokenType != "refresh" {
		t.Errorf("claims.TokenType = %s, want refresh", claims.TokenType)
	}

	if claims.ExpiresAt == nil {
		t.Fatal("claims.ExpiresAt is nil")
	}
}

func TestParseInvalidToken(t *testing.T) {
	setupTestConfig(t)

	_, err := ParseToken("invalid-token-string")
	if err == nil {
		t.Error("ParseToken() should return error for invalid token")
	}

	_, err = ParseToken("")
	if err == nil {
		t.Error("ParseToken() should return error for empty token")
	}
}

func TestParseRefreshTokenType(t *testing.T) {
	setupTestConfig(t)

	_, claims, err := GenerateRefreshToken(1, "user", "user@example.com")
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	// The token parses but has type "refresh" - middleware should check this
	if claims.TokenType != "refresh" {
		t.Errorf("claims.TokenType = %s, want refresh", claims.TokenType)
	}
}

func TestGenerateOAuthToken(t *testing.T) {
	setupTestConfig(t)

	_, claims, err := GenerateOAuthToken(1, "user", "user@example.com", "read write", "client_abc")
	if err != nil {
		t.Fatalf("GenerateOAuthToken() error = %v", err)
	}

	if claims.Scope != "read write" {
		t.Errorf("claims.Scope = %s, want 'read write'", claims.Scope)
	}

	if claims.ClientID != "client_abc" {
		t.Errorf("claims.ClientID = %s, want 'client_abc'", claims.ClientID)
	}

	if claims.Issuer != "user-system-oauth" {
		t.Errorf("claims.Issuer = %s, want 'user-system-oauth'", claims.Issuer)
	}
}

func TestGenerateTokenWithExpiry(t *testing.T) {
	setupTestConfig(t)

	customExpiry := 5 * time.Minute
	_, claims, err := GenerateTokenWithExpiry(1, "user", "user@example.com", customExpiry)
	if err != nil {
		t.Fatalf("GenerateTokenWithExpiry() error = %v", err)
	}

	if claims.ExpiresAt == nil {
		t.Fatal("claims.ExpiresAt is nil")
	}

	expiresIn := time.Until(claims.ExpiresAt.Time)
	if expiresIn < 4*time.Minute || expiresIn > 6*time.Minute {
		t.Errorf("token expires in %v, expected approximately 5 minutes", expiresIn)
	}
}

func TestTokenJTI(t *testing.T) {
	setupTestConfig(t)

	_, claims1, _ := GenerateToken(1, "user", "user@example.com")
	_, claims2, _ := GenerateToken(1, "user", "user@example.com")

	if claims1.JTI == "" {
		t.Error("JTI is empty")
	}

	if claims1.JTI == claims2.JTI {
		t.Error("Two tokens have the same JTI")
	}
}
