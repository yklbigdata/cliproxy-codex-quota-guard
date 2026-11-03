package main

import (
	"encoding/json"
	"time"
)

const (
	actionDisabled = "disabled"
	actionEnabled  = "enabled"
	actionNone     = "none"
	actionSkipped  = "skipped"
	actionError    = "error"
)

// guardMarker records which policy reasons made this plugin disable a
// credential. It is stored in the credential file because host.auth.save now
// applies and persists the disabled flag atomically in CPA v8.
type guardMarker struct {
	Managed       bool           `json:"managed"`
	Reasons       []managedBlock `json:"reasons,omitempty"`
	UpdatedAt     string         `json:"updated_at,omitempty"`
	PluginVersion string         `json:"plugin_version,omitempty"`

	// Legacy v1 fields remain readable during migration.
	DisabledAt       string   `json:"disabled_at,omitempty"`
	Reason           string   `json:"reason,omitempty"`
	RemainingPercent *float64 `json:"remaining_percent,omitempty"`
	WindowResetAt    string   `json:"window_reset_at,omitempty"`
}

func readGuardMarker(document *authDocument) *guardMarker {
	if document == nil {
		return nil
	}
	raw := document.rawValue(guardMetadataKey)
	if len(raw) == 0 {
		return nil
	}
	var marker guardMarker
	if err := json.Unmarshal(raw, &marker); err != nil || !marker.Managed {
		return nil
	}
	return &marker
}

type credentialReport struct {
	Ref               string
	AuthID            string
	AuthIndex         string
	Name              string
	Email             string
	Provider          string
	PlanType          string
	RuntimeDisabled   bool
	FileDisabled      bool
	Managed           bool
	ManualDisabled    bool
	ManualEnabled     bool
	QuotaBlocked      bool
	RateLimitBlocked  bool
	PrimaryUsed       *float64
	PrimaryResetIn    int64
	WeeklyUsed        *float64
	WeeklyResetIn     int64
	RateLimitResetAt  time.Time
	RateLimitWindow   string
	WarmupLastRunAt   time.Time
	WarmupLastSuccess bool
	WarmupLastStatus  int
	WarmupLastError   string
	Action            string
	Reason            string
	Error             string
	CheckedAt         time.Time
}

type credentialDecision struct {
	entry           pluginAuthEntry
	document        *authDocument
	ref             string
	stateKey        string
	token           string
	account         string
	quota           quotaSnapshot
	state           *credentialState
	quotaBlock      *managedBlock
	rateBlock       *managedBlock
	manual          bool
	manualEnabled   bool
	managed         bool
	fileDisabled    bool
	runtimeDisabled bool
	rateActive      bool
	quotaActive     bool
	protected       bool
	report          credentialReport
}

type pluginAuthEntry = authListEntry
