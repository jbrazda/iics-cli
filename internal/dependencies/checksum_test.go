package dependencies

import "testing"

func TestParseChecksumEntries(t *testing.T) {
	data := `#
#Tue May 05 18:48:04 UTC 2026
Explore/ClaimCenter_GW.Project.json=ABC123
Explore/ClaimCenter_GW/SmartComm_v1/SmartComm\ File\ Transfers.GUIDE.xml=DEF456
exportMetadata.v2.json=ZZZ
`
	got := ParseChecksumEntries(data)
	if !got["Explore/ClaimCenter_GW.Project.json"] {
		t.Fatalf("missing project checksum entry")
	}
	if !got["Explore/ClaimCenter_GW/SmartComm_v1/SmartComm File Transfers.GUIDE.xml"] {
		t.Fatalf("missing guide checksum entry with unescaped spaces")
	}
	if !got["exportMetadata.v2.json"] {
		t.Fatalf("missing metadata checksum entry")
	}
}

func TestIsObjectChecksumBacked(t *testing.T) {
	entries := map[string]bool{
		"Explore/ClaimCenter_GW.Project.json":                                    true,
		"Explore/ClaimCenter_GW/SmartComm_v1.Folder.json":                        true,
		"Explore/ClaimCenter_GW/SmartComm_v1/SmartComm File Transfers.GUIDE.xml": true,
	}

	if !IsObjectChecksumBacked("/Explore", "ClaimCenter_GW", "Project", entries) {
		t.Fatalf("expected project to be checksum-backed")
	}
	if !IsObjectChecksumBacked("/Explore/ClaimCenter_GW", "SmartComm_v1", "Folder", entries) {
		t.Fatalf("expected folder to be checksum-backed")
	}
	if !IsObjectChecksumBacked("/Explore/ClaimCenter_GW/SmartComm_v1", "SmartComm File Transfers", "GUIDE", entries) {
		t.Fatalf("expected guide to be checksum-backed")
	}
	if IsObjectChecksumBacked("/Explore/Connections/Informatica", "IICS-TaskFlowService", "AI_CONNECTION", entries) {
		t.Fatalf("did not expect AI_CONNECTION to be checksum-backed")
	}
}

func TestObjectChecksumCandidates(t *testing.T) {
	got := ObjectChecksumCandidates("/Explore/ClaimCenter_GW/SmartComm_v1", "SmartComm File Transfers", "GUIDE")
	if len(got) != 3 {
		t.Fatalf("expected 3 candidates, got %d", len(got))
	}
	want := map[string]bool{
		"Explore/ClaimCenter_GW/SmartComm_v1/SmartComm File Transfers.GUIDE.xml":  true,
		"Explore/ClaimCenter_GW/SmartComm_v1/SmartComm File Transfers.GUIDE.zip":  true,
		"Explore/ClaimCenter_GW/SmartComm_v1/SmartComm File Transfers.GUIDE.json": true,
	}
	for _, c := range got {
		if !want[c] {
			t.Fatalf("unexpected candidate %q", c)
		}
	}
}
