package jwt

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTokenBlacklist_DefaultPrefix(t *testing.T) {
	bl := &TokenBlacklist{prefix: ""}
	if bl.prefix == "" {
		bl.prefix = "token_blacklist"
	}
	assert.Equal(t, "token_blacklist", bl.prefix)
}

func TestTokenBlacklist_KeyFormat(t *testing.T) {
	bl := &TokenBlacklist{prefix: "myapp_bl"}
	key := bl.blacklistKey("jti-123")
	assert.Equal(t, "myapp_bl:jti-123", key)
}

func TestTokenBlacklist_KeyFormat_DefaultPrefix(t *testing.T) {
	bl := &TokenBlacklist{prefix: "token_blacklist"}
	key := bl.blacklistKey("jti-abc")
	assert.Equal(t, "token_blacklist:jti-abc", key)
}

func TestErrorSentinels(t *testing.T) {
	assert.Error(t, ErrTokenExpired)
	assert.Error(t, ErrTokenInvalid)
	assert.Error(t, ErrTokenInBlacklist)
	assert.Error(t, ErrTokenNotRefreshable)
	assert.Error(t, ErrRefreshTokenExpired)

	errors := []error{ErrTokenExpired, ErrTokenInvalid, ErrTokenInBlacklist, ErrTokenNotRefreshable, ErrRefreshTokenExpired}
	for i := 0; i < len(errors); i++ {
		for j := i + 1; j < len(errors); j++ {
			assert.NotEqual(t, errors[i], errors[j], "error %d should differ from %d", i, j)
		}
	}
}

func TestTokenType(t *testing.T) {
	assert.Equal(t, TokenType("access"), TokenTypeAccess)
	assert.Equal(t, TokenType("refresh"), TokenTypeRefresh)
}

func TestTokenPair_Fields(t *testing.T) {
	pair := &TokenPair{
		AccessToken:  "at",
		RefreshToken: "rt",
		ExpiresIn:    3600,
		RefreshIn:    86400,
	}
	assert.Equal(t, "at", pair.AccessToken)
	assert.Equal(t, "rt", pair.RefreshToken)
	assert.Equal(t, int64(3600), pair.ExpiresIn)
	assert.Equal(t, int64(86400), pair.RefreshIn)
}

func TestExtendedClaims(t *testing.T) {
	claims := &ExtendedClaims{
		UserID:    1,
		TokenType: TokenTypeAccess,
	}
	assert.Equal(t, uint64(1), claims.UserID)
	assert.Equal(t, TokenTypeAccess, claims.TokenType)
}

func TestConfig(t *testing.T) {
	cfg := Config{
		Secret:        "secret",
		Issuer:        "issuer",
		Audience:      "audience",
		TokenExpire:   time.Hour,
		RefreshExpire: 24 * time.Hour,
	}
	assert.Equal(t, "secret", cfg.Secret)
	assert.Equal(t, time.Hour, cfg.TokenExpire)
}

func TestNewTokenManager(t *testing.T) {
	cfg := testConfig()
	tm := NewTokenManager(cfg, nil)
	assert.NotNil(t, tm)
	assert.Equal(t, cfg, tm.cfg)
	assert.Nil(t, tm.blacklist)
}

func TestNewTokenManager_WithBlacklist(t *testing.T) {
	cfg := testConfig()
	bl := &TokenBlacklist{prefix: "test"}
	tm := NewTokenManager(cfg, bl)
	assert.NotNil(t, tm)
	assert.NotNil(t, tm.blacklist)
}

func TestParseAccessToken_InvalidSigningMethod(t *testing.T) {
	cfg := testConfig()
	tm := NewTokenManager(cfg, nil)

	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)

	claims, err := tm.ParseAccessToken(context.Background(), pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), claims.UserID)
}
