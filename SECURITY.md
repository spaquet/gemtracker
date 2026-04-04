# Security Policy

## Supported Versions

gemtracker follows semantic versioning. Security updates are provided for the latest stable release.

| Version | Supported |
|---------|-----------|
| Latest  | ✅ Yes    |
| Previous major.minor | ⚠️ Case-by-case |
| Older   | ❌ No     |

## Reporting a Vulnerability

If you discover a security vulnerability in gemtracker, please do **not** open a public GitHub issue. Instead, report it privately using one of these methods:

### Option 1: GitHub Security Advisory (Recommended)
1. Go to https://github.com/spaquet/gemtracker
2. Click **Security** → **Report a vulnerability**
3. Describe the vulnerability in detail

### Option 2: Email
Send an email to the maintainer with:
- Description of the vulnerability
- Steps to reproduce (if applicable)
- Potential impact assessment
- Suggested fix (if you have one)

## Security Response Timeline

- **Report received**: We will acknowledge receipt within 48 hours
- **Validation**: We will attempt to reproduce and validate within 1 week
- **Fix development**: We will create a fix and release a patch version
- **Disclosure**: Public disclosure happens after a patch is released (responsible disclosure)

## What We Do

- Promptly investigate all reported vulnerabilities
- Create a patch and release it as soon as possible
- Keep the reporter informed of progress
- Credit the reporter (unless they prefer anonymity) in the release notes

## Scope

This policy applies to:
- The gemtracker CLI tool itself
- Dependencies in `go.mod`

This policy does **not** cover:
- Vulnerabilities in user's Ruby project dependencies (gemtracker simply reports what's in their Gemfile.lock)
- Issues with terminal/system compatibility
- Feature requests or enhancement suggestions

## Security Best Practices for Users

- Keep gemtracker updated to the latest version
- Use `gemtracker` on a system with a recent Go runtime
- Be cautious when analyzing untrusted Gemfile.lock files
- Review detected CVEs carefully in your project context

## Contact

For security questions or to report a vulnerability, contact the maintainers privately via GitHub Security Advisory.
