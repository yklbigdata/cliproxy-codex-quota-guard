package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

type scheduledWarmup struct {
	ID   string
	At   string
	Time time.Time
}

type warmupResult struct {
	AuthID      string
	Name        string
	ScheduleKey string
	Model       string
	StatusCode  int
	Success     bool
	Error       string
	ExecutedAt  time.Time
}

func (s *guardState) warmupLoop(ctx context.Context, cfg pluginConfig) {
	for {
		next, ok := nextWarmupRun(cfg, time.Now())
		if !ok {
			select {
			case <-ctx.Done():
				return
			case <-time.After(24 * time.Hour):
				continue
			}
		}
		wait := time.Until(next.Time)
		if wait < 0 {
			continue
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-timer.C:
		}
		scheduleKey := "schedule:" + next.ID + ":" + next.Time.In(mustLocation(cfg.WarmupTimezone)).Format("2006-01-02")
		if _, errRun := s.runWarmup(ctx, cfg, scheduleKey, "schedule:"+next.ID); errRun != nil && ctx.Err() == nil {
			hostLog("error", "codex-quota-guard: scheduled warm-up failed", map[string]any{
				"schedule_id": next.ID,
				"error":       errRun.Error(),
			})
		}
	}
}

func nextWarmupRun(cfg pluginConfig, now time.Time) (scheduledWarmup, bool) {
	if !cfg.warmupEnabled() || len(cfg.WarmupSchedules) == 0 {
		return scheduledWarmup{}, false
	}
	location, errLocation := time.LoadLocation(cfg.WarmupTimezone)
	if errLocation != nil {
		return scheduledWarmup{}, false
	}
	current := now.In(location)
	var next scheduledWarmup
	found := false
	for _, item := range cfg.WarmupSchedules {
		if !item.enabled() {
			continue
		}
		clock, errParse := time.Parse("15:04", item.At)
		if errParse != nil {
			continue
		}
		candidate := time.Date(current.Year(), current.Month(), current.Day(), clock.Hour(), clock.Minute(), 0, 0, location)
		if !candidate.After(current) {
			candidate = candidate.AddDate(0, 0, 1)
		}
		if !found || candidate.Before(next.Time) {
			next = scheduledWarmup{ID: item.ID, At: item.At, Time: candidate}
			found = true
		}
	}
	return next, found
}

func mustLocation(name string) *time.Location {
	location, errLoad := time.LoadLocation(name)
	if errLoad != nil {
		return time.UTC
	}
	return location
}

func (s *guardState) runWarmup(ctx context.Context, cfg pluginConfig, scheduleKey, source string) ([]warmupResult, error) {
	if !s.warmupMu.TryLock() {
		return nil, errWarmupInProgress
	}
	defer s.warmupMu.Unlock()
	return s.runWarmupLocked(ctx, cfg, scheduleKey, source)
}

func (s *guardState) runWarmupLocked(ctx context.Context, cfg pluginConfig, scheduleKey, source string) ([]warmupResult, error) {
	if !cfg.warmupEnabled() {
		return nil, errWarmupDisabled
	}
	s.mu.Lock()
	s.warmupRunning = true
	s.warmupLastSource = source
	s.warmupLastError = ""
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.warmupRunning = false
		s.mu.Unlock()
	}()

	list, errList := hostAuthList(cfg.httpTimeout() + 10*time.Second)
	if errList != nil {
		s.setWarmupError(errList.Error())
		return nil, fmt.Errorf("host.auth.list: %w", errList)
	}
	results := make([]warmupResult, 0)
	now := time.Now().UTC()
	skipped := 0
	for _, auth := range list.Files {
		if errContext := ctx.Err(); errContext != nil {
			return results, errContext
		}
		if !strings.EqualFold(strings.TrimSpace(auth.Provider), "codex") {
			continue
		}
		if strings.TrimSpace(auth.ID) == "" {
			skipped++
			continue
		}
		state := s.warmupStateForAuth(auth)
		eligible, _ := warmupEligible(auth, state, now, scheduleKey)
		if !eligible {
			skipped++
			continue
		}
		result := warmupCredential(ctx, cfg, auth, scheduleKey)
		results = append(results, result)
		s.recordWarmupResult(auth, result)
		_ = s.persistState("warmup-result")
	}

	success := 0
	for _, result := range results {
		if result.Success {
			success++
		}
	}
	s.mu.Lock()
	s.warmupLastRunAt = time.Now().UTC()
	s.warmupLastResults = len(results)
	s.warmupLastSuccess = success
	s.warmupLastSkipped = skipped
	s.state.Warmup = warmupSummaryState{
		LastRunAt:  s.warmupLastRunAt,
		LastSource: s.warmupLastSource,
		Results:    s.warmupLastResults,
		Success:    s.warmupLastSuccess,
		Skipped:    s.warmupLastSkipped,
	}
	s.mu.Unlock()
	_ = s.persistState("warmup-summary")
	hostLog("info", "codex-quota-guard: warm-up finished", map[string]any{
		"source":           source,
		"credential_count": len(results),
		"success_count":    success,
		"failure_count":    len(results) - success,
	})
	return results, nil
}

func warmupEligible(auth pluginapi.HostAuthFileEntry, state *credentialState, now time.Time, scheduleKey string) (bool, string) {
	if !strings.EqualFold(strings.TrimSpace(auth.Provider), "codex") {
		return false, "provider is not Codex"
	}
	if strings.TrimSpace(auth.ID) == "" {
		return false, "credential has no auth ID"
	}
	if auth.Disabled {
		return false, "credential is disabled"
	}
	if state == nil {
		return true, "new credential"
	}
	if state.ManualDisabled {
		return false, "credential is manually disabled"
	}
	if !state.ManualEnabled && state.blocked(now) {
		return false, "credential has an active guard block"
	}
	if scheduleKey != "" && state.Warmup.LastRunKey == scheduleKey {
		return false, "schedule already ran for this credential"
	}
	return true, "eligible"
}

func (s *guardState) warmupStateForAuth(auth pluginapi.HostAuthFileEntry) *credentialState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneCredentialState(s.state.lookup(auth.ID, auth.AuthIndex, auth.Name))
}

func warmupCredential(ctx context.Context, cfg pluginConfig, auth pluginapi.HostAuthFileEntry, scheduleKey string) warmupResult {
	result := warmupResult{
		AuthID:      auth.ID,
		Name:        auth.Name,
		ScheduleKey: scheduleKey,
		Model:       cfg.WarmupModel,
		ExecutedAt:  time.Now().UTC(),
	}
	body, errMarshal := json.Marshal(map[string]any{
		"model": cfg.WarmupModel,
		"messages": []map[string]string{{
			"role":    "user",
			"content": cfg.WarmupPrompt,
		}},
		"stream": false,
	})
	if errMarshal != nil {
		result.Error = "encode warm-up request failed"
		return result
	}
	if errContext := ctx.Err(); errContext != nil {
		result.Error = "warm-up canceled"
		return result
	}
	request := pluginapi.HostModelExecutionRequest{
		EntryProtocol:  "openai",
		ExitProtocol:   "openai",
		Model:          cfg.WarmupModel,
		Stream:         false,
		Body:           body,
		Headers:        http.Header{"Content-Type": []string{"application/json"}},
		ForcedProvider: "codex",
		AuthID:         auth.ID,
	}
	raw, errCall := callHost(pluginabi.MethodHostModelExecute, request, cfg.httpTimeout()+15*time.Second)
	if errCall != nil {
		result.Error = "host model execution failed"
		return result
	}
	var response pluginapi.HostModelExecutionResponse
	if errUnmarshal := json.Unmarshal(raw, &response); errUnmarshal != nil {
		result.Error = "decode model execution response failed"
		return result
	}
	result.StatusCode = response.StatusCode
	result.Success = response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices
	if !result.Success {
		result.Error = fmt.Sprintf("model execution returned HTTP %d", response.StatusCode)
	}
	return result
}

func (s *guardState) recordWarmupResult(auth pluginapi.HostAuthFileEntry, result warmupResult) {
	s.mu.Lock()
	item := s.state.ensure(auth.ID, auth.AuthIndex, auth.Name, auth.Email, firstNonEmpty(auth.Provider, auth.Type))
	if item != nil {
		item.Warmup = warmupCredentialState{
			LastRunAt:   result.ExecutedAt,
			LastRunKey:  result.ScheduleKey,
			LastSource:  s.warmupLastSource,
			LastModel:   result.Model,
			LastStatus:  result.StatusCode,
			LastSuccess: result.Success,
			LastError:   result.Error,
		}
	}
	for key, report := range s.reports {
		if strings.EqualFold(report.AuthID, auth.ID) || strings.EqualFold(report.AuthIndex, auth.AuthIndex) || strings.EqualFold(report.Name, auth.Name) {
			report.WarmupLastRunAt = result.ExecutedAt
			report.WarmupLastSuccess = result.Success
			report.WarmupLastStatus = result.StatusCode
			report.WarmupLastError = result.Error
			s.reports[key] = report
			break
		}
	}
	s.mu.Unlock()
}

func (s *guardState) setWarmupError(message string) {
	s.mu.Lock()
	s.warmupLastError = message
	s.warmupLastRunAt = time.Now().UTC()
	s.state.Warmup.LastRunAt = s.warmupLastRunAt
	s.state.Warmup.LastSource = s.warmupLastSource
	s.state.Warmup.LastError = message
	s.mu.Unlock()
	_ = s.persistState("warmup-error")
}
