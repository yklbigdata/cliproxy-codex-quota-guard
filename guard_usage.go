package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

func (s *guardState) HandleUsage(_ context.Context, record pluginapi.UsageRecord) {
	if !strings.EqualFold(strings.TrimSpace(record.Provider), "codex") {
		return
	}
	if !record.Failed || record.Failure.StatusCode != http.StatusTooManyRequests {
		return
	}
	authID := strings.TrimSpace(record.AuthID)
	authIndex := strings.TrimSpace(record.AuthIndex)
	if authID == "" && authIndex == "" {
		hostLog("warn", "codex-quota-guard: 429 has no credential identity", nil)
		return
	}
	cfg := s.config()
	now := time.Now().UTC()
	block := rateLimitBlockFromHeaders(record.ResponseHeaders, now, cfg.rateLimitFallback())

	s.mu.Lock()
	item := s.state.ensure(authID, authIndex, authID, "", "codex")
	if item == nil {
		s.mu.Unlock()
		return
	}
	item.RateLimitBlock = mergeManagedBlock(item.RateLimitBlock, block, now)
	item.LastAction = "rate_limited"
	item.LastReason = block.Reason
	item.LastActionAt = now
	key := item.Key
	s.mu.Unlock()

	if report, ok := s.reportByStateKey(key); ok {
		report.RateLimitBlocked = true
		report.RateLimitResetAt = block.ResetAt
		report.RateLimitWindow = block.Window
		report.Reason = block.Reason
		report.CheckedAt = now
		s.storeReport(report.Ref, report)
	}
	if errPersist := s.persistState("429"); errPersist != nil {
		hostLog("error", "codex-quota-guard: 429 block was not persisted", map[string]any{"error": errPersist.Error()})
	}
	hostLog("warn", "codex-quota-guard: credential blocked after 429", map[string]any{
		"auth_id":  authID,
		"window":   block.Window,
		"reset_at": block.ResetAt.Format(time.RFC3339),
	})
}

func (s *guardState) reportByStateKey(key string) (credentialReport, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item := s.state.Credentials[key]
	if item == nil {
		return credentialReport{}, false
	}
	for _, report := range s.reports {
		if credentialStateKey(report.AuthID, report.AuthIndex, report.Name) == key {
			return report, true
		}
	}
	return credentialReport{}, false
}

func rateLimitBlockFromHeaders(headers http.Header, now time.Time, fallback time.Duration) managedBlock {
	if fallback <= 0 {
		fallback = defaultRateLimitFallback
	}
	primaryUsed := headerFloat(headers, "x-codex-primary-used-percent")
	secondaryUsed := headerFloat(headers, "x-codex-secondary-used-percent")
	primaryReset := headerUnixTime(headers, "x-codex-primary-reset-at")
	secondaryReset := headerUnixTime(headers, "x-codex-secondary-reset-at")
	primaryMinutes := headerInt(headers, "x-codex-primary-window-minutes")
	secondaryMinutes := headerInt(headers, "x-codex-secondary-window-minutes")

	primaryFull := primaryUsed >= 99.5
	secondaryFull := secondaryUsed >= 99.5
	window := ""
	resetAt := time.Time{}
	switch {
	case secondaryFull && !primaryFull && !secondaryReset.IsZero():
		window, resetAt = "7d", secondaryReset
	case primaryFull && !secondaryFull && !primaryReset.IsZero():
		window, resetAt = "5h", primaryReset
	case primaryFull && secondaryFull:
		switch {
		case !secondaryReset.IsZero() && !primaryReset.IsZero():
			resetAt = laterTime(secondaryReset, primaryReset)
			if resetAt.Equal(secondaryReset) {
				window = "7d"
			} else {
				window = "5h"
			}
		case !secondaryReset.IsZero():
			window, resetAt = "7d", secondaryReset
		case !primaryReset.IsZero():
			window, resetAt = "5h", primaryReset
		}
	case !secondaryReset.IsZero() && (secondaryMinutes == 10080 || secondaryFull):
		window, resetAt = "7d", secondaryReset
	case !primaryReset.IsZero() && (primaryMinutes == 300 || primaryFull):
		window, resetAt = "5h", primaryReset
	}
	if resetAt.IsZero() || !resetAt.After(now) {
		window = "5h"
		resetAt = now.Add(fallback)
	}
	return managedBlock{
		Kind:      blockKindRateLimit,
		Reason:    fmt.Sprintf("HTTP 429 from Codex; %s window resets at %s", window, resetAt.Format(time.RFC3339)),
		Window:    window,
		Status:    blockStatusActive,
		BlockedAt: now,
		ResetAt:   resetAt.UTC(),
	}
}

func laterTime(left, right time.Time) time.Time {
	if right.After(left) {
		return right
	}
	return left
}

func headerFloat(headers http.Header, name string) float64 {
	value := strings.TrimSpace(headers.Get(name))
	value = strings.TrimSuffix(value, "%")
	parsed, errParse := strconv.ParseFloat(value, 64)
	if errParse != nil {
		return -1
	}
	if parsed < 0 {
		return 0
	}
	if parsed > 100 {
		return 100
	}
	return parsed
}

func headerInt(headers http.Header, name string) int64 {
	value := strings.TrimSpace(headers.Get(name))
	parsed, errParse := strconv.ParseInt(value, 10, 64)
	if errParse != nil {
		return 0
	}
	return parsed
}

func headerUnixTime(headers http.Header, name string) time.Time {
	value := strings.TrimSpace(headers.Get(name))
	if value == "" {
		return time.Time{}
	}
	if parsed, errParse := strconv.ParseInt(value, 10, 64); errParse == nil {
		if parsed > 100000000000 {
			parsed /= 1000
		}
		return time.Unix(parsed, 0).UTC()
	}
	if parsed, errParse := time.Parse(time.RFC3339, value); errParse == nil {
		return parsed.UTC()
	}
	return time.Time{}
}

func (s *guardState) Pick(ctx context.Context, request pluginapi.SchedulerPickRequest) (pluginapi.SchedulerPickResponse, error) {
	if errContext := ctx.Err(); errContext != nil {
		return pluginapi.SchedulerPickResponse{}, errContext
	}
	now := time.Now().UTC()
	available := make([]pluginapi.SchedulerAuthCandidate, 0, len(request.Candidates))
	dirty := false
	s.mu.Lock()
	for _, candidate := range request.Candidates {
		if !strings.EqualFold(strings.TrimSpace(candidate.Provider), "codex") {
			available = append(available, candidate)
			continue
		}
		item := s.state.lookup(candidate.ID, candidate.ID, "")
		if item == nil {
			available = append(available, candidate)
			continue
		}
		if item.refreshExpired(now) {
			dirty = true
		}
		if item.ManualDisabled || item.blocked(now) {
			continue
		}
		available = append(available, candidate)
	}
	s.mu.Unlock()
	if dirty {
		_ = s.persistState("scheduler-expiry")
	}
	if len(available) == len(request.Candidates) {
		return pluginapi.SchedulerPickResponse{
			Handled:         true,
			DelegateBuiltin: pluginapi.SchedulerBuiltinRoundRobin,
		}, nil
	}
	if len(available) == 0 {
		return pluginapi.SchedulerPickResponse{
			Handled:      true,
			Reject:       true,
			RejectCode:   "quota_guard_blocked",
			RejectReason: "all Codex candidates are blocked by quota, rate-limit, or manual-disable state",
		}, nil
	}
	chosen := available[0]
	for _, candidate := range available[1:] {
		if candidate.Priority > chosen.Priority {
			chosen = candidate
		}
	}
	return pluginapi.SchedulerPickResponse{AuthID: chosen.ID, Handled: true}, nil
}
