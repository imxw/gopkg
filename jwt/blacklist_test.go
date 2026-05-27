package jwt

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBlacklist is an in-memory Blacklist implementation for testing.
type mockBlacklist struct {
	store       map[string]time.Time
	addErr      error
	containsErr error
}

func newMockBlacklist() *mockBlacklist {
	return &mockBlacklist{store: make(map[string]time.Time)}
}

func (m *mockBlacklist) AddToken(_ context.Context, jti string, expiresAt time.Time) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.store[jti] = expiresAt
	return nil
}

func (m *mockBlacklist) Contains(_ context.Context, jti string) (bool, error) {
	if m.containsErr != nil {
		return false, m.containsErr
	}
	_, ok := m.store[jti]
	return ok, nil
}

func TestErrorSentinels(t *testing.T) {
	sentinels := []error{ErrTokenExpired, ErrTokenInvalid, ErrTokenInBlacklist, ErrTokenNotRefreshable}
	for _, e := range sentinels {
		assert.Error(t, e)
	}
	for i := 0; i < len(sentinels); i++ {
		for j := i + 1; j < len(sentinels); j++ {
			assert.NotEqual(t, sentinels[i], sentinels[j])
		}
	}
}

func TestTokenType(t *testing.T) {
	assert.Equal(t, TokenType("access"), TokenTypeAccess)
	assert.Equal(t, TokenType("refresh"), TokenTypeRefresh)
}

func TestTokenPair_Fields(t *testing.T) {
	pair := &TokenPair{AccessToken: "at", RefreshToken: "rt", ExpiresIn: 3600, RefreshIn: 86400}
	assert.Equal(t, "at", pair.AccessToken)
	assert.Equal(t, int64(3600), pair.ExpiresIn)
}

func TestExtendedClaims(t *testing.T) {
	claims := &ExtendedClaims{UserID: 1, TokenType: TokenTypeAccess}
	assert.Equal(t, int64(1), claims.UserID)
}

func TestConfig(t *testing.T) {
	cfg := Config{Secret: "s", Issuer: "i", Audience: "a", TokenExpire: time.Hour, RefreshExpire: 24 * time.Hour}
	assert.Equal(t, "s", cfg.Secret)
}

func TestNewTokenManager_WithBlacklist(t *testing.T) {
	cfg := testConfig()
	bl := newMockBlacklist()
	tm := NewTokenManager(cfg, bl)
	assert.NotNil(t, tm)
	assert.NotNil(t, tm.blacklist)
}

func TestNewTokenManager_NilBlacklist(t *testing.T) {
	cfg := testConfig()
	tm := NewTokenManager(cfg, nil)
	assert.NotNil(t, tm)
	assert.Nil(t, tm.blacklist)
}

func TestParseAccessToken_RevokedInBlacklist(t *testing.T) {
	bl := newMockBlacklist()
	tm := NewTokenManager(testConfig(), bl)

	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)

	claims, err := tm.ParseAccessToken(context.Background(), pair.AccessToken)
	require.NoError(t, err)
	bl.store[claims.ID] = time.Now().Add(time.Hour)

	_, err = tm.ParseAccessToken(context.Background(), pair.AccessToken)
	assert.ErrorIs(t, err, ErrTokenInBlacklist)
}

func TestParseAccessToken_BlacklistCheckFails(t *testing.T) {
	bl := &mockBlacklist{
		store:       make(map[string]time.Time),
		containsErr: assert.AnError,
	}
	tm := NewTokenManager(testConfig(), bl)

	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)

	_, err = tm.ParseAccessToken(context.Background(), pair.AccessToken)
	assert.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestRevokeToken_WithBlacklist(t *testing.T) {
	bl := newMockBlacklist()
	tm := NewTokenManager(testConfig(), bl)

	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)

	claims, err := tm.parseExtendedToken(pair.AccessToken)
	require.NoError(t, err)
	jti := claims.ID

	err = tm.RevokeToken(context.Background(), pair.AccessToken)
	assert.NoError(t, err)

	_, ok := bl.store[jti]
	assert.True(t, ok, "revoked token should be in blacklist")
}

func TestRevokeToken_BlacklistAddFails(t *testing.T) {
	bl := &mockBlacklist{
		store:  make(map[string]time.Time),
		addErr: assert.AnError,
	}
	tm := NewTokenManager(testConfig(), bl)

	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)

	err = tm.RevokeToken(context.Background(), pair.AccessToken)
	assert.Error(t, err)
}

func TestRefreshAccessToken_BlacklistRotation(t *testing.T) {
	bl := newMockBlacklist()
	tm := NewTokenManager(testConfig(), bl)

	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)

	newPair, err := tm.RefreshAccessToken(context.Background(), pair.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, newPair.AccessToken)

	claims, err := tm.parseExtendedToken(pair.RefreshToken)
	require.NoError(t, err)
	_, ok := bl.store[claims.ID]
	assert.True(t, ok, "old refresh token should be blacklisted after rotation")
}

func TestRefreshAccessToken_RevokedRefreshToken(t *testing.T) {
	bl := newMockBlacklist()
	tm := NewTokenManager(testConfig(), bl)

	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)

	claims, err := tm.parseExtendedToken(pair.RefreshToken)
	require.NoError(t, err)
	bl.store[claims.ID] = claims.ExpiresAt.Time

	_, err = tm.RefreshAccessToken(context.Background(), pair.RefreshToken)
	assert.ErrorIs(t, err, ErrTokenInBlacklist)
}

func TestRefreshAccessToken_BlacklistCheckFails(t *testing.T) {
	bl := &mockBlacklist{
		store:       make(map[string]time.Time),
		containsErr: assert.AnError,
	}
	tm := NewTokenManager(testConfig(), bl)

	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)

	_, err = tm.RefreshAccessToken(context.Background(), pair.RefreshToken)
	assert.Error(t, err)
}

func TestRefreshAccessToken_BlacklistAddFails(t *testing.T) {
	bl := &mockBlacklist{
		store:  make(map[string]time.Time),
		addErr: assert.AnError,
	}
	tm := NewTokenManager(testConfig(), bl)

	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)

	_, err = tm.RefreshAccessToken(context.Background(), pair.RefreshToken)
	assert.Error(t, err)
}
