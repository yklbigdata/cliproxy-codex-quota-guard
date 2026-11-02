package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	pluginID      = "codex-quota-guard"
	pluginName    = "Codex Quota Guard"
	pluginVersion = "2.2.0"
	pluginAuthor  = "yklbigdata"
	repositoryURL = "https://github.com/yklbigdata/cliproxy-codex-quota-guard"

	defaultUsageURL  = "https://chatgpt.com/backend-api/wham/usage"
	defaultUserAgent = "codex-cli/0.154.0"

	guardMetadataKey     = "codex_quota_guard"
	primaryWindowSeconds = int64(18000)
	weeklyWindowSeconds  = int64(604800)

	defaultPollInterval      = 60 * time.Second
	defaultInitialDelay      = 15 * time.Second
	defaultHTTPTimeout       = 25 * time.Second
	defaultRateLimitFallback = 5 * time.Hour
	minPollInterval          = 15 * time.Second
	maxCredentials           = 500
	defaultWarmupModel       = "gpt-6-luna"
	defaultWarmupPrompt      = "Reply with exactly: quota window activated"
	defaultWarmupTimezone    = "Asia/Singapore"
)

// duration parses YAML durations such as "60s" as well as bare seconds.
type duration struct {
	value time.Duration
}

func (d *duration) UnmarshalYAML(node *yaml.Node) error {
	var text string
	if errDecode := node.Decode(&text); errDecode == nil {
		text = strings.TrimSpace(text)
		if text == "" {
			d.value = 0
			return nil
		}
		parsed, errParse := time.ParseDuration(text)
		if errParse != nil {
			return fmt.Errorf("invalid duration %q: %w", text, errParse)
		}
		d.value = parsed
		return nil
	}
	var seconds float64
	if errDecode := node.Decode(&seconds); errDecode != nil {
		return fmt.Errorf("invalid duration value: %w", errDecode)
	}
	d.value = time.Duration(seconds * float64(time.Second))
	return nil
}

func (d duration) or(fallback time.Duration) time.Duration {
	if d.value <= 0 {
		return fallback
	}
	return d.value
}

type pluginConfig struct {
	Providers []string `yaml:"providers"`

	DisableBelowRemainingPercent float64 `yaml:"disable_below_remaining_percent"`
	ResumeAboveRemainingPercent  float64 `yaml:"resume_above_remaining_percent"`
	EnforceWeeklyWindow          *bool   `yaml:"enforce_weekly_window"`
	WeeklyDisableBelowPercent    float64 `yaml:"weekly_disable_below_remaining_percent"`
	WeeklyResumeAbovePercent     float64 `yaml:"weekly_resume_above_remaining_percent"`

	PollInterval duration `yaml:"poll_interval"`
	InitialDelay duration `yaml:"initial_delay"`
	HTTPTimeout  duration `yaml:"http_timeout"`

	KeepAtLeastOne *bool `yaml:"keep_at_least_one"`
	DryRun         *bool `yaml:"dry_run"`

	MaxCredentials    int      `yaml:"max_credentials"`
	UsageURL          string   `yaml:"usage_url"`
	UserAgent         string   `yaml:"user_agent"`
	StateFile         string   `yaml:"state_file"`
	RateLimitFallback duration `yaml:"rate_limit_fallback"`

	WarmupEnabled   *bool            `yaml:"warmup_enabled"`
	WarmupModel     string           `yaml:"warmup_model"`
	WarmupPrompt    string           `yaml:"warmup_prompt"`
	WarmupTimezone  string           `yaml:"warmup_timezone"`
	WarmupSchedules []warmupSchedule `yaml:"warmup_schedules"`
}

type warmupSchedule struct {
	ID      string `yaml:"id" json:"id"`
	At      string `yaml:"at" json:"at"`
	Enabled *bool  `yaml:"enabled" json:"enabled,omitempty"`
}

func (s warmupSchedule) enabled() bool {
	return s.Enabled == nil || *s.Enabled
}

func defaultConfig() pluginConfig {
	keepOne := true
	dryRun := false
	weekly := false
	warmupEnabled := true
	return pluginConfig{
		Providers:                    []string{"codex"},
		DisableBelowRemainingPercent: 1,
		ResumeAboveRemainingPercent:  5,
		EnforceWeeklyWindow:          &weekly,
		WeeklyDisableBelowPercent:    1,
		WeeklyResumeAbovePercent:     5,
		PollInterval:                 duration{value: defaultPollInterval},
		InitialDelay:                 duration{value: defaultInitialDelay},
		HTTPTimeout:                  duration{value: defaultHTTPTimeout},
		KeepAtLeastOne:               &keepOne,
		DryRun:                       &dryRun,
		MaxCredentials:               50,
		UsageURL:                     defaultUsageURL,
		UserAgent:                    defaultUserAgent,
		StateFile:                    defaultPersistentStateFile(),
		RateLimitFallback:            duration{value: defaultRateLimitFallback},
		WarmupEnabled:                &warmupEnabled,
		WarmupModel:                  defaultWarmupModel,
		WarmupPrompt:                 defaultWarmupPrompt,
		WarmupTimezone:               defaultWarmupTimezone,
	}
}

func (c pluginConfig) enforceWeekly() bool {
	return c.EnforceWeeklyWindow != nil && *c.EnforceWeeklyWindow
}

func (c pluginConfig) keepAtLeastOne() bool {
	return c.KeepAtLeastOne == nil || *c.KeepAtLeastOne
}

func (c pluginConfig) dryRun() bool {
	return c.DryRun != nil && *c.DryRun
}

func (c pluginConfig) pollInterval() time.Duration {
	interval := c.PollInterval.or(defaultPollInterval)
	if interval < minPollInterval {
		interval = minPollInterval
	}
	return interval
}

func (c pluginConfig) initialDelay() time.Duration {
	delay := c.InitialDelay.or(defaultInitialDelay)
	if delay < 0 {
		delay = 0
	}
	return delay
}

func (c pluginConfig) httpTimeout() time.Duration {
	return c.HTTPTimeout.or(defaultHTTPTimeout)
}

func (c pluginConfig) rateLimitFallback() time.Duration {
	return c.RateLimitFallback.or(defaultRateLimitFallback)
}

func (c pluginConfig) warmupEnabled() bool {
	return c.WarmupEnabled == nil || *c.WarmupEnabled
}

func (c pluginConfig) providerAllowed(provider string) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return false
	}
	if len(c.Providers) == 0 {
		return provider == "codex"
	}
	for _, candidate := range c.Providers {
		if strings.EqualFold(strings.TrimSpace(candidate), provider) {
			return true
		}
	}
	return false
}

func configure(raw []byte) error {
	var req lifecycleRequest
	if len(raw) > 0 {
		if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
			return fmt.Errorf("decode lifecycle request: %w", errUnmarshal)
		}
	}
	cfg := defaultConfig()
	if len(req.ConfigYAML) > 0 {
		if errUnmarshal := yaml.Unmarshal(req.ConfigYAML, &cfg); errUnmarshal != nil {
			return fmt.Errorf("decode plugin config: %w", errUnmarshal)
		}
	}
	if errValidate := validateConfig(&cfg); errValidate != nil {
		return errValidate
	}
	guard.applyConfig(cfg)
	return nil
}

func validateConfig(cfg *pluginConfig) error {
	cfg.Providers = cleanList(cfg.Providers)
	if len(cfg.Providers) == 0 {
		cfg.Providers = []string{"codex"}
	}
	if cfg.DisableBelowRemainingPercent < 0 || cfg.DisableBelowRemainingPercent > 100 {
		return fmt.Errorf("disable_below_remaining_percent must be between 0 and 100")
	}
	if cfg.ResumeAboveRemainingPercent < 0 || cfg.ResumeAboveRemainingPercent > 100 {
		return fmt.Errorf("resume_above_remaining_percent must be between 0 and 100")
	}
	if cfg.ResumeAboveRemainingPercent <= cfg.DisableBelowRemainingPercent {
		return fmt.Errorf("resume_above_remaining_percent (%v) must be greater than disable_below_remaining_percent (%v)",
			cfg.ResumeAboveRemainingPercent, cfg.DisableBelowRemainingPercent)
	}
	if cfg.WeeklyDisableBelowPercent < 0 || cfg.WeeklyDisableBelowPercent > 100 {
		return fmt.Errorf("weekly_disable_below_remaining_percent must be between 0 and 100")
	}
	if cfg.WeeklyResumeAbovePercent < 0 || cfg.WeeklyResumeAbovePercent > 100 {
		return fmt.Errorf("weekly_resume_above_remaining_percent must be between 0 and 100")
	}
	if cfg.WeeklyResumeAbovePercent <= cfg.WeeklyDisableBelowPercent {
		return fmt.Errorf("weekly_resume_above_remaining_percent (%v) must be greater than weekly_disable_below_remaining_percent (%v)",
			cfg.WeeklyResumeAbovePercent, cfg.WeeklyDisableBelowPercent)
	}
	if cfg.MaxCredentials <= 0 || cfg.MaxCredentials > maxCredentials {
		return fmt.Errorf("max_credentials must be between 1 and %d", maxCredentials)
	}
	cfg.UsageURL = strings.TrimSpace(cfg.UsageURL)
	if cfg.UsageURL == "" {
		cfg.UsageURL = defaultUsageURL
	}
	if !strings.HasPrefix(cfg.UsageURL, "https://") {
		return fmt.Errorf("usage_url must use https")
	}
	cfg.UserAgent = strings.TrimSpace(cfg.UserAgent)
	if cfg.UserAgent == "" {
		cfg.UserAgent = defaultUserAgent
	}
	if strings.ContainsAny(cfg.UserAgent, "\r\n") {
		return fmt.Errorf("user_agent contains invalid characters")
	}
	cfg.StateFile = strings.TrimSpace(cfg.StateFile)
	if cfg.StateFile == "" {
		cfg.StateFile = defaultPersistentStateFile()
	}
	stateFile, errPath := filepath.Abs(cfg.StateFile)
	if errPath != nil {
		return fmt.Errorf("resolve state_file: %w", errPath)
	}
	cfg.StateFile = stateFile
	if cfg.rateLimitFallback() < time.Minute || cfg.rateLimitFallback() > 7*24*time.Hour {
		return fmt.Errorf("rate_limit_fallback must be between 1m and 168h")
	}
	cfg.WarmupModel = strings.TrimSpace(cfg.WarmupModel)
	if cfg.WarmupModel == "" {
		cfg.WarmupModel = defaultWarmupModel
	}
	cfg.WarmupPrompt = strings.TrimSpace(cfg.WarmupPrompt)
	if cfg.WarmupPrompt == "" {
		cfg.WarmupPrompt = defaultWarmupPrompt
	}
	cfg.WarmupTimezone = strings.TrimSpace(cfg.WarmupTimezone)
	if cfg.WarmupTimezone == "" {
		cfg.WarmupTimezone = defaultWarmupTimezone
	}
	if _, errLocation := time.LoadLocation(cfg.WarmupTimezone); errLocation != nil {
		return fmt.Errorf("warmup_timezone %q is invalid: %w", cfg.WarmupTimezone, errLocation)
	}
	seenIDs := make(map[string]struct{}, len(cfg.WarmupSchedules))
	seenTimes := make(map[string]struct{}, len(cfg.WarmupSchedules))
	for index := range cfg.WarmupSchedules {
		item := &cfg.WarmupSchedules[index]
		item.ID = strings.TrimSpace(item.ID)
		item.At = strings.TrimSpace(item.At)
		if item.ID == "" {
			return fmt.Errorf("warmup_schedules[%d].id is required", index)
		}
		if _, duplicate := seenIDs[strings.ToLower(item.ID)]; duplicate {
			return fmt.Errorf("warmup_schedules[%d].id %q is duplicated", index, item.ID)
		}
		seenIDs[strings.ToLower(item.ID)] = struct{}{}
		if item.At == "" {
			return fmt.Errorf("warmup_schedules[%d].at is required", index)
		}
		parsed, errParse := time.Parse("15:04", item.At)
		if errParse != nil {
			return fmt.Errorf("warmup_schedules[%d].at must use HH:MM", index)
		}
		item.At = parsed.Format("15:04")
		if _, duplicate := seenTimes[item.At]; duplicate {
			return fmt.Errorf("warmup_schedules[%d].at %q is duplicated", index, item.At)
		}
		seenTimes[item.At] = struct{}{}
	}
	return nil
}

func cleanList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}
