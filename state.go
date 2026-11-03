package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const persistentStateSchemaVersion = 1

const (
	blockKindQuota5h   = "quota_5h"
	blockKindQuota7d   = "quota_7d"
	blockKindRateLimit = "rate_limit"
	blockKindManual    = "manual_disabled"
	blockStatusActive  = "active"
	blockStatusCleared = "cleared"
)

// managedBlock is durable, non-secret state for one reason that keeps a
// credential out of scheduling. Times are stored in UTC and reset timestamps
// are retained so a restart cannot accidentally re-enable a credential early.
type managedBlock struct {
	Kind      string    `json:"kind"`
	Reason    string    `json:"reason,omitempty"`
	Window    string    `json:"window,omitempty"`
	Status    string    `json:"status"`
	BlockedAt time.Time `json:"blocked_at"`
	ResetAt   time.Time `json:"reset_at,omitempty"`
}

func (b *managedBlock) active(now time.Time) bool {
	if b == nil || b.Status == blockStatusCleared {
		return false
	}
	return b.ResetAt.IsZero() || now.Before(b.ResetAt)
}

type observedQuotaState struct {
	PlanType         string    `json:"plan_type,omitempty"`
	PrimaryRemaining *float64  `json:"primary_remaining_percent,omitempty"`
	PrimaryResetAt   time.Time `json:"primary_reset_at,omitempty"`
	WeeklyRemaining  *float64  `json:"weekly_remaining_percent,omitempty"`
	WeeklyResetAt    time.Time `json:"weekly_reset_at,omitempty"`
	ObservedAt       time.Time `json:"observed_at,omitempty"`
}

type warmupCredentialState struct {
	LastRunAt   time.Time `json:"last_run_at,omitempty"`
	LastRunKey  string    `json:"last_run_key,omitempty"`
	LastSource  string    `json:"last_source,omitempty"`
	LastModel   string    `json:"last_model,omitempty"`
	LastStatus  int       `json:"last_status,omitempty"`
	LastSuccess bool      `json:"last_success,omitempty"`
	LastError   string    `json:"last_error,omitempty"`
}

type warmupSummaryState struct {
	LastRunAt  time.Time `json:"last_run_at,omitempty"`
	LastSource string    `json:"last_source,omitempty"`
	Results    int       `json:"results,omitempty"`
	Success    int       `json:"success,omitempty"`
	Skipped    int       `json:"skipped,omitempty"`
	LastError  string    `json:"last_error,omitempty"`
}

type credentialState struct {
	Key              string                `json:"key"`
	AuthID           string                `json:"auth_id,omitempty"`
	AuthIndex        string                `json:"auth_index,omitempty"`
	Name             string                `json:"name,omitempty"`
	Email            string                `json:"email,omitempty"`
	Provider         string                `json:"provider,omitempty"`
	ManualDisabled   bool                  `json:"manual_disabled,omitempty"`
	ManualEnabled    bool                  `json:"manual_enabled,omitempty"`
	DisabledByPlugin bool                  `json:"disabled_by_plugin,omitempty"`
	QuotaBlock       *managedBlock         `json:"quota_block,omitempty"`
	RateLimitBlock   *managedBlock         `json:"rate_limit_block,omitempty"`
	ObservedQuota    observedQuotaState    `json:"observed_quota,omitempty"`
	Warmup           warmupCredentialState `json:"warmup,omitempty"`
	LastAction       string                `json:"last_action,omitempty"`
	LastReason       string                `json:"last_reason,omitempty"`
	LastActionAt     time.Time             `json:"last_action_at,omitempty"`
	LastError        string                `json:"last_error,omitempty"`
}

func (s *credentialState) activeBlocks(now time.Time) []*managedBlock {
	if s == nil {
		return nil
	}
	out := make([]*managedBlock, 0, 2)
	if s.QuotaBlock.active(now) {
		out = append(out, s.QuotaBlock)
	}
	if s.RateLimitBlock.active(now) {
		out = append(out, s.RateLimitBlock)
	}
	return out
}

func (s *credentialState) blocked(now time.Time) bool {
	return len(s.activeBlocks(now)) > 0
}

func (s *credentialState) refreshExpired(now time.Time) bool {
	changed := false
	if s.QuotaBlock != nil && s.QuotaBlock.active(now) == false && s.QuotaBlock.Status != blockStatusCleared {
		s.QuotaBlock.Status = blockStatusCleared
		changed = true
	}
	if s.RateLimitBlock != nil && s.RateLimitBlock.active(now) == false && s.RateLimitBlock.Status != blockStatusCleared {
		s.RateLimitBlock.Status = blockStatusCleared
		changed = true
	}
	return changed
}

type persistentState struct {
	SchemaVersion  int                         `json:"schema_version"`
	UpdatedAt      time.Time                   `json:"updated_at,omitempty"`
	LastSweepAt    time.Time                   `json:"last_sweep_at,omitempty"`
	LastSweepCause string                      `json:"last_sweep_cause,omitempty"`
	LastSweepError string                      `json:"last_sweep_error,omitempty"`
	Warmup         warmupSummaryState          `json:"warmup,omitempty"`
	Credentials    map[string]*credentialState `json:"credentials"`
}

func newPersistentState() persistentState {
	return persistentState{
		SchemaVersion: persistentStateSchemaVersion,
		Credentials:   make(map[string]*credentialState),
	}
}

func stateLookupKeys(authID, authIndex, name string) []string {
	keys := make([]string, 0, 3)
	seen := make(map[string]struct{}, 3)
	for _, value := range []string{authID, authIndex, name} {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys
}

func credentialStateKey(authID, authIndex, name string) string {
	for _, value := range []string{authID, authIndex, name} {
		if value = strings.TrimSpace(value); value != "" {
			return strings.ToLower(value)
		}
	}
	return ""
}

func (p *persistentState) lookup(authID, authIndex, name string) *credentialState {
	for _, key := range stateLookupKeys(authID, authIndex, name) {
		if item := p.Credentials[key]; item != nil {
			return item
		}
	}
	authID = strings.ToLower(strings.TrimSpace(authID))
	authIndex = strings.ToLower(strings.TrimSpace(authIndex))
	name = strings.ToLower(strings.TrimSpace(name))
	for _, item := range p.Credentials {
		if item == nil {
			continue
		}
		if authID != "" && strings.EqualFold(item.AuthID, authID) {
			return item
		}
		if authIndex != "" && strings.EqualFold(item.AuthIndex, authIndex) {
			return item
		}
		if name != "" && strings.EqualFold(item.Name, name) {
			return item
		}
	}
	return nil
}

func (p *persistentState) ensure(authID, authIndex, name, email, provider string) *credentialState {
	key := credentialStateKey(authID, authIndex, name)
	if key == "" {
		return nil
	}
	if item := p.lookup(authID, authIndex, name); item != nil {
		if item.Key == "" {
			item.Key = key
		}
		return item
	}
	item := &credentialState{
		Key:       key,
		AuthID:    strings.TrimSpace(authID),
		AuthIndex: strings.TrimSpace(authIndex),
		Name:      strings.TrimSpace(name),
		Email:     strings.TrimSpace(email),
		Provider:  strings.TrimSpace(provider),
	}
	p.Credentials[key] = item
	return item
}

func (p *persistentState) sortedCredentials() []*credentialState {
	out := make([]*credentialState, 0, len(p.Credentials))
	for _, item := range p.Credentials {
		if item != nil {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func cloneCredentialState(source *credentialState) *credentialState {
	if source == nil {
		return nil
	}
	out := *source
	if source.QuotaBlock != nil {
		block := *source.QuotaBlock
		out.QuotaBlock = &block
	}
	if source.RateLimitBlock != nil {
		block := *source.RateLimitBlock
		out.RateLimitBlock = &block
	}
	if source.ObservedQuota.PrimaryRemaining != nil {
		value := *source.ObservedQuota.PrimaryRemaining
		out.ObservedQuota.PrimaryRemaining = &value
	}
	if source.ObservedQuota.WeeklyRemaining != nil {
		value := *source.ObservedQuota.WeeklyRemaining
		out.ObservedQuota.WeeklyRemaining = &value
	}
	return &out
}

func clonePersistentState(source persistentState) persistentState {
	out := persistentState{
		SchemaVersion:  source.SchemaVersion,
		UpdatedAt:      source.UpdatedAt,
		LastSweepAt:    source.LastSweepAt,
		LastSweepCause: source.LastSweepCause,
		LastSweepError: source.LastSweepError,
		Warmup:         source.Warmup,
		Credentials:    make(map[string]*credentialState, len(source.Credentials)),
	}
	for key, item := range source.Credentials {
		out.Credentials[key] = cloneCredentialState(item)
	}
	return out
}

func mergeManagedBlock(existing *managedBlock, next managedBlock, now time.Time) *managedBlock {
	if existing != nil && existing.active(now) {
		if existing.ResetAt.After(next.ResetAt) {
			next.ResetAt = existing.ResetAt
			next.Window = existing.Window
			next.Reason = existing.Reason
		}
	}
	copy := next
	copy.Status = blockStatusActive
	return &copy
}

func validatePersistentState(state *persistentState) error {
	if state.SchemaVersion == 0 {
		state.SchemaVersion = persistentStateSchemaVersion
	}
	if state.SchemaVersion != persistentStateSchemaVersion {
		return fmt.Errorf("unsupported persistent state schema %d", state.SchemaVersion)
	}
	if state.Credentials == nil {
		state.Credentials = make(map[string]*credentialState)
	}
	return nil
}
