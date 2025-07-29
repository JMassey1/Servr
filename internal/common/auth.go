package common

import (
	"fmt"

	"github.com/charmbracelet/log"
)

type User struct {
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
}

func ValidateAuth(token string) (*User, error) {
	log.Info("Validating auth token: %s", token)

	if token == "valid-token" || token == "testpass" {
		return &User{
			Username: "testuser",
			Email:    "test@example.com",
			Roles:    []string{"admin", "user"},
		}, nil
	}

	return nil, fmt.Errorf("invalid token")
}
