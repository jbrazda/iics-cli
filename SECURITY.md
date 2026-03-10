# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public GitHub issue
2. Email the maintainer directly or use [GitHub's private vulnerability reporting](https://github.com/jbrazda/iics-cli/security/advisories/new)
3. Include a description of the vulnerability and steps to reproduce

We will acknowledge receipt within 48 hours and provide a timeline for a fix.

## Credential Safety

This CLI handles IICS credentials. Please follow these practices:

- **Never** commit passwords or session tokens to version control
- Use environment variables (`IICS_PASSWORD`) instead of storing passwords in `~/.iics/config.yaml`
- The session cache (`~/.iics/sessions.yaml`) contains session tokens - ensure it is not shared
- The `.gitignore` excludes `.iics/` and `.env` files by default

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest  | Yes       |
