package client

import "testing"

func TestDetectArtifactsFormat(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: "txt"},
		{name: "json object", in: `{"objects":[{"id":"x"}]}`, want: "json"},
		{name: "json array", in: `[{"path":"A/B","type":"MTT"}]`, want: "json"},
		{name: "csv header", in: "ID,PATH,TYPE\n1,A/B,MTT\n", want: "csv"},
		{name: "csv single location header", in: "LOCATION\nExplore/Proj/Task.MTT\n", want: "csv"},
		{name: "yaml list", in: "- id: abc123\n- path: A/B\n", want: "yaml"},
		{name: "yaml doc", in: "---\nobjects:\n  - id: x\n", want: "yaml"},
		{name: "txt", in: "Explore/Proj/Task.MTT\n", want: "txt"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetectArtifactsFormat([]byte(tc.in)); got != tc.want {
				t.Fatalf("DetectArtifactsFormat(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
