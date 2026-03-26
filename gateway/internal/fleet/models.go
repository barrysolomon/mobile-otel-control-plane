package fleet

import (
	"time"
)

// FleetEvent represents a fleet-relevant event from a device or backend.
type FleetEvent struct {
	ID             string    `json:"id"`
	DeviceID       string    `json:"device_id"`
	CohortID       string    `json:"cohort_id,omitempty"`
	TriggerType    string    `json:"trigger_type"`
	AttributesJSON string    `json:"attributes_json,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	DeviceTimestamp time.Time `json:"device_timestamp,omitempty"`
	ClockOffsetMs  int64     `json:"clock_offset_ms,omitempty"`
	Processed      bool      `json:"processed"`
}

// FleetAlert is the payload delivered to devices via push channels.
type FleetAlert struct {
	Type              string        `json:"type"` // "fleet_alert"
	AlertID           string        `json:"alert_id"`
	CascadeChainID    string        `json:"cascade_chain_id"`
	Hop               int           `json:"hop"`
	Priority          int           `json:"priority"` // 0=critical, 1=high, 2=medium, 3=low
	SourceTrigger     string        `json:"source_trigger"`
	SourceCohort      string        `json:"source_cohort"`
	SourceDeviceCount int           `json:"source_device_count"`
	Actions           []AlertAction `json:"actions"`
	ExpiresAt         time.Time     `json:"expires_at"`
	Signature         string        `json:"signature"`
	IssuedAt          time.Time     `json:"issued_at"`
	Truncated         bool          `json:"truncated,omitempty"`
	TotalActions      int           `json:"total_actions,omitempty"`
}

// AlertAction is a single action within a fleet alert.
type AlertAction struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// CascadeChain tracks a full cascade lifecycle.
type CascadeChain struct {
	ChainID              string     `json:"chain_id"`
	RootTriggerType      string     `json:"root_trigger_type"`
	RootCohortID         string     `json:"root_cohort_id"`
	RootDeviceID         string     `json:"root_device_id,omitempty"`
	StartedAt            time.Time  `json:"started_at"`
	EndedAt              *time.Time `json:"ended_at,omitempty"`
	Status               string     `json:"status"` // active, completed, halted, killed, shadow
	MaxHopReached        int        `json:"max_hop_reached"`
	TotalDevicesAffected int        `json:"total_devices_affected"`
	KillSwitchTripped    bool       `json:"kill_switch_tripped"`
	ChainTimeoutMin      int        `json:"chain_timeout_minutes"`
	HopTimeoutMin        int        `json:"hop_timeout_minutes"`
	LastHopAt            time.Time  `json:"last_hop_at"`
	WorkflowID           string     `json:"workflow_id"`
}

// CascadeHop tracks a single hop within a cascade chain.
type CascadeHop struct {
	HopID            string    `json:"hop_id"`
	ChainID          string    `json:"chain_id"`
	HopNumber        int       `json:"hop_number"`
	TriggerType      string    `json:"trigger_type"`
	SourceCohortID   string    `json:"source_cohort_id"`
	TargetCohortID   string    `json:"target_cohort_id"`
	DevicesTargeted  int       `json:"devices_targeted"`
	DevicesDelivered int       `json:"devices_delivered"`
	ActionsJSON      string    `json:"actions_json"`
	BreakersJSON     string    `json:"breakers_evaluated_json"`
	CreatedAt        time.Time `json:"created_at"`
}

// CascadeDeviceEntry tracks per-device delivery within a cascade.
type CascadeDeviceEntry struct {
	ChainID             string     `json:"chain_id"`
	HopID               string     `json:"hop_id"`
	DeviceID            string     `json:"device_id"`
	DeliveryChannel     string     `json:"delivery_channel"` // websocket, fcm, poll
	DeliveredAt         *time.Time `json:"delivered_at,omitempty"`
	AcknowledgedAt      *time.Time `json:"acknowledged_at,omitempty"`
	ReactionCompletedAt *time.Time `json:"reaction_completed_at,omitempty"`
}

// FleetRule defines a fleet-level detection rule.
type FleetRule struct {
	ID         string `json:"id"`
	WorkflowID string `json:"workflow_id"`
	RuleType   string `json:"rule_type"` // fleet_threshold, fleet_rate, fleet_absence
	CohortID   string `json:"cohort_id"`
	ConfigJSON string `json:"config_json"`
	Enabled    bool   `json:"enabled"`
	Priority   int    `json:"priority"` // 0=critical, 1=high, 2=medium, 3=low
}

// ThresholdConfig is the parsed config for a fleet_threshold rule.
type ThresholdConfig struct {
	TriggerType   string `json:"trigger_type"`
	Threshold     int    `json:"threshold"`
	WindowMinutes int    `json:"window_minutes"`
}

// RateConfig is the parsed config for a fleet_rate rule.
type RateConfig struct {
	TriggerType       string  `json:"trigger_type"`
	BaselineWindowMin int     `json:"baseline_window_minutes"`
	CurrentWindowMin  int     `json:"current_window_minutes"`
	Factor            float64 `json:"factor"`
}

// AbsenceConfig is the parsed config for a fleet_absence rule.
type AbsenceConfig struct {
	MinSilentDevices int `json:"min_silent_devices"`
	WindowMinutes    int `json:"window_minutes"`
}

// CircuitBreakerConfig defines cascade safety limits.
type CircuitBreakerConfig struct {
	MaxCascadeDepth    int     `json:"max_cascade_depth"`
	CooldownMinutes    int     `json:"cooldown_minutes"`
	MaxPercentAffected float64 `json:"max_percent_affected"`
	MaxAbsoluteDevices int     `json:"max_absolute_devices"`
	BudgetWindowMin    int     `json:"budget_window_minutes"`
	MaxAlertsPerHour   int     `json:"max_alerts_per_hour"`
	ChainTimeoutMin    int     `json:"chain_timeout_minutes"`
	HopTimeoutMin      int     `json:"hop_timeout_minutes"`
}

// FleetWorkflowAudit records who published/changed fleet workflows.
type FleetWorkflowAudit struct {
	ID                int       `json:"id"`
	WorkflowID        string    `json:"workflow_id"`
	Action            string    `json:"action"`
	Actor             string    `json:"actor"`
	PreviousJSON      string    `json:"previous_version_json,omitempty"`
	NewJSON           string    `json:"new_version_json,omitempty"`
	DiffSummary       string    `json:"diff_summary,omitempty"`
	CascadeImpactJSON string    `json:"cascade_impact_json,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// Priority constants for fleet alerts.
const (
	PriorityCritical = 0
	PriorityHigh     = 1
	PriorityMedium   = 2
	PriorityLow      = 3
)

// MaxAlertPayloadBytes is the maximum alert payload size (3.5KB for FCM headroom).
const MaxAlertPayloadBytes = 3500

// MaxActionsPerAlert is the max actions before truncation.
const MaxActionsPerAlert = 10
