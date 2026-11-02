package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

type usageWindow struct {
	UsedPercent        *float64 `json:"used_percent"`
	LimitWindowSeconds int64    `json:"limit_window_seconds"`
	ResetAfterSeconds  int64    `json:"reset_after_seconds"`
	ResetAt            int64    `json:"reset_at"`
}

type usagePayload struct {
	PlanType  string `json:"plan_type"`
	RateLimit struct {
		PrimaryWindow   *usageWindow `json:"primary_window"`
		SecondaryWindow *usageWindow `json:"secondary_window"`
	} `json:"rate_limit"`
}

// quotaWindow is the normalized view of one Codex rate limit window.
type quotaWindow struct {
	Present          bool
	UsedPercent      float64
	RemainingPercent float64
	WindowSeconds    int64
	ResetAfter       int64
	ResetAt          time.Time
}

type quotaSnapshot struct {
	PlanType  string
	Primary   quotaWindow
	Weekly    quotaWindow
	FetchedAt time.Time
}

func normalizeWindow(window *usageWindow) quotaWindow {
	if window == nil || window.UsedPercent == nil {
		return quotaWindow{}
	}
	used := *window.UsedPercent
	if used < 0 {
		used = 0
	}
	if used > 100 {
		used = 100
	}
	out := quotaWindow{
		Present:          true,
		UsedPercent:      used,
		RemainingPercent: 100 - used,
		WindowSeconds:    window.LimitWindowSeconds,
		ResetAfter:       window.ResetAfterSeconds,
	}
	if window.ResetAt > 0 {
		out.ResetAt = time.Unix(window.ResetAt, 0).UTC()
	} else if window.ResetAfterSeconds > 0 {
		out.ResetAt = time.Now().UTC().Add(time.Duration(window.ResetAfterSeconds) * time.Second)
	}
	return out
}

// fetchQuota reads the same Codex usage endpoint the Codex client and the
// management UI display, using the credential access token held by the host.
func fetchQuota(ctx context.Context, cfg pluginConfig, token, accountID string) (quotaSnapshot, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return quotaSnapshot{}, fmt.Errorf("credential has no access token")
	}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)
	headers.Set("Accept", "application/json")
	headers.Set("Content-Type", "application/json")
	headers.Set("User-Agent", cfg.UserAgent)
	if trimmed := strings.TrimSpace(accountID); trimmed != "" {
		headers.Set("Chatgpt-Account-Id", trimmed)
	}
	request := pluginapi.HTTPRequest{
		Method:  http.MethodGet,
		URL:     cfg.UsageURL,
		Headers: headers,
	}
	if errContext := ctx.Err(); errContext != nil {
		return quotaSnapshot{}, errContext
	}
	response, errCall := hostHTTPDo(request, cfg.httpTimeout())
	if errCall != nil {
		return quotaSnapshot{}, errCall
	}
	if response.StatusCode != http.StatusOK {
		return quotaSnapshot{}, fmt.Errorf("usage endpoint returned HTTP %d", response.StatusCode)
	}
	var payload usagePayload
	if errUnmarshal := json.Unmarshal(response.Body, &payload); errUnmarshal != nil {
		return quotaSnapshot{}, fmt.Errorf("decode usage payload: %w", errUnmarshal)
	}
	snapshot := quotaSnapshot{
		PlanType:  strings.TrimSpace(payload.PlanType),
		Primary:   normalizeWindow(payload.RateLimit.PrimaryWindow),
		Weekly:    normalizeWindow(payload.RateLimit.SecondaryWindow),
		FetchedAt: time.Now().UTC(),
	}
	if !snapshot.Primary.Present && !snapshot.Weekly.Present {
		return quotaSnapshot{}, fmt.Errorf("usage payload carried no rate limit windows")
	}
	return snapshot, nil
}

func windowLabel(window quotaWindow) string {
	if !window.Present {
		return "n/a"
	}
	switch {
	case window.WindowSeconds == primaryWindowSeconds:
		return "5h"
	case window.WindowSeconds == weeklyWindowSeconds:
		return "7d"
	case window.WindowSeconds > 0:
		return fmt.Sprintf("%dm", window.WindowSeconds/60)
	default:
		return "window"
	}
}

func formatPercent(value float64) string {
	if value == float64(int64(value)) {
		return fmt.Sprintf("%d%%", int64(value))
	}
	return fmt.Sprintf("%.1f%%", value)
}

func formatResetIn(seconds int64) string {
	if seconds <= 0 {
		return "-"
	}
	duration := time.Duration(seconds) * time.Second
	hours := int64(duration.Hours())
	minutes := int64(duration.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("%dh%02dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
