package client

import (
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Custom properties ("Custom Configuration Details" in the Administrator UI)
// are extra properties added to a service, such as maxDTMProcesses for the
// Data Integration Server. Each one is a row of form fields:
//
//	customConfigEngine     service key, e.g. Data_Integration_Server77.0
//	customConfigType       legacy type, always -1 for services
//	customConfigTypeEngine property type, e.g. TOMCAT_CFG
//	customConfigSubType    -1
//	customConfigName       name (maxlength 255)
//	customConfigValue      value (maxlength 512)
//	customConfigSensitive  checkbox, sent only when checked
//	engineCustomConfigIdInput  row ID; new rows use the time in milliseconds
//	isSensitiveConfig      stored sensitive flag
//
// The form always submits at least one row; with no custom properties it is
// an empty template row with -1 values. The UI's add and remove actions are
// reproduced in submitFields.

const (
	maxCustomNameLength  = 255
	maxCustomValueLength = 512
)

// CustomConfigType is a property type a service accepts for custom
// properties, e.g. {TOMCAT_CFG, Tomcat}.
type CustomConfigType struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// customRow is one custom property row: fields[start] to fields[end].
type customRow struct {
	start, end int
	template   bool
}

var (
	customTypeSelRe   = regexp.MustCompile(`(?s)<select[^>]*name="customConfigTypeEngine"[^>]*>(.*?)</select>`)
	customEngineSelRe = regexp.MustCompile(`(?s)<select[^>]*name="customConfigEngine"[^>]*>(.*?)</select>`)
	customOptionRe    = regexp.MustCompile(`(?s)<option\b([^>]*)>([^<]*)`)
)

// propertyNotFoundError is returned by Find when no property matches.
type propertyNotFoundError struct {
	sel   PropertySelector
	agent string
}

func (e *propertyNotFoundError) Error() string {
	return fmt.Sprintf("property %s not found on agent %s", e.sel, e.agent)
}

// parseCustomRow reads the custom property row starting at fields[i] and
// returns the index of its last field.
func (f *AgentConfigForm) parseCustomRow(i int) (int, error) {
	row := customRow{start: i}
	vals := map[string]string{}
	valueIdx := -1
	sensitive := false
	j := i
	for ; j < len(f.fields); j++ {
		fld := f.fields[j]
		switch fld.name {
		case "customConfigValue":
			valueIdx = j
		case "customConfigSensitive":
			sensitive = true
		}
		if _, dup := vals[fld.name]; dup {
			return 0, fmt.Errorf("unexpected agent form layout in custom property row at field %d", i)
		}
		vals[fld.name] = fld.value
		if fld.name == "isSensitiveConfig" {
			break
		}
	}
	if j == len(f.fields) || valueIdx < 0 {
		return 0, fmt.Errorf("unexpected agent form layout in custom property row at field %d", i)
	}
	row.end = j
	engine := vals["customConfigEngine"]
	if engine == "-1" || engine == "" {
		row.template = true
		f.customRows = append(f.customRows, row)
		return j, nil
	}
	typ := vals["customConfigTypeEngine"]
	if typ == "-1" || typ == "" {
		typ = vals["customConfigType"]
	}
	sub := vals["customConfigSubType"]
	if sub == "-1" {
		sub = ""
	}
	f.customRows = append(f.customRows, row)
	f.Properties = append(f.Properties, AgentConfigProperty{
		Service:    versionTail.ReplaceAllString(engine, ""),
		Type:       typ,
		SubType:    sub,
		Name:       vals["customConfigName"],
		Value:      vals["customConfigValue"],
		Sensitive:  sensitive || vals["isSensitiveConfig"] == "true",
		Custom:     true,
		valueIdx:   valueIdx,
		rowStart:   i,
		serviceKey: engine,
	})
	return j, nil
}

// parseCustomTypes reads the custom property types each service offers, and
// the service names, from the first custom row's selects.
func (f *AgentConfigForm) parseCustomTypes(section string) {
	f.customTypes = map[string][]CustomConfigType{}
	if m := customTypeSelRe.FindStringSubmatch(section); m != nil {
		for _, om := range customOptionRe.FindAllStringSubmatch(m[1], -1) {
			a := parseAttrs(om[1])
			engine, value := a["data-engine"], a["value"]
			if engine == "" || value == "" || value == "-1" {
				continue
			}
			label := strings.TrimSpace(html.UnescapeString(om[2]))
			if label == "" {
				label = value
			}
			f.customTypes[engine] = append(f.customTypes[engine], CustomConfigType{Value: value, Label: label})
		}
	}
	if m := customEngineSelRe.FindStringSubmatch(section); m != nil {
		for _, om := range customOptionRe.FindAllStringSubmatch(m[1], -1) {
			a := parseAttrs(om[1])
			if v := a["value"]; v != "" && v != "-1" && f.serviceNames[v] == "" {
				f.serviceNames[v] = strings.TrimSpace(html.UnescapeString(om[2]))
			}
		}
	}
}

// typeValue returns the type value whose UI label is label for the service
// key, or "" when there is none.
func (f *AgentConfigForm) typeValue(serviceKey, label string) string {
	for _, t := range f.customTypes[serviceKey] {
		if strings.EqualFold(t.Label, label) {
			return t.Value
		}
	}
	return ""
}

// CustomTypes returns the custom property types by service name (without
// version), for the services on this agent.
func (f *AgentConfigForm) CustomTypes() map[string][]CustomConfigType {
	out := map[string][]CustomConfigType{}
	for key, types := range f.customTypes {
		out[versionTail.ReplaceAllString(key, "")] = types
	}
	return out
}

// customService finds the service key for a service name, versioned key or
// display name among the services that accept custom properties.
func (f *AgentConfigForm) customService(name string) (string, error) {
	var keys []string
	for key := range f.customTypes {
		if strings.EqualFold(name, key) || strings.EqualFold(name, versionTail.ReplaceAllString(key, "")) ||
			strings.EqualFold(name, f.serviceNames[key]) {
			return key, nil
		}
		keys = append(keys, versionTail.ReplaceAllString(key, ""))
	}
	sort.Strings(keys)
	return "", fmt.Errorf("service %q does not take custom properties on agent %s; services: %s", name, f.AgentName, strings.Join(keys, ", "))
}

// addCustom adds a custom property for sel.
func (f *AgentConfigForm) addCustom(sel PropertySelector, value string, sensitive bool) (AgentConfigDiff, error) {
	if len(f.customRows) == 0 {
		return AgentConfigDiff{}, fmt.Errorf("the agent form has no custom properties section")
	}
	key, err := f.customService(sel.Service)
	if err != nil {
		return AgentConfigDiff{}, err
	}
	if sel.SubType != "" {
		return AgentConfigDiff{}, fmt.Errorf("custom property %s: sub-types are not supported", sel)
	}
	typ := ""
	var labels []string
	for _, t := range f.customTypes[key] {
		if strings.EqualFold(sel.Type, t.Value) || strings.EqualFold(sel.Type, t.Label) {
			typ = t.Value
		}
		labels = append(labels, t.Value)
	}
	if typ == "" {
		return AgentConfigDiff{}, fmt.Errorf("custom property %s: type %q is not available for %s; types: %s",
			sel, sel.Type, f.serviceNames[key], strings.Join(labels, ", "))
	}
	name := strings.TrimSpace(sel.Name)
	switch {
	case name == "":
		return AgentConfigDiff{}, fmt.Errorf("custom property %s: the name is empty", sel)
	case len([]rune(name)) > maxCustomNameLength:
		return AgentConfigDiff{}, fmt.Errorf("custom property %s: the name is longer than %d characters", sel, maxCustomNameLength)
	case len([]rune(value)) > maxCustomValueLength:
		return AgentConfigDiff{}, fmt.Errorf("custom property %s: the value is longer than %d characters", sel, maxCustomValueLength)
	case strings.Contains(value, "\n"):
		return AgentConfigDiff{}, fmt.Errorf("custom property %s: the value must be a single line", sel)
	}
	p := AgentConfigProperty{
		Service:     versionTail.ReplaceAllString(key, ""),
		ServiceName: f.serviceNames[key],
		Type:        typ,
		Name:        name,
		Value:       value,
		Sensitive:   sensitive,
		Custom:      true,
		rowStart:    -1,
		serviceKey:  key,
	}
	f.Properties = append(f.Properties, p)
	return AgentConfigDiff{Property: p.Key(), ServiceName: p.ServiceName, NewValue: value,
		Sensitive: sensitive, Custom: true, Change: "add"}, nil
}

// submitFields returns the fields to submit: removed custom rows are left
// out, added custom properties follow the last custom row, and the empty
// template row is kept only while there is no custom property.
func (f *AgentConfigForm) submitFields() []formField {
	removedRows := map[int]bool{}
	var added []AgentConfigProperty
	for _, p := range f.Properties {
		switch {
		case p.Custom && p.removed && p.rowStart >= 0:
			removedRows[p.rowStart] = true
		case p.Custom && !p.removed && p.rowStart < 0:
			added = append(added, p)
		}
	}
	if len(removedRows) == 0 && len(added) == 0 {
		return f.fields
	}
	remaining := len(added)
	template := -1
	for _, r := range f.customRows {
		switch {
		case r.template:
			template = r.start
		case !removedRows[r.start]:
			remaining++
		}
	}
	skip := map[int]int{} // row start -> row end
	for _, r := range f.customRows {
		if removedRows[r.start] || r.template && remaining > 0 {
			skip[r.start] = r.end
		}
	}
	last := f.customRows[len(f.customRows)-1].end
	id := time.Now().UnixMilli()
	out := make([]formField, 0, len(f.fields)+9*len(added))
	for i := 0; i < len(f.fields); i++ {
		if end, ok := skip[i]; ok {
			i = end
		} else {
			out = append(out, f.fields[i])
		}
		if i != last {
			continue
		}
		for _, p := range added {
			out = append(out, customRowFields(p.serviceKey, p.Type, p.Name, p.Value, p.Sensitive, id)...)
			id++
		}
		if remaining == 0 && template < 0 {
			out = append(out, customRowFields("-1", "-1", "", "", false, id)...)
		}
	}
	return out
}

// customRowFields builds the fields of one custom property row as the UI
// submits a new row.
func customRowFields(serviceKey, typ, name, value string, sensitive bool, id int64) []formField {
	ids := strconv.FormatInt(id, 10)
	fields := []formField{
		{"customConfigEngine", serviceKey},
		{"customConfigType", "-1"},
		{"customConfigTypeEngine", typ},
		{"customConfigSubType", "-1"},
		{"customConfigName", name},
		{"customConfigValue", value},
	}
	if sensitive {
		fields = append(fields, formField{"customConfigSensitive", ids})
	}
	return append(fields,
		formField{"engineCustomConfigIdInput", ids},
		formField{"isSensitiveConfig", "false"})
}
