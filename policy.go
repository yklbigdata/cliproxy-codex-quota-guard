package main

import (
	"fmt"
	"strings"
	"time"
)

func blockResetAt(block *managedBlock) time.Time {
	if block == nil {
		return time.Time{}
	}
	return block.ResetAt
}

func blockWindow(block *managedBlock) string {
	if block == nil {
		return ""
	}
	return block.Window
}

func activeBlockLabels(state *credentialState, now time.Time) []string {
	if state == nil {
		return nil
	}
	out := make([]string, 0, 2)
	if state.QuotaBlock.active(now) {
		out = append(out, state.QuotaBlock.Reason)
	}
	if state.RateLimitBlock.active(now) {
		out = append(out, state.RateLimitBlock.Reason)
	}
	return out
}

func (cfg pluginConfig) blockedBy(quota quotaSnapshot) (bool, string, string, time.Time) {
	if quota.Primary.Present && quota.Primary.RemainingPercent <= cfg.DisableBelowRemainingPercent {
		return true,
			fmt.Sprintf("5h window remaining %s <= %s", formatPercent(quota.Primary.RemainingPercent), formatPercent(cfg.DisableBelowRemainingPercent)),
			"5h",
			quota.Primary.ResetAt
	}
	if cfg.enforceWeekly() && quota.Weekly.Present && quota.Weekly.RemainingPercent <= cfg.WeeklyDisableBelowPercent {
		return true,
			fmt.Sprintf("7d window remaining %s <= %s", formatPercent(quota.Weekly.RemainingPercent), formatPercent(cfg.WeeklyDisableBelowPercent)),
			"7d",
			quota.Weekly.ResetAt
	}
	return false, "within policy", "", time.Time{}
}

func (cfg pluginConfig) recoveredBy(quota quotaSnapshot) (bool, string) {
	if !quota.Primary.Present {
		return false, "waiting for a 5h window reading"
	}
	if quota.Primary.RemainingPercent < cfg.ResumeAboveRemainingPercent {
		return false, fmt.Sprintf("5h window remaining %s < %s", formatPercent(quota.Primary.RemainingPercent), formatPercent(cfg.ResumeAboveRemainingPercent))
	}
	if cfg.enforceWeekly() && quota.Weekly.Present && quota.Weekly.RemainingPercent < cfg.WeeklyResumeAbovePercent {
		return false, fmt.Sprintf("7d window remaining %s < %s", formatPercent(quota.Weekly.RemainingPercent), formatPercent(cfg.WeeklyResumeAbovePercent))
	}
	return true, fmt.Sprintf("5h window recovered to %s remaining", formatPercent(quota.Primary.RemainingPercent))
}

func updateQuotaBlock(cfg pluginConfig, quota quotaSnapshot, existing *managedBlock, now time.Time) (*managedBlock, string) {
	if existing != nil && existing.active(now) {
		if recovered, reason := cfg.recoveredBy(quota); recovered {
			cleared := *existing
			cleared.Status = blockStatusCleared
			return &cleared, reason
		} else {
			return existing, reason
		}
	}
	if blocked, reason, window, resetAt := cfg.blockedBy(quota); blocked {
		kind := blockKindQuota5h
		if window == "7d" {
			kind = blockKindQuota7d
		}
		next := managedBlock{
			Kind:      kind,
			Reason:    reason,
			Window:    window,
			Status:    blockStatusActive,
			BlockedAt: now,
			ResetAt:   resetAt,
		}
		return &next, reason
	}
	if existing != nil {
		cleared := *existing
		cleared.Status = blockStatusCleared
		return &cleared, "within policy"
	}
	return nil, "within policy"
}

func markerFromBlocks(state *credentialState, now time.Time) guardMarker {
	marker := guardMarker{
		Managed:       true,
		UpdatedAt:     now.UTC().Format(time.RFC3339),
		PluginVersion: pluginVersion,
	}
	if state == nil {
		return marker
	}
	for _, block := range state.activeBlocks(now) {
		if block == nil {
			continue
		}
		marker.Reasons = append(marker.Reasons, *block)
	}
	if state.RateLimitBlock != nil && state.RateLimitBlock.active(now) {
		marker.DisabledAt = state.RateLimitBlock.BlockedAt.Format(time.RFC3339)
		marker.Reason = state.RateLimitBlock.Reason
		marker.WindowResetAt = state.RateLimitBlock.ResetAt.Format(time.RFC3339)
	}
	if state.QuotaBlock != nil && state.QuotaBlock.active(now) {
		marker.DisabledAt = state.QuotaBlock.BlockedAt.Format(time.RFC3339)
		marker.Reason = state.QuotaBlock.Reason
		marker.WindowResetAt = state.QuotaBlock.ResetAt.Format(time.RFC3339)
		if state.ObservedQuota.PrimaryRemaining != nil {
			value := *state.ObservedQuota.PrimaryRemaining
			marker.RemainingPercent = &value
		}
	}
	return marker
}

func markerHasBlocks(marker *guardMarker) bool {
	return marker != nil && marker.Managed
}

func compareBlocks(left []managedBlock, right []managedBlock) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		a, b := left[index], right[index]
		if a.Kind != b.Kind || a.Status != b.Status || !a.ResetAt.Equal(b.ResetAt) || a.Reason != b.Reason {
			return false
		}
	}
	return true
}

func quotaChoice(decision credentialDecision) float64 {
	if decision.report.PrimaryUsed != nil {
		return 100 - *decision.report.PrimaryUsed
	}
	if decision.state != nil && decision.state.ObservedQuota.PrimaryRemaining != nil {
		return *decision.state.ObservedQuota.PrimaryRemaining
	}
	return -1
}

func protectedQuotaCredential(cfg pluginConfig, candidates []credentialDecision, now time.Time) string {
	if !cfg.keepAtLeastOne() {
		return ""
	}
	type group struct {
		eligible int
		blocked  int
		name     string
		key      string
		best     float64
	}
	groups := make(map[string]*group)
	for index := range candidates {
		item := &candidates[index]
		provider := strings.ToLower(strings.TrimSpace(item.report.Provider))
		if provider == "" || item.manual || item.rateActive || item.report.Action == actionError {
			continue
		}
		if item.quotaBlock == nil && item.report.PrimaryUsed == nil {
			continue
		}
		g := groups[provider]
		if g == nil {
			g = &group{best: -1}
			groups[provider] = g
		}
		g.eligible++
		if !item.manualEnabled && item.quotaBlock.active(now) {
			g.blocked++
		}
		remaining := quotaChoice(*item)
		if remaining > g.best {
			g.best = remaining
			g.name = item.report.Name
			g.key = item.stateKey
		}
	}
	for _, g := range groups {
		if g.eligible > 0 && g.blocked == g.eligible && g.key != "" {
			return g.key
		}
	}
	return ""
}
