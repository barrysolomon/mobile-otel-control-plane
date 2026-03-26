package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// AuditDB manages the cascade_audit.db split database.
type AuditDB struct {
	db *sql.DB
}

// NewAuditDB opens and migrates the cascade audit database.
func NewAuditDB(path string) (*AuditDB, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("open audit db: %w", err)
	}
	adb := &AuditDB{db: db}
	if err := adb.migrate(); err != nil {
		return nil, fmt.Errorf("migrate audit db: %w", err)
	}
	return adb, nil
}

func (a *AuditDB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS cascade_chains (
		chain_id TEXT PRIMARY KEY,
		root_trigger_type TEXT NOT NULL,
		root_cohort_id TEXT NOT NULL,
		root_device_id TEXT,
		started_at TIMESTAMP NOT NULL,
		ended_at TIMESTAMP,
		status TEXT NOT NULL CHECK(status IN ('active','completed','halted','killed','shadow','timeout_chain_duration','timeout_hop_wait')),
		max_hop_reached INTEGER DEFAULT 0,
		total_devices_affected INTEGER DEFAULT 0,
		kill_switch_tripped BOOLEAN DEFAULT FALSE,
		chain_timeout_minutes INTEGER DEFAULT 60,
		hop_timeout_minutes INTEGER DEFAULT 10,
		last_hop_at TIMESTAMP,
		workflow_id TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS cascade_hops (
		hop_id TEXT PRIMARY KEY,
		chain_id TEXT REFERENCES cascade_chains(chain_id),
		hop_number INTEGER NOT NULL,
		trigger_type TEXT NOT NULL,
		source_cohort_id TEXT NOT NULL,
		target_cohort_id TEXT NOT NULL,
		devices_targeted INTEGER NOT NULL,
		devices_delivered INTEGER DEFAULT 0,
		actions_json TEXT NOT NULL,
		breakers_evaluated_json TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL
	);

	CREATE TABLE IF NOT EXISTS cascade_device_log (
		chain_id TEXT REFERENCES cascade_chains(chain_id),
		hop_id TEXT REFERENCES cascade_hops(hop_id),
		device_id TEXT NOT NULL,
		delivery_channel TEXT NOT NULL,
		delivered_at TIMESTAMP,
		acknowledged_at TIMESTAMP,
		reaction_completed_at TIMESTAMP,
		PRIMARY KEY (chain_id, device_id)
	);

	CREATE TABLE IF NOT EXISTS fleet_workflow_audit (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		workflow_id TEXT NOT NULL,
		action TEXT NOT NULL CHECK(action IN ('created','updated','published','rollout_advanced','rollout_halted','rollout_rolled_back','disabled','deleted')),
		actor TEXT NOT NULL,
		previous_version_json TEXT,
		new_version_json TEXT,
		diff_summary TEXT,
		cascade_impact_json TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_fleet_audit_workflow ON fleet_workflow_audit(workflow_id, created_at);
	`
	_, err := a.db.Exec(schema)
	return err
}

// InsertCascadeChain creates a new cascade chain record.
func (a *AuditDB) InsertCascadeChain(chain CascadeChainRow) error {
	_, err := a.db.Exec(
		`INSERT INTO cascade_chains (chain_id, root_trigger_type, root_cohort_id, root_device_id,
		 started_at, status, chain_timeout_minutes, hop_timeout_minutes, last_hop_at, workflow_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		chain.ChainID, chain.RootTriggerType, chain.RootCohortID, chain.RootDeviceID,
		chain.StartedAt, chain.Status, chain.ChainTimeoutMin, chain.HopTimeoutMin, chain.StartedAt, chain.WorkflowID,
	)
	return err
}

// UpdateCascadeChainStatus updates the status and ended_at of a chain.
func (a *AuditDB) UpdateCascadeChainStatus(chainID, status string, endedAt *time.Time) error {
	_, err := a.db.Exec(
		"UPDATE cascade_chains SET status = ?, ended_at = ? WHERE chain_id = ?",
		status, endedAt, chainID,
	)
	return err
}

// InsertCascadeHop records a single hop within a cascade.
func (a *AuditDB) InsertCascadeHop(hop CascadeHopRow) error {
	_, err := a.db.Exec(
		`INSERT INTO cascade_hops (hop_id, chain_id, hop_number, trigger_type, source_cohort_id,
		 target_cohort_id, devices_targeted, devices_delivered, actions_json, breakers_evaluated_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		hop.HopID, hop.ChainID, hop.HopNumber, hop.TriggerType, hop.SourceCohortID,
		hop.TargetCohortID, hop.DevicesTargeted, hop.DevicesDelivered, hop.ActionsJSON, hop.BreakersJSON, hop.CreatedAt,
	)
	return err
}

// InsertCascadeDeviceLog records per-device delivery status.
func (a *AuditDB) InsertCascadeDeviceLog(entry CascadeDeviceRow) error {
	_, err := a.db.Exec(
		`INSERT OR IGNORE INTO cascade_device_log (chain_id, hop_id, device_id, delivery_channel, delivered_at)
		 VALUES (?, ?, ?, ?, ?)`,
		entry.ChainID, entry.HopID, entry.DeviceID, entry.DeliveryChannel, entry.DeliveredAt,
	)
	return err
}

// UpdateDeviceAck records the acknowledgment time for a device.
func (a *AuditDB) UpdateDeviceAck(chainID, deviceID string, ackedAt time.Time) error {
	_, err := a.db.Exec(
		"UPDATE cascade_device_log SET acknowledged_at = ? WHERE chain_id = ? AND device_id = ?",
		ackedAt, chainID, deviceID,
	)
	return err
}

// GetActiveChains returns all chains with status 'active'.
func (a *AuditDB) GetActiveChains() ([]CascadeChainRow, error) {
	rows, err := a.db.Query("SELECT chain_id, root_trigger_type, root_cohort_id, root_device_id, started_at, status, chain_timeout_minutes, hop_timeout_minutes, last_hop_at, workflow_id FROM cascade_chains WHERE status = 'active'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chains []CascadeChainRow
	for rows.Next() {
		var c CascadeChainRow
		if err := rows.Scan(&c.ChainID, &c.RootTriggerType, &c.RootCohortID, &c.RootDeviceID, &c.StartedAt, &c.Status, &c.ChainTimeoutMin, &c.HopTimeoutMin, &c.LastHopAt, &c.WorkflowID); err != nil {
			return nil, err
		}
		chains = append(chains, c)
	}
	return chains, nil
}

// GetTargetedDevices returns all device IDs targeted by a chain.
func (a *AuditDB) GetTargetedDevices(chainID string) ([]string, error) {
	rows, err := a.db.Query("SELECT DISTINCT device_id FROM cascade_device_log WHERE chain_id = ?", chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// InsertWorkflowAudit records a fleet workflow change.
func (a *AuditDB) InsertWorkflowAudit(entry WorkflowAuditRow) error {
	_, err := a.db.Exec(
		`INSERT INTO fleet_workflow_audit (workflow_id, action, actor, previous_version_json, new_version_json, diff_summary, cascade_impact_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entry.WorkflowID, entry.Action, entry.Actor, entry.PreviousJSON, entry.NewJSON, entry.DiffSummary, entry.CascadeImpactJSON,
	)
	return err
}

// ListWorkflowAudit returns the audit trail for a workflow.
func (a *AuditDB) ListWorkflowAudit(workflowID string) ([]WorkflowAuditRow, error) {
	rows, err := a.db.Query(
		"SELECT id, workflow_id, action, actor, previous_version_json, new_version_json, diff_summary, cascade_impact_json, created_at FROM fleet_workflow_audit WHERE workflow_id = ? ORDER BY created_at DESC",
		workflowID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []WorkflowAuditRow
	for rows.Next() {
		var e WorkflowAuditRow
		if err := rows.Scan(&e.ID, &e.WorkflowID, &e.Action, &e.Actor, &e.PreviousJSON, &e.NewJSON, &e.DiffSummary, &e.CascadeImpactJSON, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// DeleteCascadeChainsOlderThan removes completed chains older than the given time.
func (a *AuditDB) DeleteCascadeChainsOlderThan(before time.Time) (int64, error) {
	result, err := a.db.Exec(
		"DELETE FROM cascade_chains WHERE ended_at IS NOT NULL AND ended_at < ?",
		before,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CascadeChainRow is the DB row type for cascade_chains.
type CascadeChainRow struct {
	ChainID         string
	RootTriggerType string
	RootCohortID    string
	RootDeviceID    string
	StartedAt       time.Time
	Status          string
	ChainTimeoutMin int
	HopTimeoutMin   int
	LastHopAt       time.Time
	WorkflowID      string
}

// CascadeHopRow is the DB row type for cascade_hops.
type CascadeHopRow struct {
	HopID            string
	ChainID          string
	HopNumber        int
	TriggerType      string
	SourceCohortID   string
	TargetCohortID   string
	DevicesTargeted  int
	DevicesDelivered int
	ActionsJSON      string
	BreakersJSON     string
	CreatedAt        time.Time
}

// CascadeDeviceRow is the DB row type for cascade_device_log.
type CascadeDeviceRow struct {
	ChainID         string
	HopID           string
	DeviceID        string
	DeliveryChannel string
	DeliveredAt     *time.Time
}

// WorkflowAuditRow is the DB row type for fleet_workflow_audit.
type WorkflowAuditRow struct {
	ID                int
	WorkflowID        string
	Action            string
	Actor             string
	PreviousJSON      string
	NewJSON           string
	DiffSummary       string
	CascadeImpactJSON string
	CreatedAt         time.Time
}

// Close closes the database connection.
func (a *AuditDB) Close() error {
	return a.db.Close()
}
