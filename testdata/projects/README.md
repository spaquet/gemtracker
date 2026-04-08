# Test Fixture Projects

This directory contains test projects for testing gemtracker's parsing and analysis capabilities across different project types and dependency formats.

## Projects

### 1. `standard-project/`
A standard Rails 7.0 application with a traditional `Gemfile.lock` file.

**Features:**
- 40 direct gem dependencies
- Mix of outdated, recent, and security-critical gems
- Includes transitive dependencies
- Real-world Rails stack: ActionPack, Devise, Elasticsearch, etc.

**Test Coverage:**
- Standard Gemfile.lock parsing
- Dependency tree analysis
- Version outdatedness detection
- Gem relationship mapping

### 2. `bundled-gemfile/`
Alternative bundle format using `gems.locked` instead of `Gemfile.lock`.

**Features:**
- 40 direct gem dependencies in `gems.locked` format
- Mix of infrastructure and utility gems
- Rails 6.1 stack (slightly older)
- Includes less common gems (Azure, AWS, analytics tools)

**Test Coverage:**
- Alternative lock file format support
- Backward compatibility with older lock files
- Format normalization

### 3. `gem-project/`
A gem package project with both `gemspec` and `Gemfile.lock`.

**Features:**
- Contains `my_gem.gemspec` with 40 declared dependencies
- Full `Gemfile.lock` for development environment
- Mixed dependency types (core Rails, databases, testing, monitoring)
- Simulates a redistributable gem package

**Test Coverage:**
- Gemspec parsing and dependency extraction
- Gem metadata handling
- Development vs runtime dependency distinction

## Gem Characteristics

Each project includes gems with varied characteristics:

- **Recent versions:** `rails ~> 7.0.4`, `graphql ~> 2.0`, `faker ~> 3.1`
- **Outdated versions:** `rack 2.2.6.4`, `devise 4.8.1`, `nokogiri 1.14.2`
- **Security-aware:** `bcrypt`, `jwt`, `devise` (authentication)
- **Infrastructure:** AWS, Azure, Elasticsearch, Redis, PostgreSQL, MySQL
- **Testing:** RSpec, Factory Bot, Faker, WebMock
- **Code Quality:** Rubocop, Guard, Solargraph
- **Analytics:** Sentry, NewRelic, Datadog, LaunchDarkly

## Usage in Tests

```ruby
# Parse standard Gemfile.lock
parser = GemfileParser.new("testdata/projects/standard-project/Gemfile.lock")
gems = parser.parse

# Parse gems.locked format
parser = GemfileParser.new("testdata/projects/bundled-gemfile/gems.locked")
gems = parser.parse

# Parse gemspec
parser = GemfileParser.new("testdata/projects/gem-project/my_gem.gemspec")
gems = parser.parse
```

## Adding More Test Data

When adding new fixtures:
1. Create a new directory under `testdata/projects/`
2. Include appropriate lock/spec file
3. Keep to 40 direct dependencies for consistency
4. Document special cases in a comment at the top of the file
5. Update this README with the new project

## Known Issues & Notes

- Platform-specific gems (e.g., `x86_64-linux`) are preserved as-is
- Pre-release versions (e.g., `3.0.0.rc1`) are included for version parsing tests
- Some gems intentionally have outdated versions for outdated detection tests
- Vulnerability detection tests should reference the documented CVEs in comments within lock files
