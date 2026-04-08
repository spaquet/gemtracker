# frozen_string_literal: true

require_relative "lib/my_gem/version"

Gem::Specification.new do |spec|
  spec.name          = "my-gem"
  spec.version       = MyGem::VERSION
  spec.authors       = ["Test Author"]
  spec.email         = ["test@example.com"]

  spec.summary       = "A test gem with 40 direct dependencies"
  spec.description   = "Used for testing gemtracker's handling of gemspec files"
  spec.homepage      = "https://github.com/spaquet/gemtracker"
  spec.license       = "MIT"

  spec.metadata["allowed_push_host"] = "https://rubygems.org"
  spec.metadata["changelog_uri"] = "https://github.com/spaquet/gemtracker/blob/main/CHANGELOG.md"
  spec.metadata["homepage_uri"] = spec.homepage
  spec.metadata["source_code_uri"] = "https://github.com/spaquet/gemtracker"

  spec.files = Dir.chdir(File.expand_path(__dir__)) do
    `git ls-files -z`.split("\x0").reject { |f| f.match(%r{\A(?:test|spec|features)/}) }
  end
  spec.bindir        = "exe"
  spec.executables   = spec.files.grep(%r{\Aexe/}) { |f| File.basename(f) }
  spec.require_paths = ["lib"]

  # Core dependencies
  spec.add_dependency "rails", "~> 7.0"
  spec.add_dependency "activemodel", "~> 7.0"
  spec.add_dependency "activerecord", "~> 7.0"
  spec.add_dependency "activesupport", "~> 7.0"

  # Web framework & server
  spec.add_dependency "puma", "~> 5.6"
  spec.add_dependency "sinatra", "~> 3.0"
  spec.add_dependency "webrick", "~> 1.7"

  # Database
  spec.add_dependency "pg", "~> 1.1"
  spec.add_dependency "mysql2", "~> 0.5"
  spec.add_dependency "sqlite3", "~> 1.6"
  spec.add_dependency "sequel", "~> 5.0"

  # Authentication & Authorization
  spec.add_dependency "devise", "~> 4.8"
  spec.add_dependency "pundit", "~> 2.3"
  spec.add_dependency "jwt", "~> 2.6"
  spec.add_dependency "bcrypt", "~> 3.1"

  # API & Serialization
  spec.add_dependency "jsonapi-serializer", "~> 2.2"
  spec.add_dependency "jbuilder", "~> 2.11"
  spec.add_dependency "graphql", "~> 2.0"
  spec.add_dependency "rest-client", "~> 2.1"

  # Search & Caching
  spec.add_dependency "elasticsearch", "~> 8.5"
  spec.add_dependency "redis", "~> 5.0"
  spec.add_dependency "memcached", "~> 1.8"

  # Image Processing
  spec.add_dependency "image_processing", "~> 1.12"
  spec.add_dependency "ruby-vips", "~> 2.1"
  spec.add_dependency "imagemagick", "~> 0.0"

  # Monitoring & Analytics
  spec.add_dependency "sentry-rails", "~> 5.8"
  spec.add_dependency "newrelic_rpm", "~> 8.0"
  spec.add_dependency "datadog", "~> 0.54"

  # Code Quality & Testing
  spec.add_dependency "rspec-rails", "~> 5.1"
  spec.add_dependency "factory_bot_rails", "~> 6.2"
  spec.add_dependency "faker", "~> 3.1"
  spec.add_dependency "rubocop", "~> 1.48"

  # Development tools
  spec.add_dependency "pry", "~> 0.14"
  spec.add_dependency "guard", "~> 2.18"
  spec.add_dependency "solargraph", "~> 0.48"

  spec.add_development_dependency "bundler", "~> 2.0"
  spec.add_development_dependency "rake", "~> 13.0"
  spec.add_development_dependency "rspec", "~> 3.12"
end
