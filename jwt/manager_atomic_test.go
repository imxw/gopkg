package jwt

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Config 校验：Secret 下限（HS256 ≥ 32 字节；空钥能签出合法过验签的 token）----

func TestNewTokenManager_ShortSecretRejected(t *testing.T) {
	for _, secret := range []string{"", "short", string(make([]byte, 31))} {
		tm, err := NewTokenManager(Config{Secret: secret}, nil)
		assert.Nil(t, tm)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least 32 bytes")
	}
}

func TestNewTokenManager_MinSecretAccepted(t *testing.T) {
	tm, err := NewTokenManager(Config{Secret: string(make([]byte, 32))}, nil)
	require.NoError(t, err)
	assert.NotNil(t, tm)
}

// ---- 原子轮换：AtomicBlacklist 消除 check-then-act 重放窗口 ----

// fakeAtomicBlacklist 内存实现：AddIfAbsent 用 map+锁模拟 SET NX 语义。
type fakeAtomicBlacklist struct {
	mu      sync.Mutex
	entries map[string]time.Time
}

func newFakeAtomicBlacklist() *fakeAtomicBlacklist {
	return &fakeAtomicBlacklist{entries: make(map[string]time.Time)}
}

func (f *fakeAtomicBlacklist) AddToken(_ context.Context, jti string, expiresAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries[jti] = expiresAt
	return nil
}

func (f *fakeAtomicBlacklist) Contains(_ context.Context, jti string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.entries[jti]
	return ok, nil
}

func (f *fakeAtomicBlacklist) AddIfAbsent(_ context.Context, jti string, expiresAt time.Time) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.entries[jti]; ok {
		return false, nil
	}
	f.entries[jti] = expiresAt
	return true, nil
}

var _ AtomicBlacklist = (*fakeAtomicBlacklist)(nil)

func TestRefreshAccessToken_AtomicRotationReplayRejected(t *testing.T) {
	bl := newFakeAtomicBlacklist()
	tm := mustManager(t, testConfig(), bl)
	ctx := context.Background()

	pair, err := tm.GenerateTokenPair(ctx, 1, "admin")
	require.NoError(t, err)

	// 首刷成功并占位。
	pair2, err := tm.RefreshAccessToken(ctx, pair.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, pair2.AccessToken)

	// 用旧 refresh token 重放：原子占位已存在 → 判重放拒绝。
	_, err = tm.RefreshAccessToken(ctx, pair.RefreshToken)
	assert.ErrorIs(t, err, ErrTokenInBlacklist)
}

// 并发同 token 双刷：恰好一个成功、一个重放失败——被盗 token 的重用
// 检测依赖「第二次刷新必失败」，两步 Contains→AddToken 在此窗口双双通过。
func TestRefreshAccessToken_AtomicRotationConcurrentReplay(t *testing.T) {
	bl := newFakeAtomicBlacklist()
	tm := mustManager(t, testConfig(), bl)
	ctx := context.Background()

	pair, err := tm.GenerateTokenPair(ctx, 1, "admin")
	require.NoError(t, err)

	var wg sync.WaitGroup
	var okCount, replayCount atomic.Int32
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := tm.RefreshAccessToken(ctx, pair.RefreshToken)
			switch {
			case err == nil:
				okCount.Add(1)
			case errors.Is(err, ErrTokenInBlacklist):
				replayCount.Add(1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), okCount.Load(), "exactly one refresh must win the rotation race")
	assert.Equal(t, int32(15), replayCount.Load())
}

// 降级路径：只实现 Blacklist 的实现方照旧两步走（非原子，语义兼容）。
type legacyBlacklistOnly struct {
	deny bool
}

func (l *legacyBlacklistOnly) AddToken(_ context.Context, _ string, _ time.Time) error { return nil }
func (l *legacyBlacklistOnly) Contains(_ context.Context, _ string) (bool, error) {
	return l.deny, nil
}

var _ Blacklist = (*legacyBlacklistOnly)(nil)

func TestRefreshAccessToken_LegacyBlacklistPath(t *testing.T) {
	tm := mustManager(t, testConfig(), &legacyBlacklistOnly{})
	ctx := context.Background()

	pair, err := tm.GenerateTokenPair(ctx, 1, "admin")
	require.NoError(t, err)

	pair2, err := tm.RefreshAccessToken(ctx, pair.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, pair2.AccessToken)
	_ = pair2

	deny := mustManager(t, testConfig(), &legacyBlacklistOnly{deny: true})
	_, err = deny.RefreshAccessToken(ctx, pair.RefreshToken)
	assert.ErrorIs(t, err, ErrTokenInBlacklist)
}

// 原子路径失败 fail-closed：AddIfAbsent 出错时拒绝轮换。
type failingAtomicBlacklist struct {
	legacyBlacklistOnly
}

func (f *failingAtomicBlacklist) AddIfAbsent(_ context.Context, _ string, _ time.Time) (bool, error) {
	return false, errors.New("redis down")
}

var _ AtomicBlacklist = (*failingAtomicBlacklist)(nil)

func TestRefreshAccessToken_AtomicFailureClosed(t *testing.T) {
	tm := mustManager(t, testConfig(), &failingAtomicBlacklist{})
	pair, err := tm.GenerateTokenPair(context.Background(), 1, "admin")
	require.NoError(t, err)

	_, err = tm.RefreshAccessToken(context.Background(), pair.RefreshToken)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis down")
}
