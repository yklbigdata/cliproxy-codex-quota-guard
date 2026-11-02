package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

type resetCredentialTarget struct {
	stateKey  string
	reportKey string
	ref       string
	state     *credentialState
	report    credentialReport
}

func (s *guardState) resetManagedBlocks(ctx context.Context, reference string) (managementCredential, error) {
	if !s.sweepMu.TryLock() {
		return managementCredential{}, errSweepInProgress
	}
	defer s.sweepMu.Unlock()

	s.mu.Lock()
	if !s.actionsOpen || s.actionCtx == nil {
		s.mu.Unlock()
		return managementCredential{}, errGuardNotRunning
	}
	target, errTarget := findResetTarget(reference, s.reports, s.state)
	if errTarget != nil {
		s.mu.Unlock()
		return managementCredential{}, errTarget
	}
	cfg := clonePluginConfig(s.cfg)
	actionCtx := s.actionCtx
	s.workerWG.Add(1)
	s.mu.Unlock()
	defer s.workerWG.Done()

	if ctx == nil {
		ctx = actionCtx
	}
	if errContext := ctx.Err(); errContext != nil {
		return managementCredential{}, errContext
	}
	now := time.Now().UTC()
	hasActiveBlock := target.state.QuotaBlock.active(now) || target.state.RateLimitBlock.active(now)
	if target.state.ManualDisabled {
		return managementCredential{}, errCredentialManual
	}
	if !hasActiveBlock && !target.state.DisabledByPlugin && !target.report.Managed {
		return managementCredential{}, errNoManagedBlock
	}

	hostTimeout := cfg.httpTimeout() + 10*time.Second
	getRef, record, errGet := resetCredentialRecord(target, hostTimeout)
	if errGet != nil {
		return managementCredential{}, fmt.Errorf("read credential: %w", errGet)
	}
	document, errParse := parseAuthDocument(record.JSON)
	if errParse != nil {
		return managementCredential{}, fmt.Errorf("parse credential: %w", errParse)
	}
	marker := readGuardMarker(document)
	fileDisabled := document.boolValue("disabled")
	if fileDisabled && marker == nil {
		return managementCredential{}, errCredentialManual
	}

	nextState := cloneCredentialState(target.state)
	clearManagedBlock(&nextState.QuotaBlock)
	clearManagedBlock(&nextState.RateLimitBlock)
	nextState.DisabledByPlugin = false

	if marker != nil {
		decision := credentialDecision{
			entry: pluginAuthEntry{
				ID:        nextState.AuthID,
				AuthIndex: nextState.AuthIndex,
				Name:      nextState.Name,
				Email:     nextState.Email,
				Provider:  nextState.Provider,
				Disabled:  true,
			},
			document: document,
			ref:      getRef,
			stateKey: nextState.Key,
			state:    nextState,
		}
		if errEnable := s.enableCredential(cfg, decision, hostTimeout); errEnable != nil {
			return managementCredential{}, fmt.Errorf("enable credential: %w", errEnable)
		}
	}

	report := target.report
	report.Ref = firstNonEmpty(report.Ref, target.ref)
	report.Name = firstNonEmpty(report.Name, nextState.Name)
	report.Email = firstNonEmpty(report.Email, nextState.Email)
	report.Provider = firstNonEmpty(report.Provider, nextState.Provider)
	report.FileDisabled = false
	report.RuntimeDisabled = false
	report.Managed = false
	report.ManualDisabled = false
	report.QuotaBlocked = false
	report.RateLimitBlocked = false
	report.RateLimitResetAt = time.Time{}
	report.RateLimitWindow = ""
	report.Error = ""
	report.Reason = "manual plugin block reset"
	report.CheckedAt = now
	if marker != nil {
		report.Action = actionEnabled
		s.bumpEnable()
	} else {
		report.Action = actionNone
	}

	s.mu.Lock()
	item := s.state.ensure(nextState.AuthID, nextState.AuthIndex, nextState.Name, nextState.Email, nextState.Provider)
	if item == nil {
		s.mu.Unlock()
		return managementCredential{}, fmt.Errorf("credential disappeared while resetting blocks")
	}
	item.QuotaBlock = cloneManagedBlock(nextState.QuotaBlock)
	item.RateLimitBlock = cloneManagedBlock(nextState.RateLimitBlock)
	item.DisabledByPlugin = false
	item.ManualDisabled = false
	item.LastAction = report.Action
	item.LastReason = report.Reason
	item.LastActionAt = now
	item.LastError = ""
	reportKey := firstNonEmpty(target.reportKey, report.Ref, item.Key)
	if s.reports == nil {
		s.reports = make(map[string]credentialReport)
	}
	s.reports[reportKey] = report
	s.mu.Unlock()

	if errPersist := s.persistState("manual-block-reset"); errPersist != nil {
		return managementCredential{}, fmt.Errorf("credential state was changed but state persistence failed: %w", errPersist)
	}
	hostLog("info", "codex-quota-guard: plugin blocks cleared", map[string]any{
		"credential": report.Name,
		"enabled":    marker != nil,
	})
	return managementCredentialFromReport(report), nil
}

func findResetTarget(reference string, reports map[string]credentialReport, state persistentState) (resetCredentialTarget, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return resetCredentialTarget{}, errCredentialNotFound
	}
	type match struct {
		score  int
		target resetCredentialTarget
	}
	matches := make([]match, 0, 1)
	for stateKey, item := range state.Credentials {
		if item == nil {
			continue
		}
		reportKey, report := findCredentialReport(stateKey, item, reports)
		score := resetReferenceScore(reference, stateKey, report, item)
		if score == 0 {
			continue
		}
		matches = append(matches, match{
			score: score,
			target: resetCredentialTarget{
				stateKey:  stateKey,
				reportKey: reportKey,
				ref:       firstNonEmpty(report.Ref, item.AuthIndex, item.AuthID, item.Name, stateKey),
				state:     cloneCredentialState(item),
				report:    report,
			},
		})
	}
	if len(matches) == 0 {
		return resetCredentialTarget{}, errCredentialNotFound
	}
	best := matches[0]
	ambiguous := false
	for _, candidate := range matches[1:] {
		if candidate.score > best.score {
			best = candidate
			ambiguous = false
			continue
		}
		if candidate.score == best.score && candidate.target.stateKey != best.target.stateKey {
			ambiguous = true
		}
	}
	if ambiguous {
		return resetCredentialTarget{}, errCredentialAmbiguous
	}
	return best.target, nil
}

func findCredentialReport(stateKey string, item *credentialState, reports map[string]credentialReport) (string, credentialReport) {
	if report, ok := reports[stateKey]; ok {
		return stateKey, report
	}
	for key, report := range reports {
		if strings.EqualFold(report.AuthID, item.AuthID) && strings.TrimSpace(item.AuthID) != "" {
			return key, report
		}
		if strings.EqualFold(report.AuthIndex, item.AuthIndex) && strings.TrimSpace(item.AuthIndex) != "" {
			return key, report
		}
		if strings.EqualFold(report.Name, item.Name) && strings.TrimSpace(item.Name) != "" {
			return key, report
		}
	}
	now := time.Now().UTC()
	return stateKey, credentialReport{
		Ref:              firstNonEmpty(item.AuthIndex, item.AuthID, item.Name, stateKey),
		AuthID:           item.AuthID,
		AuthIndex:        item.AuthIndex,
		Name:             item.Name,
		Email:            item.Email,
		Provider:         item.Provider,
		Managed:          item.DisabledByPlugin,
		ManualDisabled:   item.ManualDisabled,
		QuotaBlocked:     item.QuotaBlock.active(now),
		RateLimitBlocked: item.RateLimitBlock.active(now),
		Action:           item.LastAction,
		Reason:           item.LastReason,
		Error:            item.LastError,
		CheckedAt:        item.LastActionAt,
	}
}

func resetReferenceScore(reference, stateKey string, report credentialReport, item *credentialState) int {
	if reference == stateKey || reference == report.Ref || reference == item.Key {
		return 4
	}
	for _, value := range []string{report.AuthID, report.AuthIndex, report.Name, item.AuthID, item.AuthIndex, item.Name} {
		if strings.TrimSpace(value) != "" && strings.EqualFold(reference, value) {
			return 3
		}
	}
	return 0
}

func resetCredentialRecord(target resetCredentialTarget, timeout time.Duration) (string, pluginapi.HostAuthGetResponse, error) {
	seen := make(map[string]struct{})
	candidates := []string{target.ref, target.state.AuthIndex, target.state.AuthID, target.state.Name}
	var lastErr error
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		record, errGet := hostAuthGet(candidate, timeout)
		if errGet == nil {
			return candidate, record, nil
		}
		lastErr = errGet
	}
	if lastErr == nil {
		lastErr = errCredentialNotFound
	}
	return "", pluginapi.HostAuthGetResponse{}, lastErr
}

func clearManagedBlock(block **managedBlock) {
	if block == nil || *block == nil {
		return
	}
	cleared := **block
	cleared.Status = blockStatusCleared
	*block = &cleared
}

func managementCredentialFromReport(report credentialReport) managementCredential {
	return managementCredential{
		Ref:               report.Ref,
		Name:              report.Name,
		Email:             report.Email,
		Provider:          report.Provider,
		PlanType:          report.PlanType,
		FileDisabled:      report.FileDisabled,
		RuntimeDisabled:   report.RuntimeDisabled,
		Managed:           report.Managed,
		ManualDisabled:    report.ManualDisabled,
		QuotaBlocked:      report.QuotaBlocked,
		RateLimitBlocked:  report.RateLimitBlocked,
		CanResetBlock:     false,
		PrimaryUsed:       cloneFloatPointer(report.PrimaryUsed),
		PrimaryResetIn:    report.PrimaryResetIn,
		WeeklyUsed:        cloneFloatPointer(report.WeeklyUsed),
		WeeklyResetIn:     report.WeeklyResetIn,
		RateLimitWindow:   report.RateLimitWindow,
		RateLimitResetAt:  formatManagementTime(report.RateLimitResetAt),
		WarmupLastRunAt:   formatManagementTime(report.WarmupLastRunAt),
		WarmupLastSuccess: report.WarmupLastSuccess,
		WarmupLastStatus:  report.WarmupLastStatus,
		WarmupLastError:   report.WarmupLastError,
		Action:            report.Action,
		Reason:            report.Reason,
		Error:             report.Error,
		CheckedAt:         formatManagementTime(report.CheckedAt),
	}
}
