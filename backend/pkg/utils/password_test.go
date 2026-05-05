package utils

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "TestPassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == "" {
		t.Fatal("HashPassword() returned empty hash")
	}

	if hash == password {
		t.Fatal("HashPassword() returned plaintext")
	}
}

func TestCheckPassword(t *testing.T) {
	password := "TestPassword123"
	hash, _ := HashPassword(password)

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{"correct password", password, hash, true},
		{"wrong password", "WrongPassword123", hash, false},
		{"empty password", "", hash, false},
		{"empty hash", password, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckPassword(tt.password, tt.hash)
			if got != tt.want {
				t.Errorf("CheckPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHashAndVerifySecret(t *testing.T) {
	secret := "my-oauth-client-secret"

	hash, err := HashSecret(secret)
	if err != nil {
		t.Fatalf("HashSecret() error = %v", err)
	}

	if !VerifySecret(secret, hash) {
		t.Error("VerifySecret() = false for correct secret")
	}

	if VerifySecret("wrong-secret", hash) {
		t.Error("VerifySecret() = true for wrong secret")
	}
}

func TestGenerateRandomString(t *testing.T) {
	s1, err := GenerateRandomString(16)
	if err != nil {
		t.Fatalf("GenerateRandomString() error = %v", err)
	}

	if len(s1) != 32 { // hex encoding doubles the length
		t.Errorf("GenerateRandomString(16) returned %d chars, want 32", len(s1))
	}

	s2, _ := GenerateRandomString(16)
	if s1 == s2 {
		t.Error("Two consecutive GenerateRandomString() calls returned the same value")
	}
}

func TestRandomSuffix(t *testing.T) {
	suffix, err := RandomSuffix(6)
	if err != nil {
		t.Fatalf("RandomSuffix() error = %v", err)
	}

	if len(suffix) != 6 {
		t.Errorf("RandomSuffix(6) returned %d chars, want 6", len(suffix))
	}

	for _, c := range suffix {
		if c < '0' || c > '9' {
			t.Errorf("RandomSuffix() returned non-digit character: %c", c)
		}
	}
}
