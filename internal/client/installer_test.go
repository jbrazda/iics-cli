package client

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSHA256File(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.bin")
	if err := os.WriteFile(p, []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	sum, n, err := SHA256File(p)
	if err != nil {
		t.Fatal(err)
	}
	if sum != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" || n != 3 {
		t.Errorf("got %s, %d", sum, n)
	}
	if _, _, err := SHA256File(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("expected error for a missing file")
	}
}

func TestCompareChecksum(t *testing.T) {
	if CompareChecksum("aa", "aa") != ChecksumMatch || CompareChecksum("aa", "bb") != ChecksumMismatch ||
		CompareChecksum("aa", "") != ChecksumUnknown {
		t.Error("unexpected CompareChecksum result")
	}
}

func TestDownloadOverExisting(t *testing.T) {
	var gotDefault *bool
	confirm := func(answer bool) func(bool) (bool, error) {
		return func(def bool) (bool, error) { gotDefault = &def; return answer, nil }
	}
	tests := []struct {
		name        string
		state       ChecksumState
		force       bool
		interactive bool
		answer      bool
		want        bool
		wantDefault *bool
	}{
		{"force", ChecksumMatch, true, true, false, true, nil},
		{"script match skips", ChecksumMatch, false, false, false, false, nil},
		{"script mismatch downloads", ChecksumMismatch, false, false, false, true, nil},
		{"script unknown downloads", ChecksumUnknown, false, false, false, true, nil},
		{"interactive match defaults no", ChecksumMatch, false, true, false, false, ptr(false)},
		{"interactive mismatch defaults yes", ChecksumMismatch, false, true, true, true, ptr(true)},
	}
	for _, tt := range tests {
		gotDefault = nil
		got, err := DownloadOverExisting(tt.state, tt.force, tt.interactive, confirm(tt.answer))
		if err != nil || got != tt.want {
			t.Errorf("%s: got %v, %v; want %v", tt.name, got, err, tt.want)
		}
		if (tt.wantDefault == nil) != (gotDefault == nil) || (tt.wantDefault != nil && *tt.wantDefault != *gotDefault) {
			t.Errorf("%s: confirm default = %v, want %v", tt.name, gotDefault, tt.wantDefault)
		}
	}
}

func ptr(b bool) *bool { return &b }
