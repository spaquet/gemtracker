# Gemtracker Test Data

This directory contains test fixtures for gemtracker's parsing, analysis, and reporting features.

## Directory Structure

```
testdata/
├── README.md                    # This file
└── projects/                    # Test fixture projects
    ├── README.md               # Project-specific documentation
    ├── minimal-example/         # Small Rails app (unit testing)
    ├── simple-deps/            # Minimal deps (basic parsing)
    ├── standard-project/       # Full Rails 7.0 app (40 gems)
    ├── bundled-gemfile/        # gems.locked format (40 gems)
    └── gem-project/            # Gem package with gemspec (40 gems)
```

## Test Projects Overview

### `minimal-example/` - Unit Testing
**Purpose:** Small, focused fixture for unit tests

**Files:**
- `Gemfile.lock` - Basic Rails app with 15 gems

**Gems Included:**
- Core: `rails`, `actionpack`, `activesupport`, `railties`
- Auth: `devise`, `bcrypt`, `warden`
- DB: `rack`, `rack-test`
- Transitive deps: `concurrent-ruby`, `i18n`, `minitest`, `tzinfo`, `orm_adapter`, `responders`, `thor`, `rake`

**Use Cases:**
- Parser unit tests (valid file parsing, gem extraction)
- Analyzer tests (forward/reverse dependencies, tree building)
- Dependency analysis (first-level gem identification)
- Basic relationship mapping

### `simple-deps/` - Parser Edge Cases
**Purpose:** Minimal test case for basic parsing and dependency chains

**Files:**
- `Gemfile.lock` - 2 gems with simple dependency

**Gems:**
- `simple-gem` (1.0.0) - no dependencies
- `another-gem` (2.0.0) - depends on simple-gem

**Use Cases:**
- Testing basic parsing without noise
- Dependency extraction and validation
- Testing gems with/without dependencies
- Edge case handling

### `standard-project/` - Integration Testing
**Purpose:** Realistic Rails 7.0 application with comprehensive gem coverage

**Files:**
- `Gemfile.lock` - Rails 7.0.4 stack with 40+ direct gems

**Key Gems:**
- **Framework:** actioncable, actionmailbox, actionmailer, actionpack, actiontext, actionview, activejob, activemodel, activerecord, activestorage, activesupport
- **Auth:** devise, jwt, bcrypt
- **API:** jsonapi-serializer, jbuilder, graphql, rest-client
- **Search/Cache:** elasticsearch, redis
- **Image Processing:** ruby-vips, mini_magick
- **Testing:** rspec-rails, factory_bot_rails, faker, capybara, webmock
- **Code Quality:** rubocop, rubocop-rails, rubocop-rspec, guard, solargraph
- **Monitoring:** sentry-rails
- **Utilities:** kaminari, friendly_id, aws-sdk-s3, icalendar, shrine, simple_form

**Use Cases:**
- Full integration tests
- Outdated detection testing
- Large gem set handling
- Performance testing
- Real-world scenario simulations

### `bundled-gemfile/` - Alternative Format Testing
**Purpose:** Test `gems.locked` format (Bundler alternative lock file)

**Files:**
- `gems.locked` - Rails 6.1 stack using alternative lock format

**Key Gems:**
- **Rails 6.1** action/active* modules
- **Infrastructure:** AWS (SDK, ACM, Cognito, X-Ray), Azure (Identity, KeyVault, Storage)
- **Utilities:** Analytics, annotations, authentication, authorization, automation, AWS features
- **Database:** SQL support, caching, messaging

**Use Cases:**
- Lock file format compatibility
- Bundler alternative support
- Version compatibility testing (Rails 6.1)
- Non-standard project structure handling

### `gem-project/` - Gem Specification Testing
**Purpose:** Test gemspec file parsing and gem package analysis

**Files:**
- `my_gem.gemspec` - Ruby gem specification with 40 declared dependencies
- `Gemfile.lock` - Development/test dependencies

**Gemspec Dependencies (40):**
- Core Rails 7.0: rails, activemodel, activerecord, activesupport
- **Web:** puma, sinatra, webrick
- **Databases:** pg, mysql2, sqlite3, sequel
- **Auth/Security:** devise, pundit, jwt, bcrypt
- **API:** jsonapi-serializer, jbuilder, graphql, rest-client
- **Search/Cache:** elasticsearch, redis, memcached
- **Image Processing:** image_processing, ruby-vips, imagemagick
- **Monitoring:** sentry-rails, newrelic_rpm, datadog
- **Testing:** rspec-rails, factory_bot_rails, faker, rubocop
- **Development:** pry, guard, solargraph

**Use Cases:**
- Gemspec parsing (not just lock files)
- Gem packaging analysis
- Development vs runtime dependency distinction
- Gem distribution scenario testing

## Usage in Tests

### Running Tests

```bash
# Run all tests
make test

# Run specific test file
go test ./internal/gemfile -v

# Run specific test
go test ./internal/gemfile -run TestParse_ValidFile -v
```

### Test File References

| Test File | Projects Used |
|-----------|---------------|
| `internal/gemfile/parser_test.go` | minimal-example, simple-deps |
| `internal/gemfile/analyzer_test.go` | minimal-example |
| `internal/gemfile/dependencies_test.go` | minimal-example, simple-deps |
| `internal/ui/report_test.go` | minimal-example |

### Example Test Patterns

**Parsing a test project:**
```go
path := "testdata/projects/standard-project/Gemfile.lock"
gf, err := Parse(path)
if err != nil {
    t.Fatalf("Parse failed: %v", err)
}
```

**Analyzing dependencies:**
```go
path := "testdata/projects/minimal-example/Gemfile.lock"
gf, err := Parse(path)
result := AnalyzeDependencies(gf, "rails")
```

**Testing gemspec:**
```go
path := "testdata/projects/gem-project/my_gem.gemspec"
gf, err := Parse(path)
// Verify gemspec dependencies are parsed
```

## Adding New Test Fixtures

When adding new test projects:

1. **Create directory** under `testdata/projects/`
2. **Add appropriate files:**
   - `Gemfile.lock` or `gems.locked` for lock file tests
   - `my_gem.gemspec` for gem package tests
   - `Gemfile` (optional) for group tests
3. **Document in `projects/README.md`:**
   - Purpose of the fixture
   - Key gems and their versions
   - Test scenarios covered
   - Any special characteristics
4. **Use in tests** via relative path: `testdata/projects/your-project/Gemfile.lock`

## Test Data Characteristics

### Version Strategies
- **Minimal:** simple-deps (2 gems)
- **Small:** minimal-example (15 gems)
- **Large:** standard-project, bundled-gemfile, gem-project (40+ gems each)

### Framework Versions
- **Rails 7.0.x:** standard-project, gem-project
- **Rails 6.1.x:** bundled-gemfile
- **Mini:** minimal-example, simple-deps (no full Rails)

### Special Cases
- **Git dependencies:** Acts-as-taggable-on with GIT section in parser_test.go examples
- **Platform-specific gems:** Handled in all projects
- **Gem groups:** default, test, development (where applicable)
- **Transitive dependencies:** All projects show gem chains

## Performance Notes

- **minimal-example** and **simple-deps** - Fast, suitable for quick unit tests
- **standard-project**, **bundled-gemfile**, **gem-project** - Moderate size, good for integration tests
- No projects exceed 100 gems to keep tests fast

## Notes

- Lock files are generated to match real Bundler output
- Versions reflect actual gems available on rubygems.org at fixture creation
- All fixtures use `BUNDLED WITH` markers for version compatibility
- Gemspec in gem-project uses realistic dependency ranges
