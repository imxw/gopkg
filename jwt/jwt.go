// Package jwt provides JWT token management and validation with dependency-injected TokenManager.
package jwt

import (
	"time"

	"github.com/google/uuid"
)

// Config holds JWT configuration parameters.
type Config struct {
	Secret        string
	Issuer        string
	Audience      string
	TokenExpire   time.Duration
	RefreshExpire time.Duration
}

// GenerateJTI generates a unique JWT ID.
func GenerateJTI() string {
	return uuid.NewString()
}
