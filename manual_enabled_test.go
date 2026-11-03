package main

import (
	"testing"
	"time"
)

func TestReconcileCredentialModeTracksManualEnableAndManualDisable(t *testing.T) {
	pluginDisabled := &credentialState{DisabledByPlugin: true}
	manual, clearMarker, reason := reconcileCredentialMode(pluginDisabled, true, false, false)
	if manual || !clearMarker || !pluginDisabled.ManualEnabled || pluginDisabled.DisabledByPlugin {
		t.Fatalf("manual enable transition = manual:%t clear:%t state:%#v reason:%q", manual, clearMarker, pluginDisabled, reason)
	}

	reason = ""
	manual, clearMarker, reason = reconcileCredentialMode(pluginDisabled, true, true, true)
	if !manual || !clearMarker || !pluginDisabled.ManualDisabled || pluginDisabled.ManualEnabled || pluginDisabled.DisabledByPlugin {
		t.Fatalf("manual disable transition = manual:%t clear:%t state:%#v reason:%q", manual, clearMarker, pluginDisabled, reason)
	}

	manuallyDisabled := &credentialState{ManualDisabled: true}
	manual, clearMarker, reason = reconcileCredentialMode(manuallyDisabled, false, false, false)
	if manual || clearMarker || !manuallyDisabled.ManualEnabled || manuallyDisabled.ManualDisabled {
		t.Fatalf("user re-enable transition = manual:%t clear:%t state:%#v reason:%q", manual, clearMarker, manuallyDisabled, reason)
	}
}

func TestManualEnabledOverrideSuppressesQuotaDisable(t *testing.T) {
	now := time.Now().UTC()
	persistent := newPersistentState()
	item := persistent.ensure("auth-1", "index-1", "codex-user.json", "user@example.com", "codex")
	item.ManualEnabled = true
	item.QuotaBlock = &managedBlock{
		Kind:      blockKindQuota5h,
		Window:    "5h",
		Status:    blockStatusActive,
		BlockedAt: now,
		ResetAt:   now.Add(time.Hour),
	}
	guard := &guardState{
		cfg:     defaultConfig(),
		state:   persistent,
		reports: make(map[string]credentialReport),
	}
	decision := credentialDecision{
		ref:           "index-1",
		stateKey:      item.Key,
		state:         cloneCredentialState(item),
		quotaBlock:    item.QuotaBlock,
		quotaActive:   true,
		manualEnabled: true,
		report: credentialReport{
			Ref:       "index-1",
			AuthID:    item.AuthID,
			AuthIndex: item.AuthIndex,
			Name:      item.Name,
			Provider:  item.Provider,
		},
	}

	guard.applyDecision(defaultConfig(), decision, time.Second)

	updated := persistent.lookup(item.AuthID, item.AuthIndex, item.Name)
	if updated == nil || !updated.ManualEnabled || updated.DisabledByPlugin {
		t.Fatalf("credential state after override = %#v", updated)
	}
	report := guard.reports[item.Key]
	if report.Action != actionNone || report.Reason != "manual enable override: quota disable suppressed" {
		t.Fatalf("manual override report = %#v", report)
	}
}
