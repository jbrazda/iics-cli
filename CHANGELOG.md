# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- `usergroup list` displayed empty NAME column - corrected JSON tag on `UserGroupName` from `"name"` to `"userGroupName"` to match the IICS v3 API response field

### Changed

- `usergroup list` now shows member count and role count columns (`countMembers`, `countRoles`) by default instead of `description`
- `usergroup list` CSV output includes `description`, `countMembers`, and `countRoles` by default

### Added

- `usergroup list --query` / `-q` flag for server-side filtering (e.g. `userGroupName=="Administrator"`)
- `usergroup list --fields` flag for custom column selection; accepts any combination of `id`, `userGroupName`, `description`, `updatedBy`, `updateTime`, `createdBy`, `createTime`, `countMembers`, `countRoles`
- `UserGroupMember` struct captures the `users` array returned by the userGroups API

### Added (initial)

- Initial CLI with Cobra framework and Viper configuration
- Multi-profile configuration (`~/.iics/config.yaml`) with environment variable overrides
- Session caching with 30-minute expiry and automatic 401 retry
- Table and JSON output formats
- Commands: login, logout
- Commands: objects list, objects dependencies
- Commands: lookup
- Commands: connection list, get, create, update, delete
- Commands: export create, status, download
- Commands: import upload, start, status
- Commands: schedule list, get, create, update, delete
- Commands: project create, update, delete
- Commands: folder create, update, delete
- Commands: user list, get, create, update, delete
- Commands: usergroup list, get, create, update, delete
- Commands: role list, get, create, update, delete
- Commands: privilege list
- Commands: runtime list, get, create, update
- Commands: agent list, start, stop
- Commands: tag assign, remove
- Commands: permission get, set, delete
- Commands: securitylog list
- Commands: metering get, download
- Commands: sourcecontrol checkout, checkin, pull, commit
- Commands: state fetch, load
- Support for all IICS regions (US, EMEA, APJ, etc.)
- Cross-platform builds (Linux, macOS, Windows; amd64, arm64)
- GitHub Actions CI pipeline
- golangci-lint configuration
