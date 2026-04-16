// Package statemachine provides a generic state machine engine
// for managing state transitions with validation and event-driven actions.
package statemachine

import (
	"errors"
	"fmt"
)

var (
	// ErrTransitionNotFound 没有找到匹配的转换规则
	ErrTransitionNotFound = errors.New("transition not found")
)

// State 表示状态机中的一个状态
type State struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	IsConfirmed bool   `json:"isConfirmed"`
	Color       string `json:"color,omitempty"`
}

// Transition 表示状态之间的转换规则
type Transition struct {
	ID          string `json:"id"`
	FromStateID string `json:"fromStateId"`
	ToStateID   string `json:"toStateId"`
	Operation   string `json:"operation"`
	Permission  string `json:"permission,omitempty"`
	Action      string `json:"action,omitempty"`
	Label       string `json:"label,omitempty"`
	IsMultiple  bool   `json:"isMultiple"`
	OrderNum    int    `json:"orderNum"`
}

// TransitionResult 转换结果
type TransitionResult struct {
	FromState   string `json:"fromState"`
	ToState     string `json:"toState"`
	ToStateID   string `json:"toStateId"`
	Operation   string `json:"operation"`
	Label       string `json:"label,omitempty"`
	Permission  string `json:"permission,omitempty"`
	Action      string `json:"action,omitempty"`
}

// Machine 表示一个完整的状态机定义
type Machine struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	DisplayName  string        `json:"displayName"`
	InitialState string        `json:"initialState"`
	States       []State       `json:"states"`
	Transitions  []Transition  `json:"transitions"`
}

// stateMap 状态 ID/Name → State 的索引
type stateMap struct {
	byID   map[string]State
	byName map[string]State
}

// Engine 状态机引擎，提供状态转换校验
type Engine struct {
	machine Machine
	states  stateMap
	transByFrom map[string][]Transition // from_state_id → transitions
}

// NewEngine 创建状态机引擎
func NewEngine(machine Machine) *Engine {
	sm := stateMap{
		byID:   make(map[string]State, len(machine.States)),
		byName: make(map[string]State, len(machine.States)),
	}
	for _, s := range machine.States {
		sm.byID[s.ID] = s
		sm.byName[s.Name] = s
	}

	tm := make(map[string][]Transition, len(machine.Transitions))
	for _, t := range machine.Transitions {
		tm[t.FromStateID] = append(tm[t.FromStateID], t)
	}

	return &Engine{
		machine:     machine,
		states:      sm,
		transByFrom: tm,
	}
}

// InitialState 返回初始状态名称
func (e *Engine) InitialState() string {
	return e.machine.InitialState
}

// StateByName 根据名称获取状态
func (e *Engine) StateByName(name string) (State, bool) {
	s, ok := e.states.byName[name]
	return s, ok
}

// StateByID 根据 ID 获取状态
func (e *Engine) StateByID(id string) (State, bool) {
	s, ok := e.states.byID[id]
	return s, ok
}

// CanTransition 检查是否可以从当前状态执行操作
func (e *Engine) CanTransition(currentStateName, operation string) (*Transition, error) {
	state, ok := e.states.byName[currentStateName]
	if !ok {
		return nil, fmt.Errorf("unknown state: %s", currentStateName)
	}

	transitions := e.transByFrom[state.ID]
	for i := range transitions {
		t := transitions[i]
		if t.Operation == operation {
			return &t, nil
		}
	}
	return nil, ErrTransitionNotFound
}

// Transition 执行状态转换，返回转换结果
func (e *Engine) Transition(currentStateName, operation string) (*TransitionResult, error) {
	t, err := e.CanTransition(currentStateName, operation)
	if err != nil {
		return nil, err
	}

	toState, ok := e.states.byID[t.ToStateID]
	if !ok {
		return nil, fmt.Errorf("target state not found: %s", t.ToStateID)
	}

	return &TransitionResult{
		FromState:  currentStateName,
		ToState:    toState.Name,
		ToStateID:  toState.ID,
		Operation:  t.Operation,
		Label:      t.Label,
		Permission: t.Permission,
		Action:     t.Action,
	}, nil
}

// AvailableOperations 返回当前状态可执行的操作列表
func (e *Engine) AvailableOperations(currentStateName string) ([]Transition, error) {
	state, ok := e.states.byName[currentStateName]
	if !ok {
		return nil, fmt.Errorf("unknown state: %s", currentStateName)
	}

	transitions := e.transByFrom[state.ID]
	result := make([]Transition, 0, len(transitions))
	for _, t := range transitions {
		result = append(result, t)
	}
	return result, nil
}

// AllStates 返回所有状态
func (e *Engine) AllStates() []State {
	return e.machine.States
}

// AllTransitions 返回所有转换规则
func (e *Engine) AllTransitions() []Transition {
	return e.machine.Transitions
}
