package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestManagementResourceAndRegistration(t *testing.T) {
	state := newGuardState()
	response := dispatchManagement(managementRequest{
		Method: http.MethodGet,
		Path:   "/v0/resource/plugins/codex-quota-guard/status",
	}, state)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("resource status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	page := string(response.Body)
	for _, expected := range []string{
		"Codex Quota Guard 2.3.0",
		"Quota policy",
		"Scheduled warm-up",
		"Credentials and last decisions",
		`report.manual_enabled && !report.manual_disabled`,
		`report.manual_enabled ? "Resume policy" : "Reset block"`,
		`const API = "/v0/management/plugins/codex-quota-guard";`,
		`request(API + "/apply-config"`,
		"Configuration saved, applied, and one quota sweep started.",
		`id="language"`,
		`const LANGUAGE_KEY = "codexQuotaGuardLanguage";`,
		`function applyLanguage(next, persist) {`,
		"/v0/management/plugins/codex-quota-guard/config",
	} {
		if !strings.Contains(page, expected) {
			t.Fatalf("resource page is missing %q", expected)
		}
	}
	if strings.Contains(page, "innerHTML") {
		t.Fatal("resource page must not use dynamic innerHTML assignment")
	}
	if strings.Contains(page, "fixture-management-key-3f071e4f") {
		t.Fatal("resource page contains a management key")
	}

	registration := managementRegistrationPayload()
	if len(registration.Routes) != 6 {
		t.Fatalf("management routes = %d, want 6", len(registration.Routes))
	}
	if len(registration.Resources) != 1 || registration.Resources[0].Path != statusResourcePath {
		t.Fatalf("management resources = %#v", registration.Resources)
	}
}

func TestManagementApplyConfigRouteUpdatesRunningStateImmediately(t *testing.T) {
	state := newGuardState()
	response := dispatchManagement(managementRequest{
		Method: http.MethodPost,
		Path:   "/v0/management/plugins/codex-quota-guard/apply-config",
		Body:   []byte(`{"keep_at_least_one":true}`),
	}, state)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("apply-config response = %d, body=%s", response.StatusCode, response.Body)
	}
	if !state.config().keepAtLeastOne() {
		t.Fatal("apply-config did not update the running plugin configuration")
	}
	var applied struct {
		Status   string             `json:"status"`
		Settings managementSettings `json:"settings"`
	}
	if errDecode := json.Unmarshal(response.Body, &applied); errDecode != nil {
		t.Fatalf("decode apply-config response: %v", errDecode)
	}
	if applied.Status != "applied" || !applied.Settings.KeepAtLeastOne {
		t.Fatalf("apply-config payload = %#v", applied)
	}
}

func TestFinishedDecisionUsesCanonicalPersistentReportKey(t *testing.T) {
	persistent := newPersistentState()
	item := persistent.ensure("auth-id", "auth-index", "codex-user.json", "user@example.com", "codex")
	state := &guardState{
		cfg:     defaultConfig(),
		state:   persistent,
		reports: map[string]credentialReport{item.Key: {Ref: "auth-index", Name: item.Name}},
	}
	decision := credentialDecision{
		ref:      "auth-index",
		stateKey: item.Key,
		state:    cloneCredentialState(item),
		report: credentialReport{
			Ref:       "auth-index",
			AuthID:    item.AuthID,
			AuthIndex: item.AuthIndex,
			Name:      item.Name,
			Action:    actionNone,
		},
	}

	state.finishDecision(decision, decision.report)

	if len(state.reports) != 1 {
		t.Fatalf("report count = %d, want one canonical report: %#v", len(state.reports), state.reports)
	}
	if report, ok := state.reports["auth-id"]; !ok || report.Ref != "auth-index" {
		t.Fatalf("canonical report = %#v, present=%t", report, ok)
	}
}

func TestManagementDispatchStatusAndUnknownRoute(t *testing.T) {
	state := newGuardState()
	response := dispatchManagement(managementRequest{
		Method: http.MethodGet,
		Path:   "/v0/management/plugins/codex-quota-guard/status",
	}, state)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status response = %d, body=%s", response.StatusCode, response.Body)
	}
	var status managementStatus
	if errDecode := json.Unmarshal(response.Body, &status); errDecode != nil {
		t.Fatalf("decode status: %v", errDecode)
	}
	if status.PluginVersion != pluginVersion || status.Settings.PollInterval == "" {
		t.Fatalf("status payload = %#v", status)
	}

	response = dispatchManagement(managementRequest{
		Method: http.MethodPost,
		Path:   "/v0/management/plugins/codex-quota-guard/not-found",
	}, state)
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown route status = %d, want %d", response.StatusCode, http.StatusNotFound)
	}
}

func TestManagementSettingsPatchValidation(t *testing.T) {
	patch, errDecode := decodeSettingsPatch([]byte(`{
  "disable_below_remaining_percent": 2,
  "resume_above_remaining_percent": 8,
  "enforce_weekly_window": true,
  "weekly_disable_below_remaining_percent": 3,
  "weekly_resume_above_remaining_percent": 9,
  "poll_interval": "90s",
  "initial_delay": "5s",
  "keep_at_least_one": false,
  "dry_run": true,
  "rate_limit_fallback": "4h",
  "warmup_enabled": true,
  "warmup_model": "test-model",
  "warmup_prompt": "test prompt",
  "warmup_timezone": "UTC",
  "warmup_schedules": [{"id":"morning","at":"09:30","enabled":false}]
}`))
	if errDecode != nil {
		t.Fatalf("decode valid patch: %v", errDecode)
	}
	cfg, errApply := applyManagementSettingsPatch(defaultConfig(), patch)
	if errApply != nil {
		t.Fatalf("apply valid patch: %v", errApply)
	}
	if !cfg.enforceWeekly() || cfg.keepAtLeastOne() || !cfg.dryRun() || cfg.pollInterval() != 90*time.Second {
		t.Fatalf("applied config = %#v", cfg)
	}
	if len(cfg.WarmupSchedules) != 1 || cfg.WarmupSchedules[0].At != "09:30" || cfg.WarmupSchedules[0].enabled() {
		t.Fatalf("warmup schedules = %#v", cfg.WarmupSchedules)
	}

	if _, errUnknown := decodeSettingsPatch([]byte(`{"state_file":"/tmp/escape.json"}`)); errUnknown == nil ||
		!strings.Contains(errUnknown.Error(), "unknown field") {
		t.Fatalf("unknown field error = %v", errUnknown)
	}
	if _, errTrailing := decodeSettingsPatch([]byte(`{"dry_run":true} {"dry_run":false}`)); errTrailing == nil {
		t.Fatal("trailing JSON value must be rejected")
	}
}

func TestManagementSettingsHysteresisValidation(t *testing.T) {
	disable := 5.0
	resume := 5.0
	_, errApply := applyManagementSettingsPatch(defaultConfig(), managementSettingsPatch{
		DisableBelowRemainingPercent: &disable,
		ResumeAboveRemainingPercent:  &resume,
	})
	if errApply == nil || !strings.Contains(errApply.Error(), "must be greater") {
		t.Fatalf("5h hysteresis error = %v", errApply)
	}

	weeklyDisable := 10.0
	weeklyResume := 2.0
	weeklyEnabled := true
	_, errApply = applyManagementSettingsPatch(defaultConfig(), managementSettingsPatch{
		EnforceWeeklyWindow:       &weeklyEnabled,
		WeeklyDisableBelowPercent: &weeklyDisable,
		WeeklyResumeAbovePercent:  &weeklyResume,
	})
	if errApply == nil || !strings.Contains(errApply.Error(), "weekly_resume_above_remaining_percent") {
		t.Fatalf("7d hysteresis error = %v", errApply)
	}
}

func TestManagementActionConcurrencyGuards(t *testing.T) {
	state := &guardState{
		cfg:         defaultConfig(),
		reports:     make(map[string]credentialReport),
		state:       newPersistentState(),
		actionCtx:   context.Background(),
		actionsOpen: true,
	}
	state.sweepMu.Lock()
	if errAction := state.startSweepAction(); !errors.Is(errAction, errSweepInProgress) {
		state.sweepMu.Unlock()
		t.Fatalf("parallel sweep error = %v", errAction)
	}
	state.sweepMu.Unlock()

	state.warmupMu.Lock()
	if errAction := state.startWarmupAction(); !errors.Is(errAction, errWarmupInProgress) {
		state.warmupMu.Unlock()
		t.Fatalf("parallel warm-up error = %v", errAction)
	}
	state.warmupMu.Unlock()

	stopped := &guardState{cfg: defaultConfig(), reports: make(map[string]credentialReport), state: newPersistentState()}
	if errAction := stopped.startSweepAction(); !errors.Is(errAction, errGuardNotRunning) {
		t.Fatalf("stopped sweep error = %v", errAction)
	}
	warmupDisabled := defaultConfig()
	disabled := false
	warmupDisabled.WarmupEnabled = &disabled
	stopped.cfg = warmupDisabled
	if errAction := stopped.startWarmupAction(); !errors.Is(errAction, errWarmupDisabled) {
		t.Fatalf("disabled warm-up error = %v", errAction)
	}
}

func TestResetTargetLookupAndManualDisableProtection(t *testing.T) {
	now := time.Now().UTC()
	one := &credentialState{
		Key:              "one",
		Name:             "same-name.json",
		Provider:         "codex",
		ManualDisabled:   true,
		DisabledByPlugin: true,
		QuotaBlock: &managedBlock{
			Kind:      blockKindQuota5h,
			Status:    blockStatusActive,
			BlockedAt: now.Add(-time.Minute),
			ResetAt:   now.Add(time.Hour),
		},
	}
	two := &credentialState{Key: "two", Name: "same-name.json", Provider: "codex"}
	persistent := newPersistentState()
	persistent.Credentials[one.Key] = one
	persistent.Credentials[two.Key] = two
	reports := map[string]credentialReport{
		"one": {Ref: "one", Name: one.Name, Managed: true},
		"two": {Ref: "two", Name: two.Name},
	}
	if _, errTarget := findResetTarget("same-name.json", reports, persistent); !errors.Is(errTarget, errCredentialAmbiguous) {
		t.Fatalf("ambiguous target error = %v", errTarget)
	}
	target, errTarget := findResetTarget("one", reports, persistent)
	if errTarget != nil || target.stateKey != "one" {
		t.Fatalf("exact target = %#v, error=%v", target, errTarget)
	}

	state := &guardState{
		cfg:         defaultConfig(),
		reports:     reports,
		state:       persistent,
		actionCtx:   context.Background(),
		actionsOpen: true,
	}
	if _, errReset := state.resetManagedBlocks(context.Background(), "one"); !errors.Is(errReset, errCredentialManual) {
		t.Fatalf("manual disable reset error = %v", errReset)
	}
}

func TestResetRequiresPluginOwnedBlock(t *testing.T) {
	item := &credentialState{Key: "available", Name: "available.json", Provider: "codex"}
	persistent := newPersistentState()
	persistent.Credentials[item.Key] = item
	state := &guardState{
		cfg:         defaultConfig(),
		reports:     map[string]credentialReport{"available": {Ref: "available", Name: "available.json", Action: actionNone}},
		state:       persistent,
		actionCtx:   context.Background(),
		actionsOpen: true,
	}
	if _, errReset := state.resetManagedBlocks(context.Background(), "available"); !errors.Is(errReset, errNoManagedBlock) {
		t.Fatalf("unmanaged reset error = %v", errReset)
	}
}
