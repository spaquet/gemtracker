# Gemtracker Release Guide

This document summarizes the distribution pipeline setup and how to make releases.

## What's Been Set Up

### Automated Release Workflow
- **Trigger**: Push a git tag matching `v*.*.*` (e.g., `v1.0.0`)
- **GitHub Actions**: Automatically runs tests, builds binaries for 6 platforms, creates GitHub Release
- **Homebrew**: Formula auto-updated in `spaquet/homebrew-gemtracker` tap after each release

### Supported Platforms
- macOS (Intel x86-64 + Apple Silicon ARM64)
- Linux (x86-64 + ARM64)
- Windows (x86-64 + ARM64)

### CI/CD
- **Branch protection**: Requires PRs + passing tests on `main`
- **Dependabot**: Auto-creates PRs for dependency updates
- **Test gating**: No release happens if tests fail

---

## Before Your First Release

### 1. Create Homebrew Tap Repository
```bash
# On GitHub.com:
1. Go to https://github.com/new
2. Create repo: spaquet/homebrew-gemtracker (public)
3. Leave empty, goreleaser will populate it

# Users will then install with:
# brew tap spaquet/gemtracker
# brew install gemtracker
# (Homebrew automatically strips the 'homebrew-' prefix)
```

### 2. Create GitHub Personal Access Token (PAT)
```bash
1. Go to https://github.com/settings/tokens
2. Generate new token (classic)
3. Name: HOMEBREW_TAP_TOKEN
4. Scope: repo (full control)
5. Copy the token
```

### 3. Add Secret to Repository
```bash
1. Go to https://github.com/spaquet/gemtracker/settings/secrets/actions
2. New repository secret
3. Name: HOMEBREW_TAP_TOKEN
4. Paste the token from step 2
```

### 4. Configure Branch Protection (Optional but Recommended)
```bash
1. Go to https://github.com/spaquet/gemtracker/settings/branches
2. Add rule for "main"
3. Require PR + require CI pass + require code review
```

See `DISTRIBUTION_SETUP.md` for detailed instructions with screenshots.

---

## Making a Release

### Quick Release (v1.0.0 example)

```bash
# 1. Ensure everything is committed and pushed
git status
git push origin main

# 2. Create and push the version tag
git tag v1.0.0 -m "Release v1.0.0: Describe what's new"
git push origin v1.0.0
```

### What Happens Automatically

1. GitHub Actions detects the `v1.0.0` tag
2. Runs all tests
3. If tests pass:
   - Builds binaries for all 6 platforms
   - Creates GitHub Release with archives + checksums
   - Updates Homebrew formula in `spaquet/homebrew-gemtracker`
4. If tests fail:
   - Release is blocked (you'll see the error in Actions)
   - Fix the issue and re-tag

### Verify the Release (2-5 minutes later)

```bash
# Check GitHub Release page
open https://github.com/spaquet/gemtracker/releases

# Check Homebrew formula was updated
open https://github.com/spaquet/homebrew-gemtracker

# Test Homebrew install (simple command for users!)
brew tap spaquet/gemtracker
brew install gemtracker
gemtracker --version
```

---

## Version Numbering

Use [semantic versioning](https://semver.org/):
- `v1.0.0` - Initial release / major version
- `v1.1.0` - New features (minor bump)
- `v1.1.1` - Bug fixes (patch bump)
- `v2.0.0` - Breaking changes (major bump)

---

## Handling Issues During Release

### "Tests failed" error in GitHub Actions
- Check the workflow logs: https://github.com/spaquet/gemtracker/actions
- Fix the code locally
- Push the fix: `git push origin main`
- Re-tag: `git tag v1.0.0` and `git push origin v1.0.0`

### "HOMEBREW_TAP_TOKEN not found"
- Verify the secret is added: https://github.com/spaquet/gemtracker/settings/secrets/actions
- Check the secret name is exactly `HOMEBREW_TAP_TOKEN`

### Homebrew formula not updating
- Check GitHub Actions logs for errors
- Verify `spaquet/homebrew-gemtracker` repo exists
- Verify the PAT has `repo` scope

### Need to delete/re-release a tag
```bash
# Delete local tag
git tag -d v1.0.0

# Delete remote tag
git push origin --delete v1.0.0

# Re-create and push
git tag v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

---

## Local Testing (Before Release)

If you want to test the build process locally before pushing a tag:

```bash
# Build all platforms using Makefile
make build-release

# Or install goreleaser and test the full pipeline
# (goreleaser release --snapshot --clean)

# Verify binaries work
./dist/gemtracker-darwin-amd64 --version
./dist/gemtracker-linux-amd64 --version
./dist/gemtracker-windows-amd64.exe --version
```

---

## Future: Homebrew Core

Once the project has ~5-10 stable releases and reasonable traction, you can submit to the official Homebrew repository:

1. Fork https://github.com/Homebrew/homebrew-core
2. Create `Formula/gemtracker.rb` (copy from the tap formula)
3. Submit a PR
4. Homebrew maintainers review and merge
5. Users can then just `brew install gemtracker` (no tap needed)

**Current setup** (`brew tap spaquet/gemtracker && brew install gemtracker`) is already user-friendly.
**Future setup** (`brew install gemtracker`) will be even simpler once accepted into Homebrew Core.

---

## Files Reference

| File | Purpose |
|------|---------|
| `.goreleaser.yml` | Main release config (platforms, archives, Homebrew) |
| `.github/workflows/release.yml` | Triggered on tag push |
| `.github/workflows/ci.yml` | Tests on PR/push to main |
| `.github/dependabot.yml` | Auto-dependency updates |
| `SECURITY.md` | Vulnerability disclosure policy |
| `DISTRIBUTION_SETUP.md` | Detailed manual setup guide |
| `RELEASE_GUIDE.md` | This file |

---

## Checklist for First Release

- [ ] `spaquet/homebrew-gemtracker` repo created on GitHub
- [ ] GitHub PAT created with `repo` scope
- [ ] `HOMEBREW_TAP_TOKEN` secret added to `spaquet/gemtracker`
- [ ] Branch protection configured on `main` (optional)
- [ ] All changes pushed: `git push origin main`
- [ ] Version tag created: `git tag v1.0.0 -m "..."`
- [ ] Tag pushed: `git push origin v1.0.0`
- [ ] GitHub Actions completes successfully
- [ ] GitHub Release created: https://github.com/spaquet/gemtracker/releases
- [ ] Homebrew formula updated: https://github.com/spaquet/homebrew-gemtracker
- [ ] Homebrew install works: `brew install spaquet/gemtracker/gemtracker`

---

## Summary

You now have a **production-grade distribution pipeline** for gemtracker:

✅ **Multi-platform support** - macOS, Linux, Windows
✅ **Automated builds** - No manual compilation needed
✅ **Homebrew integration** - Easy installation via `brew install`
✅ **GitHub Releases** - Direct download with checksums
✅ **CI/CD** - Tests gate releases, Dependabot keeps deps updated
✅ **Security** - Branch protection, vulnerability policy, tag protection

When you're ready to release, just push a tag and everything else happens automatically!
