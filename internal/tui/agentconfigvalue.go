package tui

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Helpers for editing agent property values. Process Server values are YAML
// fragments: scalars are often single-quoted ('10G'), and multi-value
// properties such as LDAP or database connection properties are lists of
// small maps:
//
//	- key: timeBetweenEvictionRunsMillis
//	  value: 300000

// unquoteValue removes YAML single quotes around v. quoted reports whether
// v was quoted, so the quotes can be added back with quoteValue.
func unquoteValue(v string) (s string, quoted bool) {
	if len(v) >= 2 && strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'") {
		return strings.ReplaceAll(v[1:len(v)-1], "''", "'"), true
	}
	return v, false
}

// quoteValue is the inverse of unquoteValue.
func quoteValue(s string, quoted bool) string {
	if !quoted {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// entryList is a property value that is a YAML list of maps with scalar
// values. The document node is kept so unchanged entries keep their style.
type entryList struct {
	doc *yaml.Node // document node; doc.Content[0] is the sequence
}

// parseEntryList parses v as a list of maps. ok is false for any other value.
func parseEntryList(v string) (*entryList, bool) {
	if strings.TrimSpace(v) == "" {
		return nil, false
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(v), &doc); err != nil || len(doc.Content) != 1 {
		return nil, false
	}
	seq := doc.Content[0]
	if seq.Kind != yaml.SequenceNode || len(seq.Content) == 0 {
		return nil, false
	}
	for _, item := range seq.Content {
		if item.Kind != yaml.MappingNode {
			return nil, false
		}
		for _, n := range item.Content {
			if n.Kind != yaml.ScalarNode {
				return nil, false
			}
		}
	}
	return &entryList{doc: &doc}, true
}

func (l *entryList) items() []*yaml.Node { return l.doc.Content[0].Content }

// Len returns the number of entries.
func (l *entryList) Len() int { return len(l.items()) }

// Fields returns entry i as ordered key/value pairs.
func (l *entryList) Fields(i int) [][2]string {
	item := l.items()[i]
	out := make([][2]string, 0, len(item.Content)/2)
	for k := 0; k+1 < len(item.Content); k += 2 {
		out = append(out, [2]string{item.Content[k].Value, item.Content[k+1].Value})
	}
	return out
}

// Label is a one-line summary of entry i, for example
// "key=testOnBorrow  value=true".
func (l *entryList) Label(i int) string {
	parts := make([]string, 0, 2)
	for _, f := range l.Fields(i) {
		parts = append(parts, f[0]+"="+f[1])
	}
	return strings.Join(parts, "  ")
}

// Keys returns the field names of the first entry, used for new entries.
func (l *entryList) Keys() []string {
	fields := l.Fields(0)
	keys := make([]string, len(fields))
	for i, f := range fields {
		keys[i] = f[0]
	}
	return keys
}

// Set replaces the values of entry i. Values that changed are written plain
// when YAML allows it, as typed.
func (l *entryList) Set(i int, values []string) {
	item := l.items()[i]
	for k := 0; k+1 < len(item.Content) && k/2 < len(values); k += 2 {
		n := item.Content[k+1]
		if n.Value != values[k/2] {
			n.Value, n.Tag, n.Style = values[k/2], "", 0
		}
	}
}

// Add appends an entry with keys and values.
func (l *entryList) Add(keys, values []string) {
	item := &yaml.Node{Kind: yaml.MappingNode}
	for i, k := range keys {
		v := ""
		if i < len(values) {
			v = values[i]
		}
		item.Content = append(item.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: k},
			&yaml.Node{Kind: yaml.ScalarNode, Value: v})
	}
	seq := l.doc.Content[0]
	seq.Content = append(seq.Content, item)
}

// Remove deletes entry i.
func (l *entryList) Remove(i int) {
	seq := l.doc.Content[0]
	seq.Content = append(seq.Content[:i], seq.Content[i+1:]...)
}

// String returns the list as YAML, two-space indented, as the agent stores
// it.
func (l *entryList) String() (string, error) {
	if l.Len() == 0 {
		return "", nil
	}
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(l.doc); err != nil {
		return "", fmt.Errorf("writing list: %w", err)
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	return b.String(), nil
}

// previewValue is a one-line preview of a property value for list labels.
func previewValue(v string, sensitive bool, width int) string {
	switch {
	case sensitive:
		return "***"
	case strings.TrimSpace(v) == "":
		return "(empty)"
	}
	if l, ok := parseEntryList(v); ok {
		return fmt.Sprintf("[%d entries] %s", l.Len(), truncate(l.Label(0), width-14))
	}
	if strings.Contains(v, "\n") {
		return truncate(strings.ReplaceAll(strings.TrimRight(v, "\n"), "\n", " / "), width)
	}
	return truncate(v, width)
}

func truncate(s string, width int) string {
	r := []rune(s)
	if width < 2 || len(r) <= width {
		return s
	}
	return string(r[:width-1]) + "…"
}

// entryDiff lists the entry changes between two list values, matching
// entries by their first field (key or name): "~ key=a  value: 1 → 2",
// "+ key=b  value=3", "- key=c  value=4". ok is false when either value is
// not a list.
func entryDiff(oldV, newV string) (lines []string, ok bool) {
	oldL, okOld := parseEntryList(oldV)
	newL, okNew := parseEntryList(newV)
	if !okOld && !okNew || (!okOld && strings.TrimSpace(oldV) != "") || (!okNew && strings.TrimSpace(newV) != "") {
		return nil, false
	}
	type entry struct {
		id     string
		fields [][2]string
	}
	collect := func(l *entryList) []entry {
		if l == nil {
			return nil
		}
		out := make([]entry, l.Len())
		for i := range out {
			f := l.Fields(i)
			out[i] = entry{fields: f}
			if len(f) > 0 {
				out[i].id = f[0][0] + "=" + f[0][1]
			}
		}
		return out
	}
	label := func(fields [][2]string) string {
		parts := make([]string, len(fields))
		for i, f := range fields {
			parts[i] = f[0] + "=" + f[1]
		}
		return strings.Join(parts, "  ")
	}
	olds, news := collect(oldL), collect(newL)
	used := make([]bool, len(olds))
	for _, n := range news {
		match := -1
		for i, o := range olds {
			if !used[i] && o.id == n.id {
				match = i
				break
			}
		}
		if match < 0 {
			lines = append(lines, "+ "+label(n.fields))
			continue
		}
		used[match] = true
		var changes []string
		of := olds[match].fields
		for k, f := range n.fields {
			switch {
			case k >= len(of):
				changes = append(changes, f[0]+"="+f[1])
			case of[k][1] != f[1]:
				changes = append(changes, fmt.Sprintf("%s: %s → %s", f[0], of[k][1], f[1]))
			}
		}
		if len(changes) > 0 {
			lines = append(lines, "~ "+n.id+"  "+strings.Join(changes, "  "))
		}
	}
	for i, o := range olds {
		if !used[i] {
			lines = append(lines, "- "+label(o.fields))
		}
	}
	return lines, true
}
