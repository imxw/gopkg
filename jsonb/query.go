// Package jsonb provides utilities for PostgreSQL JSONB query building.
package jsonb

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var validFieldName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

var placeholderRe = regexp.MustCompile(`\$(\d+)`)

const (
	OpEQ         = "$eq"
	OpNE         = "$ne"
	OpGT         = "$gt"
	OpGTE        = "$gte"
	OpLT         = "$lt"
	OpLTE        = "$lte"
	OpIn         = "$in"
	OpNin        = "$nin"
	OpLike       = "$like"
	OpContains   = "$contains"
	OpStartsWith = "$startsWith"
	OpEndsWith   = "$endsWith"
	OpIsNull     = "$isNull"
	OpExists     = "$exists"
	OpAnd        = "$and"
	OpOr         = "$or"
)

const maxNestingDepth = 10

// ParseQuery parses a MongoDB-style JSON filter into a PostgreSQL JSONB SQL WHERE clause
// with parameterized placeholders. Uses "attributes" as the default JSONB column name.
func ParseQuery(cond json.RawMessage) (string, []any, error) {
	return ParseQueryColumn(cond, "attributes")
}

// ParseQueryColumn is like ParseQuery but allows specifying the JSONB column name.
func ParseQueryColumn(cond json.RawMessage, column string) (string, []any, error) {
	return parseQuery(cond, column, 0)
}

func parseQuery(cond json.RawMessage, column string, depth int) (string, []any, error) {
	if depth >= maxNestingDepth {
		return "", nil, fmt.Errorf("query nesting exceeds maximum depth of %d", maxNestingDepth)
	}

	if len(cond) == 0 {
		return "", nil, nil
	}

	var conditions map[string]any
	if err := json.Unmarshal(cond, &conditions); err != nil {
		return "", nil, fmt.Errorf("invalid query format: %w", err)
	}

	var args []any
	var clauses []string

	requireNumeric := func(op string, val any, field string) error {
		switch val.(type) {
		case float64, int, int64, int32:
			return nil
		default:
			return fmt.Errorf("%s requires numeric value for field %s, got %T", op, field, val)
		}
	}

	requireString := func(op string, val any, field string) error {
		if _, ok := val.(string); !ok {
			return fmt.Errorf("%s requires string value for field %s, got %T", op, field, val)
		}
		return nil
	}

	parseCondition := func(field string, value any) error {
		if !validFieldName.MatchString(field) {
			return fmt.Errorf("invalid field name: %q (only alphanumeric and underscore allowed)", field)
		}
		switch v := value.(type) {
		case string:
			args = append(args, v)
			clauses = append(clauses, fmt.Sprintf("%s->>'%s' = $%d", column, field, len(args)))

		case map[string]any:
			for op, opVal := range v {
				switch op {
				case OpEQ:
					args = append(args, opVal)
					clauses = append(clauses, fmt.Sprintf("%s->>'%s' = $%d", column, field, len(args)))

				case OpNE:
					args = append(args, opVal)
					clauses = append(clauses, fmt.Sprintf("(%s->>'%s' != $%d OR %s->>'%s' IS NULL)", column, field, len(args), column, field))

				case OpGT:
					if err := requireNumeric(OpGT, opVal, field); err != nil {
						return err
					}
					args = append(args, opVal)
					clauses = append(clauses, fmt.Sprintf("(%s->>'%s')::numeric > $%d", column, field, len(args)))

				case OpGTE:
					if err := requireNumeric(OpGTE, opVal, field); err != nil {
						return err
					}
					args = append(args, opVal)
					clauses = append(clauses, fmt.Sprintf("(%s->>'%s')::numeric >= $%d", column, field, len(args)))

				case OpLT:
					if err := requireNumeric(OpLT, opVal, field); err != nil {
						return err
					}
					args = append(args, opVal)
					clauses = append(clauses, fmt.Sprintf("(%s->>'%s')::numeric < $%d", column, field, len(args)))

				case OpLTE:
					if err := requireNumeric(OpLTE, opVal, field); err != nil {
						return err
					}
					args = append(args, opVal)
					clauses = append(clauses, fmt.Sprintf("(%s->>'%s')::numeric <= $%d", column, field, len(args)))

				case OpLike:
					if err := requireString(OpLike, opVal, field); err != nil {
						return err
					}
					args = append(args, opVal)
					clauses = append(clauses, fmt.Sprintf("%s->>'%s' LIKE $%d", column, field, len(args)))

				case OpStartsWith:
					if err := requireString(OpStartsWith, opVal, field); err != nil {
						return err
					}
					args = append(args, opVal)
					clauses = append(clauses, fmt.Sprintf("%s->>'%s' LIKE ($%d || '%%')", column, field, len(args)))

				case OpEndsWith:
					if err := requireString(OpEndsWith, opVal, field); err != nil {
						return err
					}
					args = append(args, opVal)
					clauses = append(clauses, fmt.Sprintf("%s->>'%s' LIKE ('%%' || $%d)", column, field, len(args)))

				case OpIn:
					arr, ok := opVal.([]any)
					if !ok {
						return fmt.Errorf("$in requires array value for field %s", field)
					}
					if len(arr) == 0 {
						clauses = append(clauses, "1=0")
						continue
					}
					for _, item := range arr {
						args = append(args, item)
					}
					placeholders := make([]string, len(arr))
					for i := range arr {
						placeholders[i] = fmt.Sprintf("$%d", len(args)-len(arr)+i+1)
					}
					clauses = append(clauses, fmt.Sprintf("%s->>'%s' IN (%s)", column, field, strings.Join(placeholders, ", ")))

				case OpNin:
					arr, ok := opVal.([]any)
					if !ok {
						return fmt.Errorf("$nin requires array value for field %s", field)
					}
					if len(arr) == 0 {
						continue
					}
					args = append(args, arr...)
					placeholders := make([]string, len(arr))
					for i := range arr {
						placeholders[i] = fmt.Sprintf("$%d", len(args)-len(arr)+i+1)
					}
					clauses = append(clauses, fmt.Sprintf("(%s->>'%s' NOT IN (%s) AND %s->>'%s' IS NOT NULL)", column, field, strings.Join(placeholders, ", "), column, field))

				case OpContains:
					jsonVal, err := json.Marshal(opVal)
					if err != nil {
						return fmt.Errorf("%s requires JSON-serializable value for field %s: %w", OpContains, field, err)
					}
					args = append(args, string(jsonVal))
					clauses = append(clauses, fmt.Sprintf("%s @> $%d::jsonb", column, len(args)))

				case OpIsNull:
					boolVal, ok := opVal.(bool)
					if !ok {
						return fmt.Errorf("$isNull requires boolean value for field %s", field)
					}
					if boolVal {
						clauses = append(clauses, fmt.Sprintf("%s->>'%s' IS NULL", column, field))
					} else {
						clauses = append(clauses, fmt.Sprintf("%s->>'%s' IS NOT NULL", column, field))
					}

				case OpExists:
					boolVal, ok := opVal.(bool)
					if !ok {
						return fmt.Errorf("$exists requires boolean value for field %s", field)
					}
					if boolVal {
						clauses = append(clauses, fmt.Sprintf("%s->>'%s' IS NOT NULL", column, field))
					} else {
						clauses = append(clauses, fmt.Sprintf("NOT (%s ? '%s')", column, field))
					}

				default:
					return fmt.Errorf("unsupported operator: %s", op)
				}
			}

		default:
			args = append(args, fmt.Sprintf("%v", v))
			clauses = append(clauses, fmt.Sprintf("%s->>'%s' = $%d", column, field, len(args)))
		}
		return nil
	}

	for field, value := range conditions {
		switch field {
		case OpAnd:
			arr, ok := value.([]any)
			if !ok {
				return "", nil, fmt.Errorf("$and requires array value")
			}
			var andClauses []string
			for _, item := range arr {
				itemJSON, err := json.Marshal(item)
				if err != nil {
					return "", nil, fmt.Errorf("failed to marshal $and item: %w", err)
				}
				subClause, subArgs, err := parseQuery(itemJSON, column, depth+1)
				if err != nil {
					return "", nil, err
				}
				if subClause != "" {
					subClause = RenumberPlaceholders(subClause, len(args))
					andClauses = append(andClauses, subClause)
					args = append(args, subArgs...)
				}
			}
			if len(andClauses) > 0 {
				clauses = append(clauses, "("+strings.Join(andClauses, " AND ")+")")
			}

		case OpOr:
			arr, ok := value.([]any)
			if !ok {
				return "", nil, fmt.Errorf("$or requires array value")
			}
			var orClauses []string
			for _, item := range arr {
				itemJSON, err := json.Marshal(item)
				if err != nil {
					return "", nil, fmt.Errorf("failed to marshal $or item: %w", err)
				}
				subClause, subArgs, err := parseQuery(itemJSON, column, depth+1)
				if err != nil {
					return "", nil, err
				}
				if subClause != "" {
					subClause = RenumberPlaceholders(subClause, len(args))
					orClauses = append(orClauses, subClause)
					args = append(args, subArgs...)
				}
			}
			if len(orClauses) > 0 {
				clauses = append(clauses, "("+strings.Join(orClauses, " OR ")+")")
			}

		default:
			if err := parseCondition(field, value); err != nil {
				return "", nil, err
			}
		}
	}

	if len(clauses) == 0 {
		return "", nil, nil
	}

	return strings.Join(clauses, " AND "), args, nil
}

// RenumberPlaceholders renumbers $1, $2, ... in a sub-clause to $base+1, $base+2, ...
func RenumberPlaceholders(clause string, baseArgsLen int) string {
	return placeholderRe.ReplaceAllStringFunc(clause, func(match string) string {
		var idx int
		fmt.Sscanf(match[1:], "%d", &idx)
		return fmt.Sprintf("$%d", baseArgsLen+idx)
	})
}
