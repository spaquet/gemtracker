# Solo Contributor Guide for gemtracker

As the sole maintainer of gemtracker, you need to know how to work with branch protection rules that require code reviews.

## The Situation

You've enabled branch protection with a rule requiring "1 approving review from reviewers with write access." This is good for future contributors, but you need to be able to merge your own work.

## Solutions

### Option 1: Self-Approve Your Own PRs (Recommended for now)

Since you have admin access and write permissions, GitHub allows you to approve your own PRs after they pass CI checks.

**How to approve your own PR:**

1. Open your PR on GitHub
2. Wait for GitHub Actions to complete (tests, gofmt, vet)
3. Go to **Files changed** tab
4. Scroll to the bottom
5. Click **Review changes**
6. Select **Approve**
7. Click **Submit review**
8. Now you can merge! ✅

**Workflow:**

```bash
# 1. Create a feature branch
git checkout -b feature/my-feature

# 2. Make changes, commit, push
git add .
git commit -m "feat: add new feature"
git push origin feature/my-feature

# 3. Create PR on GitHub (it will show CI status)

# 4. Wait for CI to pass (GitHub Actions)

# 5. Self-approve the PR (via GitHub UI)
# Go to PR → Review changes → Approve

# 6. Click "Merge pull request" button
```

### Option 2: Add Yourself to Bypass List (For Emergencies)

If you want to bypass the review requirement in emergencies:

1. Go to **Settings → Rules**
2. Click your main branch ruleset
3. Scroll to **Bypass list**
4. Click **Add bypass**
5. Select yourself (your GitHub username)
6. Choose **For all pull requests** or **For pull requests authored by**
7. Click **Add**

Now you can merge without review if needed (but still recommended to review your own work).

### Option 3: Remove Review Requirement When Ready (Future)

If you want to remove the review requirement when you have contributors:

1. Go to **Settings → Rules**
2. Click your main branch ruleset
3. Find **Require pull request reviews before merging**
4. Uncheck it
5. Click **Update**

Later, when you have contributors, enable it back.

## Recommended Workflow (Right Now)

Since you're the only contributor but want to maintain good practices for the future:

**Before merging your own PR:**
1. ✅ Ensure GitHub Actions passes (tests, formatting, vet)
2. ✅ Review your own code carefully in the PR
3. ✅ Self-approve the PR in the GitHub UI
4. ✅ Merge

This way:
- You maintain good review discipline
- When contributors join, they'll see the process you use
- You're practicing good code review habits
- The ruleset is already in place and enforced

## When Contributors Join

Once you have contributors:
- They'll create PRs with their code
- You'll review and approve their PRs
- Automated CI checks still run (required for everyone)
- Everyone follows the same process

The review requirement becomes valuable then!

## GitHub Actions CI Checks (Still Required)

These **always** run and **must pass**, regardless of reviews:

```yaml
✅ Tests: go test ./...
✅ Formatting: gofmt -s -l .
✅ Linting: go vet ./...
```

Even if you're the only reviewer, CI must still pass before merging. This is good — it catches bugs early.

## Quick Reference

| Situation | Action |
|-----------|--------|
| **Your PR, CI passes** | Self-approve in GitHub UI, then merge |
| **Your PR, CI fails** | Fix code, push, wait for CI, then approve & merge |
| **Urgent fix needed** | Use bypass list (if configured), or just approve & merge normally |
| **Want strict rules** | Keep current setup; always self-review carefully |
| **Want faster merging** | Remove review requirement temporarily (edit ruleset) |

## Tips

**For clear code reviews:**
- Create descriptive PR titles and descriptions
- Review your own code in the "Files changed" tab before approving
- Make sure commit messages explain the "why"
- Test locally before pushing

**For maintaining discipline:**
- Even though you can approve yourself, take time to review
- Comment on your own code if something is unclear
- This trains good habits for future contributors

**For future contributors:**
- Document the review process clearly (you just did!)
- Be consistent in your own reviews
- Set an example of thorough, helpful reviews

## Troubleshooting

### "I can't approve my own PR"
- Make sure you're logged in as the repo owner
- You need "Write" or "Admin" access to approve
- Wait for CI checks to complete first
- Check the PR review section — it should have an "Approve" button

### "Merge button is greyed out"
- Check that CI passed (green checkmarks on GitHub Actions)
- Check that at least 1 approval exists (your review)
- Check that branch is up-to-date with main

### "I want to disable reviews temporarily"
Go to **Settings → Rules** and toggle off the ruleset temporarily:
1. Click the ruleset name
2. Scroll to top
3. Change **Enforcement status** to **Disabled**
4. Click **Update**
5. Re-enable when ready

---

## Summary

✅ You can approve your own PRs
✅ GitHub Actions still requires all checks to pass
✅ Self-review your code before approving
✅ When contributors join, they follow the same process
✅ This sets good precedent for collaborative development

You're all set! 🚀
