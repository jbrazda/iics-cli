# Release Manifest

## Deployment Options

> **Instructions** - Fill in before merging to control the automated Build Deploy pipeline.
> Leave the defaults (Full Deployment, TST + QA) for a standard release.

### Deploy Mode *(check exactly one)*

- [ ] Full Deployment *(all assets - uses `conf/all_exclude_connections.package.csv`)*
- [x] Selective - Tag-Based *(assets tagged in DEV with the tag below + their dependencies)*

### Deployment Tag *(required for Tag-Based mode - must match an IICS DEV tag exactly)*

Tag: `ZZ_TEST_CLI` <!-- enter single-word tag here, e.g. sprint-42 -->

> **Note:**  Tag-based deployments will automatically include all missing dependencies of tagged assets to specific org. Dependencies are determined by analyzing asset references and API metadata, and may include connections or connectors if tagged assets rely on them. The generated package file for tag-based deployments will include both the tagged assets and their dependencies to ensure a successful deployment to the target environments. Build pipeline will built specific package for each environment based on the dependencies and options selected in the PR description, ensuring that the correct set of assets and their dependencies are included for deployment to each environment.

### Target Environments

- [x] TST
- [x] QA
- [ ] STG
- [ ] PROD

### Connectors Package *(optional - typically managed manually post-deploy)*

- [x] Connectors (CAI connector assets that may require manual intervention after deployment)
- [x] Connections (CAI connection assets that may require manual intervention after deployment)

> **Note:**  Connectors and connections are typically not included in the default package configurations and require manual intervention after deployment (e.g. setting up credentials or environment-specific parameters). This connector Package is only built and staged for deployment but must be deployed manually to allow for review and control over when and how connectors are deployed based on the target environment and deployment strategy.
