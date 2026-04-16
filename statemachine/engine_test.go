package statemachine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMachine() Machine {
	return Machine{
		ID:           "sm-1",
		Name:         "create_destroy",
		DisplayName:  "创建-销毁",
		InitialState: "draft",
		States: []State{
			{ID: "s-draft", Name: "draft", DisplayName: "草稿"},
			{ID: "s-created", Name: "created", DisplayName: "已创建"},
			{ID: "s-confirmed", Name: "confirmed", DisplayName: "已确认"},
			{ID: "s-destroyed", Name: "destroyed", DisplayName: "已销毁"},
		},
		Transitions: []Transition{
			{ID: "t-1", FromStateID: "s-draft", ToStateID: "s-created", Operation: "create", Label: "创建", OrderNum: 1},
			{ID: "t-2", FromStateID: "s-created", ToStateID: "s-confirmed", Operation: "confirm", Label: "确认", OrderNum: 2},
			{ID: "t-3", FromStateID: "s-confirmed", ToStateID: "s-destroyed", Operation: "destroy", Label: "销毁", OrderNum: 3},
			{ID: "t-4", FromStateID: "s-created", ToStateID: "s-destroyed", Operation: "destroy", Label: "销毁", OrderNum: 4},
		},
	}
}

func TestEngine_InitialState(t *testing.T) {
	e := NewEngine(newTestMachine())
	assert.Equal(t, "draft", e.InitialState())
}

func TestEngine_StateByName(t *testing.T) {
	e := NewEngine(newTestMachine())

	s, ok := e.StateByName("draft")
	assert.True(t, ok)
	assert.Equal(t, "s-draft", s.ID)

	_, ok = e.StateByName("unknown")
	assert.False(t, ok)
}

func TestEngine_CanTransition(t *testing.T) {
	e := NewEngine(newTestMachine())

	tests := []struct {
		name    string
		state   string
		op      string
		want    bool
		wantErr bool
	}{
		{"draft → create", "draft", "create", true, false},
		{"created → confirm", "created", "confirm", true, false},
		{"confirmed → destroy", "confirmed", "destroy", true, false},
		{"created → destroy", "created", "destroy", true, false},
		{"draft → confirm (invalid)", "draft", "confirm", false, true},
		{"unknown state", "unknown", "create", false, true},
		{"unknown operation", "draft", "destroy", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trans, err := e.CanTransition(tt.state, tt.op)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, trans)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, trans)
				assert.Equal(t, tt.op, trans.Operation)
			}
		})
	}
}

func TestEngine_Transition(t *testing.T) {
	e := NewEngine(newTestMachine())

	result, err := e.Transition("draft", "create")
	require.NoError(t, err)
	assert.Equal(t, "draft", result.FromState)
	assert.Equal(t, "created", result.ToState)
	assert.Equal(t, "s-created", result.ToStateID)
	assert.Equal(t, "创建", result.Label)
}

func TestEngine_Transition_Invalid(t *testing.T) {
	e := NewEngine(newTestMachine())

	_, err := e.Transition("draft", "destroy")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTransitionNotFound)
}

func TestEngine_AvailableOperations(t *testing.T) {
	e := NewEngine(newTestMachine())

	ops, err := e.AvailableOperations("created")
	require.NoError(t, err)
	assert.Len(t, ops, 2) // confirm + destroy

	opNames := make(map[string]bool)
	for _, op := range ops {
		opNames[op.Operation] = true
	}
	assert.True(t, opNames["confirm"])
	assert.True(t, opNames["destroy"])

	// draft 只有一个操作
	ops, err = e.AvailableOperations("draft")
	require.NoError(t, err)
	assert.Len(t, ops, 1)
	assert.Equal(t, "create", ops[0].Operation)
}

func TestEngine_AvailableOperations_DestroyedState(t *testing.T) {
	e := NewEngine(newTestMachine())

	// destroyed 是终态，没有可执行操作
	ops, err := e.AvailableOperations("destroyed")
	require.NoError(t, err)
	assert.Len(t, ops, 0)
}

func TestEngine_AllStates(t *testing.T) {
	e := NewEngine(newTestMachine())
	states := e.AllStates()
	assert.Len(t, states, 4)
}

func TestEngine_AllTransitions(t *testing.T) {
	e := NewEngine(newTestMachine())
	transitions := e.AllTransitions()
	assert.Len(t, transitions, 4)
}
