package client

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// Agent-level Secure Agent service properties have no public REST API. The
// Administrator UI edits them through a server-rendered form at
// /runtime/app/agent.do: GET ?actionMethod=displayUpdate&id=<agentId> renders
// every property as form fields, and POST actionMethod=update submits the whole
// form back. This file reads and writes that form the same way the browser
// does. The endpoint is undocumented and may change without notice.
//
// Authentication: the REST session ID is accepted as the USER_SESSION cookie.
// CSRF protection is a double-submit check: the XSRF_TOKEN cookie must match
// the csrfToken form field, so a random token is sent in both.

const agentFormPath = "/runtime/app/agent.do"

// SystemService is the service name used for the Secure Agent's own
// (agent core, agent manager, Tomcat) properties.
const SystemService = "agent"

// AgentConfigProperty is one editable property on an agent's configuration
// form.
type AgentConfigProperty struct {
	// Service is the service name without its version suffix, for example
	// process-engine, or SystemService for the agent's own properties.
	Service string `json:"service"`
	// ServiceName is the display name, for example "Process Server".
	ServiceName string `json:"serviceName,omitempty"`
	// Type is the property type, for example cluster or AGENTCORE_CFG.
	Type string `json:"type"`
	// SubType is INFO or DEBUG for agent log4j and other system properties.
	SubType      string `json:"subType,omitempty"`
	Name         string `json:"name"`
	Value        string `json:"value"`
	DefaultValue string `json:"defaultValue"`
	Overridden   bool   `json:"overridden"`
	Sensitive    bool   `json:"sensitive,omitempty"`

	// Indexes into AgentConfigForm.fields for the value and overridden flag.
	valueIdx, overIdx int
	// serviceKey is the versioned service key used in the form, for example
	// process-engine15124581.
	serviceKey string
}

// Key returns service:type:name, adding /subType to the type when set.
func (p AgentConfigProperty) Key() string {
	t := p.Type
	if p.SubType != "" {
		t += "/" + p.SubType
	}
	return p.Service + ":" + t + ":" + p.Name
}

// formField is one name=value pair the browser would submit, in form order.
type formField struct {
	name, value string
}

// AgentConfigForm is a parsed agent configuration form.
type AgentConfigForm struct {
	AgentID    string                `json:"agentId"`
	AgentName  string                `json:"agentName"`
	Version    string                `json:"version"`
	Properties []AgentConfigProperty `json:"properties"`

	fields  []formField
	cookies []*http.Cookie
	csrf    string
}

var (
	formTagRe   = regexp.MustCompile(`(?is)<(input|textarea|select)\b([^>]*)>`)
	formAttrRe  = regexp.MustCompile(`(?s)([a-zA-Z_:][-a-zA-Z0-9_:.]*)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'>]+)))?`)
	formOptRe   = regexp.MustCompile(`(?is)<option\b([^>]*)>`)
	engineRowRe = regexp.MustCompile(`<tr data-engine="([^"]*)" data-config-type="([^"]*)" class="engineConfigProperty">\s*<input name="engineConfigKey" type="hidden" value="([^"]*)"`)
	engineSelRe = regexp.MustCompile(`(?s)<select[^>]*id="engineConfigSel"[^>]*>(.*?)</select>`)
	optionRe    = regexp.MustCompile(`(?s)<option value="([^"]*)"[^>]*>(.*?)</option>`)
	sensitiveRe = regexp.MustCompile(`<input name="engineConfigSensitive"[^>]*>`)
	hasErrorsRe = regexp.MustCompile(`var hasErrors = "(true|false)"`)
	errorDivRe  = regexp.MustCompile(`(?s)<div id="InfErrorDiv"[^>]*>(.*?)</div>`)
	tagRe       = regexp.MustCompile(`<[^>]*>`)
	versionTail = regexp.MustCompile(`[0-9.]+$`)
)

// normalizeNewlines turns CRLF and CR into LF. Values are kept with LF and
// sent with CRLF, as browsers do.
func normalizeNewlines(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

func parseAttrs(s string) map[string]string {
	attrs := map[string]string{}
	for _, m := range formAttrRe.FindAllStringSubmatch(s, -1) {
		name := strings.ToLower(m[1])
		if _, ok := attrs[name]; ok {
			continue
		}
		attrs[name] = normalizeNewlines(html.UnescapeString(m[2] + m[3] + m[4]))
	}
	return attrs
}

// parseFormFields returns the fields a browser submits for the form body,
// in document order: inputs (checkboxes only when checked), textareas and
// the selected option of each select. Disabled and nameless fields are
// skipped.
func parseFormFields(body string) []formField {
	var fields []formField
	pos := 0
	for {
		loc := formTagRe.FindStringSubmatchIndex(body[pos:])
		if loc == nil {
			return fields
		}
		tag := strings.ToLower(body[pos+loc[2] : pos+loc[3]])
		attrs := parseAttrs(body[pos+loc[4] : pos+loc[5]])
		end := pos + loc[1]
		pos = end
		name := attrs["name"]
		if _, disabled := attrs["disabled"]; name == "" || disabled {
			continue
		}
		switch tag {
		case "input":
			switch strings.ToLower(attrs["type"]) {
			case "checkbox", "radio":
				if _, ok := attrs["checked"]; !ok {
					continue
				}
				v, ok := attrs["value"]
				if !ok {
					v = "on"
				}
				fields = append(fields, formField{name, v})
			case "submit", "button", "image", "reset", "file":
			default:
				fields = append(fields, formField{name, attrs["value"]})
			}
		case "textarea":
			closeIdx := strings.Index(strings.ToLower(body[end:]), "</textarea>")
			if closeIdx < 0 {
				return fields
			}
			// Browsers drop one newline right after the opening tag.
			v := strings.TrimPrefix(normalizeNewlines(html.UnescapeString(body[end:end+closeIdx])), "\n")
			fields = append(fields, formField{name, v})
			pos = end + closeIdx
		case "select":
			closeIdx := strings.Index(strings.ToLower(body[end:]), "</select>")
			if closeIdx < 0 {
				return fields
			}
			var first, selected string
			haveFirst, haveSelected := false, false
			for _, om := range formOptRe.FindAllStringSubmatch(body[end:end+closeIdx], -1) {
				oa := parseAttrs(om[1])
				if !haveFirst {
					first, haveFirst = oa["value"], true
				}
				if _, ok := oa["selected"]; ok && !haveSelected {
					selected, haveSelected = oa["value"], true
				}
			}
			switch {
			case haveSelected:
				fields = append(fields, formField{name, selected})
			case haveFirst:
				fields = append(fields, formField{name, first})
			}
			pos = end + closeIdx
		}
	}
}

// agentFormSection returns the agentForm element's contents, or "" if the
// page has no agent form (for example a login page).
func agentFormSection(page string) string {
	start := strings.Index(page, `<form id="agentForm"`)
	if start < 0 {
		return ""
	}
	end := strings.Index(page[start:], "</form>")
	if end < 0 {
		return page[start:]
	}
	return page[start : start+end]
}

// parseAgentConfigForm parses the agent configuration page.
func parseAgentConfigForm(page string) (*AgentConfigForm, error) {
	section := agentFormSection(page)
	if section == "" {
		return nil, fmt.Errorf("agent configuration form not found in response")
	}
	form := &AgentConfigForm{fields: parseFormFields(section)}

	type engineRow struct{ serviceKey, typ string }
	rows := map[string]engineRow{}
	for _, m := range engineRowRe.FindAllStringSubmatch(section, -1) {
		rows[html.UnescapeString(m[3])] = engineRow{html.UnescapeString(m[1]), html.UnescapeString(m[2])}
	}
	serviceNames := map[string]string{}
	if m := engineSelRe.FindStringSubmatch(page); m != nil {
		for _, om := range optionRe.FindAllStringSubmatch(m[1], -1) {
			serviceNames[html.UnescapeString(om[1])] = strings.TrimSpace(html.UnescapeString(om[2]))
		}
	}
	sensitive := map[string]bool{}
	for _, tag := range sensitiveRe.FindAllString(section, -1) {
		a := parseAttrs(strings.TrimSuffix(strings.TrimPrefix(tag, "<input"), ">"))
		if _, ok := a["checked"]; ok {
			sensitive[a["value"]] = true
		}
	}

	f := form.fields
	for i := 0; i < len(f); i++ {
		switch f[i].name {
		case "id":
			form.AgentID = f[i].value
		case "name":
			form.AgentName = f[i].value
		case "version":
			form.Version = f[i].value
		case "csrfToken":
			form.csrf = f[i].value
		case "sysConfigType":
			if i+5 >= len(f) || f[i+1].name != "sysConfigSubType" || f[i+2].name != "sysConfigName" ||
				f[i+3].name != "sysConfigValue" || f[i+4].name != "sysConfigDefault" || f[i+5].name != "sysConfigOverriden" {
				return nil, fmt.Errorf("unexpected agent form layout at field %d (%s)", i, f[i].name)
			}
			form.Properties = append(form.Properties, AgentConfigProperty{
				Service:      SystemService,
				ServiceName:  "Secure Agent",
				Type:         f[i].value,
				SubType:      f[i+1].value,
				Name:         f[i+2].value,
				Value:        f[i+3].value,
				DefaultValue: f[i+4].value,
				Overridden:   f[i+5].value == "true",
				valueIdx:     i + 3,
				overIdx:      i + 5,
			})
			i += 5
		case "engineConfigKey":
			if i+3 >= len(f) || f[i+1].name != "engineConfigValue" || f[i+2].name != "engineConfigDefault" || f[i+3].name != "engineConfigOverriden" {
				return nil, fmt.Errorf("unexpected agent form layout at field %d (%s)", i, f[i].name)
			}
			key := f[i].value
			row, ok := rows[key]
			if !ok {
				return nil, fmt.Errorf("no service row found for agent form property %q", key)
			}
			form.Properties = append(form.Properties, AgentConfigProperty{
				Service:      versionTail.ReplaceAllString(row.serviceKey, ""),
				ServiceName:  serviceNames[row.serviceKey],
				Type:         row.typ,
				Name:         strings.TrimPrefix(key, row.serviceKey+row.typ),
				Value:        f[i+1].value,
				DefaultValue: f[i+2].value,
				Overridden:   f[i+3].value == "true",
				Sensitive:    sensitive[key],
				valueIdx:     i + 1,
				overIdx:      i + 3,
				serviceKey:   row.serviceKey,
			})
			i += 3
		}
	}
	if form.AgentID == "" || form.Version == "" {
		return nil, fmt.Errorf("agent form is missing the agent id or version")
	}
	return form, nil
}

// PropertySelector identifies agent form properties: Service matches the
// service name, its versioned key or display name; Type may carry a /SubType
// suffix. Matching is case-insensitive.
type PropertySelector struct {
	Service, Type, SubType, Name string
}

// ParsePropertySelector parses service:type[/subtype]:name.
func ParsePropertySelector(s string) (PropertySelector, error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return PropertySelector{}, fmt.Errorf("invalid property %q: expected service:type:name", s)
	}
	sel := PropertySelector{Service: parts[0], Type: parts[1], Name: parts[2]}
	if t, sub, ok := strings.Cut(sel.Type, "/"); ok {
		sel.Type, sel.SubType = t, sub
	}
	return sel, nil
}

func (s PropertySelector) matches(p AgentConfigProperty) bool {
	svc := strings.EqualFold(s.Service, p.Service) || strings.EqualFold(s.Service, p.serviceKey) ||
		(p.ServiceName != "" && strings.EqualFold(s.Service, p.ServiceName))
	return svc && strings.EqualFold(s.Type, p.Type) && strings.EqualFold(s.Name, p.Name) &&
		(s.SubType == "" || strings.EqualFold(s.SubType, p.SubType))
}

func (s PropertySelector) String() string {
	t := s.Type
	if s.SubType != "" {
		t += "/" + s.SubType
	}
	return s.Service + ":" + t + ":" + s.Name
}

// Find returns the index of the one property matching sel.
func (f *AgentConfigForm) Find(sel PropertySelector) (int, error) {
	var hits []int
	for i, p := range f.Properties {
		if sel.matches(p) {
			hits = append(hits, i)
		}
	}
	switch len(hits) {
	case 1:
		return hits[0], nil
	case 0:
		return -1, fmt.Errorf("property %s not found on agent %s", sel, f.AgentName)
	}
	keys := make([]string, len(hits))
	for i, h := range hits {
		keys[i] = f.Properties[h].Key()
	}
	sort.Strings(keys)
	return -1, fmt.Errorf("property %s is ambiguous, use one of: %s", sel, strings.Join(keys, ", "))
}

// Set changes property i to value and marks it overridden, as the UI's edit
// action does.
func (f *AgentConfigForm) Set(i int, value string) {
	p := &f.Properties[i]
	p.Value, p.Overridden = value, true
	f.fields[p.valueIdx].value = value
	f.fields[p.overIdx].value = "true"
}

// Reset sets property i back to its default, as the UI's reset action does.
func (f *AgentConfigForm) Reset(i int) {
	p := &f.Properties[i]
	p.Value, p.Overridden = p.DefaultValue, false
	f.fields[p.valueIdx].value = p.DefaultValue
	f.fields[p.overIdx].value = "false"
}

// encode builds the form body for actionMethod=update, keeping field order.
func (f *AgentConfigForm) encode() string {
	var b strings.Builder
	for i, fld := range f.fields {
		v := fld.value
		switch fld.name {
		case "actionMethod":
			v = "update"
		case "csrfToken":
			v = f.csrf
		}
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(url.QueryEscape(fld.name))
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(strings.ReplaceAll(normalizeNewlines(v), "\n", "\r\n")))
	}
	return b.String()
}

func newCSRFToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating CSRF token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// agentFormURL returns the agent.do URL on the host of the base API URL.
func (c *Client) agentFormURL(query url.Values) (string, error) {
	base, err := url.Parse(c.BaseAPIURL())
	if err != nil {
		return "", fmt.Errorf("parsing base API URL: %w", err)
	}
	u := url.URL{Scheme: base.Scheme, Host: base.Host, Path: agentFormPath, RawQuery: query.Encode()}
	return u.String(), nil
}

// doAgentForm sends one request to agent.do with the session and CSRF
// cookies and returns the page.
func (c *Client) doAgentForm(ctx context.Context, method, u, body, csrf string, cookies []*http.Cookie) (string, []*http.Cookie, error) {
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rd)
	if err != nil {
		return "", nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "text/html")
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", req.URL.Scheme+"://"+req.URL.Host)
	}
	req.AddCookie(&http.Cookie{Name: "USER_SESSION", Value: c.SessionID()})
	req.AddCookie(&http.Cookie{Name: "XSRF_TOKEN", Value: csrf})
	for _, ck := range cookies {
		req.AddCookie(&http.Cookie{Name: ck.Name, Value: ck.Value})
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("%s %s: %w", method, agentFormPath, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("reading response: %w", err)
	}
	if c.debug {
		debugPrintHTTP(req, nil, resp, nil)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, newAPIError(resp, data)
	}
	return string(data), resp.Cookies(), nil
}

// GetAgentConfigForm reads the configuration form of agent agentID.
// If the session is not accepted it logs in again once.
func (c *Client) GetAgentConfigForm(ctx context.Context, agentID string) (*AgentConfigForm, error) {
	if err := c.ensureSession(ctx); err != nil {
		return nil, err
	}
	csrf, err := newCSRFToken()
	if err != nil {
		return nil, err
	}
	for attempt := 0; ; attempt++ {
		u, err := c.agentFormURL(url.Values{"actionMethod": {"displayUpdate"}, "id": {agentID}})
		if err != nil {
			return nil, err
		}
		page, cookies, err := c.doAgentForm(ctx, http.MethodGet, u, "", csrf, nil)
		if err != nil {
			return nil, err
		}
		if agentFormSection(page) == "" && attempt == 0 {
			if _, loginErr := c.Login(ctx); loginErr != nil {
				return nil, &SessionExpiredError{Wrapped: fmt.Errorf("%w; run 'iics login' to re-authenticate", loginErr)}
			}
			continue
		}
		form, err := parseAgentConfigForm(page)
		if err != nil {
			return nil, fmt.Errorf("reading agent %s configuration: %w", agentID, err)
		}
		form.cookies = cookies
		if form.csrf == "" {
			form.csrf = csrf
		}
		return form, nil
	}
}

// SaveAgentConfigForm submits form, with all its properties, as the UI's
// Save button does. It returns an error when the page reports one.
func (c *Client) SaveAgentConfigForm(ctx context.Context, form *AgentConfigForm) error {
	u, err := c.agentFormURL(nil)
	if err != nil {
		return err
	}
	page, _, err := c.doAgentForm(ctx, http.MethodPost, u, form.encode(), form.csrf, form.cookies)
	if err != nil {
		return err
	}
	if agentFormSection(page) == "" {
		return fmt.Errorf("saving agent %s configuration: unexpected response (session or CSRF token not accepted)", form.AgentID)
	}
	if m := hasErrorsRe.FindStringSubmatch(page); m != nil && m[1] == "true" {
		msg := "the server reported an error"
		if d := errorDivRe.FindStringSubmatch(page); d != nil {
			if t := strings.Join(strings.Fields(html.UnescapeString(tagRe.ReplaceAllString(d[1], " "))), " "); t != "" {
				msg = t
			}
		}
		return fmt.Errorf("saving agent %s configuration: %s", form.AgentID, msg)
	}
	return nil
}

// AgentConfigFilter selects properties for display. Empty fields match all.
type AgentConfigFilter struct {
	Service    string // service name, versioned key or display name
	Type       string // property type, optionally type/subType
	Name       string // regular expression on the property name, case-insensitive
	Overridden bool   // only overridden properties
}

// Filter returns the properties matching flt.
func (f *AgentConfigForm) Filter(flt AgentConfigFilter) ([]AgentConfigProperty, error) {
	var nameRe *regexp.Regexp
	if flt.Name != "" {
		re, err := regexp.Compile("(?i)" + flt.Name)
		if err != nil {
			return nil, fmt.Errorf("invalid name pattern: %w", err)
		}
		nameRe = re
	}
	typ, sub, _ := strings.Cut(flt.Type, "/")
	var out []AgentConfigProperty
	for _, p := range f.Properties {
		if flt.Service != "" && !strings.EqualFold(flt.Service, p.Service) && !strings.EqualFold(flt.Service, p.serviceKey) &&
			!strings.EqualFold(flt.Service, p.ServiceName) {
			continue
		}
		if typ != "" && !strings.EqualFold(typ, p.Type) || sub != "" && !strings.EqualFold(sub, p.SubType) {
			continue
		}
		if nameRe != nil && !nameRe.MatchString(p.Name) || flt.Overridden && !p.Overridden {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

// AgentConfigChange is one requested property change: a new value, or a
// reset to the default when Reset is true.
type AgentConfigChange struct {
	Property string `json:"property"` // service:type[/subType]:name
	Value    string `json:"value,omitempty"`
	Reset    bool   `json:"reset,omitempty"`
}

// AgentConfigDiff describes a change applied to the form.
type AgentConfigDiff struct {
	Property      string `json:"property"`
	ServiceName   string `json:"serviceName,omitempty"`
	OldValue      string `json:"oldValue"`
	NewValue      string `json:"newValue"`
	OldOverridden bool   `json:"oldOverridden"`
	NewOverridden bool   `json:"newOverridden"`
	Sensitive     bool   `json:"sensitive,omitempty"`
}

// Apply applies changes to the form and returns the properties that change.
// Setting a property to its current value leaves it untouched, so a default
// value does not become an override.
func (f *AgentConfigForm) Apply(changes []AgentConfigChange) ([]AgentConfigDiff, error) {
	var diffs []AgentConfigDiff
	seen := map[int]bool{}
	for _, ch := range changes {
		sel, err := ParsePropertySelector(ch.Property)
		if err != nil {
			return nil, err
		}
		i, err := f.Find(sel)
		if err != nil {
			return nil, err
		}
		if f.Properties[i].Service == SystemService {
			return nil, fmt.Errorf("property %s: changing the agent's own (%s) properties is not supported; the server ignores them on save", f.Properties[i].Key(), SystemService)
		}
		if seen[i] {
			return nil, fmt.Errorf("property %s is changed more than once", f.Properties[i].Key())
		}
		seen[i] = true
		old := f.Properties[i]
		switch value := normalizeNewlines(ch.Value); {
		case ch.Reset:
			f.Reset(i)
		case value != old.Value:
			f.Set(i, value)
		}
		p := f.Properties[i]
		if p.Value == old.Value && p.Overridden == old.Overridden {
			continue
		}
		diffs = append(diffs, AgentConfigDiff{
			Property: p.Key(), ServiceName: p.ServiceName, OldValue: old.Value, NewValue: p.Value,
			OldOverridden: old.Overridden, NewOverridden: p.Overridden, Sensitive: p.Sensitive,
		})
	}
	return diffs, nil
}

// VerifyAgentConfigDiffs reads the agent form again and checks that each
// change was stored. Sensitive values are only checked for the overridden
// flag, as the form may not return them.
func (c *Client) VerifyAgentConfigDiffs(ctx context.Context, agentID string, diffs []AgentConfigDiff) error {
	form, err := c.GetAgentConfigForm(ctx, agentID)
	if err != nil {
		return fmt.Errorf("verifying changes: %w", err)
	}
	var failed []string
	for _, d := range diffs {
		sel, err := ParsePropertySelector(d.Property)
		if err != nil {
			return err
		}
		i, err := form.Find(sel)
		if err != nil {
			failed = append(failed, d.Property+" (not found)")
			continue
		}
		p := form.Properties[i]
		if p.Overridden != d.NewOverridden || !d.Sensitive && p.Value != d.NewValue {
			failed = append(failed, fmt.Sprintf("%s (now %q, overridden %t)", d.Property, p.Value, p.Overridden))
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("the agent did not store these changes: %s", strings.Join(failed, "; "))
	}
	return nil
}
