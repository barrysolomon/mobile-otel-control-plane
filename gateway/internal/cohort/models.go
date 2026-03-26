package cohort

import (
	"time"
)

// Cohort represents a group of devices for fleet targeting.
type Cohort struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"` // static, dynamic, discovered
	RulesJSON       string    `json:"rules_json,omitempty"`
	SourceClusterID string    `json:"source_cluster_id,omitempty"`
	DeviceCount     int       `json:"device_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CohortRule defines the filter expression for dynamic cohorts.
type CohortRule struct {
	Operator   string          `json:"operator"` // AND, OR, NOT
	Conditions []RuleCondition `json:"conditions,omitempty"`
	Children   []CohortRule    `json:"children,omitempty"` // nested rules
}

// RuleCondition is a single device attribute filter.
type RuleCondition struct {
	Field string `json:"field"` // device_model, os_version, app_version, locale, etc.
	Op    string `json:"op"`    // equals, glob, prefix, semver_range, in, lt, gt
	Value string `json:"value"`
}

// CohortMembership tracks device-to-cohort assignment.
type CohortMembership struct {
	CohortID string    `json:"cohort_id"`
	DeviceID string    `json:"device_id"`
	JoinedAt time.Time `json:"joined_at"`
}

// MinCohortSize is the privacy minimum for dynamic cohorts.
const MinCohortSize = 10
