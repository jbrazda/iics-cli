package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jbrazda/iics-cli/internal/release"
)

func bindManifestLogFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVar(target, "log-file", "", "append Markdown release log to this file; use --log-file without a value for "+release.DefaultManifestLogPath)
	if flag := cmd.Flags().Lookup("log-file"); flag != nil {
		flag.NoOptDefVal = release.DefaultManifestLogPath
	}
}

func manifestLogPath(cmd *cobra.Command, value string) (bool, string) {
	return release.ResolveManifestLogPath(cmd.Flags().Changed("log-file"), value)
}

func appendManifestLogWarning(cmd *cobra.Command, path, markdown string) {
	if err := release.AppendManifestLog(path, markdown); err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not append release log %s: %v\n", path, err)
	}
}

func releaseAssetsToManifestLog(assets []release.Asset) []release.ManifestLogAsset {
	rows := make([]release.ManifestLogAsset, 0, len(assets))
	for _, asset := range assets {
		rows = append(rows, release.ManifestLogAsset{
			ID:         asset.ID,
			Location:   asset.Location,
			Type:       asset.Type,
			Path:       asset.Path,
			Dependency: asset.Dependency,
			Status:     asset.Status,
		})
	}
	return rows
}
