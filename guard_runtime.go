package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	errGuardNotRunning     = errors.New("codex-quota-guard is not running")
	errSweepInProgress     = errors.New("a quota sweep is already running")
	errWarmupInProgress    = errors.New("warm-up is already running")
	errWarmupDisabled      = errors.New("warm-up is disabled")
	errCredentialNotFound  = errors.New("credential was not found")
	errCredentialAmbiguous = errors.New("credential reference is ambiguous")
	errCredentialManual    = errors.New("credential is disabled outside this plugin")
	errNoManagedBlock      = errors.New("credential has no active plugin-owned block")
)

type guardState struct {
	mu sync.RWMutex

	cfg     pluginConfig
	reports map[string]credentialReport
	state   persistentState

	stateStore    *stateFileStore
	statePath     string
	stateReadOnly bool
	stateError    string
	stateSavedAt  time.Time
	stateSaves    int64
	persistMu     sync.Mutex

	lastSweep      time.Time
	lastSweepError string
	lastSweepCause string
	sweepRunning   bool
	sweeps         int64
	disableActions int64
	enableActions  int64
	errorCount     int64

	cancel      context.CancelFunc
	actionCtx   context.Context
	actionsOpen bool
	running     bool
	startedAt   time.Time
	workerWG    sync.WaitGroup
	sweepMu     sync.Mutex
	warmupMu    sync.Mutex

	warmupRunning     bool
	warmupLastRunAt   time.Time
	warmupLastSource  string
	warmupLastResults int
	warmupLastSuccess int
	warmupLastSkipped int
	warmupLastError   string
}

func newGuardState() *guardState {
	cfg := defaultConfig()
	return &guardState{
		cfg:        cfg,
		reports:    make(map[string]credentialReport),
		state:      newPersistentState(),
		stateStore: newStateFileStore(cfg.StateFile),
		statePath:  cfg.StateFile,
	}
}

func clonePluginConfig(source pluginConfig) pluginConfig {
	out := source
	out.Providers = append([]string(nil), source.Providers...)
	out.WarmupSchedules = append([]warmupSchedule(nil), source.WarmupSchedules...)
	return out
}

func (s *guardState) applyConfig(cfg pluginConfig) {
	s.stop()
	cfg = clonePluginConfig(cfg)
	store := newStateFileStore(cfg.StateFile)
	state, errLoad := store.load()

	s.mu.Lock()
	s.cfg = cfg
	s.stateStore = store
	s.statePath = cfg.StateFile
	s.stateReadOnly = errLoad != nil
	s.stateError = ""
	if errLoad != nil {
		s.state = newPersistentState()
		s.stateError = errLoad.Error()
		s.reports = make(map[string]credentialReport)
	} else {
		s.state = clonePersistentState(state)
		s.stateSavedAt = state.UpdatedAt
		s.lastSweep = state.LastSweepAt
		s.lastSweepCause = state.LastSweepCause
		s.lastSweepError = state.LastSweepError
		s.warmupLastRunAt = state.Warmup.LastRunAt
		s.warmupLastSource = state.Warmup.LastSource
		s.warmupLastResults = state.Warmup.Results
		s.warmupLastSuccess = state.Warmup.Success
		s.warmupLastSkipped = state.Warmup.Skipped
		s.warmupLastError = state.Warmup.LastError
		s.reports = reportsFromPersistentState(s.state)
	}
	s.mu.Unlock()

	if errLoad != nil {
		hostLog("error", "codex-quota-guard: persistent state is read-only", map[string]any{
			"path":  cfg.StateFile,
			"error": errLoad.Error(),
		})
	}
	s.restart()
}

func (s *guardState) config() pluginConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return clonePluginConfig(s.cfg)
}

func (s *guardState) stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.actionCtx = nil
	s.actionsOpen = false
	s.running = false
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.workerWG.Wait()
}

func (s *guardState) restart() {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancel = cancel
	s.actionCtx = ctx
	s.actionsOpen = true
	s.running = true
	s.startedAt = time.Now().UTC()
	cfg := clonePluginConfig(s.cfg)
	s.mu.Unlock()

	s.workerWG.Add(1)
	go func() {
		defer s.workerWG.Done()
		s.loop(ctx, cfg)
	}()

	if cfg.warmupEnabled() && len(cfg.WarmupSchedules) > 0 {
		s.workerWG.Add(1)
		go func() {
			defer s.workerWG.Done()
			s.warmupLoop(ctx, cfg)
		}()
	}
}

func (s *guardState) loop(ctx context.Context, cfg pluginConfig) {
	if delay := cfg.initialDelay(); delay > 0 {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
	s.sweepExclusive(ctx, "startup")

	ticker := time.NewTicker(cfg.pollInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweepExclusive(ctx, "interval")
		}
	}
}

func (s *guardState) sweepExclusive(ctx context.Context, cause string) {
	if ctx.Err() != nil {
		return
	}
	s.sweepMu.Lock()
	defer s.sweepMu.Unlock()
	s.runSweepLocked(ctx, cause)
}

func (s *guardState) runSweepLocked(ctx context.Context, cause string) {
	if ctx.Err() != nil {
		return
	}
	s.mu.Lock()
	s.sweepRunning = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.sweepRunning = false
		s.mu.Unlock()
	}()
	s.sweep(ctx, cause)
}

func (s *guardState) startSweepAction() error {
	if !s.sweepMu.TryLock() {
		return errSweepInProgress
	}
	s.mu.Lock()
	if !s.actionsOpen || s.actionCtx == nil {
		s.mu.Unlock()
		s.sweepMu.Unlock()
		return errGuardNotRunning
	}
	ctx := s.actionCtx
	s.workerWG.Add(1)
	s.mu.Unlock()
	go func() {
		defer s.workerWG.Done()
		defer s.sweepMu.Unlock()
		s.runSweepLocked(ctx, "manual")
	}()
	return nil
}

func (s *guardState) startWarmupAction() error {
	cfg := s.config()
	if !cfg.warmupEnabled() {
		return errWarmupDisabled
	}
	if !s.warmupMu.TryLock() {
		return errWarmupInProgress
	}
	s.mu.Lock()
	if !s.actionsOpen || s.actionCtx == nil {
		s.mu.Unlock()
		s.warmupMu.Unlock()
		return errGuardNotRunning
	}
	ctx := s.actionCtx
	s.workerWG.Add(1)
	s.mu.Unlock()
	go func() {
		defer s.workerWG.Done()
		defer s.warmupMu.Unlock()
		if _, errRun := s.runWarmupLocked(ctx, cfg, "", "manual"); errRun != nil && ctx.Err() == nil {
			hostLog("error", "codex-quota-guard: manual warm-up failed", map[string]any{
				"error": errRun.Error(),
			})
		}
	}()
	return nil
}

func (s *guardState) persistState(reason string) error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()

	s.mu.Lock()
	if s.stateReadOnly || s.stateStore == nil {
		err := fmt.Errorf("persistent state is read-only: %s", s.stateError)
		s.mu.Unlock()
		return err
	}
	state := clonePersistentState(s.state)
	store := s.stateStore
	path := s.statePath
	s.mu.Unlock()

	if errSave := store.save(state); errSave != nil {
		s.mu.Lock()
		s.stateError = errSave.Error()
		s.errorCount++
		s.mu.Unlock()
		hostLog("error", "codex-quota-guard: persistent state save failed", map[string]any{
			"path":   path,
			"reason": reason,
			"error":  errSave.Error(),
		})
		return errSave
	}

	s.mu.Lock()
	s.stateError = ""
	s.stateSavedAt = time.Now().UTC()
	s.stateSaves++
	s.mu.Unlock()
	return nil
}

func (s *guardState) setSweepError(cause, message string) {
	s.mu.Lock()
	s.lastSweep = time.Now().UTC()
	s.lastSweepError = message
	s.lastSweepCause = cause
	s.state.LastSweepAt = s.lastSweep
	s.state.LastSweepError = message
	s.state.LastSweepCause = cause
	s.sweeps++
	s.errorCount++
	s.mu.Unlock()
	_ = s.persistState("sweep-error")
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

func (s *guardState) storeReport(key string, report credentialReport) {
	s.mu.Lock()
	if s.reports == nil {
		s.reports = make(map[string]credentialReport)
	}
	if key = strings.ToLower(strings.TrimSpace(key)); key != "" {
		s.reports[key] = report
	}
	s.mu.Unlock()
}

func reportsFromPersistentState(state persistentState) map[string]credentialReport {
	reports := make(map[string]credentialReport)
	latestByCredential := make(map[string]string)
	for _, item := range state.Credentials {
		if item == nil {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(item.Key))
		if key == "" {
			key = credentialStateKey(item.AuthID, item.AuthIndex, item.Name)
		}
		if key == "" {
			continue
		}
		report := credentialReport{
			Ref:               firstNonEmpty(item.Key, key),
			AuthID:            item.AuthID,
			AuthIndex:         item.AuthIndex,
			Name:              item.Name,
			Email:             item.Email,
			Provider:          item.Provider,
			PlanType:          item.ObservedQuota.PlanType,
			Managed:           item.DisabledByPlugin,
			ManualDisabled:    item.ManualDisabled,
			ManualEnabled:     item.ManualEnabled,
			RateLimitResetAt:  blockResetAt(item.RateLimitBlock),
			RateLimitWindow:   blockWindow(item.RateLimitBlock),
			WarmupLastRunAt:   item.Warmup.LastRunAt,
			WarmupLastSuccess: item.Warmup.LastSuccess,
			WarmupLastStatus:  item.Warmup.LastStatus,
			WarmupLastError:   item.Warmup.LastError,
			Action:            item.LastAction,
			Reason:            item.LastReason,
			Error:             item.LastError,
			CheckedAt:         item.LastActionAt,
		}
		report.QuotaBlocked = item.QuotaBlock.active(time.Now().UTC())
		report.RateLimitBlocked = item.RateLimitBlock.active(time.Now().UTC())
		if item.ObservedQuota.PrimaryRemaining != nil {
			used := 100 - *item.ObservedQuota.PrimaryRemaining
			report.PrimaryUsed = &used
		}
		if item.ObservedQuota.WeeklyRemaining != nil {
			used := 100 - *item.ObservedQuota.WeeklyRemaining
			report.WeeklyUsed = &used
		}
		identity := strings.ToLower(firstNonEmpty(item.Name, item.AuthID, item.AuthIndex, item.Key))
		if previousKey, ok := latestByCredential[identity]; ok {
			if !report.CheckedAt.After(reports[previousKey].CheckedAt) {
				continue
			}
			delete(reports, previousKey)
		}
		reports[key] = report
		latestByCredential[identity] = key
	}
	return reports
}

type statusSnapshot struct {
	PluginVersion  string
	Config         pluginConfig
	Running        bool
	SweepRunning   bool
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
	StatePath      string
	StateError     string
	StateReadOnly  bool
	StateSavedAt   time.Time
	StateSaves     int64
	WarmupRunning  bool
	WarmupLastRun  time.Time
	WarmupSource   string
	WarmupResults  int
	WarmupSuccess  int
	WarmupSkipped  int
	WarmupError    string
	NextWarmup     time.Time
	NextWarmupID   string
}

func (s *guardState) snapshot() statusSnapshot {
	s.mu.RLock()
	reports := make([]credentialReport, 0, len(s.reports))
	for _, report := range s.reports {
		reports = append(reports, report)
	}
	cfg := clonePluginConfig(s.cfg)
	out := statusSnapshot{
		PluginVersion:  pluginVersion,
		Config:         cfg,
		Running:        s.running,
		SweepRunning:   s.sweepRunning,
		StartedAt:      s.startedAt,
		LastSweep:      s.lastSweep,
		LastSweepError: s.lastSweepError,
		LastSweepCause: s.lastSweepCause,
		Sweeps:         s.sweeps,
		Disables:       s.disableActions,
		Enables:        s.enableActions,
		Errors:         s.errorCount,
		StatePath:      s.statePath,
		StateError:     s.stateError,
		StateReadOnly:  s.stateReadOnly,
		StateSavedAt:   s.stateSavedAt,
		StateSaves:     s.stateSaves,
		WarmupRunning:  s.warmupRunning,
		WarmupLastRun:  s.warmupLastRunAt,
		WarmupSource:   s.warmupLastSource,
		WarmupResults:  s.warmupLastResults,
		WarmupSuccess:  s.warmupLastSuccess,
		WarmupSkipped:  s.warmupLastSkipped,
		WarmupError:    s.warmupLastError,
	}
	s.mu.RUnlock()
	sort.Slice(reports, func(i, j int) bool {
		return strings.ToLower(reports[i].Name) < strings.ToLower(reports[j].Name)
	})
	out.Reports = reports
	if !out.LastSweep.IsZero() {
		out.NextSweep = out.LastSweep.Add(cfg.pollInterval())
	}
	if next, ok := nextWarmupRun(cfg, time.Now()); ok {
		out.NextWarmup = next.Time
		out.NextWarmupID = next.ID
	}
	return out
}
