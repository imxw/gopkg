package jwt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Classification errors — use errors.Is(err, ErrXxx) to check.
var (
	ErrTokenExpired        = errors.New("token has expired")
	ErrTokenInvalid        = errors.New("token is invalid")
	ErrTokenInBlacklist    = errors.New("token has been revoked")
	ErrTokenNotRefreshable = errors.New("token is not refreshable")
)

// TokenType distinguishes access tokens from refresh tokens.
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// ExtendedClaims extends standard JWT claims with token type information.
// UserName is stored in RegisteredClaims.Subject to avoid redundancy.
// OrgID carries the current organization context (multi-org session):
// switching orgs means re-issuing the token pair.
type ExtendedClaims struct {
	UserID    int64     `json:"userID"`
	OrgID     int64     `json:"orgID,omitempty"`
	TokenType TokenType `json:"tokenType"`
	jwt.RegisteredClaims
}

// TokenPair holds an access/refresh token pair with expiry information.
type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	AccessJTI    string `json:"-"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
	RefreshIn    int64  `json:"refreshIn"`
	// OrgID echoes the org context bound into the tokens (0 if none),
	// so callers can surface it without re-parsing the access token.
	OrgID int64 `json:"orgId,omitempty"`
}

// Blacklist is the interface for checking and managing revoked tokens.
// TokenBlacklist (backed by Redis) satisfies this interface.
type Blacklist interface {
	AddToken(ctx context.Context, jti string, expiresAt time.Time) error
	Contains(ctx context.Context, jti string) (bool, error)
}

// AtomicBlacklist 是 Blacklist 的可选原子扩展：把「查在黑 + 入黑」折叠为
// 一次原子操作。刷新轮换走 Contains→AddToken 两步存在 check-then-act
// 竞态——同一 refresh token 并发刷两次可双双通过，被盗 token 的重用检测
// 完全失效。实现方支持本接口时 TokenManager 自动走原子路径。
type AtomicBlacklist interface {
	Blacklist
	// AddIfAbsent 原子地把 jti 入黑：首次调用返回 absent=true（本调用
	// 完成占位，允许轮换），jti 已在黑时返回 absent=false（判定重放）。
	AddIfAbsent(ctx context.Context, jti string, expiresAt time.Time) (absent bool, err error)
}

// minSecretLen 是 HS256 签名密钥的下限（256 bit）：HMAC 密钥短于摘要
// 输出长度会实质削弱强度，空密钥甚至能产出合法过验签的 token。
const minSecretLen = 32

// TokenManager manages JWT token lifecycle with injected config and optional blacklist.
type TokenManager struct {
	cfg       Config
	blacklist Blacklist
}

// NewTokenManager creates a new TokenManager with the given config and
// optional blacklist. cfg.Secret must be at least 32 bytes (HS256 minimum).
func NewTokenManager(cfg Config, blacklist Blacklist) (*TokenManager, error) {
	if len(cfg.Secret) < minSecretLen {
		return nil, fmt.Errorf("jwt secret must be at least %d bytes (got %d)", minSecretLen, len(cfg.Secret))
	}
	return &TokenManager{
		cfg:       cfg,
		blacklist: blacklist,
	}, nil
}

// GenerateTokenPair generates an access and refresh token pair without an org
// context. Prefer GenerateTokenPairInOrg for HTTP sessions so downstream
// authorization can scope by org.
func (tm *TokenManager) GenerateTokenPair(ctx context.Context, userID int64, userName string) (*TokenPair, error) {
	return tm.GenerateTokenPairInOrg(ctx, userID, userName, 0)
}

// GenerateTokenPairInOrg generates an access and refresh token pair bound to
// the given organization context. orgID <= 0 omits the org claim.
func (tm *TokenManager) GenerateTokenPairInOrg(ctx context.Context, userID int64, userName string, orgID int64) (*TokenPair, error) {
	accessToken, accessJTI, err := tm.generateTokenWithType(userID, userName, orgID, TokenTypeAccess, tm.cfg.TokenExpire)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, _, err := tm.generateTokenWithType(userID, userName, orgID, TokenTypeRefresh, tm.cfg.RefreshExpire)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// TokenPair 有效期元数据与 token 内嵌 exp 同源：config 零值时
	// generateTokenWithType 兜底 24h，元数据若仍读 raw config 就会报 0——
	// 客户端凭它排刷新计划，0 直接破坏该协议（测试逮住的真缺陷）。
	accessExpire, refreshExpire := tm.cfg.TokenExpire, tm.cfg.RefreshExpire
	if accessExpire <= 0 {
		accessExpire = time.Hour * 24
	}
	if refreshExpire <= 0 {
		refreshExpire = time.Hour * 24
	}

	return &TokenPair{
		AccessToken:  accessToken,
		AccessJTI:    accessJTI,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(accessExpire.Seconds()),
		RefreshIn:    int64(refreshExpire.Seconds()),
		OrgID:        orgID,
	}, nil
}

func (tm *TokenManager) generateTokenWithType(userID int64, userName string, orgID int64, tokenType TokenType, expire time.Duration) (string, string, error) {
	if expire <= 0 {
		expire = time.Hour * 24
	}

	jti := GenerateJTI()
	// exp/nbf/iat 取同一 now：三次 time.Now() 之间有微秒级漂移，可能让
	// nbf 晚于 iat 的极端对不齐（同一时间基线是协议要求）。
	now := time.Now()
	claims := ExtendedClaims{
		UserID:    userID,
		OrgID:     orgID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tm.cfg.Issuer,
			Audience:  jwt.ClaimStrings{tm.cfg.Audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(expire)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   userName,
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(tm.cfg.Secret))
	if err != nil {
		return "", "", fmt.Errorf("sign token: %w", err)
	}

	return tokenStr, jti, nil
}

// ParseAccessToken parses and validates an access token, checking the blacklist.
// Use errors.Is to classify the returned error:
//
//	errors.Is(err, jwt.ErrTokenExpired)     — token has expired
//	errors.Is(err, jwt.ErrTokenInvalid)     — malformed or bad signature
//	errors.Is(err, jwt.ErrTokenInBlacklist) — token has been revoked
func (tm *TokenManager) ParseAccessToken(ctx context.Context, tokenStr string) (*ExtendedClaims, error) {
	claims, err := tm.parseExtendedToken(tokenStr)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != TokenTypeAccess {
		return nil, fmt.Errorf("%w: expected access, got %s", ErrTokenInvalid, claims.TokenType)
	}

	if tm.blacklist != nil {
		inBlacklist, checkErr := tm.blacklist.Contains(ctx, claims.ID)
		if checkErr != nil {
			// fail-closed on blacklist check failure: reject token
			return nil, fmt.Errorf("token revocation check failed: %w", checkErr)
		}
		if inBlacklist {
			return nil, ErrTokenInBlacklist
		}
	}

	return claims, nil
}

// ParseRefreshToken parses a refresh token and returns its claims without blacklist checking.
func (tm *TokenManager) ParseRefreshToken(tokenStr string) (*ExtendedClaims, error) {
	claims, err := tm.parseExtendedToken(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeRefresh {
		return nil, fmt.Errorf("%w: expected refresh, got %s", ErrTokenInvalid, claims.TokenType)
	}
	return claims, nil
}

// RefreshAccessToken uses a refresh token to generate a new token pair.
// The old refresh token is revoked as part of rotation: blacklists exposing
// the AtomicBlacklist extension claim the jti atomically, so a concurrent
// replay of the same refresh token deterministically loses the race and is
// rejected — replay detection depends on the second refresh failing.
func (tm *TokenManager) RefreshAccessToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := tm.parseExtendedToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("parse refresh token: %w", err)
	}

	if claims.TokenType != TokenTypeRefresh {
		return nil, ErrTokenNotRefreshable
	}

	if tm.blacklist != nil {
		if atomicBL, ok := tm.blacklist.(AtomicBlacklist); ok {
			absent, claimErr := atomicBL.AddIfAbsent(ctx, claims.ID, claims.ExpiresAt.Time)
			if claimErr != nil {
				return nil, fmt.Errorf("refresh token revocation claim failed: %w", claimErr)
			}
			if !absent {
				return nil, ErrTokenInBlacklist
			}
		} else {
			// 降级路径：两步 Contains→AddToken 非原子，并发重放窗口仍在
			// （实现 AtomicBlacklist 可消除；见 AtomicBlacklist 注释）。
			inBlacklist, checkErr := tm.blacklist.Contains(ctx, claims.ID)
			if checkErr != nil {
				return nil, fmt.Errorf("refresh token revocation check failed: %w", checkErr)
			}
			if inBlacklist {
				return nil, ErrTokenInBlacklist
			}
			if err := tm.blacklist.AddToken(ctx, claims.ID, claims.ExpiresAt.Time); err != nil {
				return nil, fmt.Errorf("failed to blacklist old refresh token: %w", err)
			}
		}
	}

	// Carry the org context into the rotated pair: refresh must not silently
	// drop or change the session's organization binding.
	return tm.GenerateTokenPairInOrg(ctx, claims.UserID, claims.Subject, claims.OrgID)
}

// RevokeToken revokes a token by adding it to the blacklist.
func (tm *TokenManager) RevokeToken(ctx context.Context, tokenStr string) error {
	claims, err := tm.parseExtendedToken(tokenStr)
	if err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}

	if tm.blacklist == nil {
		return nil
	}

	err = tm.blacklist.AddToken(ctx, claims.ID, claims.ExpiresAt.Time)
	if err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}

	return nil
}

func (tm *TokenManager) parseExtendedToken(tokenStr string) (*ExtendedClaims, error) {
	if tokenStr == "" {
		return nil, ErrTokenInvalid
	}

	// jwt/v5：算法约束优先用 WithValidMethods 白名单（解析器在验签前强制），
	// keyfunc 内的 HMAC 断言作为二道防线保留。
	token, err := jwt.ParseWithClaims(tokenStr, &ExtendedClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unsupported signing method", ErrTokenInvalid)
		}
		return []byte(tm.cfg.Secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		// v5 用哨兵错误分类（ValidationError 位掩码已移除）。
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("%w: %v", ErrTokenExpired, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	claims, ok := token.Claims.(*ExtendedClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}
