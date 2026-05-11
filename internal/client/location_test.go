package client

import "testing"

func TestNormalizeLocationPath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "Project/Asset", want: "Project/Asset"},
		{name: "leading slash", in: "/Project/Asset", want: "Project/Asset"},
		{name: "explore prefix", in: "Explore/Project/Asset", want: "Project/Asset"},
		{name: "sys prefix", in: "SYS/Project/Asset", want: "Project/Asset"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeLocationPath(tc.in); got != tc.want {
				t.Fatalf("NormalizeLocationPath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestBuildLocation(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		typ      string
		expected string
	}{
		{name: "default explore", path: "Default/MyTask", typ: "MTT", expected: "Explore/Default/MyTask.MTT"},
		{name: "connection sys", path: "SYS/Connections/MyConn", typ: "Connection", expected: "SYS/Connections/MyConn.Connection"},
		{name: "agentgroup sys", path: "/Explore/Agents/MyGroup", typ: "AgentGroup", expected: "SYS/Agents/MyGroup.AgentGroup"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := BuildLocation(tc.path, tc.typ); got != tc.expected {
				t.Fatalf("BuildLocation(%q, %q) = %q, want %q", tc.path, tc.typ, got, tc.expected)
			}
		})
	}
}
