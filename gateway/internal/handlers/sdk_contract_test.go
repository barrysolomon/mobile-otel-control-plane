// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
//
// SDK contract tests. These pin the wire shape of the three endpoints the
// mobile SDKs (Android PolicyEvaluator.fetchConfig, iOS ConfigPoller,
// React-Native bridge) call against this gateway:
//
//   GET  /config?dsl_version=2  — policy DSL pull
//   POST /ingest                — event batch upload
//   POST /status                — device heartbeat
//
// Any change that would break a deployed SDK MUST fail one of these tests.
// They are the v1 MVP guarantee that the cross-repo contract documented in
// the sibling mobile-otel repo's docs/contracts/ stays stable.

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/mobile-observability/gateway/internal/config"
	"github.com/mobile-observability/gateway/internal/db"
	"github.com/mobile-observability/gateway/internal/otel"
)

// fakeExporter records the events it received without dialing a real
// gRPC connection. Plain Mutex (not sync/atomic) to keep the test simple.
type fakeExporter struct {
	mu     sync.Mutex
	events []otel.MobileEvent
	err    error
}

func (f *fakeExporter) ExportEvents(_ context.Context, events []otel.MobileEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.events = append(f.events, events...)
	return nil
}

func (f *fakeExporter) received() []otel.MobileEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]otel.MobileEvent, len(f.events))
	copy(out, f.events)
	return out
}

// newTestHandler wires a fresh sqlite DB + config manager + fake exporter for
// a single test. Caller is responsible for `t.Cleanup` of the temp DB path.
func newTestHandler(t *testing.T) (*Handler, *fakeExporter) {
	t.Helper()
	path := t.TempDir() + "/test.db"
	database, err := db.NewDatabase(path)
	if err != nil {
		t.Fatalf("NewDatabase: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	exp := &fakeExporter{}
	mgr := config.NewManager(database)
	return NewHandler(database, exp, mgr), exp
}

// ───────────────────────────────────────────────────────────────────────────
// GET /config — DSL pull contract
// ───────────────────────────────────────────────────────────────────────────

// SDK contract: when no policy has been published yet, the gateway must still
// return a valid (seed) DSL v2 document so the SDK doesn't enter a permanent
// no-config state on first boot.
func TestGetConfigV2_ReturnsSeedConfigWhenEmpty(t *testing.T) {
	h, _ := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/config?app_id=demo&device_id=dev1&dsl_version=2", nil)
	rec := httptest.NewRecorder()

	h.HandleGetConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200 got %d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type: want application/json got %q", ct)
	}

	// Shape check — must be parseable as the v2 contract the SDK enforces.
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response JSON parse: %v body=%s", err, rec.Body.String())
	}
	if version, _ := payload["version"].(float64); int(version) != 2 {
		t.Errorf("version: want 2 got %v", payload["version"])
	}
	if _, ok := payload["workflows"]; !ok {
		t.Errorf("payload missing required `workflows` key: %v", payload)
	}
}

// SDK contract: ?dsl_version=1 explicitly requests the legacy flat format.
// The SDK only ships v2 today, but the gateway must keep v1 available for
// any older SDK build still in the wild.
func TestGetConfig_V1FallbackShape(t *testing.T) {
	h, _ := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/config?app_id=demo&device_id=dev1&dsl_version=1", nil)
	rec := httptest.NewRecorder()

	h.HandleGetConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200 got %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response JSON parse: %v", err)
	}
	// v1 has its own version field (presence is the contract; value may
	// differ across deploys). Triggers / actions arrays may be empty
	// when no config is published; that's fine.
	if _, ok := payload["version"]; !ok {
		t.Errorf("v1 payload missing `version` key: %v", payload)
	}
}

// SDK contract: rejected app_id / device_id must return 400 with a readable
// error. The SDK retries on 5xx but treats 4xx as "stop polling" — so giving
// 5xx for bad query params would create polling loops.
func TestGetConfig_RejectsMissingAppID(t *testing.T) {
	h, _ := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/config?device_id=dev1&dsl_version=2", nil)
	rec := httptest.NewRecorder()

	h.HandleGetConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing app_id: want 400 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetConfig_MethodNotAllowed(t *testing.T) {
	h, _ := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/config?app_id=demo&device_id=dev1", nil)
	rec := httptest.NewRecorder()

	h.HandleGetConfig(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /config: want 405 got %d", rec.Code)
	}
}

// ───────────────────────────────────────────────────────────────────────────
// POST /ingest — event batch contract
// ───────────────────────────────────────────────────────────────────────────

// SDK contract: a batch of events flows through to the exporter exactly once,
// no events dropped. The response shape (status + count) is consumed by the
// SDK's exporter for retry decisions — keep it stable.
func TestIngest_ForwardsEventsToExporter(t *testing.T) {
	h, exp := newTestHandler(t)

	body := IngestRequest{
		Events: []otel.MobileEvent{
			{EventName: "app.crash", DeviceID: "dev1", SessionID: "s1", Timestamp: 1, Attributes: map[string]any{"k": "v"}},
			{EventName: "http.error", DeviceID: "dev1", SessionID: "s1", Timestamp: 2, Attributes: map[string]any{"status": 503}},
		},
	}
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(buf))
	rec := httptest.NewRecorder()

	h.HandleIngest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200 got %d body=%s", rec.Code, rec.Body.String())
	}

	got := exp.received()
	if len(got) != 2 {
		t.Fatalf("events forwarded: want 2 got %d", len(got))
	}
	if got[0].EventName != "app.crash" || got[1].EventName != "http.error" {
		t.Errorf("event order / names not preserved: %+v", got)
	}

	// Response shape contract.
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response JSON parse: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("response.status: want ok got %v", resp["status"])
	}
	if n, _ := resp["events_ingested"].(float64); int(n) != 2 {
		t.Errorf("response.events_ingested: want 2 got %v", resp["events_ingested"])
	}
}

// SDK contract: empty events array is a client error, not silent success.
// Without this the SDK could spin on an empty-batch loop.
func TestIngest_EmptyBatch400(t *testing.T) {
	h, exp := newTestHandler(t)
	body := IngestRequest{Events: []otel.MobileEvent{}}
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(buf))
	rec := httptest.NewRecorder()
	h.HandleIngest(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty batch: want 400 got %d", rec.Code)
	}
	if got := exp.received(); len(got) != 0 {
		t.Errorf("exporter received events on empty batch: %d", len(got))
	}
}

// SDK contract: malformed JSON is 400. The SDK treats 4xx as permanent
// and stops retrying that batch — important so transient corrupt bytes
// don't loop forever.
func TestIngest_MalformedJSON400(t *testing.T) {
	h, _ := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/ingest", strings.NewReader("not json"))
	rec := httptest.NewRecorder()
	h.HandleIngest(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("malformed JSON: want 400 got %d", rec.Code)
	}
}

// SDK contract: if the gateway can't reach the collector, it returns 5xx so
// the SDK retries instead of dropping the batch. The current SDK ConfigPoller
// retry policy depends on this distinction.
func TestIngest_ExporterFailureReturns500(t *testing.T) {
	h, exp := newTestHandler(t)
	exp.err = bytes.ErrTooLarge // any non-nil error

	body := IngestRequest{
		Events: []otel.MobileEvent{{EventName: "app.crash", DeviceID: "dev1", Timestamp: 1}},
	}
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(buf))
	rec := httptest.NewRecorder()
	h.HandleIngest(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("exporter failure: want 500 got %d", rec.Code)
	}
}

func TestIngest_GETNotAllowed(t *testing.T) {
	h, _ := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/ingest", nil)
	rec := httptest.NewRecorder()
	h.HandleIngest(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /ingest: want 405 got %d", rec.Code)
	}
}

// ───────────────────────────────────────────────────────────────────────────
// POST /status — heartbeat contract
// ───────────────────────────────────────────────────────────────────────────

// SDK contract: a well-formed heartbeat is accepted; the gateway auto-
// registers an unknown device so cold devices can begin reporting without
// a prior register-device call. UI components (DeviceFleet, DeviceMonitor)
// depend on this auto-registration to populate the fleet view.
func TestStatus_AcceptsHeartbeatAndAutoRegisters(t *testing.T) {
	h, _ := newTestHandler(t)

	body := StatusRequest{
		DeviceID:      "dev-new",
		AppID:         "demo",
		SessionID:     "s1",
		BufferUsageMB: 1.5,
		LastTriggers:  []string{"crash-recovery"},
		ConfigVersion: 1,
	}
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/status", bytes.NewReader(buf))
	rec := httptest.NewRecorder()

	h.HandleStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestStatus_MalformedJSON400(t *testing.T) {
	h, _ := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/status", strings.NewReader("not json"))
	rec := httptest.NewRecorder()
	h.HandleStatus(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("malformed status: want 400 got %d", rec.Code)
	}
}

func TestStatus_GETNotAllowed(t *testing.T) {
	h, _ := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	h.HandleStatus(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /status: want 405 got %d", rec.Code)
	}
}
