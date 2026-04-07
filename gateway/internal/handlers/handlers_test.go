// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ── Utility function tests ──────────────────────────────────────────────────

func TestValidateID_Valid(t *testing.T) {
	w := httptest.NewRecorder()
	if !validateID(w, "test_id", "abc-123") {
		t.Error("expected valid ID to pass")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestValidateID_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	if validateID(w, "test_id", "") {
		t.Error("expected empty ID to fail")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestValidateID_TooLong(t *testing.T) {
	w := httptest.NewRecorder()
	longID := strings.Repeat("a", maxIDLength+1)
	if validateID(w, "test_id", longID) {
		t.Error("expected overlong ID to fail")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestValidateID_MaxLength(t *testing.T) {
	w := httptest.NewRecorder()
	maxID := strings.Repeat("a", maxIDLength)
	if !validateID(w, "test_id", maxID) {
		t.Error("expected max-length ID to pass")
	}
}

func TestClampLimit_Default(t *testing.T) {
	if got := clampLimit("", 50); got != 50 {
		t.Errorf("expected 50, got %d", got)
	}
}

func TestClampLimit_ValidValue(t *testing.T) {
	if got := clampLimit("25", 50); got != 25 {
		t.Errorf("expected 25, got %d", got)
	}
}

func TestClampLimit_ExceedsMax(t *testing.T) {
	if got := clampLimit("500", 50); got != maxQueryLimit {
		t.Errorf("expected %d, got %d", maxQueryLimit, got)
	}
}

func TestClampLimit_Negative(t *testing.T) {
	if got := clampLimit("-1", 50); got != 50 {
		t.Errorf("expected default 50, got %d", got)
	}
}

func TestClampLimit_Invalid(t *testing.T) {
	if got := clampLimit("abc", 50); got != 50 {
		t.Errorf("expected default 50, got %d", got)
	}
}

func TestClampLimit_Zero(t *testing.T) {
	if got := clampLimit("0", 50); got != 50 {
		t.Errorf("expected default 50 for zero, got %d", got)
	}
}

// ── readLimitedBody tests ───────────────────────────────────────────────────

func TestReadLimitedBody_ValidBody(t *testing.T) {
	body := bytes.NewBufferString(`{"test": "data"}`)
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	w := httptest.NewRecorder()

	data, err := readLimitedBody(w, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"test": "data"}` {
		t.Errorf("unexpected body: %s", string(data))
	}
}

func TestReadLimitedBody_OversizedBody(t *testing.T) {
	// Create a body larger than maxRequestBody (10 MB)
	bigBody := bytes.NewBuffer(make([]byte, maxRequestBody+1))
	req := httptest.NewRequest(http.MethodPost, "/test", bigBody)
	w := httptest.NewRecorder()

	_, err := readLimitedBody(w, req)
	if err == nil {
		t.Error("expected error for oversized body")
	}
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", w.Code)
	}
}

func TestReadLimitedBody_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBuffer(nil))
	w := httptest.NewRecorder()

	data, err := readLimitedBody(w, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty body, got %d bytes", len(data))
	}
}

func TestReadLimitedBody_ExactlyAtLimit(t *testing.T) {
	body := bytes.NewBuffer(make([]byte, maxRequestBody))
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	w := httptest.NewRecorder()

	data, err := readLimitedBody(w, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != maxRequestBody {
		t.Errorf("expected %d bytes, got %d", maxRequestBody, len(data))
	}
}
