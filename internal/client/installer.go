package client

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// SHA256File returns the lowercase hex SHA-256 digest and size of a file.
func SHA256File(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, fmt.Errorf("reading %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// ChecksumState describes how an existing file compares to the expected
// installer checksum.
type ChecksumState int

const (
	// ChecksumUnknown means the expected checksum could not be determined.
	ChecksumUnknown ChecksumState = iota
	// ChecksumMatch means the file matches the expected checksum.
	ChecksumMatch
	// ChecksumMismatch means the file differs from the expected checksum.
	ChecksumMismatch
)

// CompareChecksum compares an actual and an expected hex digest.
func CompareChecksum(actual, expected string) ChecksumState {
	switch {
	case expected == "" || actual == "":
		return ChecksumUnknown
	case actual == expected:
		return ChecksumMatch
	default:
		return ChecksumMismatch
	}
}

// DownloadOverExisting decides whether to download over an existing file.
// With force it always downloads. Interactively it asks confirm, defaulting
// to yes unless the file already matches. Otherwise it skips only a file
// that matches the expected checksum.
func DownloadOverExisting(state ChecksumState, force, interactive bool, confirm func(defaultYes bool) (bool, error)) (bool, error) {
	if force {
		return true, nil
	}
	if interactive && confirm != nil {
		return confirm(state != ChecksumMatch)
	}
	return state != ChecksumMatch, nil
}
