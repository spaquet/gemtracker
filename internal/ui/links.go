package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/spaquet/gemtracker/internal/gemfile"
)

// gemPageURL returns the gem's homepage/source URL, falling back to its rubygems.org page.
func gemPageURL(gem *gemfile.GemStatus) string {
	if gem.HomepageURL != "" {
		return gem.HomepageURL
	}
	return fmt.Sprintf("https://rubygems.org/gems/%s", gem.Name)
}

// gemReleaseURL returns the gem's release page: GitHub releases when the source
// is on GitHub, otherwise the rubygems.org version history.
func gemReleaseURL(gem *gemfile.GemStatus) string {
	if owner, repo, ok := gemfile.ExtractGitHubOwnerRepo(gem.HomepageURL); ok {
		return fmt.Sprintf("https://github.com/%s/%s/releases", owner, repo)
	}
	return fmt.Sprintf("https://rubygems.org/gems/%s/versions", gem.Name)
}

// openGemListPage opens the page of the gem under the gem-list cursor.
func (m *Model) openGemListPage() tea.Cmd {
	if m.GemListCursor >= len(m.FirstLevelGems) {
		return nil
	}
	return openBrowserCmd(gemPageURL(m.FirstLevelGems[m.GemListCursor]))
}

// openUpgradeableReleasePage opens the release page of the upgradable gem under the cursor.
func (m *Model) openUpgradeableReleasePage() tea.Cmd {
	gems := m.allUpgradeableGems()
	if m.UpgradeableCursor >= len(gems) {
		return nil
	}
	return openBrowserCmd(gemReleaseURL(gems[m.UpgradeableCursor]))
}
