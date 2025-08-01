package common

import (
	"fmt"
	"os"
	"time"

	CharmLog "github.com/charmbracelet/log"
)

type User struct {
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
}

var logger = CharmLog.NewWithOptions(os.Stderr, CharmLog.Options{
	ReportTimestamp: true,
	TimeFormat:      time.Kitchen,
	Prefix:          "Auth Service 🔐",
})

func ValidateAuth(token string) (*User, error) {
	logger.Info("Validating auth token", "token", token)

	if token == "valid-token" || token == "testpass" {
		return &User{
			Username: "testuser",
			Email:    "test@example.com",
			Roles:    []string{"admin", "user"},
		}, nil
	}

	return nil, fmt.Errorf("invalid token")
}
