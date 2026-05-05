package service

import (
	"testing"
)

func TestIsValidRedirectURI(t *testing.T) {
	tests := []struct {
		name           string
		registeredURIs string
		requestedURI   string
		want           bool
	}{
		{
			"exact match",
			"https://app.com/callback",
			"https://app.com/callback",
			true,
		},
		{
			"no match",
			"https://app.com/callback",
			"https://evil.com/callback",
			false,
		},
		{
			"multiple URIs - match",
			"https://app.com/callback,https://app.com/other",
			"https://app.com/other",
			true,
		},
		{
			"multiple URIs - no match",
			"https://app.com/callback,https://app.com/other",
			"https://app.com/callback?extra=param",
			false,
		},
		{
			"empty registered",
			"",
			"https://app.com/callback",
			false,
		},
		{
			"empty requested",
			"https://app.com/callback",
			"",
			false,
		},
		{
			"javascript URI rejected",
			"https://app.com/callback",
			"javascript:alert(1)",
			false,
		},
		{
			"data URI rejected",
			"https://app.com/callback",
			"data:text/html,<script>alert(1)</script>",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidRedirectURI(tt.registeredURIs, tt.requestedURI)
			if got != tt.want {
				t.Errorf("isValidRedirectURI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidScope(t *testing.T) {
	tests := []struct {
		name            string
		registeredScopes string
		requestedScope  string
		want            bool
	}{
		{"single match", "read,write", "read", true},
		{"all match", "read,write", "read,write", true},
		{"partial no match", "read", "read,write", false},
		{"no match", "read,write", "admin", false},
		{"empty registered", "", "read", false},
		{"empty requested with registered", "read,write", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidScope(tt.registeredScopes, tt.requestedScope)
			if got != tt.want {
				t.Errorf("isValidScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateRedirectURIIsPublic(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantErr bool
	}{
		{"public domain", "https://example.com/callback", false},
		{"public IP", "https://8.8.8.8/callback", false},
		{"localhost", "http://localhost:3000/callback", true},
		{"127.0.0.1", "http://127.0.0.1:3000/callback", true},
		{"0.0.0.0", "http://0.0.0.0/callback", true},
		{"private 10.x", "http://10.0.0.1/callback", true},
		{"private 172.x", "http://172.16.0.1/callback", true},
		{"private 192.168.x", "http://192.168.1.1/callback", true},
		{"AWS metadata", "http://169.254.169.254/latest/meta-data/", true},
		{"GCP metadata", "http://metadata.google.internal/", true},
		{"IPv6 loopback", "http://[::1]:3000/callback", true},
		{"link-local", "http://169.254.0.1/callback", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRedirectURIIsPublic(tt.uri)
			gotErr := err != nil
			if gotErr != tt.wantErr {
				t.Errorf("validateRedirectURIIsPublic(%q) error = %v, wantErr %v", tt.uri, gotErr, tt.wantErr)
			}
		})
	}
}
