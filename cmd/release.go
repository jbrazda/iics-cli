package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	cmd.Flags().StringVar(&outputRoot, "output-root", "target/iics", "output root directory for generated files")
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
	cmd.Flags().StringVar(&manifestPath, "manifest", "target/iics/conf/release_manifest.yaml", "path to release manifest yaml")
	cmd.Flags().StringVar(&validTargets, "valid-targets", "", "comma-separated allowlist of valid targets (overrides IICS_VALID_DEPLOY_TARGETS)")
	return cmd
}

func newReleasePlanCmd() *cobra.Command {
	var (
		manifestPath     string
		outputRoot       string
		fullPackageCfg   string
		validTargets     string
		packageFieldsRaw string
		publishFieldsRaw string
	)
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Generate per-environment package and publish CSV files",
		RunE: func(cmd *cobra.Command, args []string) error {
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
			excludes, err := release.LoadExcludePatterns(opts.ExcludeFile)
			if err != nil {
				return err
			}

			packageFields := splitCSVFields(packageFieldsRaw, []string{"location", "dependency", "type", "path"})
			publishFields := splitCSVFields(publishFieldsRaw, []string{"location", "dependency"})

			if opts.Mode == release.ModeFullDeployment {
				for _, env := range opts.Targets {
					envDir := filepath.Join(outputRoot, "conf", strings.ToLower(env))
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
				}
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
			allFiltered := release.ApplyPolicies(assets, opts.IncludeConnectors, opts.ConnectorsOnly, excludes)
			publishAssets := release.PublishAssets(allFiltered)
			connectorAssets := release.ConnectorAssets(allFiltered)

			for _, env := range opts.Targets {
				envDir := filepath.Join(outputRoot, "conf", strings.ToLower(env))
				if err := os.MkdirAll(envDir, 0o755); err != nil {
					return fmt.Errorf("creating env directory: %w", err)
				}
				if err := release.WriteAssetsCSV(filepath.Join(envDir, "tag_build.package.csv"), allFiltered, packageFields); err != nil {
					return err
				}
				if err := release.WriteAssetsCSV(filepath.Join(envDir, "publish_assets.csv"), publishAssets, publishFields); err != nil {
					return err
				}
			}
			if opts.IncludeConnectors {
				if err := release.WriteAssetsCSV(filepath.Join(outputRoot, "conf", "connectors.package.csv"), connectorAssets, packageFields); err != nil {
					return err
				}
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Generated selective plan files for targets: %s\n", strings.Join(opts.Targets, ","))
			return nil
		},
	}
	cmd.Flags().StringVar(&manifestPath, "manifest", "target/iics/conf/release_manifest.yaml", "path to release manifest yaml")
	cmd.Flags().StringVar(&outputRoot, "output-root", "target/iics", "output root directory for generated files")
	cmd.Flags().StringVar(&fullPackageCfg, "full-package-config", "./conf/all_exclude_connections.package.csv", "full-deployment package config file to copy per environment")
	cmd.Flags().StringVar(&validTargets, "valid-targets", "", "comma-separated allowlist of valid targets (overrides IICS_VALID_DEPLOY_TARGETS)")
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
