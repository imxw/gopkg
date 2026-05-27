package jsonb

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestParseQuery_SimpleEq(t *testing.T) {
	input := json.RawMessage(`{"hostname": "server1"}`)
	clause, args, err := ParseQuery(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clause != "attributes->>'hostname' = $1" {
		t.Errorf("unexpected clause: %s", clause)
	}
	if len(args) != 1 || args[0] != "server1" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestParseQuery_WithOperator(t *testing.T) {
	input := json.RawMessage(`{"ip": {"$like": "192.168.%"}}`)
	clause, args, err := ParseQuery(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clause != "attributes->>'ip' LIKE $1" {
		t.Errorf("unexpected clause: %s", clause)
	}
	if len(args) != 1 || args[0] != "192.168.%" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestParseQuery_EmptyInput(t *testing.T) {
	clause, args, err := ParseQuery(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clause != "" {
		t.Errorf("expected empty clause, got: %s", clause)
	}
	if args != nil {
		t.Errorf("expected nil args, got: %v", args)
	}
}

func TestParseQuery_InOperator(t *testing.T) {
	input := json.RawMessage(`{"status": {"$in": ["active", "inactive"]}}`)
	clause, args, err := ParseQuery(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clause != "attributes->>'status' IN ($1, $2)" {
		t.Errorf("unexpected clause: %s", clause)
	}
	if len(args) != 2 {
		t.Errorf("expected 2 args, got: %d", len(args))
	}
}

func TestParseQuery_OrCondition(t *testing.T) {
	input := json.RawMessage(`{
		"$or": [
			{"status": "active"},
			{"status": "inactive"}
		]
	}`)
	clause, args, err := ParseQuery(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clause != "(attributes->>'status' = $1 OR attributes->>'status' = $2)" {
		t.Errorf("unexpected clause: %s", clause)
	}
	if len(args) != 2 {
		t.Errorf("expected 2 args, got: %d", len(args))
	}
}

func TestParseQuery_AndCondition(t *testing.T) {
	input := json.RawMessage(`{"hostname": {"$like": "server%"}, "status": {"$eq": "active"}}`)
	clause, args, err := ParseQuery(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Map iteration order is non-deterministic in Go, so check both orderings
	validClauses := map[string]bool{
		"attributes->>'hostname' LIKE $1 AND attributes->>'status' = $2": true,
		"attributes->>'status' = $1 AND attributes->>'hostname' LIKE $2": true,
	}
	if !validClauses[clause] {
		t.Errorf("unexpected clause: %s", clause)
	}
	if len(args) != 2 {
		t.Errorf("expected 2 args, got: %d", len(args))
	}
	// Verify both expected values are present regardless of map iteration order
	got := map[string]bool{fmt.Sprintf("%v", args[0]): true, fmt.Sprintf("%v", args[1]): true}
	if !got["server%"] || !got["active"] {
		t.Errorf("expected args [server%%, active], got: %v", args)
	}
}

func TestParseQuery_NeOperator(t *testing.T) {
	input := json.RawMessage(`{"status": {"$ne": "deleted"}}`)
	clause, _, err := ParseQuery(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clause != "(attributes->>'status' != $1 OR attributes->>'status' IS NULL)" {
		t.Errorf("unexpected clause: %s", clause)
	}
}

func TestParseQuery_StartsWith(t *testing.T) {
	input := json.RawMessage(`{"hostname": {"$startsWith": "web"}}`)
	clause, _, err := ParseQuery(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clause != "attributes->>'hostname' LIKE ($1 || '%')" {
		t.Errorf("unexpected clause: %s", clause)
	}
}

func TestParseQuery_IsNull(t *testing.T) {
	input := json.RawMessage(`{"notes": {"$isNull": true}}`)
	clause, _, err := ParseQuery(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clause != "attributes->>'notes' IS NULL" {
		t.Errorf("unexpected clause: %s", clause)
	}
}

// --- 新增测试：字段名安全校验 ---

func TestParseQuery_InvalidFieldName(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"SQL injection attempt", `{"hostname'; DROP TABLE cmdb_cis; --": "value"}`},
		{"dollar sign in name", `{"foo$1": "value"}`},
		{"semicolon in name", `{"foo;bar": "value"}`},
		{"space in name", `{"foo bar": "value"}`},
		{"starts with digit", `{"123field": "value"}`},
		{"dot notation", `{"foo.bar": "value"}`},
		{"dash in name", `{"foo-bar": "value"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseQuery(json.RawMessage(tt.input))
			if err == nil {
				t.Error("expected error for invalid field name, got nil")
			}
		})
	}
}

func TestParseQuery_ValidFieldNames(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"underscore prefix", `{"_private": "value"}`},
		{"snake_case", `{"my_field_name": "value"}`},
		{"camelCase", `{"myFieldName": "value"}`},
		{"single letter", `{"a": "value"}`},
		{"with digits", `{"field123": "value"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseQuery(json.RawMessage(tt.input))
			if err != nil {
				t.Errorf("unexpected error for valid field name: %v", err)
			}
		})
	}
}

// --- 新增测试：占位符重编号正确性（>9 个参数时 $1 不误匹配 $10） ---

func TestParseQuery_PlaceholderRenumbering_LargeArgCount(t *testing.T) {
	// 构造一个 $and 查询，第一个子句产生 10 个参数，第二个子句的 $1 应变为 $11 而非 $10
	input := json.RawMessage(`{
		"$and": [
			{
				"f1": {"$in": ["a","b","c","d","e","f","g","h","i","j"]}
			},
			{
				"f2": "value"
			}
		]
	}`)
	clause, args, err := ParseQuery(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 第一个子句：f1 IN ($1..$10)
	// 第二个子句：f2 = $11（不是 $10）
	if len(args) != 11 {
		t.Errorf("expected 11 args, got %d", len(args))
	}
	// 确保没有 $10 被错误替换：clause 中应包含 $11 而非 $10 出现在第二个条件中
	if contains := containsPlaceholder(clause, "$10"); !contains {
		t.Errorf("expected $10 in first sub-clause, clause: %s", clause)
	}
	if contains := containsPlaceholder(clause, "$11"); !contains {
		t.Errorf("expected $11 for second sub-clause, clause: %s", clause)
	}
}

func containsPlaceholder(clause, placeholder string) bool {
	// 简单检查 placeholder 是否作为完整 token 出现
	for i := 0; i <= len(clause)-len(placeholder); i++ {
		if clause[i:i+len(placeholder)] == placeholder {
			// 确保后面不是数字（即 $10 不会误匹配 $100）
			after := i + len(placeholder)
			if after < len(clause) && clause[after] >= '0' && clause[after] <= '9' {
				continue
			}
			return true
		}
	}
	return false
}

// --- Type validation tests ---

func TestParseQuery_NumericOperators_RejectNonNumeric(t *testing.T) {
	ops := []string{"$gt", "$gte", "$lt", "$lte"}
	for _, op := range ops {
		t.Run(op+"_rejects_string", func(t *testing.T) {
			input := json.RawMessage(fmt.Sprintf(`{"count": {"%s": "not_a_number"}}`, op))
			_, _, err := ParseQuery(input)
			if err == nil {
				t.Error("expected error for non-numeric value")
			}
		})
	}
}

func TestParseQuery_NumericOperators_AcceptNumeric(t *testing.T) {
	input := json.RawMessage(`{"count": {"$gt": 10, "$lt": 100}}`)
	clause, args, err := ParseQuery(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsPlaceholder(clause, "$1") || !containsPlaceholder(clause, "$2") {
		t.Errorf("expected 2 placeholders, clause: %s", clause)
	}
	if len(args) != 2 {
		t.Errorf("expected 2 args, got %d", len(args))
	}
}

func TestParseQuery_StringOperators_RejectNonString(t *testing.T) {
	ops := []string{"$like", "$startsWith", "$endsWith"}
	for _, op := range ops {
		t.Run(op+"_rejects_number", func(t *testing.T) {
			input := json.RawMessage(fmt.Sprintf(`{"name": {"%s": 123}}`, op))
			_, _, err := ParseQuery(input)
			if err == nil {
				t.Error("expected error for non-string value")
			}
		})
	}
}
