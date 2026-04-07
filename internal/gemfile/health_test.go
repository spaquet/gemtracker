package gemfile

import (
	"testing"
	"time"
)

func TestComputeHealthScore(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		health   *GemHealth
		expected HealthScore
	}{
		{
			name: "healthy - active within 1 year with multiple maintainers",
			health: &GemHealth{
				LastRelease:     now.AddDate(0, -6, 0),
				GitHubPushedAt:  now.AddDate(0, -3, 0),
				MaintainerCount: 2,
				RateLimited:     false,
			},
			expected: HealthHealthy,
		},
		{
			name: "healthy - GitHub push within 1 year with multiple maintainers",
			health: &GemHealth{
				LastRelease:     time.Time{},
				GitHubPushedAt:  now.AddDate(0, -6, 0),
				MaintainerCount: 2,
				RateLimited:     false,
			},
			expected: HealthHealthy,
		},
		{
			name: "warning - last release 1.5 years ago",
			health: &GemHealth{
				LastRelease:     now.AddDate(-1, -6, 0),
				GitHubPushedAt:  time.Time{},
				MaintainerCount: 2,
				RateLimited:     false,
			},
			expected: HealthWarning,
		},
		{
			name: "warning - single maintainer despite recent activity",
			health: &GemHealth{
				LastRelease:     now.AddDate(0, -1, 0),
				GitHubPushedAt:  now.AddDate(0, -1, 0),
				MaintainerCount: 1,
				RateLimited:     false,
			},
			expected: HealthWarning,
		},
		{
			name: "critical - no activity for 3+ years",
			health: &GemHealth{
				LastRelease:     now.AddDate(-4, 0, 0),
				GitHubPushedAt:  time.Time{},
				MaintainerCount: 2,
				RateLimited:     false,
			},
			expected: HealthCritical,
		},
		{
			name: "critical - archived repository",
			health: &GemHealth{
				LastRelease:     now,
				GitHubPushedAt:  now,
				Archived:        true,
				MaintainerCount: 2,
				RateLimited:     false,
			},
			expected: HealthCritical,
		},
		{
			name: "critical - disabled repository",
			health: &GemHealth{
				LastRelease:     now,
				GitHubPushedAt:  now,
				Disabled:        true,
				MaintainerCount: 2,
				RateLimited:     false,
			},
			expected: HealthCritical,
		},
		{
			name: "unknown - rate limited",
			health: &GemHealth{
				LastRelease:     now,
				GitHubPushedAt:  now,
				MaintainerCount: 2,
				RateLimited:     true,
			},
			expected: HealthUnknown,
		},
		{
			name: "unknown - no activity data",
			health: &GemHealth{
				LastRelease:     time.Time{},
				GitHubPushedAt:  time.Time{},
				MaintainerCount: 2,
				RateLimited:     false,
			},
			expected: HealthUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := ComputeHealthScore(tt.health)
			if score != tt.expected {
				t.Errorf("ComputeHealthScore() = %v, want %v", score, tt.expected)
			}
		})
	}
}

func TestExtractGitHubOwnerRepo(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		wantOwner string
		wantRepo  string
		wantOk    bool
	}{
		{
			name:      "https URL",
			uri:       "https://github.com/rails/rails",
			wantOwner: "rails",
			wantRepo:  "rails",
			wantOk:    true,
		},
		{
			name:      "https URL with .git suffix",
			uri:       "https://github.com/spaquet/gemtracker.git",
			wantOwner: "spaquet",
			wantRepo:  "gemtracker",
			wantOk:    true,
		},
		{
			name:      "http URL",
			uri:       "http://github.com/sinatra/sinatra",
			wantOwner: "sinatra",
			wantRepo:  "sinatra",
			wantOk:    true,
		},
		{
			name:      "git SSH format",
			uri:       "git@github.com:ruby/ruby.git",
			wantOwner: "ruby",
			wantRepo:  "ruby",
			wantOk:    true,
		},
		{
			name:      "non-GitHub URL",
			uri:       "https://gitlab.com/owner/repo",
			wantOwner: "",
			wantRepo:  "",
			wantOk:    false,
		},
		{
			name:      "empty string",
			uri:       "",
			wantOwner: "",
			wantRepo:  "",
			wantOk:    false,
		},
		{
			name:      "malformed GitHub URL",
			uri:       "https://github.com/invalid",
			wantOwner: "",
			wantRepo:  "",
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, ok := ExtractGitHubOwnerRepo(tt.uri)
			if ok != tt.wantOk {
				t.Errorf("ExtractGitHubOwnerRepo() ok = %v, want %v", ok, tt.wantOk)
			}
			if owner != tt.wantOwner {
				t.Errorf("ExtractGitHubOwnerRepo() owner = %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("ExtractGitHubOwnerRepo() repo = %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

func TestHealthScoreString(t *testing.T) {
	tests := []struct {
		score    HealthScore
		expected string
	}{
		{HealthHealthy, "HEALTHY"},
		{HealthWarning, "WARNING"},
		{HealthCritical, "CRITICAL"},
		{HealthUnknown, "UNKNOWN"},
	}

	for _, tt := range tests {
		result := tt.score.String()
		if result != tt.expected {
			t.Errorf("HealthScore.String() = %q, want %q", result, tt.expected)
		}
	}
}
