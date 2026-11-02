package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	actionDisabled = "disabled"
	actionEnabled  = "enabled"
	actionNone     = "none"
	actionSkipped  = "skipped"
	actionError    = "error"
)

// guardMarker is written next to the credential so the guard only ever
// re-enables credentials it disabled itself.
type guardMarker struct {
	Managed          bool     `json:"managed"`
	DisabledAt       string   `json:"disabled_at,omitempty"`
	Reason           string   `json:"reason,omitempty"`
	RemainingPercent *float64 `json:"remaining_percent,omitempty"`
	WindowResetAt    string   `json:"window_reset_at,omitempty"`
	PluginVersion    string   `json:"plugin_version,omitempty"`
}

type credentialReport struct {
	Ref             string
	Name            string
	Email           string
	Provider        string
	PlanType        string
	RuntimeDisabled bool
	FileDisabled    bool
	Managed         bool
	PrimaryUsed     *float64
	PrimaryResetIn  int64
	WeeklyUsed      *float64
	WeeklyResetIn   int64
	Action          string
	Reason          string
	Error           string
	CheckedAt       time.Time
}

type credentialDecision struct {
	report   credentialReport
	document *authDocument
	token    string
	account  string
	path     string
	quota    quotaSnapshot
	blocked  bool
	recover  bool
	disabled bool
	managed  bool
	name     string
}

type guardState struct {
	mu sync.RWMutex

	cfg     pluginConfig
	reports map[string]credentialReport

	lastSweep      time.Time
	lastSweepError string
	lastSweepCause string
	sweeps         int64
	disableActions int64
	enableActions  int64
	errorCount     int64

	cancel    context.CancelFunc
	running   bool
	startedAt time.Time
}

func newGuardState() *guardState {
	return &guardState{
		cfg:     defaultConfig(),
		reports: make(map[string]credentialReport),
	}
}

func (s *guardState) applyConfig(cfg pluginConfig) {
	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
	s.restart()
}

func (s *guardState) config() pluginConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *guardState) stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.running = false
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *guardState) restart() {
	s.stop()
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancel = cancel
	s.running = true
	s.startedAt = time.Now().UTC()
	cfg := s.cfg
	s.mu.Unlock()
	go s.loop(ctx, cfg)
}

func (s *guardState) loop(ctx context.Context, cfg pluginConfig) {
	delay := cfg.initialDelay()
	if delay > 0 {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
	s.sweep(ctx, "startup")

	interval := cfg.pollInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweep(ctx, "interval")
		}
	}
}

func (s *guardState) sweep(ctx context.Context, cause string) {
	cfg := s.config()
	hostTimeout := cfg.httpTimeout() + 10*time.Second

	list, errList := hostAuthList(hostTimeout)
	if errList != nil {
		s.recordSweepFailure(cause, fmt.Sprintf("host.auth.list failed: %v", errList))
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
		provider := entry.Provider
		if strings.TrimSpace(provider) == "" {
			provider = entry.Type
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
			continue
		}
		candidates = append(candidates, s.inspect(ctx, cfg, entry, ref, hostTimeout))
	}

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].name < candidates[j].name })

	// keep_at_least_one: never disable every credential of the same provider.
	protected := s.protectLastEnabled(cfg, candidates)

	for index := range candidates {
		decision := &candidates[index]
		report := decision.report
		report.CheckedAt = time.Now().UTC()
		if decision.report.Action == actionSkipped || decision.report.Action == actionError {
			s.storeReport(decision.report.Ref, decision.report)
			continue
		}
		if decision.blocked && !decision.disabled {
			if protected == decision.name {
				report.Action = actionNone
				report.Reason = "protected: keeping the last enabled credential"
				s.storeReport(report.Ref, report)
				continue
			}
			if errApply := s.disableCredential(cfg, *decision); errApply != nil {
				report.Action = actionError
				report.Error = errApply.Error()
				report.Reason = "disable failed"
				s.bumpError()
				hostLog("error", "codex-quota-guard: disable failed", map[string]any{"credential": decision.name, "error": errApply.Error()})
			} else {
				report.Action = actionDisabled
				report.Managed = true
				report.FileDisabled = true
				s.bumpDisable()
				if cfg.dryRun() {
					report.Reason = "dry run: " + report.Reason
				} else {
					applied, observed := s.confirmRuntime(decision.report.Ref, true, 5*time.Second)
					report.RuntimeDisabled = observed
					if !applied {
						report.Action = actionError
						report.Error = "runtime credential still reports enabled after the write"
						s.bumpError()
						hostLog("error", "codex-quota-guard: disable did not reach the runtime", map[string]any{
							"credential": decision.name,
							"reason":     report.Reason,
						})
					} else {
						hostLog("warn", "codex-quota-guard: credential disabled", map[string]any{
							"credential": decision.name,
							"reason":     report.Reason,
						})
					}
				}
			}
			s.storeReport(report.Ref, report)
			continue
		}
		if decision.disabled && decision.managed && decision.recover {
			if errApply := s.enableCredential(cfg, *decision); errApply != nil {
				report.Action = actionError
				report.Error = errApply.Error()
				report.Reason = "enable failed"
				s.bumpError()
				hostLog("error", "codex-quota-guard: enable failed", map[string]any{"credential": decision.name, "error": errApply.Error()})
			} else {
				report.Action = actionEnabled
				report.Managed = false
				report.FileDisabled = false
				s.bumpEnable()
				if cfg.dryRun() {
					report.Reason = "dry run: " + report.Reason
				} else {
					applied, observed := s.confirmRuntime(decision.report.Ref, false, 5*time.Second)
					report.RuntimeDisabled = observed
					if !applied {
						report.Action = actionError
						report.Error = "runtime credential still reports disabled after the write"
						s.bumpError()
						hostLog("error", "codex-quota-guard: enable did not reach the runtime", map[string]any{
							"credential": decision.name,
							"reason":     report.Reason,
						})
					} else {
						hostLog("info", "codex-quota-guard: credential re-enabled", map[string]any{
							"credential": decision.name,
							"reason":     report.Reason,
						})
					}
				}
			}
			s.storeReport(report.Ref, report)
			continue
		}
		report.Action = actionNone
		if report.Reason == "" {
			report.Reason = "within policy"
		}
		s.storeReport(report.Ref, report)
	}

	s.mu.Lock()
	s.lastSweep = time.Now().UTC()
	s.lastSweepError = ""
	s.lastSweepCause = cause
	s.sweeps++
	s.mu.Unlock()
}

func (s *guardState) inspect(ctx context.Context, cfg pluginConfig, entry authListEntry, ref string, hostTimeout time.Duration) credentialDecision {
	decision := credentialDecision{
		name: strings.TrimSpace(entry.Name),
		report: credentialReport{
			Ref:      ref,
			Name:     strings.TrimSpace(entry.Name),
			Email:    strings.TrimSpace(entry.Email),
			Provider: strings.TrimSpace(entry.Provider),
			Action:   actionNone,
		},
	}
	if decision.report.Provider == "" {
		decision.report.Provider = strings.TrimSpace(entry.Type)
	}

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
	document.remove(guardMetadataKey + "_scratch")
	decision.document = document
	decision.path = strings.TrimSpace(record.Path)
	decision.disabled = document.boolValue("disabled")
	decision.managed = readGuardMarker(document) != nil && document.boolValue("disabled")
	decision.report.FileDisabled = decision.disabled
	// The file flag and the runtime flag can disagree for one sweep: the host
	// applies a plugin write to its in-memory registry, while the auth file
	// watcher re-synthesizes the record from disk. Report the runtime view when
	// it is available and fall back to what the file says.
	runtimeDisabled := decision.disabled || entry.Disabled
	if runtime, errRuntime := hostAuthGetRuntime(ref, hostTimeout); errRuntime == nil {
		runtimeDisabled = runtime.Auth.Disabled || strings.EqualFold(strings.TrimSpace(runtime.Auth.Status), "disabled")
	}
	decision.report.RuntimeDisabled = runtimeDisabled
	decision.report.Managed = decision.managed
	decision.token = document.stringValue("access_token")
	decision.account = document.stringValue("account_id")

	if decision.disabled && !decision.managed {
		decision.report.Action = actionSkipped
		decision.report.Reason = "disabled outside this plugin; left untouched"
		return decision
	}
	if decision.path == "" {
		decision.report.Action = actionSkipped
		decision.report.Reason = "credential has no backing file; the guard only manages file backed credentials"
		return decision
	}
	if strings.TrimSpace(decision.token) == "" {
		decision.report.Action = actionSkipped
		decision.report.Reason = "credential has no access token"
		return decision
	}

	quota, errQuota := fetchQuota(ctx, cfg, decision.token, decision.account)
	if errQuota != nil {
		decision.report.Action = actionError
		decision.report.Error = errQuota.Error()
		decision.report.Reason = "cannot read Codex quota"
		s.bumpError()
		return decision
	}
	decision.quota = quota
	decision.report.PlanType = quota.PlanType
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

	blocked, blockReason := cfg.blockedBy(quota)
	decision.blocked = blocked
	decision.report.Reason = blockReason
	if blocked {
		decision.report.Reason = blockReason
	} else if decision.disabled && decision.managed {
		recover, recoverReason := cfg.recoveredBy(quota)
		decision.recover = recover
		decision.report.Reason = recoverReason
	} else {
		decision.report.Reason = "within policy"
	}
	return decision
}

// protectLastEnabled returns the name of the single credential that must stay
// enabled when every candidate would otherwise be disabled.
func (s *guardState) protectLastEnabled(cfg pluginConfig, candidates []credentialDecision) string {
	if !cfg.keepAtLeastOne() {
		return ""
	}
	type groupState struct {
		total    int
		enabled  int
		toBlock  int
		bestName string
		best     float64
	}
	groups := make(map[string]*groupState)
	for _, candidate := range candidates {
		if candidate.report.Action == actionSkipped || candidate.report.Action == actionError {
			continue
		}
		key := strings.ToLower(candidate.report.Provider)
		group := groups[key]
		if group == nil {
			group = &groupState{best: -1}
			groups[key] = group
		}
		group.total++
		if candidate.disabled {
			continue
		}
		group.enabled++
		remaining := -1.0
		if candidate.quota.Primary.Present {
			remaining = candidate.quota.Primary.RemainingPercent
		}
		if remaining > group.best {
			group.best = remaining
			group.bestName = candidate.name
		}
		if candidate.blocked {
			group.toBlock++
		}
	}
	for _, group := range groups {
		if group.toBlock > 0 && group.toBlock >= group.enabled && group.bestName != "" {
			return group.bestName
		}
	}
	return ""
}

func (s *guardState) disableCredential(cfg pluginConfig, decision credentialDecision) error {
	remaining := decision.quota.Primary.RemainingPercent
	marker := guardMarker{
		Managed:          true,
		DisabledAt:       time.Now().UTC().Format(time.RFC3339),
		Reason:           decision.report.Reason,
		RemainingPercent: &remaining,
		PluginVersion:    pluginVersion,
	}
	if !decision.quota.Primary.ResetAt.IsZero() {
		marker.WindowResetAt = decision.quota.Primary.ResetAt.Format(time.RFC3339)
	}
	if decision.quota.PlanType != "" {
		marker.Reason = fmt.Sprintf("%s (plan %s)", marker.Reason, decision.quota.PlanType)
	}
	rawMarker, errMarker := json.Marshal(marker)
	if errMarker != nil {
		return errMarker
	}
	document := decision.document
	document.setBool("disabled", true)
	document.setRaw(guardMetadataKey, rawMarker)
	if cfg.dryRun() {
		return nil
	}
	payload, errMarshal := document.marshal()
	if errMarshal != nil {
		return errMarshal
	}
	// The credential file is written directly on purpose. host.auth.save
	// rebuilds the runtime record from the file payload without applying the
	// disabled flag, and the host then persists that record back to disk, which
	// would immediately clear the disable. Writing the file lets the auth
	// directory watcher synthesize the record with the disabled flag applied.
	return writeAuthFileAtomic(decision.path, payload)
}

func (s *guardState) enableCredential(cfg pluginConfig, decision credentialDecision) error {
	document := decision.document
	document.setBool("disabled", false)
	document.remove(guardMetadataKey)
	if cfg.dryRun() {
		return nil
	}
	payload, errMarshal := document.marshal()
	if errMarshal != nil {
		return errMarshal
	}
	return writeAuthFileAtomic(decision.path, payload)
}

// confirmRuntime polls the runtime credential record until it matches the state
// the guard just wrote, so a status page never claims a change that the host did
// not actually apply.
func (s *guardState) confirmRuntime(ref string, wantDisabled bool, budget time.Duration) (bool, bool) {
	deadline := time.Now().Add(budget)
	observed := !wantDisabled
	for {
		if runtime, errRuntime := hostAuthGetRuntime(ref, 10*time.Second); errRuntime == nil {
			observed = runtime.Auth.Disabled || strings.EqualFold(strings.TrimSpace(runtime.Auth.Status), "disabled")
			if observed == wantDisabled {
				return true, observed
			}
		}
		if time.Now().After(deadline) {
			return false, observed
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func readGuardMarker(document *authDocument) *guardMarker {
	raw := document.rawValue(guardMetadataKey)
	if len(raw) == 0 {
		return nil
	}
	var marker guardMarker
	if errUnmarshal := json.Unmarshal(raw, &marker); errUnmarshal != nil {
		return nil
	}
	if !marker.Managed {
		return nil
	}
	return &marker
}

func (cfg pluginConfig) blockedBy(quota quotaSnapshot) (bool, string) {
	if quota.Primary.Present && quota.Primary.RemainingPercent <= cfg.DisableBelowRemainingPercent {
		return true, fmt.Sprintf("5h window remaining %s <= %s", formatPercent(quota.Primary.RemainingPercent), formatPercent(cfg.DisableBelowRemainingPercent))
	}
	if cfg.enforceWeekly() && quota.Weekly.Present && quota.Weekly.RemainingPercent <= cfg.WeeklyDisableBelowPercent {
		return true, fmt.Sprintf("7d window remaining %s <= %s", formatPercent(quota.Weekly.RemainingPercent), formatPercent(cfg.WeeklyDisableBelowPercent))
	}
	return false, "within policy"
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

func (s *guardState) storeReport(ref string, report credentialReport) {
	s.mu.Lock()
	if s.reports == nil {
		s.reports = make(map[string]credentialReport)
	}
	s.reports[ref] = report
	s.mu.Unlock()
}

func (s *guardState) recordSweepFailure(cause, message string) {
	s.mu.Lock()
	s.lastSweep = time.Now().UTC()
	s.lastSweepError = message
	s.lastSweepCause = cause
	s.sweeps++
	s.errorCount++
	s.mu.Unlock()
}

func (s *guardState) bumpError() {
	s.mu.Lock()
	s.errorCount++
	s.mu.Unlock()
}

func (s *guardState) bumpDisable() {
	s.mu.Lock()
	s.disableActions++
	s.mu.Unlock()
}

func (s *guardState) bumpEnable() {
	s.mu.Lock()
	s.enableActions++
	s.mu.Unlock()
}

type statusSnapshot struct {
	PluginVersion  string
	Config         pluginConfig
	Running        bool
	StartedAt      time.Time
	LastSweep      time.Time
	LastSweepError string
	LastSweepCause string
	Sweeps         int64
	Disables       int64
	Enables        int64
	Errors         int64
	NextSweep      time.Time
	Reports        []credentialReport
}

func (s *guardState) snapshot() statusSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	reports := make([]credentialReport, 0, len(s.reports))
	for _, report := range s.reports {
		reports = append(reports, report)
	}
	sort.Slice(reports, func(i, j int) bool { return reports[i].Name < reports[j].Name })
	out := statusSnapshot{
		PluginVersion:  pluginVersion,
		Config:         s.cfg,
		Running:        s.running,
		StartedAt:      s.startedAt,
		LastSweep:      s.lastSweep,
		LastSweepError: s.lastSweepError,
		LastSweepCause: s.lastSweepCause,
		Sweeps:         s.sweeps,
		Disables:       s.disableActions,
		Enables:        s.enableActions,
		Errors:         s.errorCount,
		Reports:        reports,
	}
	if !s.lastSweep.IsZero() {
		out.NextSweep = s.lastSweep.Add(s.cfg.pollInterval())
	}
	return out
}
