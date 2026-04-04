# Contributing to gemtracker

Thank you for your interest in contributing! This guide will help you get started.

## Code of Conduct

Be respectful and welcoming to all contributors. We're building something great together.

## How to Contribute

### 1. Report Issues
Found a bug or have a feature request?
- **Bugs**: Open an [issue](https://github.com/spaquet/gemtracker/issues) with a clear title and description
- **Features**: Start a [discussion](https://github.com/spaquet/gemtracker/discussions) to discuss the idea first

### 2. Fork and Clone
```bash
git clone https://github.com/YOUR_USERNAME/gemtracker.git
cd gemtracker
```

### 3. Create a Feature Branch
```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/your-bug-fix
```

Use descriptive names that explain what you're working on.

### 4. Make Your Changes

Follow these guidelines:
- **Keep changes focused** — One feature or fix per PR
- **Write clear commit messages** — Explain the "why" not just the "what"
- **Add tests** for new functionality
- **Update docs** if you change behavior

### 5. Code Quality

Before submitting your PR, ensure your code passes all quality checks:

```bash
# Run all tests
make test

# Format your code (auto-fixes most issues)
gofmt -s -w .

# Check for issues
go vet ./...
```

**GitHub Actions will check these automatically**, but fixing locally prevents CI failures.

### 6. Testing

Write tests for your changes:

```bash
# Run tests
make test

# Run a specific test
go test -v ./internal/gemfile -run TestAnalyze
```

Aim for good test coverage on new code. Check `internal/gemfile/` for examples.

### 7. Create a Pull Request

1. Push your branch: `git push origin feature/your-feature-name`
2. Go to GitHub and click "Compare & pull request"
3. Fill in the PR description:
   - **What**: Briefly describe what you changed
   - **Why**: Explain why this change is needed
   - **How**: Explain how it works
   - **Testing**: Describe how you tested it

Example PR description:

```markdown
## What
Add support for Gemfile global options

## Why
Users with custom Gemfile configurations couldn't analyze their projects.

## How
- Updated parser to handle global options
- Added tests for global option parsing
- Updated docs with new capability

## Testing
- Tested with sample Gemfiles containing global options
- All existing tests pass
- New tests cover edge cases
```

### 8. Review Process

- **Maintainers will review** your PR within a few days
- **Respond to feedback** constructively — we're all learning
- **Make requested changes** on your feature branch (push updates)
- **CI must pass** — tests, linting, formatting
- **One approval required** — Then you can merge!

## Development Workflow

### Project Structure

```
gemtracker/
├── cmd/gemtracker/          # CLI entry point
├── internal/
│   ├── gemfile/             # Core analysis logic
│   │   ├── parser.go        # Gemfile.lock parser
│   │   ├── analyzer.go      # Dependency analysis
│   │   ├── outdated.go      # Version checking
│   │   └── vulnerabilities.go # CVE detection
│   └── ui/                  # Terminal UI (BubbleTea)
│       ├── model.go         # UI state management
│       ├── update.go        # Message handling
│       ├── view.go          # Screen rendering
│       └── styles.go        # Colors & themes
├── CLAUDE.md                # Development guidelines
├── Makefile                 # Build targets
└── go.mod                   # Dependencies
```

### Key Files

- **Core Logic**: `internal/gemfile/` — Parsing, analysis, vulnerabilities
- **UI**: `internal/ui/` — Terminal interface using BubbleTea
- **Entry Point**: `cmd/gemtracker/main.go` — CLI setup
- **Tests**: `*_test.go` files throughout `internal/`

### Common Tasks

**Add a new analysis feature**:
1. Add logic to `internal/gemfile/analyzer.go` or new file
2. Write tests in `*_test.go`
3. Hook up UI in `internal/ui/`
4. Test end-to-end with `make build && ./gemtracker`

**Fix a bug**:
1. Write a test that reproduces the bug
2. Fix the bug
3. Verify test passes
4. Run full test suite: `make test`

**Update the UI**:
1. Modify `internal/ui/view.go` for rendering
2. Modify `internal/ui/update.go` for event handling
3. Modify `internal/ui/styles.go` for colors/spacing
4. Test with `make build && ./gemtracker`

## Code Style

- Follow standard Go conventions (gofmt)
- Keep functions small and focused
- Use descriptive variable names
- Comment exported types and functions
- Group related code together

### Formatting

Code must be formatted with `gofmt`:

```bash
# Check formatting
gofmt -s -l .

# Auto-fix formatting
gofmt -s -w .
```

The `-s` flag applies simplifications where possible.

### Comments

- **Exported functions**: Add a comment starting with the function name
- **Complex logic**: Explain why, not just what
- **Public types**: Document the purpose

Good example:
```go
// GetDependencies returns all gems that depend on the given gem.
// It searches both direct and transitive dependencies.
func GetDependencies(gemName string, gf *GemFile) []string {
    // ...
}
```

## Testing Guidelines

- **Unit tests**: Test individual functions
- **Integration tests**: Test real Gemfile.lock files
- **Edge cases**: Test boundary conditions (empty, very large, invalid input)
- **Names**: Use `Test[FunctionName]_[ScenarioName]` pattern

Example:
```go
func TestGetDependencies_BasicGem(t *testing.T) {
    // Test basic case
}

func TestGetDependencies_NoMatch(t *testing.T) {
    // Test when gem doesn't exist
}
```

## Pre-Commit Checklist

Before pushing your code:

- [ ] `make test` passes
- [ ] `gofmt -s -w .` applied
- [ ] `go vet ./...` passes
- [ ] Commit messages are clear and descriptive
- [ ] No unrelated changes mixed in
- [ ] Tests added for new functionality
- [ ] Docs updated if behavior changed

**One-liner to check everything:**
```bash
make test && gofmt -s -w . && go vet ./... && echo "✓ Ready for PR!"
```

## Release Process

When maintainers are ready to release:

1. Update `CHANGELOG.md` with new features/fixes
2. Tag release: `git tag v1.0.0`
3. GitHub Actions automatically builds and publishes
4. Homebrew formula is updated automatically

See [RELEASE_GUIDE.md](RELEASE_GUIDE.md) for details.

## Questions?

- **How do I...?** → Check [README.md](README.md) and [CLAUDE.md](CLAUDE.md)
- **Bug or weird behavior?** → Open an [issue](https://github.com/spaquet/gemtracker/issues)
- **Want to discuss an idea?** → Start a [discussion](https://github.com/spaquet/gemtracker/discussions)
- **Security issue?** → See [SECURITY.md](SECURITY.md)

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (see [LICENSE](LICENSE)).

---

Thank you for helping make gemtracker better! 🚀
