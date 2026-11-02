package main

import (
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

func TestWarmupEligibilityPersistsDailyDedup(t *testing.T) {
	now := time.Now().UTC()
	auth := pluginapi.HostAuthFileEntry{ID: "auth-1", Provider: "codex"}
	if eligible, reason := warmupEligible(auth, nil, now, "schedule:daily:2026-11-02"); !eligible {
		t.Fatalf("new credential must be eligible: %s", reason)
	}

	state := &credentialState{Warmup: warmupCredentialState{LastRunKey: "schedule:daily:2026-11-02"}}
	if eligible, _ := warmupEligible(auth, state, now, "schedule:daily:2026-11-02"); eligible {
		t.Fatal("same schedule run must be skipped after restart")
	}
	if eligible, reason := warmupEligible(auth, state, now, "schedule:daily:2026-11-03"); !eligible {
		t.Fatalf("next day must be eligible: %s", reason)
	}
}

func TestWarmupEligibilityRejectsManualQuotaAndRateBlocks(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name  string
		state *credentialState
		auth  pluginapi.HostAuthFileEntry
	}{
		{name: "manual", state: &credentialState{ManualDisabled: true}},
		{name: "quota", state: &credentialState{QuotaBlock: &managedBlock{Kind: blockKindQuota5h, Status: blockStatusActive, BlockedAt: now, ResetAt: now.Add(time.Hour)}}},
		{name: "rate", state: &credentialState{RateLimitBlock: &managedBlock{Kind: blockKindRateLimit, Status: blockStatusActive, BlockedAt: now, ResetAt: now.Add(time.Hour)}}},
		{name: "disabled", state: &credentialState{}, auth: pluginapi.HostAuthFileEntry{ID: "auth-1", Provider: "codex", Disabled: true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if eligible, _ := warmupEligible(test.auth, test.state, now, "daily"); eligible {
				t.Fatal("credential must not be eligible")
			}
		})
	}
}
