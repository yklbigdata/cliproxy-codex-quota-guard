package main

import (
	"net/http"
	"testing"
	"time"
)

func TestQuotaBlockHysteresis(t *testing.T) {
	cfg := defaultConfig()
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	resetAt := now.Add(2 * time.Hour)

	block, reason := updateQuotaBlock(cfg, quotaAt(1, resetAt), nil, now)
	if block == nil || !block.active(now) {
		t.Fatalf("1%% remaining did not create an active block: block=%#v reason=%q", block, reason)
	}

	stillBlocked, _ := updateQuotaBlock(cfg, quotaAt(3, resetAt), block, now.Add(time.Minute))
	if stillBlocked == nil || !stillBlocked.active(now.Add(time.Minute)) {
		t.Fatalf("3%% remaining must remain blocked until the 5%% hysteresis threshold")
	}

	cleared, _ := updateQuotaBlock(cfg, quotaAt(5, resetAt), stillBlocked, now.Add(2*time.Minute))
	if cleared == nil || cleared.active(now.Add(2*time.Minute)) {
		t.Fatalf("5%% remaining must clear the active block: %#v", cleared)
	}
}

func TestKeepAtLeastOneProtectsOnlyQuotaBlockedCredential(t *testing.T) {
	cfg := defaultConfig()
	keepOne := true
	cfg.KeepAtLeastOne = &keepOne
	now := time.Now().UTC()
	block := &managedBlock{Kind: blockKindQuota5h, Window: "5h", Status: blockStatusActive, BlockedAt: now, ResetAt: now.Add(time.Hour)}
	used := 99.0
	blocked := func(key string) credentialDecision {
		return credentialDecision{
			stateKey:    key,
			report:      credentialReport{Name: key, Provider: "codex", PrimaryUsed: &used},
			quotaBlock:  block,
			quotaActive: true,
			state:       &credentialState{Key: key},
		}
	}
	manual := blocked("manual")
	manual.manual = true
	rateLimited := blocked("rate")
	rateLimited.rateActive = true

	protected := protectedQuotaCredential(cfg, []credentialDecision{manual, rateLimited, blocked("free")}, now)
	if protected != "free" {
		t.Fatalf("protected credential = %q, want free", protected)
	}
}

func TestKeepAtLeastOneDefaultsOffAndCanBeExplicitlyEnabled(t *testing.T) {
	cfg := defaultConfig()
	if cfg.keepAtLeastOne() {
		t.Fatal("keep_at_least_one must default to false")
	}
	enabled := true
	cfg.KeepAtLeastOne = &enabled
	if !cfg.keepAtLeastOne() {
		t.Fatal("explicit keep_at_least_one=true must remain supported")
	}
}

func TestDefaultPolicyBlocksEveryCredentialThatHitsAWindow(t *testing.T) {
	now := time.Now().UTC()
	fiveHour := &managedBlock{Kind: blockKindQuota5h, Window: "5h", Status: blockStatusActive, BlockedAt: now, ResetAt: now.Add(time.Hour)}
	sevenDay := &managedBlock{Kind: blockKindQuota7d, Window: "7d", Status: blockStatusActive, BlockedAt: now, ResetAt: now.Add(24 * time.Hour)}
	used := 99.0
	candidates := []credentialDecision{
		{
			stateKey:    "five-hour",
			report:      credentialReport{Name: "five-hour", Provider: "codex", PrimaryUsed: &used},
			quotaBlock:  fiveHour,
			quotaActive: true,
			state:       &credentialState{Key: "five-hour"},
		},
		{
			stateKey:    "seven-day",
			report:      credentialReport{Name: "seven-day", Provider: "codex", PrimaryUsed: &used},
			quotaBlock:  sevenDay,
			quotaActive: true,
			state:       &credentialState{Key: "seven-day"},
		},
	}
	if protected := protectedQuotaCredential(defaultConfig(), candidates, now); protected != "" {
		t.Fatalf("default policy protected %q; every credential must remain independently enforceable", protected)
	}
	enabled := true
	cfg := defaultConfig()
	cfg.KeepAtLeastOne = &enabled
	if protected := protectedQuotaCredential(cfg, candidates, now); protected == "" {
		t.Fatal("explicit keep-at-least-one policy must still protect one credential")
	}
}

func TestRateLimitHeaders(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	primaryReset := now.Add(2 * time.Hour)
	weeklyReset := now.Add(24 * time.Hour)

	tests := []struct {
		name       string
		headers    http.Header
		wantWindow string
		wantReset  time.Time
	}{
		{
			name: "five-hour window",
			headers: http.Header{
				"X-Codex-Primary-Used-Percent":   {"100"},
				"X-Codex-Primary-Reset-At":       {formatUnix(primaryReset)},
				"X-Codex-Primary-Window-Minutes": {"300"},
			},
			wantWindow: "5h",
			wantReset:  primaryReset,
		},
		{
			name: "weekly window",
			headers: http.Header{
				"X-Codex-Secondary-Used-Percent":   {"100"},
				"X-Codex-Secondary-Reset-At":       {formatUnix(weeklyReset)},
				"X-Codex-Secondary-Window-Minutes": {"10080"},
			},
			wantWindow: "7d",
			wantReset:  weeklyReset,
		},
		{
			name: "both full with only primary reset",
			headers: http.Header{
				"X-Codex-Primary-Used-Percent":   {"100"},
				"X-Codex-Secondary-Used-Percent": {"100"},
				"X-Codex-Primary-Reset-At":       {formatUnix(primaryReset)},
			},
			wantWindow: "5h",
			wantReset:  primaryReset,
		},
		{
			name: "both full chooses later reset",
			headers: http.Header{
				"X-Codex-Primary-Used-Percent":   {"100"},
				"X-Codex-Secondary-Used-Percent": {"100"},
				"X-Codex-Primary-Reset-At":       {formatUnix(primaryReset)},
				"X-Codex-Secondary-Reset-At":     {formatUnix(weeklyReset)},
			},
			wantWindow: "7d",
			wantReset:  weeklyReset,
		},
		{
			name:       "missing headers fallback",
			headers:    http.Header{},
			wantWindow: "5h",
			wantReset:  now.Add(5 * time.Hour),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			block := rateLimitBlockFromHeaders(test.headers, now, 5*time.Hour)
			if block.Window != test.wantWindow || !block.ResetAt.Equal(test.wantReset) {
				t.Fatalf("block window/reset = %s/%s, want %s/%s", block.Window, block.ResetAt, test.wantWindow, test.wantReset)
			}
		})
	}
}

func quotaAt(remaining float64, resetAt time.Time) quotaSnapshot {
	return quotaSnapshot{Primary: quotaWindow{Present: true, RemainingPercent: remaining, UsedPercent: 100 - remaining, ResetAt: resetAt}}
}

func formatUnix(value time.Time) string {
	return value.UTC().Format("2006-01-02T15:04:05Z")
}
