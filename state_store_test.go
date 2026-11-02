package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStateFileRoundTripAndPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "state.json")
	store := newStateFileStore(path)
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	state := newPersistentState()
	item := state.ensure("auth-1", "index-1", "codex-user.json", "user@example.com", "codex")
	item.ManualDisabled = false
	item.DisabledByPlugin = true
	item.QuotaBlock = &managedBlock{Kind: blockKindQuota5h, Window: "5h", Status: blockStatusActive, BlockedAt: now, ResetAt: now.Add(time.Hour)}
	item.RateLimitBlock = &managedBlock{Kind: blockKindRateLimit, Window: "7d", Status: blockStatusActive, BlockedAt: now, ResetAt: now.Add(24 * time.Hour)}
	item.Warmup = warmupCredentialState{LastRunAt: now, LastRunKey: "daily", LastSuccess: true, LastStatus: 200}
	state.Warmup = warmupSummaryState{LastRunAt: now, LastSource: "schedule:daily", Results: 1, Success: 1}
	state.LastSweepAt = now
	state.LastSweepCause = "interval"

	if err := store.save(state); err != nil {
		t.Fatalf("save state: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat state: %v", err)
	}
	if got := info.Mode().Perm(); got != stateFileMode {
		t.Fatalf("state mode = %04o, want %04o", got, stateFileMode)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	assertNoSecretFields(t, raw)

	loaded, err := store.load()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	got := loaded.lookup("auth-1", "index-1", "codex-user.json")
	if got == nil || got.RateLimitBlock == nil || !got.RateLimitBlock.ResetAt.Equal(item.RateLimitBlock.ResetAt) {
		t.Fatalf("loaded credential did not preserve rate-limit state: %#v", got)
	}
	if loaded.Warmup.Results != 1 || loaded.Warmup.Success != 1 || loaded.LastSweepCause != "interval" {
		t.Fatalf("loaded aggregate state mismatch: %#v", loaded)
	}
}

func TestPersistentStateRebuildsReportsAfterRestart(t *testing.T) {
	now := time.Now().UTC()
	state := newPersistentState()
	item := state.ensure("auth-1", "index-1", "codex-user.json", "", "codex")
	item.QuotaBlock = &managedBlock{Kind: blockKindQuota5h, Window: "5h", Status: blockStatusActive, BlockedAt: now, ResetAt: now.Add(time.Hour)}
	remaining := 0.0
	item.ObservedQuota.PrimaryRemaining = &remaining
	item.LastAction = actionDisabled
	item.LastReason = "quota blocked"

	reports := reportsFromPersistentState(state)
	report, ok := reports["auth-1"]
	if !ok || !report.QuotaBlocked || report.PrimaryUsed == nil || *report.PrimaryUsed != 100 {
		t.Fatalf("restored report = %#v, present=%t", report, ok)
	}
}

func TestPersistentReportsCollapseDuplicateCredentialIdentity(t *testing.T) {
	now := time.Now().UTC()
	state := newPersistentState()
	older := &credentialState{
		Key:          "old-auth-id",
		AuthID:       "old-auth-id",
		Name:         "codex-user.json",
		Provider:     "codex",
		LastAction:   actionNone,
		LastActionAt: now.Add(-time.Hour),
	}
	newer := &credentialState{
		Key:          "new-auth-id",
		AuthID:       "new-auth-id",
		Name:         "codex-user.json",
		Provider:     "codex",
		LastAction:   actionDisabled,
		LastReason:   "quota blocked",
		LastActionAt: now,
	}
	state.Credentials[older.Key] = older
	state.Credentials[newer.Key] = newer

	reports := reportsFromPersistentState(state)

	if len(reports) != 1 {
		t.Fatalf("report count = %d, want one deduplicated credential: %#v", len(reports), reports)
	}
	report, ok := reports[newer.Key]
	if !ok || !report.CheckedAt.Equal(newer.LastActionAt) || report.Action != actionDisabled {
		t.Fatalf("newest report = %#v, present=%t", report, ok)
	}
}

func TestGuardMarkerRead(t *testing.T) {
	document, err := parseAuthDocument([]byte(`{"access_token":"secret","disabled":true}`))
	if err != nil {
		t.Fatalf("parse document: %v", err)
	}
	now := time.Now().UTC()
	state := &credentialState{RateLimitBlock: &managedBlock{Kind: blockKindRateLimit, Window: "5h", Status: blockStatusActive, BlockedAt: now, ResetAt: now.Add(time.Hour)}}
	raw, err := json.Marshal(markerFromBlocks(state, now))
	if err != nil {
		t.Fatalf("marshal marker: %v", err)
	}
	document.setRaw(guardMetadataKey, raw)
	marker := readGuardMarker(document)
	if marker == nil || len(marker.Reasons) != 1 || marker.Reasons[0].Kind != blockKindRateLimit {
		t.Fatalf("marker = %#v", marker)
	}
}

func assertNoSecretFields(t *testing.T, raw []byte) {
	t.Helper()
	text := string(raw)
	for _, forbidden := range []string{"access_token", "refresh_token", "id_token", "management_key", "api_key", "authorization"} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Fatalf("persistent state contains forbidden field %q", forbidden)
		}
	}
}
