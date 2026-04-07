// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAllow_UnderLimit(t *testing.T) {
	l := New(5, time.Second)
	for i := 0; i < 5; i++ {
		if !l.Allow("client1") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
}

func TestAllow_OverLimit(t *testing.T) {
	l := New(3, time.Second)
	for i := 0; i < 3; i++ {
		l.Allow("client1")
	}
	if l.Allow("client1") {
		t.Error("4th request should be denied")
	}
}

func TestAllow_DifferentKeys(t *testing.T) {
	l := New(1, time.Second)
	if !l.Allow("client1") {
		t.Error("client1 first request should be allowed")
	}
	if !l.Allow("client2") {
		t.Error("client2 first request should be allowed (independent key)")
	}
	if l.Allow("client1") {
		t.Error("client1 second request should be denied")
	}
}

func TestAllow_WindowExpires(t *testing.T) {
	l := New(1, 50*time.Millisecond)
	if !l.Allow("client1") {
		t.Error("first request should be allowed")
	}
	if l.Allow("client1") {
		t.Error("second request should be denied")
	}
	time.Sleep(60 * time.Millisecond)
	if !l.Allow("client1") {
		t.Error("request after window should be allowed")
	}
}

func TestMiddleware_Returns429(t *testing.T) {
	l := New(1, time.Second)
	handler := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request: allowed
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "1.2.3.4:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Second request: denied
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
}

func TestMiddleware_XForwardedFor(t *testing.T) {
	l := New(1, time.Second)
	handler := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.Header.Set("X-Forwarded-For", "10.0.0.1, 192.168.1.1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req1)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Same XFF = same client = rate limited
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req1)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 for same XFF client, got %d", w.Code)
	}
}
