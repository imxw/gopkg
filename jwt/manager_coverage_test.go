package jwt

import (
	"context"
	"testing"
	"time"
)

// 覆盖缺口补测：ParseRefreshToken 的类型分支（access 冒充 refresh 拒绝 /
// 合法 refresh 通过）、GenerateTokenPair 语法糖腿、RefreshExpire 缺省。

func TestParseRefreshToken_Valid(t *testing.T) {
	tm := NewTokenManager(testConfig(), nil)
	pair, err := tm.GenerateTokenPair(context.Background(), 7, "alice")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := tm.ParseRefreshToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("valid refresh token should parse: %v", err)
	}
	if claims.UserID != 7 || claims.Subject != "alice" || claims.TokenType != TokenTypeRefresh {
		t.Fatalf("claims round trip mismatch: %+v", claims)
	}
}

func TestParseRefreshToken_AccessTokenRejected(t *testing.T) {
	tm := NewTokenManager(testConfig(), nil)
	pair, err := tm.GenerateTokenPair(context.Background(), 7, "alice")
	if err != nil {
		t.Fatal(err)
	}
	// access token 冒充 refresh：类型分支必须拒（ErrTokenInvalid 哨兵链）。
	if _, err := tm.ParseRefreshToken(pair.AccessToken); err == nil {
		t.Fatal("access token must be rejected as refresh token")
	}
}

func TestParseRefreshToken_Empty(t *testing.T) {
	tm := NewTokenManager(testConfig(), nil)
	if _, err := tm.ParseRefreshToken(""); err == nil {
		t.Fatal("empty token must be rejected")
	}
}

// GenerateTokenPair = GenerateTokenPairInOrg(0) 的语法糖：OrgID 不出线。
func TestGenerateTokenPair_NoOrg(t *testing.T) {
	tm := NewTokenManager(testConfig(), nil)
	pair, err := tm.GenerateTokenPair(context.Background(), 7, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if pair.OrgID != 0 {
		t.Fatalf("legacy pair should carry no org, got %d", pair.OrgID)
	}
	claims, err := tm.ParseAccessToken(context.Background(), pair.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if claims.OrgID != 0 {
		t.Fatalf("claims org should be omitted, got %d", claims.OrgID)
	}
}

// RefreshExpire 零值缺省 24h（generateTokenWithType 的兜底腿）。
func TestGenerateTokenPair_DefaultRefreshExpiry(t *testing.T) {
	cfg := testConfig()
	cfg.RefreshExpire = 0
	tm := NewTokenManager(cfg, nil)
	pair, err := tm.GenerateTokenPair(context.Background(), 7, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if pair.RefreshIn != int64(24*time.Hour/time.Second) {
		t.Fatalf("default refresh expiry = %ds, want 86400s", pair.RefreshIn)
	}
}
