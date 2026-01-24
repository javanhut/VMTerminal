# Contributing to VMTerminal

Thank you for your interest in contributing to VMTerminal!

## Development Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/javanstorm/vmterminal.git
   cd vmterminal
   ```

2. **Install dependencies:**
   ```bash
   make deps
   ```

3. **Build and test:**
   ```bash
   make build
   make test
   ```

## Requirements

- **Go 1.22+** (or latest)
- **macOS 12+** for testing macOS features (Virtualization.framework)
- **Linux with KVM** for testing Linux features

## Making Changes

1. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/your-feature
   ```

2. **Make your changes** and ensure tests pass:
   ```bash
   make check  # runs fmt, vet, build
   make test
   ```

3. **Commit with a clear message:**
   ```bash
   git commit -m "feat: add your feature"
   ```

4. **Push and create a pull request.**

## Code Style

- Follow standard Go formatting (`gofmt`)
- Run `go vet` before committing
- Add tests for new functionality
- Keep commits focused and atomic
- Use table-driven tests where appropriate

## Commit Messages

Use conventional commits:
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `test:` - Test additions or changes
- `refactor:` - Code refactoring
- `chore:` - Maintenance tasks

Examples:
```
feat: add snapshot verification command
fix: handle SIGHUP in login shell mode
docs: update installation instructions
test: add coverage for SnapshotManager
```

## Testing

Run tests with:
```bash
make test           # Run all tests
go test -v ./...    # Verbose output
go test -run TestName ./internal/vm/...  # Run specific test
```

Some tests require platform-specific features:
- Tests using KVM are skipped on non-Linux systems
- Tests using Virtualization.framework are skipped on non-macOS systems

## Pull Request Process

1. Ensure all tests pass
2. Update documentation if needed
3. Add tests for new functionality
4. Keep PRs focused - one feature/fix per PR
5. Respond to review feedback promptly

## Project Structure

```
cmd/vmterminal/      # Entry point
internal/
├── cli/             # CLI commands (Cobra)
├── config/          # Configuration
├── distro/          # Linux distribution providers
├── terminal/        # Terminal handling
├── timing/          # Startup timing instrumentation
├── version/         # Version information
└── vm/              # VM management, state, snapshots
pkg/
└── hypervisor/      # Hypervisor abstraction layer
    ├── driver_darwin.go  # macOS Virtualization.framework
    └── driver_linux.go   # Linux KVM
```

## Questions?

Open an issue for questions or discussions. We're happy to help!

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
