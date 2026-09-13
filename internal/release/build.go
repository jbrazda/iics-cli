package release

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// AssetWriter persists assets to disk in a caller-selected output format
// (csv/json/yaml); see WriteAssetsCSV/WriteAssetsJSON/WriteAssetsYAML.
type AssetWriter func(path string, assets []Asset, fields []string) error

// TypeCountRenderer renders a per-type asset count table (typically to
// stderr) with the given title. Implementations may depend on presentation
// concerns (theming, output format) that live outside this package.
type TypeCountRenderer func(title string, counts []AssetTypeCount, total int) error

// PlanOptions configures BuildPlan's shared per-target pipeline: resolve ->
// filter missing-transitive -> validate -> write package -> compute publish
// assets -> write publish file, run once per target, followed by an optional
// connectors package file. Full-deployment mode and selective/tag mode differ
// only in how the initial asset set (Assets/ConnectorSourceAssets) is
// resolved and in a handful of mode-specific log messages, captured here via
// the OnTargetProcessing/OnTargetFilesGenerated hooks and CompletedLabel.
type PlanOptions struct {
	// Assets is the resolved asset set used as the per-target filter/validate
	// source (full mode: seed-resolved assets; selective mode: policy-filtered
	// tag assets).
	Assets []Asset
	// ConnectorSourceAssets is the asset set the optional connectors package
	// file is built from.
	ConnectorSourceAssets []Asset

	Targets                 []string
	FilterMissingTransitive bool
	TargetResolutionOptions TargetResolutionOptions

	OutputRoot string
	// PackageFileBaseName is the per-target package file name without its
	// extension, e.g. "full_build.package" or "tag_build.package".
	PackageFileBaseName string
	PlanExt             string
	WriteAssets         AssetWriter
	PackageFields       []string
	PublishFields       []string

	IncludeConnectors  bool
	IncludeConnections bool

	InfoEnabled            bool
	RenderPackageTotals    TypeCountRenderer
	RenderPublishTotals    TypeCountRenderer
	RenderConnectorTotals  TypeCountRenderer
	OnTargetProcessing     func(env string)
	OnTargetFilesGenerated func(env, packageFile, publishFile string, publishAssetCount int)

	// CompletedLabel distinguishes the final "completed" log line, e.g.
	// "full mode" or "selective mode". Empty skips that log line.
	CompletedLabel string
}

// PlanResult carries the per-target outputs needed for the release plan
// manifest log.
type PlanResult struct {
	AssetsByTarget  map[string][]ManifestLogAsset
	PublishByTarget map[string][]ManifestLogAsset
	FilesWritten    int
	ConnectorsFile  string
}

// BuildPlan runs the shared per-target pipeline (resolve -> filter
// missing-transitive -> validate -> write package -> compute publish assets
// -> write publish file) once per target, for either full-deployment or
// selective/tag mode, differing only in how the initial asset set is
// resolved (opts.Assets/opts.ConnectorSourceAssets) and in the mode-specific
// hooks/labels on PlanOptions.
func BuildPlan(ctx context.Context, opts PlanOptions) (PlanResult, error) {
	result := PlanResult{
		AssetsByTarget:  make(map[string][]ManifestLogAsset, len(opts.Targets)),
		PublishByTarget: make(map[string][]ManifestLogAsset, len(opts.Targets)),
	}

	for _, env := range opts.Targets {
		if opts.OnTargetProcessing != nil {
			opts.OnTargetProcessing(env)
		}

		envDir := filepath.Join(opts.OutputRoot, strings.ToLower(env))
		if err := os.MkdirAll(envDir, 0o755); err != nil {
			return PlanResult{}, fmt.Errorf("creating env directory: %w", err)
		}

		envAssets := opts.Assets
		if opts.FilterMissingTransitive {
			filtered, err := FilterMissingTransitiveForTarget(ctx, env, opts.Assets, opts.TargetResolutionOptions)
			if err != nil {
				return PlanResult{}, err
			}
			envAssets = filtered
			slog.Info("release plan: missing-transitive filter applied",
				"environment", env,
				"before", len(opts.Assets),
				"after", len(envAssets),
			)
		}

		envPackageAssets, err := AnnotateAssetsWithTargetValidation(ctx, env, envAssets, opts.TargetResolutionOptions)
		if err != nil {
			return PlanResult{}, err
		}
		envPackageFields := EnsureCurrentTargetStatusField(opts.PackageFields, env)
		publishAssets := PublishAssets(envAssets)

		if opts.InfoEnabled {
			if opts.RenderPackageTotals != nil {
				if renderErr := opts.RenderPackageTotals(
					fmt.Sprintf("release plan: package totals by type for %s", env),
					AssetCountsByType(envAssets),
					len(envAssets),
				); renderErr != nil {
					return PlanResult{}, renderErr
				}
			}
			if opts.RenderPublishTotals != nil {
				if renderErr := opts.RenderPublishTotals(
					fmt.Sprintf("release plan: publish totals by type for %s", env),
					AssetCountsByType(publishAssets),
					len(publishAssets),
				); renderErr != nil {
					return PlanResult{}, renderErr
				}
			}
		}

		packageFile := filepath.Join(envDir, opts.PackageFileBaseName+"."+opts.PlanExt)
		if err := opts.WriteAssets(packageFile, envPackageAssets, envPackageFields); err != nil {
			return PlanResult{}, err
		}
		result.FilesWritten++

		publishFile := filepath.Join(envDir, "publish_assets."+opts.PlanExt)
		if err := opts.WriteAssets(publishFile, publishAssets, opts.PublishFields); err != nil {
			return PlanResult{}, err
		}
		result.FilesWritten++

		result.AssetsByTarget[env] = manifestLogAssets(envPackageAssets)
		result.PublishByTarget[env] = manifestLogAssets(publishAssets)

		if opts.OnTargetFilesGenerated != nil {
			opts.OnTargetFilesGenerated(env, packageFile, publishFile, len(publishAssets))
		}
	}

	if ShouldWriteConnectorPackage(opts.IncludeConnectors, opts.IncludeConnections) {
		connectorAssets := ConnectorPackageAssets(opts.ConnectorSourceAssets)
		connectorsFile := filepath.Join(opts.OutputRoot, "connectors.package."+opts.PlanExt)
		if err := opts.WriteAssets(connectorsFile, connectorAssets, opts.PackageFields); err != nil {
			return PlanResult{}, err
		}
		result.FilesWritten++
		result.ConnectorsFile = connectorsFile

		if opts.InfoEnabled && opts.RenderConnectorTotals != nil {
			if renderErr := opts.RenderConnectorTotals(
				"release plan: connector totals by type",
				AssetCountsByType(connectorAssets),
				len(connectorAssets),
			); renderErr != nil {
				return PlanResult{}, renderErr
			}
		}
		slog.Info("release plan: connectors file generated", "connectorsFile", connectorsFile)
	}

	if opts.CompletedLabel != "" {
		slog.Info("release plan: completed "+opts.CompletedLabel,
			"targets", strings.Join(opts.Targets, ","),
			"outputRoot", opts.OutputRoot,
			"filesWritten", result.FilesWritten,
		)
	}

	return result, nil
}

func manifestLogAssets(assets []Asset) []ManifestLogAsset {
	rows := make([]ManifestLogAsset, 0, len(assets))
	for _, a := range assets {
		rows = append(rows, ManifestLogAsset{
			ID:         a.ID,
			Location:   a.Location,
			Type:       a.Type,
			Path:       a.Path,
			Dependency: a.Dependency,
			Status:     a.Status,
		})
	}
	return rows
}
