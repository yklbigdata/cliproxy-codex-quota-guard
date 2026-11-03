package main

import (
	"context"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

func TestSchedulerFiltersAndRejectsBlockedCandidates(t *testing.T) {
	now := time.Now().UTC()
	guard := &guardState{cfg: defaultConfig(), state: newPersistentState(), reports: make(map[string]credentialReport)}
	blocked := guard.state.ensure("blocked", "", "", "", "codex")
	blocked.QuotaBlock = &managedBlock{Kind: blockKindQuota5h, Window: "5h", Status: blockStatusActive, BlockedAt: now, ResetAt: now.Add(time.Hour)}
	manual := guard.state.ensure("manual", "", "", "", "codex")
	manual.ManualDisabled = true
	open := guard.state.ensure("open", "", "", "", "codex")
	open.Key = "open"

	request := pluginapi.SchedulerPickRequest{
		Provider: "codex",
		Candidates: []pluginapi.SchedulerAuthCandidate{
			{ID: "blocked", Provider: "codex", Priority: 10},
			{ID: "manual", Provider: "codex", Priority: 20},
			{ID: "open", Provider: "codex", Priority: 1},
		},
	}
	response, err := guard.Pick(context.Background(), request)
	if err != nil {
		t.Fatalf("pick: %v", err)
	}
	if !response.Handled || response.AuthID != "open" {
		t.Fatalf("response = %#v, want open", response)
	}

	response, err = guard.Pick(context.Background(), pluginapi.SchedulerPickRequest{
		Provider: "codex",
		Candidates: []pluginapi.SchedulerAuthCandidate{
			{ID: "blocked", Provider: "codex"},
			{ID: "manual", Provider: "codex"},
		},
	})
	if err != nil {
		t.Fatalf("pick all blocked: %v", err)
	}
	if !response.Handled || !response.Reject || response.RejectCode != "quota_guard_blocked" {
		t.Fatalf("all-blocked response = %#v", response)
	}
}

func TestSchedulerAllowsManualEnabledOverride(t *testing.T) {
	now := time.Now().UTC()
	guard := &guardState{cfg: defaultConfig(), state: newPersistentState(), reports: make(map[string]credentialReport)}
	item := guard.state.ensure("manual-enabled", "", "", "", "codex")
	item.ManualEnabled = true
	item.QuotaBlock = &managedBlock{Kind: blockKindQuota5h, Window: "5h", Status: blockStatusActive, BlockedAt: now, ResetAt: now.Add(time.Hour)}

	response, err := guard.Pick(context.Background(), pluginapi.SchedulerPickRequest{
		Provider:   "codex",
		Candidates: []pluginapi.SchedulerAuthCandidate{{ID: "manual-enabled", Provider: "codex"}},
	})
	if err != nil {
		t.Fatalf("pick: %v", err)
	}
	if !response.Handled || response.Reject || response.DelegateBuiltin != pluginapi.SchedulerBuiltinRoundRobin {
		t.Fatalf("manual-enabled response = %#v", response)
	}
}

func TestSchedulerDelegatesWhenAllCandidatesEligible(t *testing.T) {
	guard := &guardState{cfg: defaultConfig(), state: newPersistentState(), reports: make(map[string]credentialReport)}
	response, err := guard.Pick(context.Background(), pluginapi.SchedulerPickRequest{
		Provider:   "codex",
		Candidates: []pluginapi.SchedulerAuthCandidate{{ID: "one", Provider: "codex"}, {ID: "two", Provider: "codex"}},
	})
	if err != nil {
		t.Fatalf("pick: %v", err)
	}
	if !response.Handled || response.DelegateBuiltin != pluginapi.SchedulerBuiltinRoundRobin {
		t.Fatalf("response = %#v", response)
	}
}
