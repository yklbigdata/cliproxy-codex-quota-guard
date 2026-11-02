package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	statusResourcePath  = "/status"
	resourceContentType = "text/html; charset=utf-8"
	jsonContentType     = "application/json; charset=utf-8"

	managementRoutePrefix  = "/plugins/" + pluginID
	managementStatusPath   = managementRoutePrefix + "/status"
	managementValidatePath = managementRoutePrefix + "/validate-config"
	managementSweepPath    = managementRoutePrefix + "/sweep"
	managementWarmupPath   = managementRoutePrefix + "/warmup"
	managementResetPath    = managementRoutePrefix + "/reset-block"
)

type managementRegistration struct {
	Routes    []managementRoute    `json:"routes,omitempty"`
	Resources []managementResource `json:"resources,omitempty"`
}

type managementRoute struct {
	Method      string `json:"Method"`
	Path        string `json:"Path"`
	Description string `json:"Description"`
}

type managementResource struct {
	Path        string `json:"Path"`
	Menu        string `json:"Menu"`
	Description string `json:"Description"`
}

type managementRequest struct {
	Method         string      `json:"Method"`
	Path           string      `json:"Path"`
	Headers        http.Header `json:"Headers"`
	Query          url.Values  `json:"Query"`
	Body           []byte      `json:"Body"`
	HostCallbackID string      `json:"host_callback_id,omitempty"`
}

type managementResponse struct {
	StatusCode int         `json:"StatusCode"`
	Headers    http.Header `json:"Headers"`
	Body       []byte      `json:"Body"`
}

type managementSettings struct {
	DisableBelowRemainingPercent float64          `json:"disable_below_remaining_percent"`
	ResumeAboveRemainingPercent  float64          `json:"resume_above_remaining_percent"`
	EnforceWeeklyWindow          bool             `json:"enforce_weekly_window"`
	WeeklyDisableBelowPercent    float64          `json:"weekly_disable_below_remaining_percent"`
	WeeklyResumeAbovePercent     float64          `json:"weekly_resume_above_remaining_percent"`
	PollInterval                 string           `json:"poll_interval"`
	InitialDelay                 string           `json:"initial_delay"`
	KeepAtLeastOne               bool             `json:"keep_at_least_one"`
	DryRun                       bool             `json:"dry_run"`
	RateLimitFallback            string           `json:"rate_limit_fallback"`
	WarmupEnabled                bool             `json:"warmup_enabled"`
	WarmupModel                  string           `json:"warmup_model"`
	WarmupPrompt                 string           `json:"warmup_prompt"`
	WarmupTimezone               string           `json:"warmup_timezone"`
	WarmupSchedules              []warmupSchedule `json:"warmup_schedules"`
}

type managementSettingsPatch struct {
	DisableBelowRemainingPercent *float64          `json:"disable_below_remaining_percent"`
	ResumeAboveRemainingPercent  *float64          `json:"resume_above_remaining_percent"`
	EnforceWeeklyWindow          *bool             `json:"enforce_weekly_window"`
	WeeklyDisableBelowPercent    *float64          `json:"weekly_disable_below_remaining_percent"`
	WeeklyResumeAbovePercent     *float64          `json:"weekly_resume_above_remaining_percent"`
	PollInterval                 *string           `json:"poll_interval"`
	InitialDelay                 *string           `json:"initial_delay"`
	KeepAtLeastOne               *bool             `json:"keep_at_least_one"`
	DryRun                       *bool             `json:"dry_run"`
	RateLimitFallback            *string           `json:"rate_limit_fallback"`
	WarmupEnabled                *bool             `json:"warmup_enabled"`
	WarmupModel                  *string           `json:"warmup_model"`
	WarmupPrompt                 *string           `json:"warmup_prompt"`
	WarmupTimezone               *string           `json:"warmup_timezone"`
	WarmupSchedules              *[]warmupSchedule `json:"warmup_schedules"`
}

type managementCredential struct {
	Ref               string   `json:"ref"`
	Name              string   `json:"name"`
	Email             string   `json:"email,omitempty"`
	Provider          string   `json:"provider,omitempty"`
	PlanType          string   `json:"plan_type,omitempty"`
	FileDisabled      bool     `json:"file_disabled"`
	RuntimeDisabled   bool     `json:"runtime_disabled"`
	Managed           bool     `json:"managed"`
	ManualDisabled    bool     `json:"manual_disabled"`
	QuotaBlocked      bool     `json:"quota_blocked"`
	RateLimitBlocked  bool     `json:"rate_limit_blocked"`
	CanResetBlock     bool     `json:"can_reset_block"`
	PrimaryUsed       *float64 `json:"primary_used_percent,omitempty"`
	PrimaryResetIn    int64    `json:"primary_reset_in_seconds,omitempty"`
	WeeklyUsed        *float64 `json:"weekly_used_percent,omitempty"`
	WeeklyResetIn     int64    `json:"weekly_reset_in_seconds,omitempty"`
	RateLimitWindow   string   `json:"rate_limit_window,omitempty"`
	RateLimitResetAt  string   `json:"rate_limit_reset_at,omitempty"`
	WarmupLastRunAt   string   `json:"warmup_last_run_at,omitempty"`
	WarmupLastSuccess bool     `json:"warmup_last_success"`
	WarmupLastStatus  int      `json:"warmup_last_status,omitempty"`
	WarmupLastError   string   `json:"warmup_last_error,omitempty"`
	Action            string   `json:"action"`
	Reason            string   `json:"reason,omitempty"`
	Error             string   `json:"error,omitempty"`
	CheckedAt         string   `json:"checked_at,omitempty"`
}

type managementStatus struct {
	PluginVersion  string                 `json:"plugin_version"`
	Running        bool                   `json:"running"`
	SweepRunning   bool                   `json:"sweep_running"`
	StartedAt      string                 `json:"started_at,omitempty"`
	LastSweep      string                 `json:"last_sweep,omitempty"`
	NextSweep      string                 `json:"next_sweep,omitempty"`
	LastSweepCause string                 `json:"last_sweep_cause,omitempty"`
	LastSweepError string                 `json:"last_sweep_error,omitempty"`
	Sweeps         int64                  `json:"sweeps"`
	Disables       int64                  `json:"disables"`
	Enables        int64                  `json:"enables"`
	Errors         int64                  `json:"errors"`
	StateReadOnly  bool                   `json:"state_read_only"`
	StateError     string                 `json:"state_error,omitempty"`
	StateSavedAt   string                 `json:"state_saved_at,omitempty"`
	StateSaves     int64                  `json:"state_saves"`
	WarmupRunning  bool                   `json:"warmup_running"`
	WarmupLastRun  string                 `json:"warmup_last_run,omitempty"`
	WarmupSource   string                 `json:"warmup_source,omitempty"`
	WarmupResults  int                    `json:"warmup_results"`
	WarmupSuccess  int                    `json:"warmup_success"`
	WarmupSkipped  int                    `json:"warmup_skipped"`
	WarmupError    string                 `json:"warmup_error,omitempty"`
	NextWarmup     string                 `json:"next_warmup,omitempty"`
	NextWarmupID   string                 `json:"next_warmup_id,omitempty"`
	Settings       managementSettings     `json:"settings"`
	Credentials    []managementCredential `json:"credentials"`
}

func managementRegistrationPayload() managementRegistration {
	return managementRegistration{
		Routes: []managementRoute{
			{Method: http.MethodGet, Path: managementStatusPath, Description: "Return live guard state, decisions, and editable settings."},
			{Method: http.MethodPost, Path: managementValidatePath, Description: "Validate and normalize a functional settings patch before it is saved."},
			{Method: http.MethodPost, Path: managementSweepPath, Description: "Start one quota sweep immediately if another sweep is not running."},
			{Method: http.MethodPost, Path: managementWarmupPath, Description: "Start one warm-up pass immediately for eligible credentials."},
			{Method: http.MethodPost, Path: managementResetPath, Description: "Clear plugin-owned quota and rate-limit blocks for one credential."},
		},
		Resources: []managementResource{{
			Path:        statusResourcePath,
			Menu:        pluginName,
			Description: "Configure and operate Codex quota enforcement, 429 blocks, and scheduled warm-up.",
		}},
	}
}

func handleManagement(raw []byte) ([]byte, error) {
	var req managementRequest
	if len(raw) > 0 {
		if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
			return nil, fmt.Errorf("decode management request: %w", errUnmarshal)
		}
	}
	return okEnvelope(dispatchManagement(req, guard))
}

func dispatchManagement(req managementRequest, state *guardState) managementResponse {
	if state == nil {
		return jsonManagementResponse(http.StatusServiceUnavailable, map[string]any{
			"error":   "plugin_unavailable",
			"message": "Codex Quota Guard is not initialized",
		})
	}
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	path := strings.TrimSpace(req.Path)

	switch {
	case method == http.MethodGet && isResourcePath(path, statusResourcePath):
		return htmlResponse(http.StatusOK, []byte(renderManagementPage()))
	case method == http.MethodGet && isManagementPath(path, managementStatusPath):
		return jsonManagementResponse(http.StatusOK, managementStatusPayload(state.snapshot()))
	case method == http.MethodPost && isManagementPath(path, managementValidatePath):
		return handleValidateSettings(state, req.Body)
	case method == http.MethodPost && isManagementPath(path, managementSweepPath):
		if errStart := state.startSweepAction(); errStart != nil {
			return actionErrorResponse(errStart)
		}
		return jsonManagementResponse(http.StatusAccepted, map[string]any{
			"status":  "accepted",
			"message": "quota sweep started",
		})
	case method == http.MethodPost && isManagementPath(path, managementWarmupPath):
		if errStart := state.startWarmupAction(); errStart != nil {
			return actionErrorResponse(errStart)
		}
		return jsonManagementResponse(http.StatusAccepted, map[string]any{
			"status":  "accepted",
			"message": "warm-up started",
		})
	case method == http.MethodPost && isManagementPath(path, managementResetPath):
		return handleResetBlock(state, req.Body)
	default:
		return jsonManagementResponse(http.StatusNotFound, map[string]any{
			"error":   "not_found",
			"message": "unknown management path",
			"method":  method,
			"path":    path,
		})
	}
}

func isResourcePath(path, suffix string) bool {
	path = strings.TrimRight(strings.TrimSpace(path), "/")
	if path == "" {
		path = "/"
	}
	return path == suffix ||
		path == "/plugins/"+pluginID+suffix ||
		path == "/v0/resource/plugins/"+pluginID+suffix
}

func isManagementPath(path, suffix string) bool {
	path = strings.TrimRight(strings.TrimSpace(path), "/")
	return path == suffix ||
		path == "/v0/management"+suffix ||
		path == "/v8/management"+suffix
}

func htmlResponse(statusCode int, body []byte) managementResponse {
	return managementResponse{
		StatusCode: statusCode,
		Headers: http.Header{
			"Content-Type":           []string{resourceContentType},
			"Cache-Control":          []string{"no-store"},
			"X-Content-Type-Options": []string{"nosniff"},
			"Referrer-Policy":        []string{"no-referrer"},
		},
		Body: body,
	}
}

func jsonManagementResponse(statusCode int, value any) managementResponse {
	body, errMarshal := json.Marshal(value)
	if errMarshal != nil {
		statusCode = http.StatusInternalServerError
		body, _ = json.Marshal(map[string]any{
			"error":   "response_encode_failed",
			"message": errMarshal.Error(),
		})
	}
	return managementResponse{
		StatusCode: statusCode,
		Headers: http.Header{
			"Content-Type":           []string{jsonContentType},
			"Cache-Control":          []string{"no-store"},
			"X-Content-Type-Options": []string{"nosniff"},
		},
		Body: body,
	}
}

func actionErrorResponse(errAction error) managementResponse {
	status := http.StatusConflict
	code := "action_unavailable"
	switch {
	case errors.Is(errAction, errGuardNotRunning):
		status = http.StatusServiceUnavailable
		code = "plugin_stopped"
	case errors.Is(errAction, errSweepInProgress):
		code = "sweep_in_progress"
	case errors.Is(errAction, errWarmupInProgress):
		code = "warmup_in_progress"
	case errors.Is(errAction, errWarmupDisabled):
		code = "warmup_disabled"
	}
	return jsonManagementResponse(status, map[string]any{
		"error":   code,
		"message": errAction.Error(),
	})
}
func handleValidateSettings(state *guardState, body []byte) managementResponse {
	patch, errDecode := decodeSettingsPatch(body)
	if errDecode != nil {
		return jsonManagementResponse(http.StatusBadRequest, map[string]any{
			"error":   "invalid_settings",
			"message": errDecode.Error(),
		})
	}
	candidate, errApply := applyManagementSettingsPatch(state.config(), patch)
	if errApply != nil {
		return jsonManagementResponse(http.StatusUnprocessableEntity, map[string]any{
			"error":   "invalid_settings",
			"message": errApply.Error(),
		})
	}
	return jsonManagementResponse(http.StatusOK, map[string]any{
		"status":   "valid",
		"settings": managementSettingsFromConfig(candidate),
	})
}

func decodeSettingsPatch(body []byte) (managementSettingsPatch, error) {
	var patch managementSettingsPatch
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if errDecode := decoder.Decode(&patch); errDecode != nil {
		return managementSettingsPatch{}, fmt.Errorf("decode settings: %w", errDecode)
	}
	var trailing any
	if errTrailing := decoder.Decode(&trailing); !errors.Is(errTrailing, io.EOF) {
		if errTrailing == nil {
			return managementSettingsPatch{}, fmt.Errorf("settings body contains more than one JSON value")
		}
		return managementSettingsPatch{}, fmt.Errorf("decode trailing settings data: %w", errTrailing)
	}
	return patch, nil
}

func applyManagementSettingsPatch(current pluginConfig, patch managementSettingsPatch) (pluginConfig, error) {
	cfg := clonePluginConfig(current)
	if value := patch.DisableBelowRemainingPercent; value != nil {
		cfg.DisableBelowRemainingPercent = *value
	}
	if value := patch.ResumeAboveRemainingPercent; value != nil {
		cfg.ResumeAboveRemainingPercent = *value
	}
	if value := patch.EnforceWeeklyWindow; value != nil {
		copied := *value
		cfg.EnforceWeeklyWindow = &copied
	}
	if value := patch.WeeklyDisableBelowPercent; value != nil {
		cfg.WeeklyDisableBelowPercent = *value
	}
	if value := patch.WeeklyResumeAbovePercent; value != nil {
		cfg.WeeklyResumeAbovePercent = *value
	}
	if value := patch.PollInterval; value != nil {
		parsed, errParse := parseManagementDuration("poll_interval", *value)
		if errParse != nil {
			return pluginConfig{}, errParse
		}
		cfg.PollInterval = duration{value: parsed}
	}
	if value := patch.InitialDelay; value != nil {
		parsed, errParse := parseManagementDuration("initial_delay", *value)
		if errParse != nil {
			return pluginConfig{}, errParse
		}
		cfg.InitialDelay = duration{value: parsed}
	}
	if value := patch.KeepAtLeastOne; value != nil {
		copied := *value
		cfg.KeepAtLeastOne = &copied
	}
	if value := patch.DryRun; value != nil {
		copied := *value
		cfg.DryRun = &copied
	}
	if value := patch.RateLimitFallback; value != nil {
		parsed, errParse := parseManagementDuration("rate_limit_fallback", *value)
		if errParse != nil {
			return pluginConfig{}, errParse
		}
		cfg.RateLimitFallback = duration{value: parsed}
	}
	if value := patch.WarmupEnabled; value != nil {
		copied := *value
		cfg.WarmupEnabled = &copied
	}
	if value := patch.WarmupModel; value != nil {
		cfg.WarmupModel = *value
	}
	if value := patch.WarmupPrompt; value != nil {
		cfg.WarmupPrompt = *value
	}
	if value := patch.WarmupTimezone; value != nil {
		cfg.WarmupTimezone = *value
	}
	if value := patch.WarmupSchedules; value != nil {
		cfg.WarmupSchedules = cloneWarmupSchedules(*value)
	}
	if errValidate := validateConfig(&cfg); errValidate != nil {
		return pluginConfig{}, errValidate
	}
	return cfg, nil
}

func parseManagementDuration(name, text string) (time.Duration, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, fmt.Errorf("%s is required", name)
	}
	parsed, errParse := time.ParseDuration(text)
	if errParse != nil {
		seconds, errSeconds := strconv.ParseFloat(text, 64)
		if errSeconds != nil {
			return 0, fmt.Errorf("%s must be a duration such as 60s: %w", name, errParse)
		}
		parsed = time.Duration(seconds * float64(time.Second))
	}
	if parsed < 0 {
		return 0, fmt.Errorf("%s must not be negative", name)
	}
	return parsed, nil
}

func managementSettingsFromConfig(cfg pluginConfig) managementSettings {
	return managementSettings{
		DisableBelowRemainingPercent: cfg.DisableBelowRemainingPercent,
		ResumeAboveRemainingPercent:  cfg.ResumeAboveRemainingPercent,
		EnforceWeeklyWindow:          cfg.enforceWeekly(),
		WeeklyDisableBelowPercent:    cfg.WeeklyDisableBelowPercent,
		WeeklyResumeAbovePercent:     cfg.WeeklyResumeAbovePercent,
		PollInterval:                 cfg.pollInterval().String(),
		InitialDelay:                 cfg.initialDelay().String(),
		KeepAtLeastOne:               cfg.keepAtLeastOne(),
		DryRun:                       cfg.dryRun(),
		RateLimitFallback:            cfg.rateLimitFallback().String(),
		WarmupEnabled:                cfg.warmupEnabled(),
		WarmupModel:                  cfg.WarmupModel,
		WarmupPrompt:                 cfg.WarmupPrompt,
		WarmupTimezone:               cfg.WarmupTimezone,
		WarmupSchedules:              cloneWarmupSchedules(cfg.WarmupSchedules),
	}
}

func managementStatusPayload(snapshot statusSnapshot) managementStatus {
	credentials := make([]managementCredential, 0, len(snapshot.Reports))
	for _, report := range snapshot.Reports {
		canReset := !report.ManualDisabled && (report.Managed || report.QuotaBlocked || report.RateLimitBlocked)
		credentials = append(credentials, managementCredential{
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
			CanResetBlock:     canReset,
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
		})
	}
	return managementStatus{
		PluginVersion:  snapshot.PluginVersion,
		Running:        snapshot.Running,
		SweepRunning:   snapshot.SweepRunning,
		StartedAt:      formatManagementTime(snapshot.StartedAt),
		LastSweep:      formatManagementTime(snapshot.LastSweep),
		NextSweep:      formatManagementTime(snapshot.NextSweep),
		LastSweepCause: snapshot.LastSweepCause,
		LastSweepError: snapshot.LastSweepError,
		Sweeps:         snapshot.Sweeps,
		Disables:       snapshot.Disables,
		Enables:        snapshot.Enables,
		Errors:         snapshot.Errors,
		StateReadOnly:  snapshot.StateReadOnly,
		StateError:     snapshot.StateError,
		StateSavedAt:   formatManagementTime(snapshot.StateSavedAt),
		StateSaves:     snapshot.StateSaves,
		WarmupRunning:  snapshot.WarmupRunning,
		WarmupLastRun:  formatManagementTime(snapshot.WarmupLastRun),
		WarmupSource:   snapshot.WarmupSource,
		WarmupResults:  snapshot.WarmupResults,
		WarmupSuccess:  snapshot.WarmupSuccess,
		WarmupSkipped:  snapshot.WarmupSkipped,
		WarmupError:    snapshot.WarmupError,
		NextWarmup:     formatManagementTime(snapshot.NextWarmup),
		NextWarmupID:   snapshot.NextWarmupID,
		Settings:       managementSettingsFromConfig(snapshot.Config),
		Credentials:    credentials,
	}
}

func formatManagementTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func cloneFloatPointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func cloneWarmupSchedules(source []warmupSchedule) []warmupSchedule {
	if len(source) == 0 {
		return []warmupSchedule{}
	}
	out := make([]warmupSchedule, len(source))
	for index := range source {
		out[index] = source[index]
		if source[index].Enabled != nil {
			enabled := *source[index].Enabled
			out[index].Enabled = &enabled
		}
	}
	return out
}

func handleResetBlock(state *guardState, body []byte) managementResponse {
	var request struct {
		Ref string `json:"ref"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if errDecode := decoder.Decode(&request); errDecode != nil {
		return jsonManagementResponse(http.StatusBadRequest, map[string]any{
			"error":   "invalid_request",
			"message": "body must contain one credential ref",
		})
	}
	request.Ref = strings.TrimSpace(request.Ref)
	if request.Ref == "" {
		return jsonManagementResponse(http.StatusBadRequest, map[string]any{
			"error":   "missing_ref",
			"message": "ref is required",
		})
	}
	report, errReset := state.resetManagedBlocks(context.Background(), request.Ref)
	if errReset != nil {
		switch {
		case errors.Is(errReset, errGuardNotRunning):
			return actionErrorResponse(errReset)
		case errors.Is(errReset, errSweepInProgress):
			return actionErrorResponse(errReset)
		case errors.Is(errReset, errCredentialNotFound), errors.Is(errReset, errCredentialAmbiguous):
			return jsonManagementResponse(http.StatusNotFound, map[string]any{
				"error":   "credential_not_found",
				"message": errReset.Error(),
			})
		case errors.Is(errReset, errNoManagedBlock):
			return jsonManagementResponse(http.StatusConflict, map[string]any{
				"error":   "no_managed_block",
				"message": errReset.Error(),
			})
		case errors.Is(errReset, errCredentialManual):
			return jsonManagementResponse(http.StatusConflict, map[string]any{
				"error":   "manual_disable",
				"message": errReset.Error(),
			})
		default:
			return jsonManagementResponse(http.StatusBadGateway, map[string]any{
				"error":   "reset_failed",
				"message": errReset.Error(),
			})
		}
	}
	return jsonManagementResponse(http.StatusOK, map[string]any{
		"status":     "cleared",
		"credential": report,
	})
}
