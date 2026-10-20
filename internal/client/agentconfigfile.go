package client

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseAgentConfigChanges parses an agent property change file, JSON or YAML,
// in one of two shapes:
//
// A nested map, as written by AgentConfigDocument. A null value resets the
// property to its default:
//
//	process-engine:
//	  cluster:
//	    primary-node: "true"
//	  jvm:
//	    max-heap: "'4G'"
//	    min-heap: null
//
// A list of changes:
//
//	[{"property": "process-engine:cluster:primary-node", "value": "true"},
//	 {"property": "process-engine:jvm:min-heap", "reset": true}]
//
// Scalars are used as written, so true or 512 become "true" and "512".
func ParseAgentConfigChanges(data []byte) ([]AgentConfigChange, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing change file: %w", err)
	}
	if len(doc.Content) == 0 {
		return nil, nil
	}
	root := doc.Content[0]
	switch root.Kind {
	case yaml.MappingNode:
		return parseChangeMap(root)
	case yaml.SequenceNode:
		return parseChangeList(root)
	}
	return nil, fmt.Errorf("change file: expected a service map or a list of changes (line %d)", root.Line)
}

func parseChangeMap(root *yaml.Node) ([]AgentConfigChange, error) {
	var changes []AgentConfigChange
	for i := 0; i+1 < len(root.Content); i += 2 {
		service, types := root.Content[i].Value, root.Content[i+1]
		if types.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("change file line %d: %s must map property types to properties", types.Line, service)
		}
		for j := 0; j+1 < len(types.Content); j += 2 {
			typ, props := types.Content[j].Value, types.Content[j+1]
			if props.Kind != yaml.MappingNode {
				return nil, fmt.Errorf("change file line %d: %s:%s must map property names to values", props.Line, service, typ)
			}
			for k := 0; k+1 < len(props.Content); k += 2 {
				name, val := props.Content[k].Value, props.Content[k+1]
				ch := AgentConfigChange{Property: service + ":" + typ + ":" + name}
				switch {
				case val.Kind == yaml.ScalarNode && val.Tag == "!!null":
					ch.Reset = true
				case val.Kind == yaml.ScalarNode:
					ch.Value = val.Value
				default:
					return nil, fmt.Errorf("change file line %d: value of %s must be a string, or null to reset", val.Line, ch.Property)
				}
				changes = append(changes, ch)
			}
		}
	}
	return changes, nil
}

func parseChangeList(root *yaml.Node) ([]AgentConfigChange, error) {
	var changes []AgentConfigChange
	for _, item := range root.Content {
		if item.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("change file line %d: expected {property, value} or {property, reset}", item.Line)
		}
		var ch AgentConfigChange
		for k := 0; k+1 < len(item.Content); k += 2 {
			key, val := item.Content[k].Value, item.Content[k+1]
			if val.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("change file line %d: %s must be a scalar", val.Line, key)
			}
			switch key {
			case "property":
				ch.Property = val.Value
			case "value":
				ch.Value = val.Value
			case "reset":
				ch.Reset = val.Value == "true"
			default:
				return nil, fmt.Errorf("change file line %d: unknown key %q", item.Content[k].Line, key)
			}
		}
		if ch.Property == "" {
			return nil, fmt.Errorf("change file line %d: property is required", item.Line)
		}
		changes = append(changes, ch)
	}
	return changes, nil
}

// AgentConfigDocument returns props as a nested service/type/name map in
// form order, the format ParseAgentConfigChanges reads. The agent's own
// (SystemService) properties are left out because they cannot be set.
func AgentConfigDocument(props []AgentConfigProperty) *yaml.Node {
	root := &yaml.Node{Kind: yaml.MappingNode}
	child := func(parent *yaml.Node, key string) *yaml.Node {
		for i := 0; i+1 < len(parent.Content); i += 2 {
			if parent.Content[i].Value == key {
				return parent.Content[i+1]
			}
		}
		n := &yaml.Node{Kind: yaml.MappingNode}
		parent.Content = append(parent.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, n)
		return n
	}
	for _, p := range props {
		if p.Service == SystemService {
			continue
		}
		val := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: p.Value}
		switch {
		case strings.Contains(p.Value, "\n"):
			val.Style = yaml.LiteralStyle
		case strings.Contains(p.Value, "'"):
			// Process Server values carry YAML quotes: write "'4G'", not '''4G'''.
			val.Style = yaml.DoubleQuotedStyle
		}
		types := child(root, p.Service)
		names := child(types, p.Type)
		names.Content = append(names.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: p.Name}, val)
	}
	return root
}

// AgentConfigMap returns the same content as AgentConfigDocument as plain
// maps, for JSON output.
func AgentConfigMap(props []AgentConfigProperty) map[string]map[string]map[string]string {
	out := map[string]map[string]map[string]string{}
	for _, p := range props {
		if p.Service == SystemService {
			continue
		}
		if out[p.Service] == nil {
			out[p.Service] = map[string]map[string]string{}
		}
		if out[p.Service][p.Type] == nil {
			out[p.Service][p.Type] = map[string]string{}
		}
		out[p.Service][p.Type][p.Name] = p.Value
	}
	return out
}
