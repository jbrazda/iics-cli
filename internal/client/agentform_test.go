package client

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// agentFormPage is a trimmed agent.do page with the same markup as the
// Administrator UI: system rows, engine rows (input, textarea, sensitive
// checkbox) and the service selector.
const agentFormPage = `<html><body>
<select style="width: 200px;" id="engineConfigSel" onchange="pnAgent.onChangeEngineList(this)">
  <option value="process-engine15124581">Process Server</option>
  <option value="OpsInsightsDataCollector25">OI Data Collector</option>
</select>
<form id="agentForm" name="AgentForm" action="/runtime/app/agent.do" method="post" onsubmit="return false;">
<input type="hidden" name="name" value="CAI-A03"/>
<select id="sysConfigSubTypeSel0" name="sysConfigSubTypeSel"><option value="DEBUG" > DEBUG </option><option value="INFO" selected> INFO </option></select>
<input type="hidden" name="sysConfigType" value="AGENTCORE_LOG4J"/>
<input type="hidden" name="sysConfigSubType" value="INFO"/>
<input type="hidden" name="sysConfigName" value="log4j.rootLogger"/>
<input id="sysConfigValueInput0-0" type="text" name="sysConfigValue" value="INFO, aclog" style="display: none"/>
<input id="sysConfigDefaultInput0-0" type="hidden" name="sysConfigDefault" value="INFO, aclog"/>
<input id="sysConfigOverriddenInput0-0" type="hidden" name="sysConfigOverriden" value="false"/>
<input type="hidden" name="sysConfigType" value="AGENTCORE_LOG4J"/>
<input type="hidden" name="sysConfigSubType" value="DEBUG"/>
<input type="hidden" name="sysConfigName" value="log4j.rootLogger"/>
<input id="sysConfigValueInput0-1" type="text" name="sysConfigValue" value="DEBUG, aclog" style="display: none"/>
<input id="sysConfigDefaultInput0-1" type="hidden" name="sysConfigDefault" value="DEBUG, aclog"/>
<input id="sysConfigOverriddenInput0-1" type="hidden" name="sysConfigOverriden" value="false"/>
<input name="customConfigName" type="text" value="" disabled/>
<tr data-engine="process-engine15124581" data-config-type="cluster" class="engineConfigProperty">
  <input name="engineConfigKey" type="hidden" value="process-engine15124581clusterprimary-node"/>
  <td>cluster</td><td>primary-node</td>
  <td><input readonly name="engineConfigValue" type=text id="x1" value="true" style='opacity: 0.5;'/>
  <input id="d1" type="hidden" name="engineConfigDefault" value="true"/>
  <input id="o1" type="hidden" name="engineConfigOverriden" value="false"/></td>
  <td><input name="engineConfigSensitive" id="s1" onclick="pnAgent.sensitiveEngineConfig('process-engine15124581clusterprimary-node', this)" value="process-engine15124581clusterprimary-node" type="checkbox" /></td>
</tr>
<tr data-engine="process-engine15124581" data-config-type="db" class="engineConfigProperty">
  <input name="engineConfigKey" type="hidden" value="process-engine15124581dbconnection-properties"/>
  <td>db</td><td>connection-properties</td>
  <td><textarea readonly name="engineConfigValue" id="x2" style="display: none;">
- key: testOnBorrow
  value: true
</textarea>
  <input id="d2" type="hidden" name="engineConfigDefault" value="- key: testOnBorrow
  value: true
"/>
  <input id="o2" type="hidden" name="engineConfigOverriden" value="false"/></td>
  <td><input name="engineConfigSensitive" id="s2" value="process-engine15124581dbconnection-properties" type="checkbox" /></td>
</tr>
<tr data-engine="OpsInsightsDataCollector25" data-config-type="OpsInsights" class="engineConfigProperty">
  <input name="engineConfigKey" type="hidden" value="OpsInsightsDataCollector25OpsInsightssecretKey"/>
  <td>OpsInsights</td><td>secretKey</td>
  <td><input name="engineConfigValue" type=text id="x3" value="&#39;********&#39;"/>
  <input id="d3" type="hidden" name="engineConfigDefault" value="&#39;&#39;"/>
  <input id="o3" type="hidden" name="engineConfigOverriden" value="true"/></td>
  <td><input name="engineConfigSensitive" id="s3" value="OpsInsightsDataCollector25OpsInsightssecretKey" type="checkbox" checked /></td>
</tr>
<input id="id" name="id" type="hidden" value="01D23X0800000000008G"/>
<input id="version" name="version" type="hidden" value="119"/>
<input type="hidden" name="actionMethod" value="displayUpdate"/>
<input type="hidden" name="csrfToken" value="%s"/>
</form>
<script>var hasErrors = "%s";</script>
<div id="InfErrorDiv" style="display:none;">%s</div>
</body></html>`

func agentPage(csrf, hasErrors, errMsg string) string {
	r := strings.NewReplacer("%s", "\x00")
	parts := strings.Split(r.Replace(agentFormPage), "\x00")
	return parts[0] + csrf + parts[1] + hasErrors + parts[2] + errMsg + parts[3]
}

func TestParseAgentConfigForm(t *testing.T) {
	form, err := parseAgentConfigForm(agentPage("tok", "false", ""))
	if err != nil {
		t.Fatal(err)
	}
	if form.AgentID != "01D23X0800000000008G" || form.AgentName != "CAI-A03" || form.Version != "119" || form.csrf != "tok" {
		t.Errorf("form header = %q %q %q %q", form.AgentID, form.AgentName, form.Version, form.csrf)
	}
	if len(form.Properties) != 5 {
		t.Fatalf("got %d properties, want 5", len(form.Properties))
	}
	want := []struct{ key, value, serviceName string }{
		{"agent:AGENTCORE_LOG4J/INFO:log4j.rootLogger", "INFO, aclog", "Secure Agent"},
		{"agent:AGENTCORE_LOG4J/DEBUG:log4j.rootLogger", "DEBUG, aclog", "Secure Agent"},
		{"process-engine:cluster:primary-node", "true", "Process Server"},
		{"process-engine:db:connection-properties", "- key: testOnBorrow\n  value: true\n", "Process Server"},
		{"OpsInsightsDataCollector:OpsInsights:secretKey", "'********'", "OI Data Collector"},
	}
	for i, w := range want {
		p := form.Properties[i]
		if p.Key() != w.key || p.Value != w.value || p.ServiceName != w.serviceName {
			t.Errorf("property %d = %q %q %q, want %q %q %q", i, p.Key(), p.Value, p.ServiceName, w.key, w.value, w.serviceName)
		}
	}
	if !form.Properties[4].Sensitive || !form.Properties[4].Overridden || form.Properties[2].Sensitive {
		t.Error("sensitive or overridden flags not parsed")
	}
	for _, f := range form.fields {
		if f.name == "customConfigName" {
			t.Error("disabled field must not be submitted")
		}
		if f.name == "engineConfigSensitive" && f.value != "OpsInsightsDataCollector25OpsInsightssecretKey" {
			t.Errorf("unchecked checkbox submitted: %q", f.value)
		}
		if f.name == "sysConfigSubTypeSel" && f.value != "INFO" {
			t.Errorf("select submitted %q, want the selected option INFO", f.value)
		}
	}
}

func TestParseAgentConfigFormNoForm(t *testing.T) {
	if _, err := parseAgentConfigForm("<html>login</html>"); err == nil {
		t.Error("expected an error for a page without the agent form")
	}
}

func TestAgentConfigFormApply(t *testing.T) {
	form, err := parseAgentConfigForm(agentPage("tok", "false", ""))
	if err != nil {
		t.Fatal(err)
	}
	diffs, err := form.Apply([]AgentConfigChange{
		{Property: "Process Server:cluster:primary-node", Value: "false"},
		{Property: "process-engine:db:connection-properties", Value: "- key: testOnBorrow\r\n  value: true\r\n"}, // unchanged
		{Property: "opsinsightsdatacollector:opsinsights:secretkey", Reset: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 2 {
		t.Fatalf("got %d diffs, want 2: %+v", len(diffs), diffs)
	}
	if d := diffs[0]; d.Property != "process-engine:cluster:primary-node" || d.OldValue != "true" || d.NewValue != "false" || !d.NewOverridden {
		t.Errorf("set diff = %+v", d)
	}
	if d := diffs[1]; d.NewValue != "''" || d.NewOverridden || !d.Sensitive {
		t.Errorf("reset diff = %+v", d)
	}

	body, err := url.ParseQuery(form.encode())
	if err != nil {
		t.Fatal(err)
	}
	if body.Get("actionMethod") != "update" || body.Get("csrfToken") != "tok" {
		t.Errorf("actionMethod=%q csrfToken=%q", body.Get("actionMethod"), body.Get("csrfToken"))
	}
	values := body["engineConfigValue"]
	if values[0] != "false" || values[1] != "- key: testOnBorrow\r\n  value: true\r\n" || values[2] != "''" {
		t.Errorf("engineConfigValue = %q", values)
	}
	if over := body["engineConfigOverriden"]; over[0] != "true" || over[2] != "false" {
		t.Errorf("engineConfigOverriden = %q", over)
	}
}

func TestAgentConfigFormApplyErrors(t *testing.T) {
	form, err := parseAgentConfigForm(agentPage("tok", "false", ""))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][]AgentConfigChange{
		"ambiguous": {{Property: "agent:AGENTCORE_LOG4J:log4j.rootLogger", Value: "x"}},
		"system":    {{Property: "agent:AGENTCORE_LOG4J/INFO:log4j.rootLogger", Value: "x"}},
		"missing":   {{Property: "process-engine:jvm:max-heap", Value: "x"}},
		"syntax":    {{Property: "process-engine:max-heap", Value: "x"}},
		"twice": {
			{Property: "process-engine:cluster:primary-node", Value: "false"},
			{Property: "process-engine:cluster:primary-node", Reset: true},
		},
	}
	for name, changes := range cases {
		if _, err := form.Apply(changes); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestAgentConfigFormFilter(t *testing.T) {
	form, err := parseAgentConfigForm(agentPage("tok", "false", ""))
	if err != nil {
		t.Fatal(err)
	}
	got, err := form.Filter(AgentConfigFilter{Service: "process-engine", Name: "^primary"})
	if err != nil || len(got) != 1 || got[0].Name != "primary-node" {
		t.Errorf("Filter by service and name = %+v, %v", got, err)
	}
	got, _ = form.Filter(AgentConfigFilter{})
	if len(got) != 3 {
		t.Errorf("default Filter should hide the agent's own properties, got %d", len(got))
	}
	got, _ = form.Filter(AgentConfigFilter{Type: "AGENTCORE_LOG4J/DEBUG", IncludeHidden: true})
	if len(got) != 1 || got[0].SubType != "DEBUG" {
		t.Errorf("Filter by type/subType = %+v", got)
	}
	got, _ = form.Filter(AgentConfigFilter{Service: "agent"})
	if len(got) != 2 {
		t.Errorf("Filter with --service agent = %d properties, want 2", len(got))
	}
	got, _ = form.Filter(AgentConfigFilter{Overridden: true})
	if len(got) != 1 || got[0].Name != "secretKey" {
		t.Errorf("Filter overridden = %+v", got)
	}
	if _, err := form.Filter(AgentConfigFilter{Name: "("}); err == nil {
		t.Error("expected an error for an invalid pattern")
	}
}

func TestGetAndSaveAgentConfigForm(t *testing.T) {
	var posted url.Values
	var postCookies map[string]string
	hasErrors, errMsg := "false", ""
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != agentFormPath {
			t.Errorf("path = %s", r.URL.Path)
		}
		cookies := map[string]string{}
		for _, c := range r.Cookies() {
			cookies[c.Name] = c.Value
		}
		if cookies["USER_SESSION"] != "test-session" || cookies["XSRF_TOKEN"] == "" {
			t.Errorf("cookies = %v", cookies)
		}
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("actionMethod") != "displayUpdate" || r.URL.Query().Get("id") != "01D23X0800000000008G" {
				t.Errorf("query = %s", r.URL.RawQuery)
			}
			http.SetCookie(w, &http.Cookie{Name: "JSESSIONID", Value: "js1"})
			// The server echoes the XSRF_TOKEN cookie into the form.
			_, _ = io.WriteString(w, agentPage(cookies["XSRF_TOKEN"], "false", ""))
		case http.MethodPost:
			if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
				t.Errorf("Content-Type = %s", ct)
			}
			data, _ := io.ReadAll(r.Body)
			posted, _ = url.ParseQuery(string(data))
			postCookies = cookies
			_, _ = io.WriteString(w, agentPage(cookies["XSRF_TOKEN"], hasErrors, errMsg))
		}
	})
	c := newTestClient(handler)
	ctx := t.Context()

	form, err := c.GetAgentConfigForm(ctx, "01D23X0800000000008G")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = form.Apply([]AgentConfigChange{{Property: "process-engine:cluster:primary-node", Value: "false"}}); err != nil {
		t.Fatal(err)
	}
	if err = c.SaveAgentConfigForm(ctx, form); err != nil {
		t.Fatal(err)
	}
	if posted.Get("actionMethod") != "update" || posted["engineConfigValue"][0] != "false" {
		t.Errorf("posted actionMethod=%q value=%q", posted.Get("actionMethod"), posted["engineConfigValue"][0])
	}
	if posted.Get("csrfToken") == "" || posted.Get("csrfToken") != postCookies["XSRF_TOKEN"] {
		t.Errorf("csrfToken %q does not match XSRF_TOKEN cookie %q", posted.Get("csrfToken"), postCookies["XSRF_TOKEN"])
	}
	if postCookies["JSESSIONID"] != "js1" {
		t.Errorf("JSESSIONID cookie not sent back: %v", postCookies)
	}

	hasErrors, errMsg = "true", "<span>For configuration primary-node invalid value</span>"
	err = c.SaveAgentConfigForm(ctx, form)
	if err == nil || !strings.Contains(err.Error(), "For configuration primary-node invalid value") {
		t.Errorf("expected the page error, got %v", err)
	}
}

func TestGetAgentConfigFormRelogin(t *testing.T) {
	logins := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		logins++
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"products":[{"name":"Integration Cloud","baseApiUrl":"`+"http://"+r.Host+`"}],"userInfo":{"sessionId":"new-session"}}`)
	})
	mux.HandleFunc(agentFormPath, func(w http.ResponseWriter, r *http.Request) {
		if ck, _ := r.Cookie("USER_SESSION"); ck == nil || ck.Value != "new-session" {
			_, _ = io.WriteString(w, "<html>login page</html>")
			return
		}
		ck, _ := r.Cookie("XSRF_TOKEN")
		_, _ = io.WriteString(w, agentPage(ck.Value, "false", ""))
	})
	c := newTestClient(mux)
	if _, err := c.GetAgentConfigForm(t.Context(), "01D23X0800000000008G"); err != nil {
		t.Fatal(err)
	}
	if logins != 1 {
		t.Errorf("logins = %d, want 1", logins)
	}
}

func TestParsePropertySelector(t *testing.T) {
	sel, err := ParsePropertySelector("agent:AGENTCORE_LOG4J/DEBUG:log4j.logger.a:b")
	if err != nil || sel.Service != "agent" || sel.Type != "AGENTCORE_LOG4J" || sel.SubType != "DEBUG" || sel.Name != "log4j.logger.a:b" {
		t.Errorf("ParsePropertySelector = %+v, %v", sel, err)
	}
	for _, bad := range []string{"", "a:b", "a::c", ":b:c"} {
		if _, err := ParsePropertySelector(bad); err == nil {
			t.Errorf("ParsePropertySelector(%q): expected an error", bad)
		}
	}
}

func TestRenameAgent(t *testing.T) {
	var postedName string
	name := "cletinfacld07"
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/agent", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `[{"id":"01D23X0800000000008G","name":"`+name+`"},{"id":"other","name":"CAI-A01","agentHost":"h1"}]`)
	})
	mux.HandleFunc("/api/v2/agent/01D23X0800000000008G", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"id":"01D23X0800000000008G","name":"`+name+`"}`)
	})
	mux.HandleFunc(agentFormPath, func(w http.ResponseWriter, r *http.Request) {
		ck, _ := r.Cookie("XSRF_TOKEN")
		if r.Method == http.MethodPost {
			data, _ := io.ReadAll(r.Body)
			v, _ := url.ParseQuery(string(data))
			postedName = v.Get("name")
			name = postedName
		}
		_, _ = io.WriteString(w, strings.Replace(agentPage(ck.Value, "false", ""), `value="CAI-A03"`, `value="`+name+`"`, 1))
	})
	c := newTestClient(mux)
	ctx := t.Context()

	agent, err := c.RenameAgent(ctx, "01D23X0800000000008G", " CAI-A03 ")
	if err != nil {
		t.Fatal(err)
	}
	if postedName != "CAI-A03" || agent.Name != "CAI-A03" {
		t.Errorf("posted %q, agent name %q", postedName, agent.Name)
	}
	if _, err := c.RenameAgent(ctx, "01D23X0800000000008G", "cai-a01"); err == nil || !strings.Contains(err.Error(), "already has the name") {
		t.Errorf("expected a duplicate name error, got %v", err)
	}
	for _, bad := range []string{"  ", strings.Repeat("x", 101)} {
		if _, err := c.RenameAgent(ctx, "01D23X0800000000008G", bad); err == nil {
			t.Errorf("RenameAgent(%q): expected an error", bad)
		}
	}
}

func TestResetAllChanges(t *testing.T) {
	form, err := parseAgentConfigForm(agentPage("tok", "false", ""))
	if err != nil {
		t.Fatal(err)
	}
	form.Set(2, "false") // process-engine:cluster:primary-node, now overridden
	changes, skipped, err := form.ResetAllChanges("")
	if err != nil || len(changes) != 1 || changes[0].Property != "process-engine:cluster:primary-node" || !changes[0].Reset || skipped != 1 {
		t.Errorf("ResetAllChanges = %+v, %d, %v", changes, skipped, err)
	}
	if changes, _, _ = form.ResetAllChanges("OI Data Collector"); len(changes) != 0 {
		t.Errorf("service filter: %+v", changes)
	}
	if _, _, err = form.ResetAllChanges("nope"); err == nil {
		t.Error("expected an error for an unknown service")
	}
}
