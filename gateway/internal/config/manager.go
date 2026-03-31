// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"fmt"

	"github.com/mobile-observability/gateway/internal/db"
)

type Manager struct {
	db *db.Database
}

// DSLConfig represents the compiled workflow configuration for devices
type DSLConfig struct {
	Version   int           `json:"version"`
	Limits    Limits        `json:"limits"`
	Workflows []Workflow    `json:"workflows"`
}

type Limits struct {
	DiskMB         int `json:"diskMb"`
	RAMEvents      int `json:"ramEvents"`
	RetentionHours int `json:"retentionHours"`
}

type Workflow struct {
	ID      string   `json:"id"`
	Enabled bool     `json:"enabled"`
	Trigger Trigger  `json:"trigger"`
	Actions []Action `json:"actions"`
}

type Trigger struct {
	Any []TriggerCondition `json:"any,omitempty"`
	All []TriggerCondition `json:"all,omitempty"`
}

type TriggerCondition struct {
	Event string      `json:"event,omitempty"`
	Where []Predicate `json:"where,omitempty"`
}

type Predicate struct {
	Attr  string      `json:"attr"`
	Op    string      `json:"op"` // ==, !=, >, >=, <, <=, contains, regex
	Value interface{} `json:"value"`
}

type Action struct {
	Type             string  `json:"type"` // annotate_trigger, flush_window, set_sampling
	TriggerID        string  `json:"trigger_id,omitempty"`
	Reason           string  `json:"reason,omitempty"`
	Minutes          int     `json:"minutes,omitempty"`
	Scope            string  `json:"scope,omitempty"` // session, device
	Rate             float64 `json:"rate,omitempty"`
	DurationMinutes  int     `json:"duration_minutes,omitempty"`
}

// ============================================================================
// DSL v2 types (state-machine-based)
// ============================================================================

type DSLConfigV2 struct {
	Version      int              `json:"version"`
	BufferConfig BufferConfig     `json:"buffer_config"`
	Targeting    *TargetingRules  `json:"targeting,omitempty"`
	Workflows    []WorkflowV2     `json:"workflows"`
}

type BufferConfig struct {
	RAMEvents      int    `json:"ram_events"`
	DiskMB         int    `json:"disk_mb"`
	RetentionHours int    `json:"retention_hours"`
	Strategy       string `json:"strategy"` // overwrite_oldest, stop_recording
}

type TargetingRules struct {
	Platform         string            `json:"platform,omitempty"`          // android, ios
	AppVersionRange  string            `json:"app_version_range,omitempty"` // semver range
	OSVersionRange   string            `json:"os_version_range,omitempty"`
	DeviceModels     []string          `json:"device_models,omitempty"`     // glob patterns
	DeviceGroup      string            `json:"device_group,omitempty"`
	CustomAttributes map[string]string `json:"custom_attributes,omitempty"`
}

type WorkflowV2 struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Enabled      bool     `json:"enabled"`
	Priority     int      `json:"priority"`
	InitialState string   `json:"initial_state"`
	States       []State  `json:"states"`
}

type State struct {
	ID        string     `json:"id"`
	Matchers  []Matcher  `json:"matchers"`
	OnMatch   MatchResult `json:"on_match"`
	OnTimeout *TimeoutResult `json:"on_timeout,omitempty"`
}

type Matcher struct {
	Type     string                 `json:"type"`
	Config   map[string]interface{} `json:"config"`
	Where    []PredicateV2          `json:"where,omitempty"`
	Combine  string                 `json:"combine,omitempty"` // any, all
	Children []Matcher              `json:"children,omitempty"`
}

type PredicateV2 struct {
	Attr  string      `json:"attr"`
	Op    string      `json:"op"`
	Value interface{} `json:"value,omitempty"`
}

type MatchResult struct {
	Actions      []ActionV2 `json:"actions"`
	TransitionTo string     `json:"transition_to,omitempty"`
}

type TimeoutResult struct {
	AfterMs      int        `json:"after_ms"`
	Actions      []ActionV2 `json:"actions"`
	TransitionTo string     `json:"transition_to,omitempty"`
}

type ActionV2 struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// GraphWorkflow represents the UI's React Flow format (for editing)
type GraphWorkflow struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Enabled      bool        `json:"enabled"`
	EntryNodeID  string      `json:"entryNodeId"`
	Nodes        []GraphNode `json:"nodes"`
	Edges        []GraphEdge `json:"edges"`
}

type GraphNode struct {
	ID   string                 `json:"id"`
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

type GraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

func NewManager(database *db.Database) *Manager {
	return &Manager{db: database}
}

func (m *Manager) GetActiveConfig() (*DSLConfig, error) {
	cv, err := m.db.GetActiveConfig()
	if err != nil {
		return nil, err
	}

	if cv == nil {
		// Return default config if none exists
		return m.getDefaultConfig(), nil
	}

	var dslConfig DSLConfig
	if err := json.Unmarshal([]byte(cv.DSLJSON), &dslConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DSL config: %w", err)
	}

	return &dslConfig, nil
}

func (m *Manager) GetActiveConfigV2() (*DSLConfigV2, error) {
	cv, err := m.db.GetActiveConfig()
	if err != nil {
		return nil, err
	}

	if cv == nil || cv.DSLV2JSON == "" {
		return m.getDefaultConfigV2(), nil
	}

	var dslConfig DSLConfigV2
	if err := json.Unmarshal([]byte(cv.DSLV2JSON), &dslConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DSL v2 config: %w", err)
	}

	return &dslConfig, nil
}

func (m *Manager) PublishWorkflow(graphJSON, dslJSON, publishedBy string) (*db.ConfigVersion, error) {
	return m.PublishWorkflowV2(graphJSON, dslJSON, "", publishedBy)
}

func (m *Manager) PublishWorkflowV2(graphJSON, dslJSON, dslV2JSON, publishedBy string) (*db.ConfigVersion, error) {
	// Validate DSL v1 JSON
	var dslConfig DSLConfig
	if err := json.Unmarshal([]byte(dslJSON), &dslConfig); err != nil {
		return nil, fmt.Errorf("invalid DSL JSON: %w", err)
	}

	// Validate DSL v2 JSON if provided
	if dslV2JSON != "" {
		var dslConfigV2 DSLConfigV2
		if err := json.Unmarshal([]byte(dslV2JSON), &dslConfigV2); err != nil {
			return nil, fmt.Errorf("invalid DSL v2 JSON: %w", err)
		}
	}

	// Validate Graph JSON
	var graphWorkflows []GraphWorkflow
	if err := json.Unmarshal([]byte(graphJSON), &graphWorkflows); err != nil {
		return nil, fmt.Errorf("invalid Graph JSON: %w", err)
	}

	// Publish to database
	return m.db.PublishConfig(graphJSON, dslJSON, dslV2JSON, publishedBy)
}

func (m *Manager) RollbackToVersion(version int) error {
	return m.db.RollbackToVersion(version)
}

func (m *Manager) ListVersions(limit int) ([]db.ConfigVersion, error) {
	return m.db.ListVersions(limit)
}

func (m *Manager) getDefaultConfigV2() *DSLConfigV2 {
	return &DSLConfigV2{
		Version: 2,
		BufferConfig: BufferConfig{
			RAMEvents:      5000,
			DiskMB:         50,
			RetentionHours: 24,
			Strategy:       "overwrite_oldest",
		},
		Workflows: []WorkflowV2{
			{
				ID:           "default-crash",
				Name:         "Default Crash Handler",
				Enabled:      true,
				Priority:     1,
				InitialState: "watching",
				States: []State{
					{
						ID: "watching",
						Matchers: []Matcher{
							{
								Type:   "crash",
								Config: map[string]interface{}{},
							},
						},
						OnMatch: MatchResult{
							Actions: []ActionV2{
								{
									Type: "annotate",
									Config: map[string]interface{}{
										"trigger_id": "default-crash",
										"reason":     "crash detected",
									},
								},
								{
									Type: "flush_buffer",
									Config: map[string]interface{}{
										"minutes": 5,
										"scope":   "session",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (m *Manager) getDefaultConfig() *DSLConfig {
	return &DSLConfig{
		Version: 1,
		Limits: Limits{
			DiskMB:         50,
			RAMEvents:      5000,
			RetentionHours: 24,
		},
		Workflows: []Workflow{
			{
				ID:      "default-crash",
				Enabled: true,
				Trigger: Trigger{
					Any: []TriggerCondition{
						{Event: "crash_marker"},
					},
				},
				Actions: []Action{
					{
						Type:      "annotate_trigger",
						TriggerID: "default-crash",
						Reason:    "crash detected",
					},
					{
						Type:    "flush_window",
						Minutes: 5,
						Scope:   "session",
					},
				},
			},
		},
	}
}
