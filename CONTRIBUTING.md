# Contributing to iics-cli

Thank you for your interest in contributing! This document provides guidelines and information for contributors.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/<your-username>/iics-cli.git`
3. Create a feature branch: `git checkout -b feature/my-feature`
4. Make your changes
5. Run tests: `make test`
6. Run linter: `make lint`
7. Commit your changes and push to your fork
8. Open a Pull Request against the `dev` branch

## Development Setup

### Prerequisites

- Go 1.25 or later
- Make
- golangci-lint (for linting)

### Building

```bash
make build
```

### Running Tests

```bash
make test
```

### Linting

```bash
make lint
```

## Code Guidelines

- Follow standard Go conventions and [Effective Go](https://go.dev/doc/effective_go)
- Run `gofmt` and `goimports` before committing
- Add tests for new functionality
- Keep the `cmd/` layer thin - business logic belongs in `internal/client/`
- Use `internal/output/` for formatting; commands should not print directly

## Pull Request Process

1. Target the `dev` branch (not `main`)
2. Ensure all tests pass and the linter is clean
3. Update documentation if your change affects CLI behavior
4. Keep PRs focused - one feature or fix per PR
5. Write clear commit messages describing the "why"

## Reporting Issues

- Use GitHub Issues to report bugs or request features
- Include the `iics` version (`iics --version`) and your OS
- For bugs, include steps to reproduce and expected vs. actual behavior

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
