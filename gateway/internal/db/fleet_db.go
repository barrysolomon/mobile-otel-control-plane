package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// FleetDB manages the fleet_events.db split database.
type FleetDB struct {
	db *sql.DB
}

// NewFleetDB opens and migrates the fleet events database.
func NewFleetDB(path string) (*FleetDB, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("open fleet db: %w", err)
	}
	fdb := &FleetDB{db: db}
	if err := fdb.migrate(); err != nil {
		return nil, fmt.Errorf("migrate fleet db: %w", err)
	}
	return fdb, nil
}

func (f *FleetDB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS fleet_events (
		id TEXT PRIMARY KEY,
		device_id TEXT NOT NULL,
		cohort_id TEXT,
		trigger_type TEXT NOT NULL,
		attributes_json TEXT,
		timestamp TIMESTAMP NOT NULL,
		device_timestamp TIMESTAMP,
		clock_offset_ms INTEGER DEFAULT 0,
		processed BOOLEAN DEFAULT FALSE
	);
	CREATE INDEX IF NOT EXISTS idx_fleet_events_cohort_trigger ON fleet_events(cohort_id, trigger_type, timestamp);
	CREATE INDEX IF NOT EXISTS idx_fleet_events_timestamp ON fleet_events(timestamp);

	CREATE TABLE IF NOT EXISTS backend_events (
		id TEXT PRIMARY KEY,
		service_name TEXT NOT NULL,
		event_type TEXT NOT NULL,
		data_json TEXT NOT NULL,
		timestamp TIMESTAMP NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_backend_events_service ON backend_events(service_name, timestamp);

	CREATE TABLE IF NOT EXISTS backend_services (
		name TEXT PRIMARY KEY,
		last_seen TIMESTAMP,
		current_health_json TEXT
	);
	`
	_, err := f.db.Exec(schema)
	return err
}

// InsertFleetEvent inserts a single fleet event.
func (f *FleetDB) InsertFleetEvent(id, deviceID, cohortID, triggerType, attrsJSON string, ts time.Time) error {
	_, err := f.db.Exec(
		"INSERT OR IGNORE INTO fleet_events (id, device_id, cohort_id, trigger_type, attributes_json, timestamp) VALUES (?, ?, ?, ?, ?, ?)",
		id, deviceID, cohortID, triggerType, attrsJSON, ts,
	)
	return err
}

// InsertFleetEventBatch inserts a batch of fleet events in a single transaction.
func (f *FleetDB) InsertFleetEventBatch(events []FleetEventRow) error {
	tx, err := f.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT OR IGNORE INTO fleet_events (id, device_id, cohort_id, trigger_type, attributes_json, timestamp) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, e := range events {
		if _, err := stmt.Exec(e.ID, e.DeviceID, e.CohortID, e.TriggerType, e.AttributesJSON, e.Timestamp); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// FleetEventRow is a DB row for fleet_events.
type FleetEventRow struct {
	ID             string
	DeviceID       string
	CohortID       string
	TriggerType    string
	AttributesJSON string
	Timestamp      time.Time
}

// DeleteFleetEventsOlderThan removes events older than the given time.
func (f *FleetDB) DeleteFleetEventsOlderThan(before time.Time) (int64, error) {
	result, err := f.db.Exec("DELETE FROM fleet_events WHERE timestamp < ?", before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Close closes the database connection.
func (f *FleetDB) Close() error {
	return f.db.Close()
}
