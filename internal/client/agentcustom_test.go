package client

import (
	"net/url"
	"strings"
	"testing"
)

// customRowHTML is one custom property row as the Administrator UI renders
// it. engine and typ are the selected options; "-1" leaves them unselected.
func customRowHTML(engine, typ, name, value, id string) string {
	sel := func(v, want string) string {
		if v == want {
			return " selected"
		}
		return ""
	}
	return `<tr>
<td><select name="customConfigEngine" index="0" class="text customConfigIndex">
  <option value="-1"/>
  <option ` + sel(engine, "Data_Integration_Server77.0") + ` value="Data_Integration_Server77.0">Data Integration Server</option>
  <option ` + sel(engine, "OpsInsightsDataCollector25") + ` value="OpsInsightsDataCollector25">OI Data Collector</option>
</select></td>
<td><div><select name="customConfigType" style="display: none;"><option value="-1"/></select></div>
<div><select name="customConfigTypeEngine" class="text customConfigIndex">
  <option value="-1"/>
  <option ` + sel(typ, "TOMCAT_CFG") + ` style="display: none;" data-engine="Data_Integration_Server77.0" value="TOMCAT_CFG">
    Tomcat
  </option>
  <option ` + sel(typ, "PMRDTM_CFG") + ` style="display: none;" data-engine="Data_Integration_Server77.0" value="PMRDTM_CFG">
    DTM
  </option>
  <option ` + sel(typ, "OpsInsights") + ` style="display: none;" data-engine="OpsInsightsDataCollector25" value="OpsInsights">
    OpsInsights
  </option>
</select></div></td>
<td><select name="customConfigSubType"><option value="-1"/></select></td>
<td><input type="text" name="customConfigName" value="` + name + `" maxlength="255"/></td>
<td><input type=text name="customConfigValue" value="` + value + `" maxlength="512"/></td>
<td><input type="checkbox" name="customConfigSensitive" value="` + id + `" /></td>
<input type="hidden" name="engineCustomConfigIdInput" value="` + id + `"/>
<input type="hidden" name="isSensitiveConfig" value="false"/>
</tr>`
}

func customFormPage(rows ...string) string {
	return `<html><body>
<select id="engineConfigSel"><option value="Data_Integration_Server77.0">Data Integration Server</option></select>
<form id="agentForm" name="AgentForm" action="/runtime/app/agent.do" method="post">
<input type="hidden" name="name" value="CDI-G01-A01"/>
<tr data-engine="Data_Integration_Server77.0" data-config-type="TOMCAT_CFG" class="engineConfigProperty">
  <input name="engineConfigKey" type="hidden" value="Data_Integration_Server77.0TOMCAT_CFGmaxThreads"/>
  <td><input name="engineConfigValue" type=text value="200"/>
  <input type="hidden" name="engineConfigDefault" value="200"/>
  <input type="hidden" name="engineConfigOverriden" value="false"/></td>
</tr>
<table id="customConfigTable">` + strings.Join(rows, "\n") + `</table>
<input name="id" type="hidden" value="01D23X08000000000003"/>
<input name="version" type="hidden" value="7"/>
<input type="hidden" name="actionMethod" value="displayUpdate"/>
<input type="hidden" name="csrfToken" value="tok"/>
</form><script>var hasErrors = "false";</script></body></html>`
}

// submitted returns the custom rows of the encoded form as
// "engine|type|name=value|id" strings.
func submitted(t *testing.T, f *AgentConfigForm) []string {
	t.Helper()
	body := f.encode()
	var rows []string
	var cur []string
	for _, kv := range strings.Split(body, "&") {
		k, v, _ := strings.Cut(kv, "=")
		v, _ = url.QueryUnescape(v)
		switch k {
		case "customConfigEngine", "customConfigTypeEngine":
			cur = append(cur, v)
		case "customConfigName":
			cur = append(cur, v)
		case "customConfigValue":
			cur[len(cur)-1] += "=" + v
		case "isSensitiveConfig":
			rows = append(rows, strings.Join(cur, "|"))
			cur = nil
		}
	}
	return rows
}

func TestParseCustomProperties(t *testing.T) {
	page := customFormPage(
		customRowHTML("Data_Integration_Server77.0", "TOMCAT_CFG", "maxDTMProcesses", "5", "1170373"),
		customRowHTML("Data_Integration_Server77.0", "PMRDTM_CFG", "MatchQuotesPastEndOfLine", "Yes", "1170385"))
	f, err := parseAgentConfigForm(page)
	if err != nil {
		t.Fatal(err)
	}
	var custom []string
	for _, p := range f.Properties {
		if p.Custom {
			custom = append(custom, p.Key()+"="+p.Value+" ("+p.ServiceName+")")
		}
	}
	want := "Data_Integration_Server:TOMCAT_CFG:maxDTMProcesses=5 (Data Integration Server)," +
		"Data_Integration_Server:PMRDTM_CFG:MatchQuotesPastEndOfLine=Yes (Data Integration Server)"
	if strings.Join(custom, ",") != want {
		t.Errorf("custom properties = %v", custom)
	}
	types := f.CustomTypes()
	if len(types["Data_Integration_Server"]) != 2 || types["Data_Integration_Server"][0] != (CustomConfigType{"TOMCAT_CFG", "Tomcat"}) {
		t.Errorf("CustomTypes = %v", types)
	}
	// Unchanged forms submit the fields as they are.
	if got := submitted(t, f); len(got) != 2 {
		t.Errorf("unchanged submit = %v", got)
	}
}

func TestApplyCustomProperties(t *testing.T) {
	page := customFormPage(
		customRowHTML("Data_Integration_Server77.0", "TOMCAT_CFG", "maxDTMProcesses", "5", "1170373"),
		customRowHTML("Data_Integration_Server77.0", "PMRDTM_CFG", "MatchQuotesPastEndOfLine", "Yes", "1170385"))
	f, err := parseAgentConfigForm(page)
	if err != nil {
		t.Fatal(err)
	}
	diffs, err := f.Apply([]AgentConfigChange{
		{Property: "Data Integration Server:Tomcat:maxDTMProcesses", Value: "6"}, // display name and type label
		{Property: "Data_Integration_Server:PMRDTM_CFG:MatchQuotesPastEndOfLine", Remove: true},
		{Property: "Data_Integration_Server:DTM:NewOne", Value: "x", Add: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 3 || diffs[0].NewValue != "6" || diffs[1].Change != "remove" || diffs[2].Change != "add" ||
		diffs[2].Property != "Data_Integration_Server:PMRDTM_CFG:NewOne" {
		t.Errorf("diffs = %+v", diffs)
	}
	got := strings.Join(submitted(t, f), ",")
	want := "Data_Integration_Server77.0|TOMCAT_CFG|maxDTMProcesses=6,Data_Integration_Server77.0|PMRDTM_CFG|NewOne=x"
	if got != want {
		t.Errorf("submitted rows = %s\nwant %s", got, want)
	}
	if !strings.Contains(f.encode(), "engineCustomConfigIdInput=1170373") {
		t.Error("existing row lost its ID")
	}
}

func TestApplyCustomTemplateRow(t *testing.T) {
	// No custom properties: the form has an empty template row.
	f, err := parseAgentConfigForm(customFormPage(customRowHTML("-1", "-1", "", "", "-1")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Apply([]AgentConfigChange{{Property: "OpsInsightsDataCollector:OpsInsights:cliTest", Value: "1", Add: true}}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(submitted(t, f), ","); got != "OpsInsightsDataCollector25|OpsInsights|cliTest=1" {
		t.Errorf("template should be replaced, got %s", got)
	}

	// Removing the last custom property submits an empty template row.
	g, err := parseAgentConfigForm(customFormPage(customRowHTML("Data_Integration_Server77.0", "TOMCAT_CFG", "a", "1", "11")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = g.Apply([]AgentConfigChange{{Property: "Data_Integration_Server:TOMCAT_CFG:a", Reset: true}}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(submitted(t, g), ","); got != "-1|-1|=" {
		t.Errorf("expected only an empty template row, got %s", got)
	}
}

func TestApplyCustomErrors(t *testing.T) {
	page := customFormPage(customRowHTML("Data_Integration_Server77.0", "TOMCAT_CFG", "maxDTMProcesses", "5", "1"))
	cases := map[string]AgentConfigChange{
		"exists":         {Property: "Data_Integration_Server:TOMCAT_CFG:maxDTMProcesses", Value: "1", Add: true},
		"standard":       {Property: "Data_Integration_Server:TOMCAT_CFG:maxThreads", Value: "1", Add: true},
		"bad type":       {Property: "Data_Integration_Server:BOGUS:x", Value: "1", Add: true},
		"bad service":    {Property: "process-engine:jvm:x", Value: "1", Add: true},
		"missing":        {Property: "Data_Integration_Server:TOMCAT_CFG:nope", Value: "1"},
		"remove std":     {Property: "Data_Integration_Server:TOMCAT_CFG:maxThreads", Remove: true},
		"multi-line":     {Property: "Data_Integration_Server:TOMCAT_CFG:x", Value: "a\nb", Add: true},
		"long value":     {Property: "Data_Integration_Server:TOMCAT_CFG:x", Value: strings.Repeat("v", 513), Add: true},
		"add and remove": {Property: "Data_Integration_Server:TOMCAT_CFG:x", Add: true, Remove: true},
	}
	for name, ch := range cases {
		f, err := parseAgentConfigForm(page)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Apply([]AgentConfigChange{ch}); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestApplyAddMissing(t *testing.T) {
	f, err := parseAgentConfigForm(customFormPage(customRowHTML("-1", "-1", "", "", "-1")))
	if err != nil {
		t.Fatal(err)
	}
	diffs, err := f.ApplyWith([]AgentConfigChange{
		{Property: "Data_Integration_Server:TOMCAT_CFG:maxThreads", Value: "300"}, // standard: set
		{Property: "Data_Integration_Server:TOMCAT_CFG:maxDTMProcesses", Value: "5"},
	}, ApplyOptions{AddMissing: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 2 || diffs[0].Change != "" || diffs[1].Change != "add" {
		t.Errorf("diffs = %+v", diffs)
	}
}
