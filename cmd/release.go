package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jbrazda/iics-cli/internal/output"
	"github.com/jbrazda/iics-cli/internal/release"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Generate CI release manifests and deployment plan files",
		Long:  "Commands for CI pipeline integration: emit release manifests, validate schema, and generate package/publish plan files.",
	}
	cmd.AddCommand(newReleaseManifestCmd())
	cmd.AddCommand(newReleaseValidateCmd())
	cmd.AddCommand(newReleasePlanCmd())
	return cmd
}

func newReleaseManifestCmd() *cobra.Command {
	var (
		fromFile          string
		outputRoot        string
		mode              string
		tag               string
		targets           []string
		validTargets      string
		includeConnectors bool
		connectorsOnly    bool
		excludeFile       string
		source            string
	)
	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Generate release_manifest.yaml and release_manifest.md",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := release.DefaultOptions()
			if fromFile != "" {
				data, err := os.ReadFile(fromFile)
				if err != nil {
					return fmt.Errorf("reading --from-file: %w", err)
				}
				parsed, err := release.ParseDeploymentOptionsMarkdown(string(data))
				if err != nil {
					return err
				}
				opts = parsed
				if source == "" {
					source = fromFile
				}
			}
			if cmd.Flags().Changed("mode") {
				opts.Mode = release.DeployMode(mode)
			}
			if cmd.Flags().Changed("tag") {
				opts.Tag = tag
			}
			if cmd.Flags().Changed("targets") {
				opts.Targets = targets
			}
			if cmd.Flags().Changed("include-connectors") {
				opts.IncludeConnectors = includeConnectors
			}
			if cmd.Flags().Changed("connectors-only") {
				opts.ConnectorsOnly = connectorsOnly
			}
			if cmd.Flags().Changed("exclude-file") {
				opts.ExcludeFile = excludeFile
			}

			policy, err := release.ResolveTargetPolicy(validTargets)
			if err != nil {
				return err
			}
			m, err := release.NewManifestWithPolicy(opts, source, policy)
			if err != nil {
				return err
			}
			yamlPath, mdPath, err := release.WriteManifestFiles(outputRoot, m)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", yamlPath)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", mdPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&fromFile, "from-file", "", "path to PR description markdown file with Deployment Options section")
	cmd.Flags().StringVar(&outputRoot, "output-root", "target/iics/import", "output root directory for generated files")
	cmd.Flags().StringVar(&mode, "mode", "", "deploy mode override: full or tag-based")
	cmd.Flags().StringVar(&tag, "tag", "", "tag value for tag-based mode")
	cmd.Flags().StringSliceVar(&targets, "targets", nil, "target environments (comma-separated): tst,qa,stg,prod")
	cmd.Flags().StringVar(&validTargets, "valid-targets", "", "comma-separated allowlist of valid targets (overrides IICS_VALID_DEPLOY_TARGETS)")
	cmd.Flags().BoolVar(&includeConnectors, "include-connectors", false, "include connector assets in generated package/publish files")
	cmd.Flags().BoolVar(&connectorsOnly, "connectors-only", false, "generate connector-only package/publish files")
	cmd.Flags().StringVar(&excludeFile, "exclude-file", "", "path to regex exclude policy file")
	cmd.Flags().StringVar(&source, "source", "", "optional source identifier written to manifest")
	return cmd
}

func newReleaseValidateCmd() *cobra.Command {
	var (
		manifestPath string
		validTargets string
	)
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a release manifest schema and options",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				return fmt.Errorf("reading manifest: %w", err)
			}
			var m release.Manifest
			if unmarshalErr := yaml.Unmarshal(data, &m); unmarshalErr != nil {
				return fmt.Errorf("parsing manifest yaml: %w", unmarshalErr)
			}
			policy, err := release.ResolveTargetPolicy(validTargets)
			if err != nil {
				return err
			}
			if err := release.ValidateManifestWithPolicy(&m, policy); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Manifest is valid: %s\n", manifestPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&manifestPath, "manifest", "target/iics/import/conf/release_manifest.yaml", "path to release manifest yaml")
	cmd.Flags().StringVar(&validTargets, "valid-targets", "", "comma-separated allowlist of valid targets (overrides IICS_VALID_DEPLOY_TARGETS)")
	return cmd
}

func newReleasePlanCmd() *cobra.Command {
	var (
		manifestPath     string
		outputRoot       string
		fullPackageCfg   string
		validTargets     string
		targetProfileMap string
		addMissingTrans  bool
		packageFieldsRaw string
		publishFieldsRaw string
	)
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Generate per-environment package and publish CSV files",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("release plan: starting",
				"manifest", manifestPath,
				"outputRoot", outputRoot,
				"fullPackageConfig", fullPackageCfg,
				"validTargets", strings.TrimSpace(validTargets),
				"targetProfileMap", strings.TrimSpace(targetProfileMap),
				"addMissingTransitiveDeps", addMissingTrans,
			)

			data, err := os.ReadFile(manifestPath)
			if err != nil {
				return fmt.Errorf("reading manifest: %w", err)
			}
			policy, err := release.ResolveTargetPolicy(validTargets)
			if err != nil {
				return err
			}
			manifest, err := release.ParseManifestYAMLWithPolicy(data, policy)
			if err != nil {
				return err
			}

			opts := manifest.Options
			slog.Info("release plan: manifest loaded",
				"mode", string(opts.Mode),
				"tag", opts.Tag,
				"targets", strings.Join(opts.Targets, ","),
				"includeConnectors", opts.IncludeConnectors,
				"connectorsOnly", opts.ConnectorsOnly,
				"excludeFile", opts.ExcludeFile,
			)
			excludes, err := release.LoadExcludePatterns(opts.ExcludeFile)
			if err != nil {
				return err
			}
			slog.Info("release plan: exclude policy loaded", "patterns", len(excludes))

			packageFields := splitCSVFields(packageFieldsRaw, []string{"location", "dependency", "type", "path"})
			publishFields := splitCSVFields(publishFieldsRaw, []string{"location", "dependency"})
			slog.Info("release plan: output fields resolved",
				"packageFields", strings.Join(packageFields, ","),
				"publishFields", strings.Join(publishFields, ","),
			)
			infoEnabled := slog.Default().Enabled(context.Background(), slog.LevelInfo)
			logWriter := cmd.ErrOrStderr()

			if opts.Mode == release.ModeFullDeployment {
				for _, env := range opts.Targets {
					envDir := filepath.Join(outputRoot, strings.ToLower(env))
					if mkErr := os.MkdirAll(envDir, 0o755); mkErr != nil {
						return fmt.Errorf("creating env directory: %w", mkErr)
					}
					targetCfg := filepath.Join(envDir, "all_exclude_connections.package.csv")
					content, readErr := os.ReadFile(fullPackageCfg)
					if readErr != nil {
						return fmt.Errorf("reading full package config %s: %w", fullPackageCfg, readErr)
					}
					if writeErr := os.WriteFile(targetCfg, content, 0o644); writeErr != nil {
						return fmt.Errorf("writing %s: %w", targetCfg, writeErr)
					}
					if writeErr := release.WriteAssetsCSV(filepath.Join(envDir, "publish_assets.csv"), nil, publishFields); writeErr != nil {
						return writeErr
					}
					slog.Info("release plan: full mode files generated",
						"environment", env,
						"packageFile", targetCfg,
						"publishFile", filepath.Join(envDir, "publish_assets.csv"),
					)
				}
				slog.Info("release plan: completed full mode",
					"targets", strings.Join(opts.Targets, ","),
					"outputRoot", outputRoot,
				)
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Generated full-deployment plan files.")
				return nil
			}

			c, err := getClient(cmd)
			if err != nil {
				return err
			}
			assets, err := release.ResolveTagAssets(context.Background(), c, opts.Tag)
			if err != nil {
				return err
			}
			slog.Info("release plan: dependencies resolved", "count", len(assets))
			if infoEnabled {
				if err := renderTypeCountTable(logWriter, "release plan: dependency totals by type", release.AssetCountsByType(assets), len(assets)); err != nil {
					return err
				}
			}

			allFiltered := release.ApplyPolicies(assets, opts.IncludeConnectors, opts.ConnectorsOnly, excludes)
			if infoEnabled {
				slog.Info("release plan: dependency status table")
				if err := renderDependencyStatusTable(
					context.Background(),
					logWriter,
					allFiltered,
					opts.Targets,
					release.TargetResolutionOptions{TargetProfileMap: targetProfileMap},
				); err != nil {
					return err
				}
				if err := renderTypeCountTable(logWriter, "release plan: policy-filtered totals by type", release.AssetCountsByType(allFiltered), len(allFiltered)); err != nil {
					return err
				}
			}
			connectorUnion := make(map[string]release.Asset)
			filesWritten := 0

			for _, env := range opts.Targets {
				slog.Info("release plan: processing target", "environment", env)
				envAssets := allFiltered
				if addMissingTrans {
					envFiltered, filterErr := release.FilterMissingTransitiveForTarget(
						context.Background(),
						env,
						allFiltered,
						release.TargetResolutionOptions{TargetProfileMap: targetProfileMap},
					)
					if filterErr != nil {
						return filterErr
					}
					envAssets = envFiltered
					slog.Info("release plan: missing-transitive filter applied",
						"environment", env,
						"before", len(allFiltered),
						"after", len(envAssets),
					)
				}
				publishAssets := release.PublishAssets(envAssets)
				connectorAssets := release.ConnectorAssets(envAssets)
				if infoEnabled {
					if err := renderTypeCountTable(
						logWriter,
						fmt.Sprintf("release plan: package totals by type for %s", env),
						release.AssetCountsByType(envAssets),
						len(envAssets),
					); err != nil {
						return err
					}
					if err := renderTypeCountTable(
						logWriter,
						fmt.Sprintf("release plan: publish totals by type for %s", env),
						release.AssetCountsByType(publishAssets),
						len(publishAssets),
					); err != nil {
						return err
					}
				}
				for _, ca := range connectorAssets {
					connectorUnion[ca.Location] = ca
				}

				envDir := filepath.Join(outputRoot, strings.ToLower(env))
				if err := os.MkdirAll(envDir, 0o755); err != nil {
					return fmt.Errorf("creating env directory: %w", err)
				}
				if err := release.WriteAssetsCSV(filepath.Join(envDir, "tag_build.package.csv"), envAssets, packageFields); err != nil {
					return err
				}
				filesWritten++
				if err := release.WriteAssetsCSV(filepath.Join(envDir, "publish_assets.csv"), publishAssets, publishFields); err != nil {
					return err
				}
				filesWritten++
				slog.Info("release plan: target files generated",
					"environment", env,
					"packageFile", filepath.Join(envDir, "tag_build.package.csv"),
					"publishFile", filepath.Join(envDir, "publish_assets.csv"),
				)
			}
			if opts.IncludeConnectors {
				connectorAssets := make([]release.Asset, 0, len(connectorUnion))
				for _, asset := range connectorUnion {
					connectorAssets = append(connectorAssets, asset)
				}
				sort.Slice(connectorAssets, func(i, j int) bool {
					if connectorAssets[i].Type != connectorAssets[j].Type {
						return connectorAssets[i].Type < connectorAssets[j].Type
					}
					return connectorAssets[i].Path < connectorAssets[j].Path
				})
				if err := release.WriteAssetsCSV(filepath.Join(outputRoot, "connectors.package.csv"), connectorAssets, packageFields); err != nil {
					return err
				}
				filesWritten++
				if infoEnabled {
					if err := renderTypeCountTable(
						logWriter,
						"release plan: connector totals by type",
						release.AssetCountsByType(connectorAssets),
						len(connectorAssets),
					); err != nil {
						return err
					}
				}
				slog.Info("release plan: connectors file generated", "connectorsFile", filepath.Join(outputRoot, "connectors.package.csv"))
			}
			slog.Info("release plan: completed selective mode",
				"targets", strings.Join(opts.Targets, ","),
				"outputRoot", outputRoot,
				"filesWritten", filesWritten,
			)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Generated selective plan files for targets: %s\n", strings.Join(opts.Targets, ","))
			return nil
		},
	}
	cmd.Flags().StringVar(&manifestPath, "manifest", "target/iics/import/conf/release_manifest.yaml", "path to release manifest yaml")
	cmd.Flags().StringVar(&outputRoot, "output-root", "target/iics/import", "output root directory for generated files")
	cmd.Flags().StringVar(&fullPackageCfg, "full-package-config", "./conf/all_exclude_connections.package.csv", "full-deployment package config file to copy per environment")
	cmd.Flags().StringVar(&validTargets, "valid-targets", "", "comma-separated allowlist of valid targets (overrides IICS_VALID_DEPLOY_TARGETS)")
	cmd.Flags().StringVar(&targetProfileMap, "target-profile-map", "", "comma-separated target to profile map (TARGET=profile), overrides IICS_TARGET_PROFILE_MAP")
	cmd.Flags().BoolVar(&addMissingTrans, "add-missing-transitive-deps", false, "include transitive dependencies only when missing in each target environment (explicit assets are always included)")
	cmd.Flags().StringVar(&packageFieldsRaw, "package-fields", "location,dependency,type,path", "fields for generated package csv files")
	cmd.Flags().StringVar(&publishFieldsRaw, "publish-fields", "location,dependency", "fields for generated publish csv files")
	return cmd
}

func splitCSVFields(raw string, fallback []string) []string {
	parts := strings.Split(raw, ",")
	fields := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			fields = append(fields, p)
		}
	}
	if len(fields) == 0 {
		return fallback
	}
	return fields
}

func renderDependencyStatusTable(
	ctx context.Context,
	w io.Writer,
	assets []release.Asset,
	targets []string,
	opts release.TargetResolutionOptions,
) error {
	tableRows := make([]map[string]interface{}, len(assets))
	for i, dep := range assets {
		row := map[string]interface{}{
			"id":         dep.Location,
			"path":       dep.Path,
			"type":       dep.Type,
			"location":   dep.Location,
			"dependency": dep.Dependency,
		}
		tableRows[i] = row
	}

	for _, target := range targets {
		validations, err := release.ValidateAssetsForTarget(ctx, target, assets, opts)
		if err != nil {
			return fmt.Errorf("profile %q: %w", target, err)
		}
		key := strings.ReplaceAll(target, "-", "_")
		for i := range tableRows {
			tableRows[i]["status_"+key] = validations[i].Status
			tableRows[i]["warning_"+key] = validations[i].Warning
		}
	}

	cols := []output.Column{
		{Header: "LOCATION", Field: "location"},
		{Header: "DEPENDENCY", Field: "dependency", Width: 12},
	}
	for _, prof := range targets {
		key := strings.ReplaceAll(prof, "-", "_")
		cols = append(cols, output.Column{
			Header: fmt.Sprintf("STATUS (%s)", prof),
			Field:  "status_" + key,
			Func:   makeProfileStatusFunc(key),
		})
	}
	return renderThemedTable(w, tableRows, cols)
}

func renderTypeCountTable(w io.Writer, title string, counts []release.AssetTypeCount, total int) error {
	slog.Info(title)
	rows := make([]map[string]interface{}, 0, len(counts)+1)
	for _, c := range counts {
		rows = append(rows, map[string]interface{}{
			"type":  c.Type,
			"count": c.Count,
		})
	}
	rows = append(rows, map[string]interface{}{
		"type":  "TOTAL",
		"count": total,
	})
	return renderThemedTable(w, rows, []output.Column{
		{Header: "TYPE", Field: "type"},
		{Header: "COUNT", Field: "count"},
	})
}

func renderThemedTable(w io.Writer, rows interface{}, columns []output.Column) error {
	cfg, _ := loadConfig() // best-effort; nil cfg falls back to defaults
	f := output.New(output.FormatTable, w, resolveTableStyle(cfg))
	return f.Format(rows, columns)
}
