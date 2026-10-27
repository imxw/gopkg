package contextx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// OrgID 双腿（74.4% 缺口的唯一来源）：nil ctx fail-closed 返 0 +
// 往返注入。授权链把 0 当「无组织上下文」处理，0 值语义必须钉死。

func TestOrgID_NilCtx_FailClosed(t *testing.T) {
	assert.Equal(t, int64(0), OrgID(nil), "nil ctx 必须返 0（fail-closed）")
}

func TestOrgID_RoundTrip(t *testing.T) {
	ctx := WithOrgID(context.Background(), 42)
	assert.Equal(t, int64(42), OrgID(ctx))
}

func TestOrgID_Absent(t *testing.T) {
	assert.Equal(t, int64(0), OrgID(context.Background()))
}

func TestWithOrgID_NilCtx(t *testing.T) {
	assert.Equal(t, int64(7), OrgID(WithOrgID(nil, 7)))
}
