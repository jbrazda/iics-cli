# CR-0021: CI Pipeline Integration

## Summary

Eliminate the need to manually cancel and re-run the Build Deploy pipeline for selective
deployments. Allow developers to declare deployment options directly in the PR Description
using a structured "Release Manifest" section. The auto-triggered build pipeline parses this
section via the ADO REST API and drives build parameters accordingly — with safe, full-deployment
defaults when the section is absent or incomplete.

---

## Problem

### Current Selective Deployment Process (painful)

The existing [Quick Guide — Selective Deployment](../../../Informatica.wiki) requires:

1. Tag assets in the IICS DEV org.
2. Merge the PR — this immediately triggers the automated **Build Deploy Main** pipeline with
   a **full** build using `full_build.package.txt`.
3. **Cancel** the auto-triggered pipeline before it completes (race condition).
4. Manually re-trigger the pipeline and supply two non-default parameters:
   - `package_config` → `./conf/tag_build.package.txt`
   - `package_tag` → the exact single-word tag string

This workflow has several problems:

- Developers must know to cancel the auto-triggered run before it deploys to TST.
- A cancelled pipeline leaves a failed/cancelled run in the history, causing confusion.
- The manual trigger UI (`build-deployment-main.yml`) only surfaces parameters when run
  manually — there is no way to parameterize an auto-triggered (push-to-main) pipeline.

- The process is poorly documented and error-prone; missing the cancellation window results
  in an unintended full deployment.

- There is no record of the intent (selective vs full) linked to the PR.

---

## Context

1. A developer preparing a selective deployment updates the **Deployment Options** section in the PR template with the appropriate checkboxes and values to indicate their intent for a selective deployment.
   description using the updated `pull_request_template.md`.
2. They tick the appropriate checkboxes and (for tag-based mode) enter the IICS tag name.
3. They merge the PR normally - **no manual pipeline intervention required**.
4. The auto-triggered Build Deploy pipeline will read the PR description via the ADO REST API,
   parse the Deployment Options section, and honor the specified options.
5. Full deployment remains the default - if the section is absent, parsing fails, incomplete, or the PR
   was not triggered by a merge (e.g. a direct push), the pipeline will use safe pre-configured defaults (full deployment targeting TST and QA only).
6. Individual repositories can update their PR templates to include the new Deployment Options section, or rely on the default  deployment if they choose not to.
7. The new `BuildManifest` stage should resolve files based on the below table layout, which is designed to be easily extendable for future deployment options (e.g. additional package configurations) in a clear and deterministic way (first match wins, with a defined fallback): and the parser will use the configuration column to determine which package file to use for the build stage. The "Description" column is for human readers and has no impact on the parsing logic.
8. Parsing Stage should generate configuration files into the `/target/iics/conf` directory for downstream stages to consume, based on the selected options. For example, if "Selective Tag-Based" is selected, it should generate a `tag_build.package.csv` file with the appropriate content for the build stage to use. If "Full Deployment" is selected, it should ensure that `full_build.package.csv` is available for the build stage. This approach keeps the parsing logic focused on interpreting the PR description and leaves the actual configuration file generation to a dedicated step, which can be easily extended in the future as new deployment options are added.
   - The logic should be flexible to allow for future options that may require additional or different configuration files, without requiring changes to the parsing logic (e.g. if a future option requires a `custom.package.csv`, the parser can simply emit that file when the option is selected, and downstream stages can be designed to look for it if it exists)
   - Several types of builds are always implicitly supported and their implicit configuration in project_root/conf is optional as the parsing stage can generate the necessary package files based on the selected options:
     - Full Deployment (default): uses pre-configured `full_build.package.csv` (includes all assets except CAI Connections and Connectors located outside of the Currently deployed project folder, which typically require manual intervention after deployment) If the full_build.package.csv file is not present in the conf directory, the parsing stage should generate it with a default configuration that includes all assets in the source folder except for any connections or connectors referenced from other projects, to ensure that the full deployment option can function correctly even if the specific package file is not pre-configured in the repository. This provides a safety net to ensure that the full deployment option is always available and can be used as a fallback if there are issues with the selective deployment options or their configurations.
     - Selective Tag-Based: uses `tag_build.package.csv` (content generated based on the specified tag)
     - Connector deployment is always optional and controlled by the "Include Connectors" checkbox, the pipeline should generate a `connectors.package.csv` file with the appropriate content (include all CAI connectors and connections that are selected). Connectors are not typically included in the other package files and are only deployed if this option is selected, allowing for flexible deployment scenarios where connectors can be managed separately from other assets as their deployment requires manual intervention after deployment (setup of credentials or environment specific parameters).
     - Connectors package is build always and staged for deployment, but not deployed automatically, to allow for manual review and deployment of connectors as needed based on the target environment and deployment strategy. This provides an additional layer of control for deployer to manage connectors effectively while still allowing them to be included in the build process for visibility and staging purposes.
   - The parsing stage should also handle the generation of any additional configuration files needed for future deployment options, ensuring that the logic for interpreting the PR description remains focused on understanding the developer's intent while the specifics of how that intent translates into configuration files is handled in a modular way that can be easily extended as new deployment options are added in the future.

9. We will need to IICS CLI to check dependencies based on the user selection for tagged builds, so the parsing stage should generate corresponding configuration files accordingly. For example, if "Selective Tag-Based" is selected, the parsing stage should generate a `tag_build.package.csv` file that includes not only the assets tagged with the specified tag but also any dependencies that are required for those assets to function correctly in the target environments. This ensures that when the build stage runs with the `tag_build.package.csv` configuration, it has all the necessary information to package the correct set of assets and their dependencies, leading to a successful deployment. The logic to determine dependencies should include following
    1. List all tagged assets based on the specified tag using the IICS CLI in DEV org.
    2. Analyze dependencies for the listed assets, including any connections or connectors they rely on.
    3. Allow for an option to include or exclude connectors and connections based on the developer's selection in the PR description (e.g. if "Include Connectors" is selected, the dependency analysis should also include any connectors and connections that are required by the tagged assets).
    4. Allow explicit exclusions of any assets using persistent config (exploit listing) or regex --filter parameter that excludes them from the generated package file, to give developers control to exclude any assets that may be tagged but are not intended for deployment or publishing in the target environments, while still including their dependencies if needed. This provides flexibility for developers to manage their deployments effectively while ensuring that all necessary assets and dependencies are included based on the selected options in the PR description.
    5. Should we generate single config with environment specific columns similar to dependencies output, or individual files for each environment and generate individual package files for each environment? The former approach (single config with environment-specific columns) may be more efficient and easier to manage as it consolidates all the relevant information into a single file, making it easier for downstream stages to access and utilize the deployment options without needing to read multiple files. The parsing stage can generate a single `tag_build.package.csv` file that includes columns for each target environment (e.g. TST, QA, STG, PROD) with indicators for which assets should be included in the deployment for each environment based on the selected options in the PR description. This allows for a more streamlined and efficient deployment process while still providing the necessary flexibility to manage deployments across multiple environments effectively.
    6. Generate a `publish_assets.csv` into the `target/iics/import/{env}` to drive the publish step in the pipeline. This file should include the list of assets to be published based on the selected options and their dependencies, ensuring that only the relevant assets are published to the target environments during deployment. This allows for a more efficient deployment process by avoiding unnecessary publishing of assets that are not part of the selective deployment, while still ensuring that all required assets and their dependencies are included in the deployment package.
10. The parsing stage should be designed to be robust and handle various edge cases gracefully. For example, if the PR description is missing the Deployment Options section, or if the section is present but incomplete (e.g. Tag-Based mode selected but no tag provided), the stage should emit clear error messages and fail gracefully, prompting the developer to correct the PR description before merging. Additionally, any issues with fetching the PR description via the ADO REST API (e.g. authentication issues, network errors) should be logged clearly, and the stage should fall back to safe defaults rather than causing a hard failure of the pipeline.
11. The new process should be clearly documented in the project wiki, including instructions for developers on how to use the new Deployment Options section in their PR descriptions, and how the pipeline will interpret those options to drive the deployment process. This documentation should also include troubleshooting tips for common issues that may arise with the new process, such as parsing errors or misconfigured options in the PR description.
12. This step should also generate `release_manifest.md`  into the `/target/iics/logs`  for reviewers and auditing purposes, capturing the exact Deployment Options as parsed from the PR description for each pipeline run. This manifest serves as a record of the deployment intent for each PR and can be used for troubleshooting and auditing to understand what options were selected for a given deployment. It also provides visibility into the deployment choices made by developers directly in the pipeline logs, which can be helpful for reviewers and maintainers to verify that the correct options were selected and parsed as intended.
13. This step should also generate `release_manifest.yaml` into the `/target/iics/conf` directory for downstream stages to consume, based on the selected options.  The  manifest captures all the relevant deployment options and their values in a machine-readable format. This allows downstream stages to easily access and utilize the deployment options without needing to re-parse the PR description, improving efficiency and reducing the risk of parsing errors in multiple stages. The manifest should include all selected options, such as deploy mode, tag (if applicable), target environments, and connectors handling preferences, providing a comprehensive overview of the deployment intent that can be used throughout the pipeline.
14. The content of the `target/iics/` directory is published as pipeline artifacts and should be accessible in subsequent stages. The parsing stage should ensure that any generated configuration files (e.g. `tag_build.package.csv`, `connectors.package.csv`) and manifests (`release_manifest.md`, `release_manifest.yaml`) are placed in the correct location and are available for downstream stages to consume as needed for the build and deployment processes.

## Design considerations

- Should we use one configuration file for all environments with environment-specific columns, or generate separate configuration files for each environment?
- The option to use separate files should generate consistent file names in the corresponding directories i.e
- `target/iics/conf/tst/tag_build.package.csv`, `target/iics/conf/qa/tag_build.package.csv`, etc. for package configurations and `target/iics/conf/tst/publish_assets.csv`, `target/iics/conf/qa/publish_assets.csv`, etc. for publish assets, to ensure that downstream stages can easily locate and utilize the correct files based on the target environment. and allow for flexibility in managing environment-specific configurations while still maintaining a clear and organized structure for the generated files.
- The pipeline should be designed to be flexible and easily extendable for future deployment options, allowing for new options to be added without requiring significant changes to the parsing code. This can be achieved by using a structured format for the Deployment Options section in the PR description, and designing the parsing logic to interpret that structure in a way that can accommodate new options as they are added in the future. For example, if a new deployment option is added that requires a different package configuration, the parsing logic can simply look for that option in the PR description and generate the corresponding configuration file without needing to change how existing options are parsed or handled.
- provide a suggestion for the pipeline steps handling ie should we pipe steps between commands or rather use file based intermediate state (e.g. generate config files that are consumed by the build stage, rather than trying to pass parameters directly between steps in the pipeline). The file-based approach may provide better traceability and decoupling between stages. The most optimal approach may be to generate configuration files based on the parsed options and have downstream stages read those files to determine their behavior, rather than trying to pass parameters directly between steps in the pipeline. This allows for better traceability of the deployment options and provides a clear record of what options were selected for each deployment, while also decoupling the stages and allowing them to operate independently based on the generated configuration files.

    **Global Config Files**

    ```shell
    target/iics/conf/full_build.package.csv # default full deployment package config copied over from project root conf directory or generated by the parsing stage if not present
    target/iics/conf/connectors.package.csv # connectors package config generated dynamically from the contents of the project dependencies and the "Include Connectors" option in the PR description
    ```

    **Package Config Files**

    ```shell
    target/iics/import/tst/tag_build.package.csv # for TST environment 
    target/iics/import/qa/tag_build.package.csv # for QA environment
    target/iics/import/stg/tag_build.package.csv # for STG environment
    target/iics/import/prod/tag_build.package.csv # for PROD environment
    ```

    **Publish Config Files**

    ```shell
    target/iics/conf/tst/publish_assets.csv # for TST environment
    target/iics/conf/qa/publish_assets.csv # for QA environment
    target/iics/conf/stg/publish_assets.csv # for STG environment
    target/iics/conf/prod/publish_assets.csv # for PROD environment
    ```

    **Deployment Archives**

    ```shell
    target/iics/deploy/tst/{ProjectName}_{Tag}_{gitcommit}_{YYYYMMDD-hhmmss}.zip # for TST environment
    target/iics/deploy/qa/{ProjectName}_{Tag}_{gitcommit}_{YYYYMMDD-hhmmss}.zip # for QA environment
    target/iics/deploy/stg/{ProjectName}_{Tag}_{gitcommit}_{YYYYMMDD-hhmmss}.zip # for STG environment
    target/iics/deploy/prod/{ProjectName}_{Tag}_{gitcommit}_{YYYYMMDD-hhmmss}.zip # for PROD environment
    ```

    **Manifest Files**

    ```shell
    target/iics/logs/release_manifest.md # human-readable manifest for logging and auditing
    target/iics/conf/release_manifest.yaml # machine-readable manifest for downstream stages
    ```

- The parsing logic should be designed to be flexible and easily extendable for future deployment options, allowing for new options to be added without requiring significant changes to the parsing code. This can be achieved by using a structured format for the Deployment Options section in the PR description, and designing the parsing logic to interpret that structure in a way that can accommodate new options as they are added in the future.

---

## CI Pipeline Integration

### Updated Pull Request Template

Add a structured **`## Deployment Options`** section to `.azuredevops/pull_request_template.md`.

Design constraints:

- Must be parseable with simple regex (no YAML front-matter, no JSON — ADO renders raw markdown).
- Checkboxes use GitHub/ADO markdown `[x]` / `[ ]` which are easy to detect with regex.
- Exactly one "Deploy Mode" checkbox should be checked. The parser treats the first checked
  option as authoritative; unrecognized combinations fall back to **Full Deployment**.

- The tag field uses an inline key–value pattern: `Tag: <value>` (single line).

```markdown
## Deployment Options

> **Instructions** - Fill in before merging to control the automated Build Deploy pipeline.
> Leave the defaults (Full Deployment, TST + QA) for a standard release.
> Delete this entire section to use pipeline defaults.

### Deploy Mode *(check exactly one)*

- [x] Full Deployment *(all assets - uses `conf/full_build.package.csv`)*
- [ ] Selective - Tag-Based *(assets tagged in DEV with the tag below + their dependencies)*

### Deployment Tag *(required for Tag-Based mode - must match an IICS DEV tag exactly)*

Tag: `sample_tag` <!-- enter single-word tag here, e.g. sprint-42 -->

> **Note:**  Tag-based deployments will automatically include all missing dependencies of tagged assets to specific org. Dependencies are determined by analyzing asset references and API metadata, and may include connections or connectors if tagged assets rely on them. The generated package file for tag-based deployments will include both the tagged assets and their dependencies to ensure a successful deployment to the target environments. Build pipeline will built specific package for each environment based on the dependencies and options selected in the PR description, ensuring that the correct set of assets and their dependencies are included for deployment to each environment.
```

### Target Environments

- [x] TST
- [x] QA
- [ ] STG
- [ ] PROD

### Connectors Package *(optional - typically managed manually post-deploy)*

- [x] Connectors and Connections (CAI assets that require manual intervention after deployment)

> **Note:**  Connectors and connections are typically not included in the default package configurations and require manual intervention after deployment (e.g. setting up credentials or environment-specific parameters). This connector Package is only built and staged for deployment but must be deployed manually to allow for review and control over when and how connectors are deployed based on the target environment and deployment strategy.
```

