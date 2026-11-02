package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

func (s *guardState) sweep(ctx context.Context, cause string) {
	cfg := s.config()
	hostTimeout := cfg.httpTimeout() + 10*time.Second
	list, errList := hostAuthList(hostTimeout)
	if errList != nil {
		message := fmt.Sprintf("host.auth.list failed: %v", errList)
		s.setSweepError(cause, message)
		hostLog("error", "codex-quota-guard: cannot enumerate credentials", map[string]any{"error": errList.Error()})
		return
	}

	candidates := make([]credentialDecision, 0, len(list.Files))
	for _, entry := range list.Files {
		if ctx.Err() != nil {
			return
		}
		if entry.RuntimeOnly {
			continue
		}
		provider := strings.TrimSpace(entry.Provider)
		if provider == "" {
			provider = strings.TrimSpace(entry.Type)
		}
		if !cfg.providerAllowed(provider) {
			continue
		}
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			continue
		}
		ref := strings.TrimSpace(entry.AuthIndex)
		if ref == "" {
			ref = strings.TrimSpace(entry.ID)
		}
		if ref == "" {
			ref = name
		}
		candidates = append(candidates, s.inspectCredential(ctx, cfg, entry, ref, hostTimeout))
	}
	sort.Slice(candidates, func(i, j int) bool {
		return strings.ToLower(candidates[i].report.Name) < strings.ToLower(candidates[j].report.Name)
	})

	protectedKey := protectedQuotaCredential(cfg, candidates, time.Now().UTC())
	for index := range candidates {
		decision := &candidates[index]
		if decision.stateKey == protectedKey {
			decision.protected = true
			if decision.quotaBlock != nil {
				cleared := *decision.quotaBlock
				cleared.Status = blockStatusCleared
				decision.quotaBlock = &cleared
			}
			decision.quotaActive = false
		}
		s.applyDecision(cfg, *decision, hostTimeout)
	}

	s.mu.Lock()
	s.lastSweep = time.Now().UTC()
	s.lastSweepError = ""
	s.lastSweepCause = cause
	s.sweeps++
	s.state.LastSweepAt = s.lastSweep
	s.state.LastSweepError = ""
	s.state.LastSweepCause = cause
	s.mu.Unlock()
	if errPersist := s.persistState("sweep"); errPersist != nil {
		hostLog("warn", "codex-quota-guard: sweep state was not persisted", map[string]any{"error": errPersist.Error()})
	}
}

func (s *guardState) stateForEntry(entry pluginAuthEntry) *credentialState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item := s.state.lookup(entry.ID, entry.AuthIndex, entry.Name)
	if item != nil {
		return cloneCredentialState(item)
	}
	key := credentialStateKey(entry.ID, entry.AuthIndex, entry.Name)
	if key == "" {
		return nil
	}
	return &credentialState{
		Key:       key,
		AuthID:    strings.TrimSpace(entry.ID),
		AuthIndex: strings.TrimSpace(entry.AuthIndex),
		Name:      strings.TrimSpace(entry.Name),
		Email:     strings.TrimSpace(entry.Email),
		Provider:  firstNonEmpty(entry.Provider, entry.Type),
	}
}

func (s *guardState) inspectCredential(ctx context.Context, cfg pluginConfig, entry pluginAuthEntry, ref string, hostTimeout time.Duration) credentialDecision {
	decision := credentialDecision{
		entry: entry,
		ref:   ref,
		state: s.stateForEntry(entry),
		report: credentialReport{
			Ref:       ref,
			AuthID:    strings.TrimSpace(entry.ID),
			AuthIndex: strings.TrimSpace(entry.AuthIndex),
			Name:      strings.TrimSpace(entry.Name),
			Email:     strings.TrimSpace(entry.Email),
			Provider:  firstNonEmpty(entry.Provider, entry.Type),
			Action:    actionNone,
		},
	}
	if decision.state == nil {
		decision.report.Action = actionError
		decision.report.Error = "credential has no stable identity"
		decision.report.Reason = "cannot identify credential"
		s.bumpError()
		return decision
	}
	decision.stateKey = decision.state.Key

	record, errGet := hostAuthGet(ref, hostTimeout)
	if errGet != nil {
		decision.report.Action = actionError
		decision.report.Error = errGet.Error()
		decision.report.Reason = "cannot read credential"
		s.bumpError()
		return decision
	}
	document, errParse := parseAuthDocument(record.JSON)
	if errParse != nil {
		decision.report.Action = actionError
		decision.report.Error = errParse.Error()
		decision.report.Reason = "cannot parse credential"
		s.bumpError()
		return decision
	}
	decision.document = document
	decision.token = document.stringValue("access_token")
	decision.account = firstNonEmpty(document.stringValue("account_id"), document.stringValue("chatgpt_account_id"))

	marker := readGuardMarker(document)
	fileDisabled := document.boolValue("disabled")
	runtimeDisabled := entry.Disabled || strings.EqualFold(strings.TrimSpace(entry.Status), "disabled")
	if runtime, errRuntime := hostAuthGetRuntime(ref, hostTimeout); errRuntime == nil {
		runtimeDisabled = runtime.Auth.Disabled || strings.EqualFold(strings.TrimSpace(runtime.Auth.Status), "disabled")
	}
	decision.fileDisabled = fileDisabled
	decision.runtimeDisabled = runtimeDisabled
	decision.managed = markerHasBlocks(marker) && fileDisabled
	decision.report.FileDisabled = fileDisabled
	decision.report.RuntimeDisabled = runtimeDisabled
	decision.report.Managed = decision.managed

	now := time.Now().UTC()
	decision.quotaBlock = cloneCredentialState(decision.state).QuotaBlock
	decision.rateBlock = cloneCredentialState(decision.state).RateLimitBlock
	if markerHasBlocks(marker) && fileDisabled {
		decision.state.DisabledByPlugin = true
		applyMarkerBlocks(&decision, marker, now)
	}
	decision.state.refreshExpired(now)
	decision.rateActive = decision.rateBlock.active(now)
	decision.quotaActive = decision.quotaBlock.active(now)

	if !fileDisabled && !runtimeDisabled {
		decision.state.ManualDisabled = false
	}
	if fileDisabled && !markerHasBlocks(marker) {
		decision.manual = true
		decision.state.ManualDisabled = true
		decision.state.DisabledByPlugin = false
		decision.report.ManualDisabled = true
		decision.report.Action = actionSkipped
		decision.report.Reason = "disabled outside this plugin; manual disable has priority"
		decision.fillBlockReport(now)
		return decision
	}

	if decision.state.ManualDisabled {
		decision.manual = true
		decision.report.ManualDisabled = true
		decision.report.Action = actionSkipped
		decision.report.Reason = "credential is manually disabled"
		decision.fillBlockReport(now)
		return decision
	}
	if runtimeDisabled && !fileDisabled && !markerHasBlocks(marker) {
		decision.report.Action = actionSkipped
		decision.report.Reason = "runtime disabled outside this plugin"
		decision.fillBlockReport(now)
		return decision
	}
	if strings.TrimSpace(decision.token) == "" {
		decision.report.Action = actionSkipped
		decision.report.Reason = "credential has no access token"
		decision.fillBlockReport(now)
		return decision
	}

	quota, errQuota := fetchQuota(ctx, cfg, decision.token, decision.account)
	if errQuota != nil {
		decision.report.Action = actionError
		decision.report.Error = errQuota.Error()
		decision.report.Reason = "cannot read Codex quota; existing blocks are preserved"
		decision.fillBlockReport(now)
		s.bumpError()
		return decision
	}
	decision.quota = quota
	decision.report.PlanType = quota.PlanType
	decision.state.ObservedQuota = observedQuotaState{
		PlanType:         quota.PlanType,
		PrimaryRemaining: percentPointer(quota.Primary),
		PrimaryResetAt:   quota.Primary.ResetAt,
		WeeklyRemaining:  percentPointer(quota.Weekly),
		WeeklyResetAt:    quota.Weekly.ResetAt,
		ObservedAt:       quota.FetchedAt,
	}
	if quota.Primary.Present {
		used := quota.Primary.UsedPercent
		decision.report.PrimaryUsed = &used
		decision.report.PrimaryResetIn = quota.Primary.ResetAfter
	}
	if quota.Weekly.Present {
		used := quota.Weekly.UsedPercent
		decision.report.WeeklyUsed = &used
		decision.report.WeeklyResetIn = quota.Weekly.ResetAfter
	}
	decision.quotaBlock, decision.report.Reason = updateQuotaBlock(cfg, quota, decision.quotaBlock, now)
	decision.quotaActive = decision.quotaBlock.active(now)
	if decision.rateActive {
		decision.report.Reason = "429 rate limit active: " + decision.rateBlock.Reason
	}
	decision.report.QuotaBlocked = decision.quotaActive
	decision.report.RateLimitBlocked = decision.rateActive
	decision.report.RateLimitResetAt = blockResetAt(decision.rateBlock)
	decision.report.RateLimitWindow = blockWindow(decision.rateBlock)
	return decision
}

func applyMarkerBlocks(decision *credentialDecision, marker *guardMarker, now time.Time) {
	if marker == nil || !marker.Managed {
		return
	}
	for _, reason := range marker.Reasons {
		if reason.Status == "" {
			reason.Status = blockStatusActive
		}
		if !reason.active(now) {
			continue
		}
		if reason.Kind == blockKindRateLimit {
			decision.rateBlock = mergeManagedBlock(decision.rateBlock, reason, now)
			continue
		}
		decision.quotaBlock = mergeManagedBlock(decision.quotaBlock, reason, now)
	}
	if len(marker.Reasons) != 0 || strings.TrimSpace(marker.DisabledAt) == "" {
		return
	}
	resetAt, _ := time.Parse(time.RFC3339, strings.TrimSpace(marker.WindowResetAt))
	blockedAt, _ := time.Parse(time.RFC3339, strings.TrimSpace(marker.DisabledAt))
	if blockedAt.IsZero() {
		blockedAt = now
	}
	legacy := managedBlock{
		Kind:      blockKindQuota5h,
		Reason:    strings.TrimSpace(marker.Reason),
		Window:    "5h",
		Status:    blockStatusActive,
		BlockedAt: blockedAt,
		ResetAt:   resetAt,
	}
	decision.quotaBlock = mergeManagedBlock(decision.quotaBlock, legacy, now)
}

func (d *credentialDecision) fillBlockReport(now time.Time) {
	d.report.QuotaBlocked = d.quotaBlock.active(now)
	d.report.RateLimitBlocked = d.rateBlock.active(now)
	d.report.RateLimitResetAt = blockResetAt(d.rateBlock)
	d.report.RateLimitWindow = blockWindow(d.rateBlock)
	if d.state != nil {
		d.report.WarmupLastRunAt = d.state.Warmup.LastRunAt
		d.report.WarmupLastSuccess = d.state.Warmup.LastSuccess
		d.report.WarmupLastStatus = d.state.Warmup.LastStatus
		d.report.WarmupLastError = d.state.Warmup.LastError
	}
}

func (s *guardState) applyDecision(cfg pluginConfig, decision credentialDecision, hostTimeout time.Duration) {
	report := decision.report
	report.CheckedAt = time.Now().UTC()
	decision.fillBlockReport(time.Now().UTC())
	if decision.manual || report.Action == actionSkipped || report.Action == actionError {
		s.finishDecision(decision, report)
		return
	}

	desiredDisabled := decision.rateActive || decision.quotaActive
	if decision.protected && desiredDisabled {
		desiredDisabled = false
		report.Reason = "protected: keeping the last Codex credential available"
	}
	now := time.Now().UTC()
	if desiredDisabled {
		report.Managed = decision.fileDisabled && decision.managed
		if !decision.fileDisabled {
			report.Action = actionDisabled
			if cfg.dryRun() {
				report.Reason = "dry run: " + report.Reason
			} else if errDisable := s.disableCredential(cfg, decision, hostTimeout); errDisable != nil {
				report.Action = actionError
				report.Error = errDisable.Error()
				report.Reason = "disable failed"
				s.bumpError()
				hostLog("error", "codex-quota-guard: disable failed", map[string]any{
					"credential": decision.report.Name,
					"error":      errDisable.Error(),
				})
			} else {
				report.FileDisabled = true
				report.RuntimeDisabled = true
				report.Managed = true
				decision.state.DisabledByPlugin = true
				s.bumpDisable()
				hostLog("warn", "codex-quota-guard: credential disabled", map[string]any{
					"credential": decision.report.Name,
					"reason":     report.Reason,
				})
			}
		} else {
			report.Action = actionNone
			report.Managed = true
			decision.state.DisabledByPlugin = true
			if decision.managed && !cfg.dryRun() {
				current := readGuardMarker(decision.document)
				next := markerFromBlocks(decision.state, now)
				if current != nil && !compareBlocks(current.Reasons, next.Reasons) {
					if errMarker := s.writeCredentialState(cfg, decision, true, hostTimeout); errMarker != nil {
						report.Action = actionError
						report.Error = errMarker.Error()
						report.Reason = "update guard marker failed"
						s.bumpError()
					}
				}
			}
		}
	} else if decision.fileDisabled && decision.managed {
		report.Action = actionEnabled
		report.Reason = firstNonEmpty(report.Reason, "block expired")
		if cfg.dryRun() {
			report.Reason = "dry run: " + report.Reason
		} else if errEnable := s.enableCredential(cfg, decision, hostTimeout); errEnable != nil {
			report.Action = actionError
			report.Error = errEnable.Error()
			report.Reason = "enable failed"
			s.bumpError()
			hostLog("error", "codex-quota-guard: enable failed", map[string]any{
				"credential": decision.report.Name,
				"error":      errEnable.Error(),
			})
		} else {
			report.FileDisabled = false
			report.RuntimeDisabled = false
			report.Managed = false
			decision.state.DisabledByPlugin = false
			s.bumpEnable()
			hostLog("info", "codex-quota-guard: credential re-enabled", map[string]any{
				"credential": decision.report.Name,
				"reason":     report.Reason,
			})
		}
	} else {
		report.Action = actionNone
		report.Managed = false
		decision.state.DisabledByPlugin = false
		if strings.TrimSpace(report.Reason) == "" {
			report.Reason = "within policy"
		}
	}
	s.finishDecision(decision, report)
}

func (s *guardState) disableCredential(cfg pluginConfig, decision credentialDecision, timeout time.Duration) error {
	return s.writeCredentialState(cfg, decision, true, timeout)
}

func (s *guardState) enableCredential(cfg pluginConfig, decision credentialDecision, timeout time.Duration) error {
	return s.writeCredentialState(cfg, decision, false, timeout)
}

func (s *guardState) writeCredentialState(cfg pluginConfig, decision credentialDecision, disabled bool, timeout time.Duration) error {
	if decision.document == nil {
		return fmt.Errorf("credential document is unavailable")
	}
	document := decision.document
	document.setBool("disabled", disabled)
	if disabled {
		marker, errMarker := json.Marshal(markerFromBlocks(decision.state, time.Now().UTC()))
		if errMarker != nil {
			return errMarker
		}
		document.setRaw(guardMetadataKey, marker)
	} else {
		document.remove(guardMetadataKey)
	}
	payload, errMarshal := document.marshal()
	if errMarshal != nil {
		return errMarshal
	}
	name := strings.TrimSpace(decision.entry.Name)
	if name == "" || !strings.HasSuffix(strings.ToLower(name), ".json") {
		return fmt.Errorf("credential has no writable .json auth name")
	}
	if errSave := hostAuthSave(name, payload, timeout); errSave != nil {
		return errSave
	}
	applied, observed := s.confirmRuntime(decision.ref, disabled, 3*time.Second)
	if !applied {
		return fmt.Errorf("runtime credential did not reflect disabled=%t; observed disabled=%t", disabled, observed)
	}
	_ = cfg
	return nil
}

func (s *guardState) confirmRuntime(ref string, wantDisabled bool, budget time.Duration) (bool, bool) {
	deadline := time.Now().Add(budget)
	observed := !wantDisabled
	for {
		if runtime, errRuntime := hostAuthGetRuntime(ref, 5*time.Second); errRuntime == nil {
			observed = runtime.Auth.Disabled || strings.EqualFold(strings.TrimSpace(runtime.Auth.Status), "disabled")
			if observed == wantDisabled {
				return true, observed
			}
		}
		if time.Now().After(deadline) {
			return false, observed
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (s *guardState) finishDecision(decision credentialDecision, report credentialReport) {
	now := time.Now().UTC()
	decision.fillBlockReport(now)
	report.QuotaBlocked = decision.quotaBlock.active(now)
	report.RateLimitBlocked = decision.rateBlock.active(now)
	report.RateLimitResetAt = blockResetAt(decision.rateBlock)
	report.RateLimitWindow = blockWindow(decision.rateBlock)

	s.mu.Lock()
	item := s.state.ensure(decision.report.AuthID, decision.report.AuthIndex, decision.report.Name, decision.report.Email, decision.report.Provider)
	if item != nil {
		item.AuthID = firstNonEmpty(item.AuthID, decision.report.AuthID)
		item.AuthIndex = firstNonEmpty(item.AuthIndex, decision.report.AuthIndex)
		item.Name = firstNonEmpty(item.Name, decision.report.Name)
		item.Email = firstNonEmpty(item.Email, decision.report.Email)
		item.Provider = firstNonEmpty(item.Provider, decision.report.Provider)
		item.ManualDisabled = decision.state.ManualDisabled
		item.DisabledByPlugin = decision.state.DisabledByPlugin
		item.QuotaBlock = cloneManagedBlock(decision.quotaBlock)
		item.RateLimitBlock = cloneManagedBlock(decision.rateBlock)
		item.ObservedQuota = decision.state.ObservedQuota
		item.Warmup = decision.state.Warmup
		item.LastAction = report.Action
		item.LastReason = report.Reason
		item.LastActionAt = now
		item.LastError = report.Error
	}
	s.reports[decision.ref] = report
	s.mu.Unlock()
}

func cloneManagedBlock(block *managedBlock) *managedBlock {
	if block == nil {
		return nil
	}
	copy := *block
	return &copy
}

func percentPointer(window quotaWindow) *float64 {
	if !window.Present {
		return nil
	}
	value := window.RemainingPercent
	return &value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
