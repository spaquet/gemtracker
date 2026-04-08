# CVE Tab Redesign - Architecture & Implementation Plan

## Overview

The CVE tab will be redesigned to:
1. Fetch vulnerability data from OSV.dev (live, no auth required)
2. Cache results per gem signature (hash of all gem names + versions)
3. Auto-refresh when gems change (graceful background refresh with old data displayed)
4. Display cache age, scan status, and severity metrics using consistent styling
5. Use exact gem list from analyzer (no file rescanning)

---

## 1. Vulnerability Data Structure Enhancement

### Current State
```go
type Vulnerability struct {
    GemName         string
    AffectedVersions []string
    Description     string
    CVE             string
}
```

### New Enhanced State
```go
type Vulnerability struct {
    GemName          string
    AffectedVersions []VersionRange  // Structured version ranges
    Description      string
    CVE              string
    CVSS             float64         // 0-10 score
    Severity         string          // "CRITICAL" | "HIGH" | "MEDIUM" | "LOW"
    FixedVersion     string          // First version that fixes it (can be empty)
    PublishedDate    time.Time
    References       []string        // Links to advisories (OSV URLs, etc)
    OSVId            string          // OSV identifier (GHSA-xxxx or CVE-2021-xxxx)
    Source           string          // "osv.dev" for tracking origin
}

type VersionRange struct {
    Min       string // "1.0.0"
    Max       string // "2.0.0"
    Inclusive bool   // whether max is inclusive
}
```

---

## 2. Cache Architecture

### Cache File Location & Naming
```
~/.cache/gemtracker/
├── vulnerabilities/
│   ├── {gems_hash}.json          // Cached results
│   └── {gems_hash}_metadata.json // Metadata (optional, can be in same file)
```

### Cache Entry Structure
```json
{
  "gems_signature": "sha256_hash_of_gems",
  "cached_at": "2026-04-08T14:30:00Z",
  "scanned_at": "2026-04-08T14:30:00Z",
  "ttl_seconds": 3600,
  "next_refresh": "2026-04-08T15:30:00Z",
  "gem_count": 42,
  "scan_status": "success",
  "error_message": null,
  "vulnerabilities": [
    {
      "gem_name": "rails",
      "cve": "CVE-2021-22942",
      "severity": "CRITICAL",
      "cvss": 9.8,
      "affected_versions": [
        { "min": "6.0.0", "max": "6.1.3", "inclusive": true },
        { "min": "7.0.0", "max": "7.0.1", "inclusive": true }
      ],
      "description": "Session fixation vulnerability",
      "published_date": "2021-06-01T00:00:00Z",
      "fixed_version": "6.1.4",
      "references": ["https://osv.dev/..."],
      "osv_id": "GHSA-xxx-xxx-xxx"
    }
  ]
}
```

### Cache Key Generation
```go
// pseudocode
gems := analyzer.GetAllGems()  // From AnalysisResult.AllGems
sortedGems := sortByName(gems)
gemStr := joinGemVersions(sortedGems) // "devise:4.8.0|rails:7.0.0|rack:2.2.0"
gemsHash := sha256(gemStr)
cachePath := "~/.cache/gemtracker/vulnerabilities/{gemsHash}.json"
```

---

## 3. Gem Change Detection & Refresh Flow

### Gem Signature Tracking
```go
type Model struct {
    // ... existing fields ...

    // CVE screen state
    VulnerableGems        []*gemfile.GemStatus
    CVECursor            int
    CVEOffset            int

    // NEW: CVE cache & refresh tracking
    LastGemsSignature    string                // SHA256 of last scanned gems
    CVERefreshInProgress bool                  // Is a refresh happening?
    CVELastScanTime      time.Time             // When was CVE data last scanned?
    CVECacheLoadedAt     time.Time             // When was cache loaded?
    CVECacheTTL          time.Duration         // Default: 1h
}
```

### Refresh Detection Logic
```go
// In model initialization and after AnalysisResult updates
func (m *Model) checkCVERefreshNeeded() bool {
    currentSig := computeGemsSignature(m.AnalysisResult.AllGems)
    return currentSig != m.LastGemsSignature ||
           time.Since(m.CVECacheLoadedAt) > m.CVECacheTTL
}
```

### Refresh Workflow
```
1. On startup or when AnalysisResult updates:
   a. Compute current gems signature
   b. If signature changed or cache expired:
      - Check disk cache (if exists & signature matches & not expired)
      - If cache hit: load it, display old data + "refreshing..." label
      - Trigger async OSV.dev scan in background
   c. If no cache or cache miss: show "Scanning..." while fetching

2. OSV.dev fetch (sequential, batch requests):
   a. Query OSV API with batch of gems
   b. Send BubbleTea Msg on completion/error
   c. Update cache file atomically
   d. Update UI with new data

3. On error/rate limit:
   a. If cache exists (even expired): show cached data + error message
   b. If no cache: show error UI with manual retry option
```

---

## 4. OSV.dev Integration

### OSV.dev API Details
- **Endpoint**: `POST https://api.osv.dev/v1/query`
- **No authentication required**
- **Rate limiting**: ~60 requests/min (generous for single batch)
- **Batch support**: Send one gem per request, but sequentially

### Query Format (per gem)
```json
POST /v1/query
{
  "package": {
    "name": "rails",
    "ecosystem": "RubyGems"
  },
  "version": "7.0.0"
}
```

### Response Format
```json
{
  "vulns": [
    {
      "id": "GHSA-fvqm-6227-5372",
      "summary": "SQL Injection",
      "details": "...",
      "affected": [
        {
          "package": {"name": "rails", "ecosystem": "RubyGems"},
          "ranges": [
            {
              "type": "SEMVER",
              "events": [
                {"introduced": "6.0.0"},
                {"fixed": "6.1.4"}
              ]
            }
          ]
        }
      ],
      "references": [...],
      "published": "2021-06-01T00:00:00Z",
      "modified": "2023-01-01T00:00:00Z",
      "severity": "CRITICAL",
      "cvss": {"score": 9.8}
    }
  ]
}
```

### Batch Processing Strategy

Using OSV.dev Batch API: `POST https://api.osv.dev/v1/query/batch`

```
Input: []Gem from analyzer (all gems, including transitive)

1. Build batch query with all gems at once:
   {
     "queries": [
       {"package": {"name": "rails", "ecosystem": "RubyGems"}, "version": "7.0.0"},
       {"package": {"name": "devise", "ecosystem": "RubyGems"}, "version": "4.8.0"},
       ...
     ]
   }

2. Send single batch request to OSV.dev

3. Parse batch response, extract all vulnerabilities

4. Filter to only gems with vulnerabilities (remove clean gems)

5. Save to cache atomically

6. Send BubbleTea completion Msg with filtered results
```

---

## 5. UI/UX Redesign

### Header Section (Above CVE List)

#### Layout
```
┌─ Vulnerabilities ────────────────────────────────────────┐
│ Severity Summary: ● CRITICAL (1) ● HIGH (3) ● MEDIUM (2) │
│                                                            │
│ Cache Status: 23 minutes old · Expires in 37 minutes      │
│ Last Scan: 2 hours ago · 42 gems scanned                  │
│                                                            │
│ 🔄 Refreshing... (18/42 gems) ← Only shown if refresh    │
└────────────────────────────────────────────────────────────┘
```

#### Color Mapping (matching health dot style)
Using existing styles from `styles.go`:
- `BadgeHealthyDotStyle` (green): LOW or no vulnerabilities
- `BadgeWarningDotStyle` (yellow): MEDIUM severity
- `BadgeCriticalDotStyle` (red): HIGH or CRITICAL severity

Example rendering:
```go
// Severity summary line
severityStr := fmt.Sprintf(
  "%s CRITICAL (%d) %s HIGH (%d) %s MEDIUM (%d)",
  BadgeCriticalDotStyle.Render("●"), critCount,
  BadgeCriticalDotStyle.Render("●"), highCount,
  BadgeWarningDotStyle.Render("●"), medCount,
)
```

### CVE List Items (Enhanced)

**Only vulnerable gems are displayed** (no clean gems in this list)

#### Current Structure (minimal)
```
CVE-2021-22942 (rails 6.1.3)
```

#### New Structure (detailed)
```
CVE-2021-22942 [GHSA-fvqm-6227-5372]
  Gem: rails
  Severity: ● CRITICAL
  Affected: 6.0.0 - 6.1.3 (your version: 7.0.0)
  Description: SQL injection vulnerability in...
  Details: https://osv.dev/...
```

### Status Messages

#### Loading (Initial Scan)
```
⏳ Scanning for vulnerabilities...
  Checking 42 gems against OSV.dev (this may take a moment)
```

#### Refreshing (Background)
```
✓ Showing cached results (refreshing in background)
🔄 Scanning... (18/42 gems)
```

#### Cache Expired
```
⚠ Cache data is 2 hours old (expired 1 hour ago)
Scan in progress... (25/42 gems)
```

#### Error with Fallback
```
⚠ Could not fetch latest data (connection timeout)
Showing cached results from 4 hours ago
[Press R to retry]
```

#### No Data
```
✗ Unable to load vulnerability data
Make sure you have internet access to osv.dev
[Press R to try again]
```

---

## 6. Implementation Files & Changes

### New Files to Create
1. `internal/gemfile/osv.go` - OSV.dev API client
   - `QueryOSVBatch(gems []Gem) ([]Vulnerability, error)`
   - `ParseOSVResponse(resp *http.Response) ([]Vulnerability, error)`

2. `internal/gemfile/vulnerability_cache.go` - Cache management
   - `LoadVulnerabilityCache(gemsHash string) (*CacheEntry, error)`
   - `SaveVulnerabilityCache(gemsHash string, entry *CacheEntry) error`
   - `ComputeGemsSignature(gems []Gem) string`
   - `IsVulnerabilityCacheValid(entry *CacheEntry) bool`

### Files to Modify

#### `internal/gemfile/vulnerabilities.go`
- Keep static list as fallback
- Update `Vulnerability` struct with new fields
- Deprecate old hardcoded checker (or keep for fallback)
- Export vulnerability constants for severity levels

#### `internal/ui/model.go`
- Add CVE cache tracking fields
- Add BubbleTea messages for CVE updates:
  - `CVEScanStartedMsg{}`
  - `CVEProgressMsg{gemIndex, totalGems int}`
  - `CVECompleteMsg{vulns []Vulnerability, err error}`
  - `CVELoadFromCacheMsg{vulns []Vulnerability}`
- Update `Update()` handler for CVE messages

#### `internal/ui/view.go`
- Enhance `renderCVEScreen()` with new header section
- Add cache status display
- Add progress indicator during refresh
- Update CVE list item rendering with severity badges

#### `internal/ui/styles.go`
- Add new styles:
  - `BadgeSeverityHeaderStyle`
  - `CacheStatusStyle`
  - `RefreshingIndicatorStyle`

#### `cmd/gemtracker/main.go`
- Initialize cache directory on startup: `~/.cache/gemtracker/vulnerabilities/`

---

## 7. State Machine for CVE Screen

```
┌─────────────────┐
│  INITIAL        │ Load cache (if exists)
└────────┬────────┘
         │
         ├─→ Cache hit & not expired
         │   └─→ Load data, show + "refreshing..."
         │       └─→ Trigger async refresh
         │
         ├─→ Cache miss or expired
         │   └─→ Show "Scanning..." placeholder
         │       └─→ Trigger OSV fetch
         │
         └─→ No internet/API error
             └─→ Show error, try cache if available

┌──────────────────────┐
│ REFRESHING           │
│ (running in bg)      │
│ Old data displayed   │
│ + "refreshing..." UI │
└──────────┬───────────┘
           │
           ├─→ Success
           │   └─→ Update cache
           │       └─→ Update UI with new data
           │       └─→ Update timestamps
           │
           └─→ Failure
               └─→ Show error in status
               └─→ Keep old data displayed
               └─→ Offer manual retry
```

---

## 8. Rate Limiting & Throttling

### OSV.dev Limits
- No official rate limit published, but ~60 req/min is safe
- Sequential processing (one gem at a time) naturally throttles

### Backoff Strategy
```go
const (
    MaxRetries = 3
    BaseDelay  = 100 * time.Millisecond
    MaxDelay   = 5 * time.Second
)

// Exponential backoff: 100ms → 200ms → 400ms
```

### Request Timeout
```go
const RequestTimeout = 10 * time.Second  // Per gem query
```

---

## 9. Data Flow Summary

```
User opens CVE tab
  ↓
Check if gems signature changed or cache expired
  ↓
  ├─→ YES: Load from disk cache (if exists)
  │   └─→ Display cached vulns + "refreshing..." label
  │   └─→ Start async OSV.dev scan
  │
  └─→ NO: Use in-memory cached data
      └─→ Display vulns

OSV.dev scan (async):
  ├─→ Query each gem sequentially
  ├─→ Merge responses
  ├─→ Parse & map to Vulnerability structs
  ├─→ Save to disk cache (atomic)
  └─→ Send BubbleTea Msg → UI update

UI handles CVE message:
  ├─→ Update data in memory
  ├─→ Remove "refreshing..." label
  ├─→ Trigger re-render with new severity counts
  └─→ Update timestamps
```

---

## 10. Error Handling

### Network/API Errors
- Log to `~/.cache/gemtracker/gemtracker.log` if verbose mode enabled
- Show user-friendly message: "Could not fetch data. Check your internet connection."
- Fall back to cache if available (even if expired)

### Invalid JSON/Parse Errors
- Log full error for debugging
- Treat as cache miss
- Attempt re-fetch on next scan

### Disk I/O Errors
- Log error
- Continue without cache (single-session operation)
- Attempt to save again next refresh

### No Gems Found
- Show: "No gems to scan for vulnerabilities"
- No cache needed

---

## 11. Testing Strategy

### Unit Tests
- `TestComputeGemsSignature()` - Verify deterministic hashing
- `TestIsVulnerabilityCacheValid()` - TTL logic
- `TestParseOSVResponse()` - Response parsing
- `TestVersionRangeMatching()` - Enhanced severity detection

### Integration Tests
- Mock OSV.dev endpoint, verify batch processing
- Test cache read/write with sample data
- Test gem signature changes trigger refresh

### Manual Testing
1. Small project (5 gems) - verify single batch
2. Large project (100+ gems) - verify multi-batch
3. Kill network mid-scan - verify fallback to cache
4. Change Gemfile.lock - verify refresh triggers
5. Same gems, different versions - verify cache invalidation

---

## 12. Future Enhancements

- [ ] Show "Fix available in version X.Y.Z" and highlight upgradable gems
- [ ] Link to detailed OSV advisory pages
- [ ] Persist scan history (trend over time)
- [ ] Alert user if new vulnerabilities introduced in an update
- [ ] Integration with GitHub Security Alerts (requires auth)
- [ ] Custom severity thresholds (ignore LOW, alert on MEDIUM+, etc)
