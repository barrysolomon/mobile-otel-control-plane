// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
//
// Publish→serve integration tests. These prove the load-bearing v1 control
// loop: a UI publishes a workflow via POST /admin/publish, and a device
// polls GET /config?dsl_version=2 and receives the published DSL.
//
// This is the canonical control-plane → SDK handoff. If this loop ever
// breaks, every deployed mobile device stops receiving new policies.

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// minimalV1DSL is the smallest DSL v1 payload the publish handler will accept.
// Production callers send a richer document; we only need something parseable.
const minimalV1DSL = `{"version":1,"workflows":[]}`

// minimalV2DSL with one workflow + matchable event. The /config?dsl_version=2
// response after publishing this must include the same workflow.
const minimalV2DSL = `{
  "version": 2,
  "buffer_config": {
    "ram_events": 5000,
    "disk_mb": 50,
    "retention_hours": 24,
    "strategy": "overwrite_oldest"
  },
  "workflows": [{
    "id": "test-crash-handler",
    "enabled": true,
    "priority": 1,
    "initial_state": "default",
    "states": [{
      "id": "default",
      "matchers": [{"type": "crash", "config": {}}],
      "on_match": {"actions": [{"type": "flush_buffer", "config": {"minutes": 5}}]}
    }]
  }]
}`

// minimalGraph: the gateway parses graph_json as []config.GraphWorkflow
// (the UI's React Flow format). Each workflow has nodes + edges.
// Empty array is valid — represents "no workflows in the editor yet".
const minimalGraph = `[]`

// THE LOAD-BEARING TEST: publish → /config?dsl_version=2 returns what we
// published. This is the single highest-value test in the gateway suite.
func TestPublishLoop_PublishedWorkflowReachesConfig(t *testing.T) {
	h, _ := newTestHandler(t)

	// Step 1: UI publishes a workflow.
	pubReq := PublishRequest{
		GraphJSON:   minimalGraph,
		DSLJSON:     minimalV1DSL,
		DSLV2JSON:   minimalV2DSL,
		PublishedBy: "test-user",
	}
	pubBody, _ := json.Marshal(pubReq)
	req := httptest.NewRequest(http.MethodPost, "/admin/publish", bytes.NewReader(pubBody))
	rec := httptest.NewRecorder()
	h.HandlePublish(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("publish: want 200 got %d body=%s", rec.Code, rec.Body.String())
	}
	var pubResp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &pubResp); err != nil {
		t.Fatalf("publish response parse: %v", err)
	}
	if pubResp["status"] != "ok" {
		t.Fatalf("publish status: want ok got %v", pubResp["status"])
	}
	// Version is monotonically increasing; first publish should be > 0.
	version, _ := pubResp["version"].(float64)
	if version <= 0 {
		t.Errorf("publish version: want > 0 got %v", pubResp["version"])
	}

	// Step 2: Device polls /config?dsl_version=2 and receives the publish.
	cfgReq := httptest.NewRequest(http.MethodGet,
		"/config?app_id=demo&device_id=dev1&dsl_version=2", nil)
	cfgRec := httptest.NewRecorder()
	h.HandleGetConfig(cfgRec, cfgReq)

	if cfgRec.Code != http.StatusOK {
		t.Fatalf("get config: want 200 got %d body=%s", cfgRec.Code, cfgRec.Body.String())
	}

	var cfgPayload map[string]any
	if err := json.Unmarshal(cfgRec.Body.Bytes(), &cfgPayload); err != nil {
		t.Fatalf("config response parse: %v body=%s", err, cfgRec.Body.String())
	}

	workflows, ok := cfgPayload["workflows"].([]any)
	if !ok || len(workflows) == 0 {
		t.Fatalf("config response missing workflows: %+v", cfgPayload)
	}
	wf, _ := workflows[0].(map[string]any)
	if wf == nil || wf["id"] != "test-crash-handler" {
		t.Errorf("published workflow id not returned: want test-crash-handler, got %v", wf)
	}
}

// Publishing twice must produce two distinct versions; the latest one is
// what /config serves. Rollback tests assume this monotonicity.
func TestPublishLoop_SecondPublishReplacesActive(t *testing.T) {
	h, _ := newTestHandler(t)

	publish := func(workflowID string) float64 {
		dsl := `{"version":2,"buffer_config":{"ram_events":5000,"disk_mb":50,"retention_hours":24,"strategy":"overwrite_oldest"},"workflows":[{"id":"` + workflowID + `","enabled":true,"priority":1,"initial_state":"default","states":[{"id":"default","matchers":[],"on_match":{"actions":[]}}]}]}`
		body, _ := json.Marshal(PublishRequest{
			GraphJSON: minimalGraph, DSLJSON: minimalV1DSL,
			DSLV2JSON: dsl, PublishedBy: "test",
		})
		req := httptest.NewRequest(http.MethodPost, "/admin/publish", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		h.HandlePublish(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("publish %s: want 200 got %d", workflowID, rec.Code)
		}
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		v, _ := resp["version"].(float64)
		return v
	}

	v1 := publish("first-handler")
	v2 := publish("second-handler")
	if !(v2 > v1) {
		t.Errorf("publish version should monotonically increase: v1=%v v2=%v", v1, v2)
	}

	// /config returns the latest publish.
	req := httptest.NewRequest(http.MethodGet,
		"/config?app_id=demo&device_id=dev1&dsl_version=2", nil)
	rec := httptest.NewRecorder()
	h.HandleGetConfig(rec, req)
	var payload map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	workflows, _ := payload["workflows"].([]any)
	if len(workflows) != 1 {
		t.Fatalf("want 1 workflow, got %d: %+v", len(workflows), workflows)
	}
	wf, _ := workflows[0].(map[string]any)
	if wf["id"] != "second-handler" {
		t.Errorf("active workflow should be second publish: got %v", wf["id"])
	}
}

// Publish rejects missing required fields. Without this, the UI could
// silently push empty configs that overwrite live policies.
func TestPublish_RequiresGraphAndDSL(t *testing.T) {
	h, _ := newTestHandler(t)

	cases := []struct {
		name string
		req  PublishRequest
	}{
		{"missing graph_json", PublishRequest{DSLJSON: minimalV1DSL}},
		{"missing dsl_json", PublishRequest{GraphJSON: minimalGraph}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.req)
			req := httptest.NewRequest(http.MethodPost, "/admin/publish",
				bytes.NewReader(body))
			rec := httptest.NewRecorder()
			h.HandlePublish(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s: want 400 got %d", tc.name, rec.Code)
			}
		})
	}
}

func TestPublish_MalformedJSON400(t *testing.T) {
	h, _ := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/admin/publish",
		bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()
	h.HandlePublish(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("malformed publish: want 400 got %d", rec.Code)
	}
}

func TestPublish_GETNotAllowed(t *testing.T) {
	h, _ := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/admin/publish", nil)
	rec := httptest.NewRecorder()
	h.HandlePublish(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /admin/publish: want 405 got %d", rec.Code)
	}
}
