# Branch Protection Setup for main

This guide walks through enabling strict branch protection on the `main` branch using **GitHub Branch Rulesets** (the modern approach) to prevent accidental force pushes, deletions, and unreviewed merges.

## Why Branch Protection Matters

- **Prevents accidental deletions** — Can't delete main branch
- **Blocks force pushes** — No `git push --force` to main
- **Requires PR reviews** — Code must be reviewed before merging
- **Requires CI/CD pass** — Tests must pass before merge
- **Requires up-to-date branch** — Force merge conflicts to be resolved
- **Protects admins too** — Even maintainers can't bypass rules

## Why Branch Rulesets (Not Classic Protection)?

**Branch Rulesets** are GitHub's newer, recommended approach:
- ✅ Can toggle on/off without deleting
- ✅ Multiple rulesets can layer together
- ✅ More flexible and powerful
- ✅ Better transparency (anyone can see rules)
- ✅ Can enforce commit metadata (future-proof)
- ✅ GitHub's direction for the future

Classic branch protection rules are legacy—rulesets are the way forward.

## Step-by-Step Setup

### 1. Go to Branch Ruleset Settings

1. Navigate to: https://github.com/spaquet/gemtracker
2. Click **Settings** (top right)
3. Click **Rules** in the left sidebar (under "Code and automation")
4. Click **New branch ruleset**

### 2. Configure the Ruleset

#### Section 1: Ruleset Details
- **Name**: `main` (or "Protect main branch")
- **Enforcement status**: `Active` (enabled by default)

#### Section 2: Target Branches
- **Target branches**:
  - Select "Include default branch" or explicitly specify `main`
  - This applies the ruleset to your main branch

#### Section 3: Rules
Check the following options to enable them:

✅ **Require pull request reviews before merging**
   - Number of required reviewers: `1`
   - ✅ Dismiss stale pull request approvals when new commits are pushed
   - ✅ Require review of the most recent reviewable push
   - (Optional) ✅ Require approval of the most recent push

✅ **Require status checks to pass**
   - ✅ Require branches to be up to date before merging
   - Required status checks: Add `test` (from `.github/workflows/ci.yml`)
   - Search for "test" and select it

✅ **Restrict deletions**
   - Prevents deletion of the branch

✅ **Restrict force pushes**
   - Prevents force pushing to the branch

✅ **Require conversation resolution**
   - Requires all conversations to be resolved before merging (optional but good)

✅ **Include administrators**
   - ✅ Enforce all the above rules for administrators too
   - (Recommended: prevents accidental admin mistakes)

#### Section 4: Bypass List (Optional)
- Leave empty for strict enforcement
- You can add specific users/teams if needed for emergencies
- For security, leave empty

### 3. Save the Ruleset

Click **Create** button at the bottom to create the ruleset.

#### Result
The ruleset is now **Active** and protecting your main branch.

**To disable later** (without deleting):
- Just click the toggle switch next to the ruleset name
- Click **Enable/Disable** to turn it on or off anytime

## Verification

After creating the ruleset:

1. Go to **Settings → Rules**
2. You should see your `main` ruleset listed with status **Active** (green)
3. The GitHub repository home page no longer shows the "Your main branch isn't protected" warning

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

## Managing the Ruleset

### Edit the Ruleset
1. Go to **Settings → Rules**
2. Click the ruleset name (`main`)
3. Make changes and click **Update**

### Disable the Ruleset Temporarily
With rulesets, you can toggle on/off easily:
1. Go to **Settings → Rules**
2. Click the toggle switch next to the ruleset
3. Select **Disable** (or **Enable** to turn back on)
4. No need to recreate it

### Delete the Ruleset
1. Go to **Settings → Rules**
2. Click the ruleset name
3. Scroll to bottom and click **Delete**

## Troubleshooting

### "I need to merge something urgent without review"
The ruleset prevents this intentionally. Options:
1. Get a quick review (fastest and best practice)
2. Temporarily disable the ruleset (use the toggle switch)
3. Add yourself to the bypass list (emergency only)
4. Re-enable protection after

### "I accidentally pushed to main, can I fix it?"
You can:
1. Open a PR from main to main (then revert the commits)
2. Create a new PR with a revert commit
3. Ask a reviewer to approve the revert

### "I want to change the ruleset settings"
1. Go to **Settings → Rules**
2. Click the ruleset name (`main`)
3. Make changes and click **Update**

### "The ruleset isn't enforcing rules"
1. Check that status is **Active** (not "Disabled")
2. Verify the target branch includes your main branch
3. Verify GitHub Actions workflow exists and runs (for CI checks)

## What This Protects Against

✅ Accidental force pushes wiping out history
✅ Accidental branch deletions
✅ Merging untested code
✅ Merging unreviewed code
✅ Merge conflicts that weren't caught
✅ Out-of-date code being merged

## Additional Rulesets (Optional)

### Protect Release Tags

You can create another ruleset to protect version tags:

1. Go to **Settings → Rules**
2. Click **New branch ruleset**
3. Name: `Protect release tags`
4. Target: Specify pattern `v*` (or `refs/tags/v*` for exact pattern)
5. Enable:
   - ✅ Require status checks to pass (test)
   - ✅ Restrict force pushes
   - ✅ Restrict deletions
   - ✅ Include administrators
6. Click **Create**

This prevents accidental deletion or overwriting of release tags.

### Protect Develop Branch (Optional)

If you have a `develop` or `staging` branch, create similar rulesets:

1. Go to **Settings → Rules**
2. Click **New branch ruleset**
3. Name: `Protect develop`
4. Target: `develop`
5. Same rules as main
6. Click **Create**

---

## Summary

Branch ruleset is now protecting `main`:
- ✅ No force pushes
- ✅ No deletions
- ✅ All code requires 1 review
- ✅ All code requires CI to pass
- ✅ Branch must be up-to-date before merge
- ✅ Rules apply to admins too
- ✅ Can toggle on/off without deleting
- ✅ Multiple rulesets can stack for layered protection

Your repository is now safer! 🔒
