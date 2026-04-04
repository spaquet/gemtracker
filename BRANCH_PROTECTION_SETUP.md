# Branch Protection Setup for main

This guide walks through enabling strict branch protection on the `main` branch to prevent accidental force pushes, deletions, and unreviewed merges.

## Why Branch Protection Matters

- **Prevents accidental deletions** — Can't delete main branch
- **Blocks force pushes** — No `git push --force` to main
- **Requires PR reviews** — Code must be reviewed before merging
- **Requires CI/CD pass** — Tests must pass before merge
- **Requires up-to-date branch** — Force merge conflicts to be resolved
- **Protects admins too** — Even maintainers can't bypass rules

## Step-by-Step Setup

### 1. Go to Branch Protection Settings

1. Navigate to: https://github.com/spaquet/gemtracker
2. Click **Settings** (top right)
3. Click **Branches** (left sidebar)
4. Under "Branch protection rules", click **Add rule**

### 2. Configure the Rule

#### Section 1: Branch Name Pattern
- **Branch name pattern**: `main`

#### Section 2: Protect Matching Branches
Check the following options:

✅ **Require a pull request before merging**
   - Minimum number of reviewers: `1`
   - ✅ Dismiss stale pull request approvals when new commits are pushed
   - ✅ Require review from Code Owners (optional, skip for now)

✅ **Require status checks to pass before merging**
   - ✅ Require branches to be up to date before merging
   - Search for required status checks:
     - `test` (from `.github/workflows/ci.yml`)
   - Make sure you see `test` in the list and select it

✅ **Require conversation resolution before merging**
   - Ensures all discussions/suggestions are resolved

✅ **Require code reviews before merging**
   - Require at least `1` reviewer
   - ✅ Dismiss stale pull request approvals when new commits are pushed
   - ✅ Require review of the most recent reviewable push

✅ **Restrict who can push to matching branches**
   - (Leave empty — allows all users with push access)

✅ **Require signed commits**
   - (Optional) Check this if you want to require GPG signatures

✅ **Require linear history**
   - (Optional) Prevents merge commits, requires rebasing

✅ **Allow force pushes**
   - ⏹️ **Do NOT check this** — we want to prevent force pushes

✅ **Allow deletions**
   - ⏹️ **Do NOT check this** — we want to prevent deletions

✅ **Include administrators**
   - ✅ Enforce all the above rules for administrators too
   - (Recommended: prevents accidental admin mistakes)

### 3. Save the Rule

Click **Create** button at the bottom.

## Verification

After creating the rule:

1. Go back to **Settings → Branches**
2. You should see a rule for `main` with a green checkmark
3. The GitHub home page no longer shows the "Your main branch isn't protected" warning

## How It Affects Workflow

### For Regular Commits
```bash
# Normal push to main is blocked
git push origin main  # ❌ Rejected

# Must use PR instead
git push origin feature-branch
# Then create PR on GitHub and merge after:
# - 1 approval ✓
# - CI passes ✓
# - Branch is up to date ✓
```

### For Force Pushes
```bash
# Force push is now forbidden
git push --force origin main  # ❌ Rejected (even for maintainers)
```

### For Deletions
```bash
# Deletion is now forbidden
git push origin --delete main  # ❌ Rejected
```

### For Releases (No Change)
Tags are not affected by branch protection, so releases still work:
```bash
git tag v1.0.0
git push origin v1.0.0  # ✓ Works fine
```

## Quick Settings Reference

| Setting | Value | Why? |
|---------|-------|------|
| Require PR | Yes, 1 reviewer | Code review quality |
| Require CI pass | Yes (test) | No broken code |
| Require up-to-date | Yes | No merge conflicts |
| Dismiss stale reviews | Yes | New code means re-review |
| Force push allowed | No | Protect history |
| Deletions allowed | No | Prevent accidents |
| Include admins | Yes | Consistency, prevent mistakes |

## Troubleshooting

### "I need to merge something urgent without review"
The rule prevents this intentionally. Options:
1. Get a quick review (fastest)
2. Temporarily disable the rule (not recommended)
3. Use a tag/release if it's for distribution

### "I accidentally pushed to main, can I fix it?"
You can:
1. Open a PR from main to main (then revert the commits)
2. Create a new PR with a revert commit
3. Ask a reviewer to approve the revert

### "I want to change the protection rule"
1. Go to **Settings → Branches**
2. Click the edit icon (pencil) next to the rule
3. Make changes and save

## Disabling Protection (Emergency Only)

If you need to temporarily disable protection:

1. Go to **Settings → Branches**
2. Click the rule for `main`
3. Uncheck all boxes (or delete the rule)
4. Click **Save changes**
5. Re-enable protection ASAP

**⚠️ Not recommended — only for emergencies!**

## What This Protects Against

✅ Accidental force pushes wiping out history
✅ Accidental branch deletions
✅ Merging untested code
✅ Merging unreviewed code
✅ Merge conflicts that weren't caught
✅ Out-of-date code being merged

## Additional Options (Optional)

### Tag Protection

You may also want to protect version tags:

1. Go to **Settings → Tags**
2. Click **Add rule**
3. **Tag name pattern**: `v*`
4. ✅ Require status checks to pass
5. ✅ Include administrators

This prevents accidental deletion or overwriting of release tags.

---

## Summary

Branch protection is now enabled on `main`:
- ✅ No force pushes
- ✅ No deletions
- ✅ All code requires 1 review
- ✅ All code requires CI to pass
- ✅ Branch must be up-to-date before merge
- ✅ Rules apply to admins too

Your repository is now safer! 🔒
