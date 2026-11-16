package contextx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// v0.12：With* 归一（非法值跳过注入）+ Has* 显式判存。
func TestWithUserID_NormalizesInvalid(t *testing.T) {
	assert.Equal(t, int64(0), UserID(WithUserID(context.Background(), 0)))
	assert.Equal(t, int64(0), UserID(WithUserID(context.Background(), -5)))
	assert.Equal(t, int64(7), UserID(WithUserID(context.Background(), 7)))
}

func TestWithOrgID_NormalizesInvalid(t *testing.T) {
	assert.Equal(t, int64(0), OrgID(WithOrgID(context.Background(), 0)))
	assert.Equal(t, int64(3), OrgID(WithOrgID(context.Background(), 3)))
}

func TestHasUserID_HasOrgID(t *testing.T) {
	assert.False(t, HasUserID(context.Background()))
	assert.False(t, HasUserID(WithUserID(context.Background(), 0)))
	assert.True(t, HasUserID(WithUserID(context.Background(), 1)))

	assert.False(t, HasOrgID(context.Background()))
	assert.True(t, HasOrgID(WithOrgID(context.Background(), 2)))
}
