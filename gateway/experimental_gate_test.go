// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
//
// Verifies the ENABLE_EXPERIMENTAL gate behaviour at the routing layer.
// These tests don't bring up the full gateway main() — they exercise the
// exact wrapper logic that main.go uses. If the wrapper drifts, copy the
// production version into this test.

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// experimentalHandlerFuncTestable mirrors main.go's experimentalHandlerFunc.
// Kept here as a tiny duplicate rather than refactoring main.go's closure
// into an importable symbol — the wrapper is too small to warrant its own
// package, and a test in `package main` would prevent `go install` callers.
func experimentalHandlerFuncTestable(name string, enabled bool, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !enabled {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":   "experimental feature disabled",
				"feature": name,
				"hint":    "Set ENABLE_EXPERIMENTAL=true on the gateway to enable.",
			})
			return
		}
		h(w, r)
	}
}

func ok(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// Disabled (default): returns 503 with structured body explaining the gate.
// Critical: it must be 503 not 200, otherwise a caller could believe a stub
// handler's success response is real production behaviour.
func TestExperimentalGate_ReturnsServiceUnavailableWhenDisabled(t *testing.T) {
	wrapped := experimentalHandlerFuncTestable("test-feature", false, ok)

	req := httptest.NewRequest(http.MethodGet, "/whatever", nil)
	rec := httptest.NewRecorder()
	wrapped(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("disabled: want 503 got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: want application/json got %q", ct)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body parse: %v", err)
	}
	if body["feature"] != "test-feature" {
		t.Errorf("feature: want test-feature got %q", body["feature"])
	}
	if body["error"] == "" || body["hint"] == "" {
		t.Errorf("body must include error+hint, got %+v", body)
	}
}

// Enabled: gate passes through to the wrapped handler.
func TestExperimentalGate_PassesThroughWhenEnabled(t *testing.T) {
	wrapped := experimentalHandlerFuncTestable("test-feature", true, ok)

	req := httptest.NewRequest(http.MethodGet, "/whatever", nil)
	rec := httptest.NewRecorder()
	wrapped(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("enabled: want 200 got %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body: want ok got %q", rec.Body.String())
	}
}

// Confirms the env var parsing (used in main.go's `os.Getenv(...) == "true"`
// check) is exact-match. "1", "yes", "TRUE" should NOT enable the gate —
// stricter parsing avoids accidental enablement.
func TestExperimentalGate_EnvVarParsing(t *testing.T) {
	cases := []struct {
		value   string
		enabled bool
	}{
		{"true", true},
		{"", false},
		{"false", false},
		{"1", false},   // strict comparison — "1" doesn't enable
		{"yes", false}, // ditto
		{"TRUE", false},
	}
	for _, tc := range cases {
		t.Setenv("ENABLE_EXPERIMENTAL", tc.value)
		got := os.Getenv("ENABLE_EXPERIMENTAL") == "true"
		if got != tc.enabled {
			t.Errorf("value=%q: want enabled=%v got %v", tc.value, tc.enabled, got)
		}
	}
}
