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

// TokenManager manages JWT token lifecycle with injected config and optional blacklist.
type TokenManager struct {
	cfg       Config
	blacklist Blacklist
}

// NewTokenManager creates a new TokenManager with the given config and optional blacklist.
func NewTokenManager(cfg Config, blacklist Blacklist) *TokenManager {
	return &TokenManager{
		cfg:       cfg,
		blacklist: blacklist,
	}
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

	return &TokenPair{
		AccessToken:  accessToken,
		AccessJTI:    accessJTI,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(tm.cfg.TokenExpire.Seconds()),
		RefreshIn:    int64(tm.cfg.RefreshExpire.Seconds()),
		OrgID:        orgID,
	}, nil
}

func (tm *TokenManager) generateTokenWithType(userID int64, userName string, orgID int64, tokenType TokenType, expire time.Duration) (string, string, error) {
	if expire <= 0 {
		expire = time.Hour * 24
	}

	jti := GenerateJTI()
	claims := ExtendedClaims{
		UserID:    userID,
		OrgID:     orgID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tm.cfg.Issuer,
			Audience:  jwt.ClaimStrings{tm.cfg.Audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expire)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
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
func (tm *TokenManager) RefreshAccessToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := tm.parseExtendedToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("parse refresh token: %w", err)
	}

	if claims.TokenType != TokenTypeRefresh {
		return nil, ErrTokenNotRefreshable
	}

	if tm.blacklist != nil {
		inBlacklist, err := tm.blacklist.Contains(ctx, claims.ID)
		if err != nil {
			return nil, fmt.Errorf("refresh token revocation check failed: %w", err)
		}
		if inBlacklist {
			return nil, ErrTokenInBlacklist
		}
	}

	if tm.blacklist != nil {
		if err := tm.blacklist.AddToken(ctx, claims.ID, claims.ExpiresAt.Time); err != nil {
			return nil, fmt.Errorf("failed to blacklist old refresh token: %w", err)
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
