# Development Guide

This document contains information for developers working on Lloader.

## Project Structure

```
lloader/
├── cmd/lload/           # CLI entry point
│   ├── main.go         # Main application entry point
│   └── commands/       # Cobra CLI commands (list, config, version)
├── internal/            # Internal packages
│   ├── app/            # Application configuration & setup
│   ├── ui/             # Bubble Tea TUI components & state management
│   ├── process/        # Process management for llama.cpp execution
│   └── models/         # Model discovery, validation & metadata
├── config/              # Configuration files & examples
├── AGENTS.md           # AI assistant directives
├── Makefile            # Build automation & development tasks
├── go.mod/go.sum       # Go module dependencies
├── README.md           # User documentation
└── DEV.md              # This development guide
```

## Prerequisites

- Go 1.24+
- llama.cpp installed and in PATH (llama-server, llama-cli)
- Optional: golangci-lint for code linting

## Building & Testing

```bash
make build        # Build binary to bin/lload
make test         # Run all tests
make test-coverage # Run tests with coverage report
make lint         # Run golangci-lint
make install      # Install to GOPATH/bin
make clean        # Clean build artifacts
```

## Development Workflow

```bash
make dev          # Run directly with go run (development mode)
make run          # Build and run binary
```

## Cross-Platform Builds

```bash
make release      # Build for Linux, macOS, and Windows
```

## Adding Dependencies

```bash
go get github.com/package/name
go mod tidy
```

## Commit Convention

This project uses [Conventional Commits](https://conventionalcommits.org/):

- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `style:` - Code style changes
- `refactor:` - Code refactoring
- `test:` - Testing
- `chore:` - Maintenance

Examples:
- `feat: add HuggingFace model search`
- `fix: resolve quantization selection bug`
- `docs: update installation instructions`
- `refactor: simplify model loading logic`

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes following the commit convention
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Style

- Follow Go conventions and idioms
- Use `gofmt` for formatting
- Run `make lint` before committing
- Add tests for new features
- Update documentation as needed

### Pull Request Guidelines

- Ensure all tests pass
- Update documentation for any user-facing changes
- Follow the commit message conventions
- Keep PRs focused on a single feature or fix
- Provide a clear description of the changes and their purpose

## Dependencies

### Core Dependencies
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Elegant terminal user interface framework
- [Cobra](https://github.com/spf13/cobra) - CLI applications with commands and flags
- [Viper](https://github.com/spf13/viper) - Configuration management with multiple sources
- [Zap](https://github.com/uber-go/zap) - High-performance structured logging
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Terminal styling and layout
- [Bubbles](https://github.com/charmbracelet/bubbles) - Common Bubble Tea components

### HuggingFace Integration
- [hf-go](https://github.com/Megatherium/hf-go) - Go client for HuggingFace Hub API

## Architecture Notes

### TUI State Management
The application uses the [Elm Architecture](https://guide.elm-lang.org/architecture/) via Bubble Tea:
- **Model**: Application state (`internal/ui/model.go`)
- **Update**: State transitions based on messages
- **View**: Rendering the UI to strings

### Process Management
Model execution is handled through subprocess management:
- Server mode: Long-running llama-server process
- CLI mode: Interactive llama-cli with stdin/stdout piping
- Automatic process lifecycle management

### Configuration Hierarchy
1. Command-line flags (highest precedence)
2. Environment variables
3. Configuration file
4. Default values (lowest precedence)

## Testing

Run the test suite:
```bash
make test
```

Run with coverage:
```bash
make test-coverage
```

### Test Structure
- Unit tests for individual functions
- Integration tests for component interaction
- End-to-end tests for critical user flows

## Release Process

1. Update version in relevant files
2. Run full test suite
3. Create git tag with version
4. Build release binaries with `make release`
5. Create GitHub release with binaries
6. Update documentation if needed