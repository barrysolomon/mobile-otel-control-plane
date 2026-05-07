// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// HandleReplayByTrace proxies a Dash0 logs query for a specific trace_id and
// returns the result to the control-plane UI. This powers the "Load from
// Dash0" path in the Journey Replay tab — the UI sends a trace_id, the
// gateway calls Dash0's REST API with the server-side auth token, and the
// browser never sees the bearer token.
//
// Configuration (env):
//   - DASH0_API_URL — e.g. "https://api.us-west-2.aws.dash0.com" (required)
//   - DASH0_AUTH_TOKEN — bearer token (required)
//   - DASH0_DATASET — defaults to "default"
//
// Query params:
//   - trace_id — required, hex-encoded 16-byte OTLP trace ID
//   - from — optional, defaults to "now-1h"
//   - to — optional, defaults to "now"
//   - limit — optional, capped at 100 (Dash0 JSON output limit)
//
// Response: forwards the Dash0 logs JSON envelope verbatim, so the UI can
// reuse the same OTLP parsing path as the paste flow.
func (h *Handler) HandleReplayByTrace(w http.ResponseWriter, r *http.Request) {
	traceID := r.URL.Query().Get("trace_id")
	if traceID == "" {
		http.Error(w, "trace_id query parameter is required", http.StatusBadRequest)
		return
	}
	// Light validation — protect against obvious abuse before opening a
	// network call. Real OTLP trace_ids are 32 lowercase hex chars; we
	// accept any 16-128 byte hex-ish string so we don't reject edge cases
	// like padded base16 from other tools.
	if len(traceID) < 16 || len(traceID) > 128 {
		http.Error(w, "trace_id must be 16-128 chars", http.StatusBadRequest)
		return
	}

	apiURL := os.Getenv("DASH0_API_URL")
	authToken := os.Getenv("DASH0_AUTH_TOKEN")
	if apiURL == "" || authToken == "" {
		http.Error(w, "DASH0_API_URL and DASH0_AUTH_TOKEN must be set on the gateway", http.StatusServiceUnavailable)
		return
	}
	dataset := os.Getenv("DASH0_DATASET")
	if dataset == "" {
		dataset = "default"
	}

	from := r.URL.Query().Get("from")
	if from == "" {
		from = "now-1h"
	}
	to := r.URL.Query().Get("to")
	if to == "" {
		to = "now"
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		// Dash0 JSON output is hard-capped at 100 records per call.
		// Pagination across multiple calls is a follow-up.
		limit = 100
	}

	// Dash0 logs query body shape per the public API. Filter trace_id
	// equality is the canonical way to fetch a journey trace.
	body := map[string]interface{}{
		"dataset":   dataset,
		"timeRange": map[string]string{"from": from, "to": to},
		"limit":     limit,
		"filterBy":  []map[string]string{{"key": "trace_id", "operator": "=", "value": traceID}},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		http.Error(w, "failed to encode upstream request body", http.StatusInternalServerError)
		return
	}

	queryURL, err := url.JoinPath(apiURL, "/api/logs")
	if err != nil {
		http.Error(w, "invalid DASH0_API_URL", http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, queryURL, bytes.NewReader(bodyBytes))
	if err != nil {
		http.Error(w, "failed to build upstream request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HandleReplayByTrace upstream error: %v", err)
		http.Error(w, fmt.Sprintf("upstream Dash0 query failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Forward status code + body verbatim. The UI reuses the same OTLP/JSON
	// parser as the paste flow, so we don't need to reshape the response.
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("HandleReplayByTrace copy error: %v", err)
	}
}
