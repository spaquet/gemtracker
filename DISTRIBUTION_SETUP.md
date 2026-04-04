# Distribution Setup Guide

This guide walks through the manual GitHub setup steps needed to enable automated releases for gemtracker.

## Prerequisites

- You must be the owner of the GitHub repository `spaquet/gemtracker`
- You have push access to create and push tags
- You have admin access to GitHub repository settings

## Step 1: Create the Homebrew Tap Repository

The Homebrew tap is a separate GitHub repository that holds the formula for installing gemtracker via Homebrew.

1. Go to https://github.com/new
2. Create a new repository with these settings:
   - **Repository name**: `homebrew-gemtracker`
   - **Description**: `Homebrew formula for gemtracker (optional)`
   - **Visibility**: Public
   - **Initialize with**: None (leave blank, we'll push content via goreleaser)
3. Click **Create repository**

The tap URL will be: `https://github.com/spaquet/homebrew-gemtracker`

**Installation command for users**: `brew tap spaquet/gemtracker && brew install gemtracker`
(Homebrew automatically strips the `homebrew-` prefix)

## Step 2: Create a GitHub Personal Access Token (PAT)

This token allows the release workflow to push the updated Homebrew formula automatically.

1. Go to https://github.com/settings/tokens
2. Click **Generate new token** → **Generate new token (classic)**
3. Fill in the form:
   - **Token name**: `HOMEBREW_TAP_TOKEN` (or similar)
   - **Expiration**: No expiration (or set yearly)
   - **Scopes**: Select **repo** (Full control of private repositories)
     - This gives access to push to the `homebrew-gemtracker` repo
4. Click **Generate token**
5. **Copy the token immediately** — you won't be able to see it again

> ⚠️ **Important**: Keep this token secret. Treat it like a password. Never commit it to Git.

## Step 3: Add the Secret to GitHub

1. Go to https://github.com/spaquet/gemtracker/settings/secrets/actions
2. Click **New repository secret**
3. Fill in:
   - **Name**: `HOMEBREW_TAP_TOKEN`
   - **Secret**: Paste the token from Step 2
4. Click **Add secret**

The release workflow will now be able to use this secret to push formula updates.

## Step 4: Configure Branch Protection

Prevent accidental pushes and ensure CI passes before merging.

1. Go to https://github.com/spaquet/gemtracker/settings/branches
2. Click **Add rule** under "Branch protection rules"
3. Fill in:
   - **Branch name pattern**: `main`
4. Check these options:
   - ✅ Require a pull request before merging
   - ✅ Require status checks to pass before merging
     - Select **ci** (the GitHub Actions workflow)
   - ✅ Require branches to be up to date before merging
   - ✅ Dismiss stale pull request approvals when new commits are pushed
   - ✅ Require code reviews before merging (set to 1)
   - ✅ Require approval of the latest reviewers
   - ✅ Include administrators
5. Click **Create**

## Step 5: Configure Tag Protection (Optional but Recommended)

Protect release tags from accidental deletion.

1. Go to https://github.com/spaquet/gemtracker/settings/tags
2. Click **New rule**
3. Fill in:
   - **Tag name pattern**: `v*`
4. Check:
   - ✅ Require status checks to pass before creation
   - ✅ Include administrators
5. Click **Create**

This prevents accidental deletion or overwriting of release tags.

## Step 6: Enable Dependabot

The `.github/dependabot.yml` file is already in place. Dependabot automatically creates PRs for dependency updates.

1. Go to https://github.com/spaquet/gemtracker/settings/security_analysis
2. Check that **Dependabot alerts** is enabled (should be by default)
3. Enable **Dependabot security updates** if desired
4. The workflow will automatically create PRs with dependency updates

## Step 7: Verify Everything is Ready

Before your first release, verify the setup:

```bash
# 1. Check that all config files are in place
ls -la .github/workflows/
ls -la .goreleaser.yml SECURITY.md

# 2. Run the goreleaser check locally (requires goreleaser installed)
goreleaser check

# 3. Do a local build of all platforms (without publishing)
goreleaser release --snapshot --clean

# 4. Verify the dist/ directory has all 6 binaries
ls -lh dist/
```

## Step 8: Make Your First Release

When you're ready to release version `v1.0.0`:

```bash
# 1. Ensure all changes are committed and pushed
git status
git push origin main

# 2. Create and push the version tag
git tag v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

This triggers the release workflow automatically:
1. Tests run first (CI)
2. goreleaser builds for all 6 platforms
3. Creates a GitHub Release with all binaries + checksums
4. Pushes the Homebrew formula to `spaquet/homebrew-gemtracker`

## Step 9: Verify the Release

After ~2-5 minutes, check:

1. **GitHub Release**: https://github.com/spaquet/gemtracker/releases
   - Should show `v1.0.0` with all 6 archives + checksums

2. **Homebrew tap repo**: https://github.com/spaquet/homebrew-gemtracker
   - Should have a `gemtracker.rb` formula file

3. **Install via Homebrew**:
   ```bash
   brew tap spaquet/gemtracker
   brew install gemtracker
   gemtracker --version
   ```

If any of these steps fail, check the GitHub Actions tab for workflow errors.

## Troubleshooting

### "HOMEBREW_TAP_TOKEN not found" error
- Verify the secret is added in repo settings: https://github.com/spaquet/gemtracker/settings/secrets/actions
- Check the secret name is exactly `HOMEBREW_TAP_TOKEN`

### Workflow doesn't trigger on tag push
- Ensure the tag matches the pattern `v*.*.*` (e.g., `v1.0.0`, not `1.0.0`)
- Verify the tag was pushed: `git push origin v1.0.0` (not just `git tag`)

### Release created but Homebrew formula not updated
- Check the release workflow logs for errors
- Verify the `spaquet/homebrew-gemtracker` repo exists and is accessible with the PAT
- Verify the PAT has the `repo` scope

### "gofmt found formatting issues"
- The CI linter checks code formatting. Before tagging, run:
  ```bash
  gofmt -s -w .
  git add -A && git commit -m "fix: gofmt formatting" && git push
  ```

## What Happens After Each Release

1. **Tests pass**: Automated check that code is not broken
2. **Binaries built**: Compiled for all 6 platform/arch combos
3. **GitHub Release created**: With archives and checksums
4. **Homebrew formula updated**: New version available via Homebrew
5. **Users can install**: `brew install spaquet/gemtracker/gemtracker`

## Next Steps: Homebrew Core (Optional, for future)

Once the project has a few stable releases (~v1.2.0+) and reasonable traction, you can submit to `homebrew/homebrew-core`:

1. Fork https://github.com/Homebrew/homebrew-core
2. Create a new file `Formula/gemtracker.rb` based on the formula goreleaser generates
3. Submit a PR to `homebrew/homebrew-core`
4. Homebrew maintainers review and merge (usually takes 1-2 weeks)
5. Users can then install with just `brew install gemtracker` (no tap needed)

**Why do this?**
- Better discoverability (users don't need to know about the tap)
- No tap management needed
- Official blessing from Homebrew community

**Why not do it now?**
- Homebrew Core has review requirements
- Better to have a few releases first
- Current tap (`brew tap spaquet/gemtracker && brew install gemtracker`) is already user-friendly

For now, the custom tap is sufficient and gives you full control.

---

## Quick Reference: Release Checklist

Before every release:
- [ ] Update version numbers if needed (in docs/comments)
- [ ] Run tests locally: `make test`
- [ ] Check formatting: `gofmt -s -l .`
- [ ] Create git tag: `git tag v1.0.0`
- [ ] Push to GitHub: `git push origin v1.0.0`
- [ ] Wait for GitHub Actions to complete
- [ ] Verify GitHub Release page and Homebrew formula

Done! Users can now install the new version.
