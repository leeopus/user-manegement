package utils

import (
	"strings"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email string
		want  bool
	}{
		{"user@example.com", true},
		{"user.name@example.com", true},
		{"user+tag@example.com", true},
		{"user@sub.example.com", true},
		{"", false},
		{"no-at-sign", false},
		{"@no-local.com", false},
		{"user@", false},
		{"user@.com", false},
		{"user@com", false},
		{"user@example", false},
		{string(make([]byte, 300)), false}, // too long
	}

	for _, tt := range tests {
		err := ValidateEmail(tt.email)
		got := err == nil
		if got != tt.want {
			t.Errorf("ValidateEmail(%q) = %v, want %v (err: %v)", tt.email, got, tt.want, err)
		}
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		username string
		want     bool
	}{
		{"john", true},
		{"john_doe", true},
		{"john-doe", true},
		{"user123", true},
		{"ab", false},   // too short
		{"admin", false}, // reserved
		{"system", false}, // reserved
		{"_john", false}, // starts with special
		{"-john", false}, // starts with special
		{"john--doe", false}, // consecutive special chars
		{"john__doe", false}, // consecutive special chars
		{"john doe", false},  // contains space
		{"", false},          // empty
	}

	// Add a long username test
	longUsername := ""
	for i := 0; i < 33; i++ {
		longUsername += "a"
	}
	tests = append(tests, struct {
		username string
		want     bool
	}{longUsername, false})

	for _, tt := range tests {
		err := ValidateUsername(tt.username)
		got := err == nil
		if got != tt.want {
			t.Errorf("ValidateUsername(%q) = %v, want %v (err: %v)", tt.username, got, tt.want, err)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		username string
		wantErr  bool
	}{
		{"valid strong", "MyPass123!", "user", false},
		{"valid without special", "MyPass123", "user", false},
		{"too short", "Ab1", "user", true},
		{"too long", "Aa1" + strings.Repeat("x", 62), "user", true},
		{"no uppercase - allowed (NIST)", "mypassword1", "user", false},
		{"no lowercase - allowed (NIST)", "MYPASSWORD1", "user", false},
		{"no number - allowed (NIST)", "MyPassword", "user", false},
		{"long passphrase (NIST)", "correcthorsebatterystaple", "user", false},
		{"common password", "password", "user", true},
		{"all same chars", "AAAAAAAAAAA", "user", true},
		{"contains username", "MyUser123", "myuser", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidatePassword(tt.password, tt.username)
			gotErr := err != nil
			if gotErr != tt.wantErr {
				t.Errorf("ValidatePassword(%q, %q) error = %v, wantErr %v", tt.password, tt.username, gotErr, tt.wantErr)
			}
		})
	}
}

func TestIsDisposableEmail(t *testing.T) {
	tests := []struct {
		email string
		want  bool
	}{
		{"user@mailinator.com", true},
		{"user@guerrillamail.com", true},
		{"user@10minutemail.com", true},
		{"user@gmail.com", false},
		{"user@company.com", false},
		{"user@example.com", false},
	}

	for _, tt := range tests {
		got := IsDisposableEmail(tt.email)
		if got != tt.want {
			t.Errorf("IsDisposableEmail(%q) = %v, want %v", tt.email, got, tt.want)
		}
	}
}

func TestGenerateUsernameFromEmail(t *testing.T) {
	tests := []struct {
		email string
		want  string
	}{
		{"john@gmail.com", "john"},
		{"john.doe@gmail.com", "johndoe"},
		{"john+tag@gmail.com", "john"},
		{"ab@x.com", "user"},           // too short (< 3)
		{"123user@gmail.com", "user_123user"}, // starts with non-alpha
	}

	for _, tt := range tests {
		got := GenerateUsernameFromEmail(tt.email)
		if got != tt.want {
			t.Errorf("GenerateUsernameFromEmail(%q) = %q, want %q", tt.email, got, tt.want)
		}
	}
}

func TestSanitizeHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<script>alert(1)</script>", "&lt;script&gt;alert(1)&lt;/script&gt;"},
		{`"onclick"='xss'`, "&quot;onclick&quot;=&#x27;xss&#x27;"},
		{"normal text", "normal text"},
		{"a & b", "a &amp; b"},
	}

	for _, tt := range tests {
		got := SanitizeHTML(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeHTML(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
