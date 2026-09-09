package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jbrazda/iics-cli/internal/client"
	"github.com/jbrazda/iics-cli/internal/config"
	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/spf13/cobra"
)

// installerOSValues are the platform identifiers accepted by the agentInstallerInfo API.
var installerOSValues = []string{"win64", "linux64"}

// resolveInstallerOS validates the --os flag, prompting for it when it is empty
// and stdin is a terminal.
func resolveInstallerOS(osFlag string) (string, error) {
	v := strings.ToLower(strings.TrimSpace(osFlag))
	if v == "" {
		if !config.IsTerminal() {
			return "", fmt.Errorf("--os is required (one of: %s)", strings.Join(installerOSValues, ", "))
		}
		in, err := promptText(fmt.Sprintf("Operating system (%s)", strings.Join(installerOSValues, "/")), "")
		if err != nil {
			return "", err
		}
		v = strings.ToLower(strings.TrimSpace(in))
	}
	for _, allowed := range installerOSValues {
		if v == allowed {
			return v, nil
		}
	}
	return "", fmt.Errorf("invalid --os %q: must be one of: %s", v, strings.Join(installerOSValues, ", "))
}

func installerInfoRows(info *client.AgentInstallerInfo) []output.KVRow {
	return []output.KVRow{
		output.KV("type", info.Type),
		output.KV("downloadUrl", info.DownloadURL),
		output.KV("checksumDownloadUrl", info.ChecksumDownloadURL),
		output.KV("installToken", info.InstallToken),
	}
}

// printInstallerInfo renders installer info: a vertical property/value table for
// the default table format, otherwise the raw struct via --output semantics.
func printInstallerInfo(info *client.AgentInstallerInfo) error {
	f, err := getFormatter()
	if err != nil {
		return err
	}
	if outputFmt == "" || outputFmt == "table" {
		return f.Format(installerInfoRows(info), output.KVCols)
	}
	cols := []output.Column{
		{Header: "type", Field: "@type"},
		{Header: "downloadUrl", Field: "downloadUrl"},
		{Header: "checksumDownloadUrl", Field: "checksumDownloadUrl"},
		{Header: "installToken", Field: "installToken"},
	}
	return f.Format(info, cols)
}

func newAgentInstallerInfoCmd() *cobra.Command {
	var osFlag string
	cmd := &cobra.Command{
		Use:   "installer-info",
		Short: "Get Secure Agent installer download information",
		Example: `  iics agent installer-info --os win64
  iics agent installer-info --os linux64 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			platform, err := resolveInstallerOS(osFlag)
			if err != nil {
				return err
			}
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			info, err := c.GetAgentInstallerInfo(context.Background(), platform)
			if err != nil {
				return err
			}
			return printInstallerInfo(info)
		},
	}
	cmd.Flags().StringVar(&osFlag, "os", "", "operating system: win64 or linux64")
	return cmd
}

// loadInstallerInfoInput returns installer info supplied via --installer-info or
// piped stdin. The second return value is false when no input was supplied.
func loadInstallerInfoInput(installerInfoFile string) (*client.AgentInstallerInfo, bool, error) {
	var data []byte
	switch {
	case installerInfoFile != "":
		b, err := os.ReadFile(installerInfoFile)
		if err != nil {
			return nil, false, fmt.Errorf("reading installer info file: %w", err)
		}
		data = b
	case hasPipedStdin():
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, false, fmt.Errorf("reading stdin: %w", err)
		}
		if len(strings.TrimSpace(string(b))) == 0 {
			return nil, false, nil
		}
		data = b
	default:
		return nil, false, nil
	}
	var info client.AgentInstallerInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, false, fmt.Errorf("parsing installer info JSON: %w", err)
	}
	if info.DownloadURL == "" {
		return nil, false, fmt.Errorf("installer info input has no downloadUrl")
	}
	return &info, true, nil
}

// installerFileName derives the installer file name from the download URL path.
func installerFileName(downloadURL string) string {
	u, err := url.Parse(downloadURL)
	if err != nil || u.Path == "" {
		return "secure-agent-installer"
	}
	name := path.Base(u.Path)
	if name == "." || name == "/" || name == "" {
		return "secure-agent-installer"
	}
	return name
}

// resolveTargetPath resolves the download destination. When target is empty the
// system temp dir is used. When target names an existing directory or ends with a
// path separator, the derived file name is appended. Otherwise target is used as-is.
func resolveTargetPath(target, fileName string) string {
	if target == "" {
		return filepath.Join(os.TempDir(), fileName)
	}
	if fi, err := os.Stat(target); err == nil && fi.IsDir() {
		return filepath.Join(target, fileName)
	}
	if strings.HasSuffix(target, "/") || strings.HasSuffix(target, string(os.PathSeparator)) {
		return filepath.Join(target, fileName)
	}
	return target
}

// parseChecksum extracts the hex digest from the contents of a checksum file,
// which is typically "<hexdigest>  <filename>" or just "<hexdigest>".
func parseChecksum(contents string) string {
	fields := strings.Fields(contents)
	if len(fields) == 0 {
		return ""
	}
	return strings.ToLower(fields[0])
}

// installerDownloadResult is the structured result for --output json/csv/yaml.
type installerDownloadResult struct {
	File                string `json:"file"`
	FileName            string `json:"fileName"`
	Size                int64  `json:"size"`
	DownloadURL         string `json:"downloadUrl"`
	ChecksumDownloadURL string `json:"checksumDownloadUrl,omitempty"`
	ChecksumAlgorithm   string `json:"checksumAlgorithm,omitempty"`
	ExpectedChecksum    string `json:"expectedChecksum,omitempty"`
	ActualChecksum      string `json:"actualChecksum,omitempty"`
	Verified            *bool  `json:"verified,omitempty"`
}

func printInstallerDownloadResult(res *installerDownloadResult) error {
	f, err := getFormatter()
	if err != nil {
		return err
	}
	if outputFmt == "" || outputFmt == "table" {
		rows := []output.KVRow{
			output.KV("file", res.File),
			output.KV("fileName", res.FileName),
			output.KV("size", fmt.Sprintf("%d", res.Size)),
			output.KV("downloadUrl", res.DownloadURL),
		}
		if res.ChecksumDownloadURL != "" {
			rows = append(rows, output.KV("checksumDownloadUrl", res.ChecksumDownloadURL))
		}
		if res.Verified != nil {
			rows = append(rows,
				output.KV("checksumAlgorithm", res.ChecksumAlgorithm),
				output.KV("expectedChecksum", res.ExpectedChecksum),
				output.KV("actualChecksum", res.ActualChecksum),
			)
			verified := "false"
			if *res.Verified {
				verified = "true"
			}
			rows = append(rows, output.KV("verified", verified))
		}
		return f.Format(rows, output.KVCols)
	}
	cols := []output.Column{
		{Header: "file", Field: "file"},
		{Header: "fileName", Field: "fileName"},
		{Header: "size", Field: "size"},
		{Header: "downloadUrl", Field: "downloadUrl"},
		{Header: "checksumDownloadUrl", Field: "checksumDownloadUrl"},
		{Header: "verified", Field: "verified"},
	}
	return f.Format(res, cols)
}

func newAgentInstallerDownloadCmd() *cobra.Command {
	var (
		osFlag            string
		installerInfoFile string
		target            string
		verify            bool
	)
	cmd := &cobra.Command{
		Use:   "installer-download",
		Short: "Download the Secure Agent installer (optionally verifying its checksum)",
		Long: `Download the Secure Agent installer binary.

Installer info may be supplied as JSON on stdin or via --installer-info. When no
installer info is supplied, it is requested for the default or --profile account
and the --os platform (prompted when running interactively).

With --target: a directory (existing or trailing slash) downloads using the file
name from the installer metadata; otherwise --target is treated as the full file
path. Without --target the file is written to the system temp directory.`,
		Example: `  iics agent installer-download --os linux64 --target ./downloads/
  iics agent installer-info --os win64 --output json | iics agent installer-download --verify
  iics agent installer-download --installer-info info.json --target /tmp/agent.exe --verify`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			info, haveInput, err := loadInstallerInfoInput(installerInfoFile)
			if err != nil {
				return err
			}
			if !haveInput {
				platform, perr := resolveInstallerOS(osFlag)
				if perr != nil {
					return perr
				}
				info, err = c.GetAgentInstallerInfo(ctx, platform)
				if err != nil {
					return err
				}
			}

			fileName := installerFileName(info.DownloadURL)
			destPath := resolveTargetPath(target, fileName)
			if dir := filepath.Dir(destPath); dir != "" {
				if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
					return fmt.Errorf("creating target directory: %w", mkErr)
				}
			}

			out := cmd.OutOrStdout()
			if verbose {
				_, _ = fmt.Fprintf(out, "Downloading %s to %s\n", info.DownloadURL, destPath)
			}

			file, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("creating output file: %w", err)
			}
			hasher := sha256.New()
			n, err := c.DownloadFile(ctx, info.DownloadURL, io.MultiWriter(file, hasher))
			if cerr := file.Close(); cerr != nil && err == nil {
				err = cerr
			}
			if err != nil {
				_ = os.Remove(destPath)
				return err
			}

			res := &installerDownloadResult{
				File:                destPath,
				FileName:            fileName,
				Size:                n,
				DownloadURL:         info.DownloadURL,
				ChecksumDownloadURL: info.ChecksumDownloadURL,
			}

			if verify {
				if info.ChecksumDownloadURL == "" {
					_ = os.Remove(destPath)
					return fmt.Errorf("--verify requested but installer info has no checksumDownloadUrl")
				}
				checksumText, cerr := c.FetchText(ctx, info.ChecksumDownloadURL)
				if cerr != nil {
					_ = os.Remove(destPath)
					return cerr
				}
				expected := parseChecksum(checksumText)
				actual := hex.EncodeToString(hasher.Sum(nil))
				match := expected != "" && expected == actual
				res.ChecksumAlgorithm = "sha256"
				res.ExpectedChecksum = expected
				res.ActualChecksum = actual
				res.Verified = &match
				if !match {
					_ = os.Remove(destPath)
					return fmt.Errorf("checksum verification failed: expected %s, got %s", expected, actual)
				}
				if verbose {
					_, _ = fmt.Fprintf(out, "Checksum verified (sha256): %s\n", actual)
				}
			}

			return printInstallerDownloadResult(res)
		},
	}
	cmd.Flags().StringVar(&osFlag, "os", "", "operating system: win64 or linux64 (used when no installer info is supplied)")
	cmd.Flags().StringVar(&installerInfoFile, "installer-info", "", "path to installer info JSON (omit to read from stdin when piped)")
	cmd.Flags().StringVar(&target, "target", "", "destination file or directory (default: system temp directory)")
	cmd.Flags().BoolVar(&verify, "verify", false, "download the checksum and verify the installer")
	return cmd
}
