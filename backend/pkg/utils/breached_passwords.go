package utils

import (
	"embed"
	"strings"
)

//go:embed data/top_passwords.txt
var passwordFS embed.FS

var breachedPasswords map[string]struct{}

func init() {
	data, err := passwordFS.ReadFile("data/top_passwords.txt")
	if err != nil {
		return
	}
	breachedPasswords = make(map[string]struct{}, 10000)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			breachedPasswords[strings.ToLower(line)] = struct{}{}
		}
	}
}

// IsBreachedPassword checks if the password appears in the top-10k common passwords list.
func IsBreachedPassword(password string) bool {
	if breachedPasswords == nil {
		return false
	}
	_, found := breachedPasswords[strings.ToLower(password)]
	return found
}
