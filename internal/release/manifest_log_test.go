package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveManifestLogPath(t *testing.T) {
	tests := []struct {
		name        string
		enabled     bool
		value       string
		wantEnabled bool
		wantPath    string
	}{
		{name: "disabled", enabled: false, value: "", wantEnabled: false, wantPath: ""},
		{name: "default", enabled: true, value: "", wantEnabled: true, wantPath: DefaultManifestLogPath},
		{name: "explicit", enabled: true, value: "out/release.md", wantEnabled: true, wantPath: "out/release.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEnabled, gotPath := ResolveManifestLogPath(tt.enabled, tt.value)
			if gotEnabled != tt.wantEnabled || gotPath != tt.wantPath {
				t.Fatalf("ResolveManifestLogPath() = (%t, %q), want (%t, %q)", gotEnabled, gotPath, tt.wantEnabled, tt.wantPath)
			}
		})
	}
}

func TestAppendManifestLogCreatesParentsAndAppends(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "release_manifest.md")
	if err := AppendManifestLog(path, "## First"); err != nil {
		t.Fatalf("AppendManifestLog() first error = %v", err)
	}
	if err := AppendManifestLog(path, "## Second"); err != nil {
		t.Fatalf("AppendManifestLog() second error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "## First") || !strings.Contains(got, "## Second") {
		t.Fatalf("append content missing:\n%s", got)
	}
	if strings.Index(got, "## First") > strings.Index(got, "## Second") {
		t.Fatalf("sections not appended in order:\n%s", got)
	}
}

func TestEscapeMarkdownCell(t *testing.T) {
	got := EscapeMarkdownCell(" value|with\r\nnewline ")
	want := `value\|with<br>newline`
	if got != want {
		t.Fatalf("EscapeMarkdownCell() = %q, want %q", got, want)
	}
	if got := EscapeMarkdownCell(""); got != " " {
		t.Fatalf("empty cell = %q, want single space", got)
	}
}

func TestFencedBlockExpandsFenceForBackticks(t *testing.T) {
	body := "line\n```inside\n"
	got := FencedBlock("txt", body)
	if !strings.HasPrefix(got, "````txt\n") || !strings.HasSuffix(got, "\n````\n") {
		t.Fatalf("unexpected fenced block:\n%s", got)
	}
}

func TestMarkdownTableAlignsColumnPipes(t *testing.T) {
	got := MarkdownTable([]string{"TYPE", "COUNT (QA)"}, [][]string{
		{"PROCESS", "12"},
		{"GUIDE", "1"},
	}, map[int]bool{1: true})

	lines := nonEmptyLines(got)
	if len(lines) != 4 {
		t.Fatalf("line count = %d, want 4:\n%s", len(lines), got)
	}
	wantPipes := pipePositions(lines[0])
	for _, line := range lines[1:] {
		if gotPipes := pipePositions(line); !equalInts(gotPipes, wantPipes) {
			t.Fatalf("pipe positions for %q = %v, want %v\nfull table:\n%s", line, gotPipes, wantPipes, got)
		}
	}
	if !strings.Contains(got, "| PROCESS |         12 |") {
		t.Fatalf("right-aligned numeric cell not padded as expected:\n%s", got)
	}
	if !strings.Contains(got, "| ------- | ---------: |") {
		t.Fatalf("right-aligned separator not padded as expected:\n%s", got)
	}
}

func TestMarkdownTableAlignmentUsesEscapedCellWidths(t *testing.T) {
	got := MarkdownTable([]string{"PATH", "VALUE"}, [][]string{
		{"folder|asset", "x"},
		{"short", "value"},
	}, nil)

	lines := nonEmptyLines(got)
	wantPipes := pipePositions(lines[0])
	for _, line := range lines[1:] {
		if gotPipes := pipePositions(line); !equalInts(gotPipes, wantPipes) {
			t.Fatalf("pipe positions for %q = %v, want %v\nfull table:\n%s", line, gotPipes, wantPipes, got)
		}
	}
	if !strings.Contains(got, `folder\|asset`) {
		t.Fatalf("escaped pipe missing:\n%s", got)
	}
}

func TestRenderReleasePlanLogDynamicTargets(t *testing.T) {
	got := RenderReleasePlanLog(ReleasePlanLog{
		Source:  "release_manifest.properties",
		Mode:    ModeFullDeployment,
		Targets: []string{"qa", "tst"},
		AssetsByTarget: map[string][]ManifestLogAsset{
			"qa":  {{Type: "PROCESS"}, {Type: "GUIDE"}},
			"tst": {{Type: "PROCESS"}},
		},
		PublishByTarget: map[string][]ManifestLogAsset{
			"qa":  {{Type: "PROCESS"}},
			"tst": {{Type: "PROCESS"}, {Type: "TASKFLOW"}},
		},
	})
	for _, want := range []string{"# Release Manifest", "COUNT (QA)", "COUNT (TST)", "PROCESS", "TASKFLOW"} {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered log missing %q:\n%s", want, got)
		}
	}
}

func nonEmptyLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func pipePositions(s string) []int {
	var positions []int
	escaped := false
	for i, r := range s {
		if r == '\\' && !escaped {
			escaped = true
			continue
		}
		if r == '|' && !escaped {
			positions = append(positions, i)
		}
		escaped = false
	}
	return positions
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
