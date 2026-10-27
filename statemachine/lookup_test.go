package statemachine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// StateByName/StateByID 查询面（覆盖率缺口补测）：命中/未命中双态。

func TestEngine_StateLookup(t *testing.T) {
	e := NewEngine(newTestMachine())

	byName, ok := e.StateByName("draft")
	require.True(t, ok)
	assert.Equal(t, "s-draft", byName.ID)

	byID, ok := e.StateByID("s-created")
	require.True(t, ok)
	assert.Equal(t, "created", byID.Name)

	_, ok = e.StateByName("nonexistent")
	assert.False(t, ok)
	_, ok = e.StateByID("s-nonexistent")
	assert.False(t, ok)
}
