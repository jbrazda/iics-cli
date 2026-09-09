// Package filter provides simple client-side filtering of API result rows using
// field==value / field!=value expressions. Keys are JSON tag names (the same
// names used by the output package for columns), with dot notation for nested
// fields. String comparison is case-insensitive; bool and numeric values are
// compared via their fmt "%v" rendering.
package filter

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Predicate is a single parsed filter expression.
type Predicate struct {
	Key   string
	Op    string // "==" or "!="
	Value string
}

// Parse parses a single expression such as "agentHost==host01" or
// "active != true". Whitespace around the key, operator, and value is trimmed.
func Parse(expr string) (Predicate, error) {
	for _, op := range []string{"==", "!="} {
		if i := strings.Index(expr, op); i >= 0 {
			key := strings.TrimSpace(expr[:i])
			val := strings.TrimSpace(expr[i+len(op):])
			if key == "" {
				return Predicate{}, fmt.Errorf("filter %q: empty field name", expr)
			}
			return Predicate{Key: key, Op: op, Value: val}, nil
		}
	}
	return Predicate{}, fmt.Errorf("filter %q: expected <field>==<value> or <field>!=<value>", expr)
}

// ParseAll parses a list of expressions.
func ParseAll(exprs []string) ([]Predicate, error) {
	preds := make([]Predicate, 0, len(exprs))
	for _, e := range exprs {
		p, err := Parse(e)
		if err != nil {
			return nil, err
		}
		preds = append(preds, p)
	}
	return preds, nil
}

// Match reports whether row satisfies every predicate (logical AND).
func Match(row map[string]interface{}, preds []Predicate) bool {
	for _, p := range preds {
		actual := lookup(row, p.Key)
		equal := strings.EqualFold(actual, p.Value)
		if (p.Op == "==" && !equal) || (p.Op == "!=" && equal) {
			return false
		}
	}
	return true
}

// Apply marshals data (a struct or slice) to JSON, decodes it to rows, and
// returns the rows that satisfy every predicate. The result can be handed
// directly to any output formatter.
func Apply(data interface{}, preds []Predicate) ([]map[string]interface{}, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("filter: marshaling data: %w", err)
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal(b, &rows); err != nil {
		var single map[string]interface{}
		if err2 := json.Unmarshal(b, &single); err2 != nil {
			return nil, fmt.Errorf("filter: data is not an object or array of objects: %w", err)
		}
		rows = []map[string]interface{}{single}
	}

	if len(preds) == 0 {
		return rows, nil
	}

	out := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		if Match(r, preds) {
			out = append(out, r)
		}
	}
	return out, nil
}

// lookup resolves a dot-separated key against a decoded JSON object and returns
// the value's "%v" rendering, or "" when any path segment is missing.
func lookup(row map[string]interface{}, key string) string {
	var cur interface{} = row
	for _, part := range strings.Split(key, ".") {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return ""
		}
		cur, ok = m[part]
		if !ok {
			return ""
		}
	}
	if cur == nil {
		return ""
	}
	if f, ok := cur.(float64); ok && f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%v", cur)
}
