package cohort

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Manager handles cohort CRUD and membership materialization.
type Manager struct {
	db *sql.DB
}

// NewManager creates a cohort manager with the given main database.
func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// Create inserts a new dynamic cohort.
func (m *Manager) Create(c Cohort) error {
	if err := ValidateCohortRule(parseRules(c.RulesJSON)); err != nil {
		return err
	}
	_, err := m.db.Exec(
		"INSERT INTO cohorts (id, name, type, rules_json, device_count, created_at, updated_at) VALUES (?, ?, ?, ?, 0, ?, ?)",
		c.ID, c.Name, c.Type, c.RulesJSON, time.Now(), time.Now(),
	)
	return err
}

// Get returns a single cohort by ID.
func (m *Manager) Get(id string) (*Cohort, error) {
	row := m.db.QueryRow("SELECT id, name, type, rules_json, device_count, created_at, updated_at FROM cohorts WHERE id = ?", id)
	var c Cohort
	err := row.Scan(&c.ID, &c.Name, &c.Type, &c.RulesJSON, &c.DeviceCount, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

// List returns all cohorts.
func (m *Manager) List() ([]Cohort, error) {
	rows, err := m.db.Query("SELECT id, name, type, rules_json, device_count, created_at, updated_at FROM cohorts ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cohorts []Cohort
	for rows.Next() {
		var c Cohort
		rows.Scan(&c.ID, &c.Name, &c.Type, &c.RulesJSON, &c.DeviceCount, &c.CreatedAt, &c.UpdatedAt)
		cohorts = append(cohorts, c)
	}
	return cohorts, nil
}

// Delete removes a cohort and its memberships.
func (m *Manager) Delete(id string) error {
	_, err := m.db.Exec("DELETE FROM cohorts WHERE id = ?", id)
	return err
}

// RefreshMembership re-evaluates cohort membership for a single device.
func (m *Manager) RefreshMembership(deviceID string, attrs map[string]string) error {
	cohorts, err := m.List()
	if err != nil {
		return err
	}

	m.db.Exec("DELETE FROM cohort_memberships WHERE device_id = ?", deviceID)

	for _, c := range cohorts {
		if c.Type != "dynamic" {
			continue
		}
		rule := parseRules(c.RulesJSON)
		if Evaluate(rule, attrs) {
			m.db.Exec(
				"INSERT OR IGNORE INTO cohort_memberships (cohort_id, device_id, joined_at) VALUES (?, ?, ?)",
				c.ID, deviceID, time.Now(),
			)
		}
	}

	for _, c := range cohorts {
		var count int
		m.db.QueryRow("SELECT COUNT(*) FROM cohort_memberships WHERE cohort_id = ?", c.ID).Scan(&count)
		m.db.Exec("UPDATE cohorts SET device_count = ?, updated_at = ? WHERE id = ?", count, time.Now(), c.ID)
	}

	return nil
}

// GetMembers returns all device IDs in a cohort.
func (m *Manager) GetMembers(cohortID string) ([]string, error) {
	rows, err := m.db.Query("SELECT device_id FROM cohort_memberships WHERE cohort_id = ?", cohortID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, nil
}

// CohortsForDevice returns all cohort IDs that a device belongs to.
func (m *Manager) CohortsForDevice(deviceID string) ([]string, error) {
	rows, err := m.db.Query("SELECT cohort_id FROM cohort_memberships WHERE device_id = ?", deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, nil
}

func parseRules(rulesJSON string) CohortRule {
	if rulesJSON == "" {
		return CohortRule{Operator: "AND"}
	}
	var rule CohortRule
	if err := json.Unmarshal([]byte(rulesJSON), &rule); err != nil {
		return CohortRule{Operator: "AND"}
	}
	return rule
}
