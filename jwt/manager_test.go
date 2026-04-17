package jwt

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig() Config {
	return Config{
		Secret:        "test-secret-key-at-least-32-bytes-long",
		Issuer:        "cmdb-test",
		Audience:      "cmdb-test-aud",
		TokenExpire:   time.Hour,
		RefreshExpire: 24 * time.Hour,
	}
}

func TestGenerateTokenPair(t *testing.T) {
	tm := NewTokenManager(testConfig(), nil)
	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, int64(3600), pair.ExpiresIn)
	assert.Equal(t, int64(86400), pair.RefreshIn)
}

func TestGenerateTokenPair_DifferentUsers(t *testing.T) {
	tm := NewTokenManager(testConfig(), nil)
	pair1, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)
	pair2, err := tm.GenerateTokenPair(context.Background(), 2, "user")
	require.NoError(t, err)
	assert.NotEqual(t, pair1.AccessToken, pair2.AccessToken)
}

func TestParseAccessToken_Valid(t *testing.T) {
	tm := NewTokenManager(testConfig(), nil)
	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)

	claims, err := tm.ParseAccessToken(context.Background(), pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), claims.UserID)
	assert.Equal(t, "admin", claims.Subject)
	assert.Equal(t, TokenTypeAccess, claims.TokenType)
	assert.Equal(t, "cmdb-test", claims.Issuer)
	assert.NotEmpty(t, claims.ID)
}

func TestParseAccessToken_EmptyToken(t *testing.T) {
	tm := NewTokenManager(testConfig(), nil)
	_, err := tm.ParseAccessToken(context.Background(), "")
	assert.Error(t, err)
}

func TestParseAccessToken_WrongSecret(t *testing.T) {
	cfg1 := testConfig()
	cfg2 := testConfig()
	cfg2.Secret = "different-secret-key-at-least-32-bytes-long"

	tm1 := NewTokenManager(cfg1, nil)
	tm2 := NewTokenManager(cfg2, nil)

	pair, err := tm1.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)

	_, err = tm2.ParseAccessToken(context.Background(), pair.AccessToken)
	assert.Error(t, err)
}

func TestParseAccessToken_RefreshTokenRejected(t *testing.T) {
	tm := NewTokenManager(testConfig(), nil)
	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)

	_, err = tm.ParseAccessToken(context.Background(), pair.RefreshToken)
	assert.Error(t, err)
}

func TestParseAccessToken_ExpiredToken(t *testing.T) {
	cfg := testConfig()
	tm := NewTokenManager(cfg, nil)

	// Manually build a token that expired in the past
	claims := ExtendedClaims{
		UserID:    1,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Audience:  jwt.ClaimStrings{cfg.Audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ID:        GenerateJTI(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(cfg.Secret))
	require.NoError(t, err)

	_, err = tm.ParseAccessToken(context.Background(), tokenStr)
	assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestParseAccessToken_WrongSigningMethod(t *testing.T) {
	cfg := testConfig()
	tm := NewTokenManager(cfg, nil)

	// Manually craft a JWT with "none" algorithm to bypass HMAC check
	claims := ExtendedClaims{
		UserID:    1,
		TokenType: TokenTypeAccess,
	}
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString(payload)
	fakeToken := header + "." + body + "."

	_, err = tm.ParseAccessToken(context.Background(), fakeToken)
	assert.Error(t, err)
}

func TestParseAccessToken_TamperedToken(t *testing.T) {
	tm := NewTokenManager(testConfig(), nil)
	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)

	_, err = tm.ParseAccessToken(context.Background(), pair.AccessToken+"tampered")
	assert.Error(t, err)
}

func TestRefreshAccessToken_Valid(t *testing.T) {
	tm := NewTokenManager(testConfig(), nil)
	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)

	newPair, err := tm.RefreshAccessToken(context.Background(), pair.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newPair.AccessToken)
	assert.NotEmpty(t, newPair.RefreshToken)
	assert.NotEqual(t, pair.AccessToken, newPair.AccessToken)
}

func TestRefreshAccessToken_WithAccessToken(t *testing.T) {
	tm := NewTokenManager(testConfig(), nil)
	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)

	_, err = tm.RefreshAccessToken(context.Background(), pair.AccessToken)
	assert.Error(t, err)
}

func TestRevokeToken_NoBlacklist(t *testing.T) {
	tm := NewTokenManager(testConfig(), nil)
	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)

	err = tm.RevokeToken(context.Background(), pair.AccessToken)
	assert.NoError(t, err)
}

func TestRevokeToken_EmptyToken(t *testing.T) {
	tm := NewTokenManager(testConfig(), nil)
	err := tm.RevokeToken(context.Background(), "")
	assert.Error(t, err)
}

func TestGenerateJTI(t *testing.T) {
	jti1 := GenerateJTI()
	jti2 := GenerateJTI()
	assert.NotEmpty(t, jti1)
	assert.NotEqual(t, jti1, jti2)
}

func TestGenerateTokenPair_DefaultExpiry(t *testing.T) {
	cfg := testConfig()
	cfg.TokenExpire = 0
	cfg.RefreshExpire = 0
	tm := NewTokenManager(cfg, nil)

	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.Equal(t, int64(0), pair.ExpiresIn)
	assert.Equal(t, int64(0), pair.RefreshIn)
}
