// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

// Dev mode (no API key, no production env): allow without auth, log warning.
// This is the local-development default — admin endpoints must remain
// usable in `npm run dev` setups without any extra config.
func TestAdminAPIKey_DevModeAllowsUnauthenticated(t *testing.T) {
	t.Setenv("GATEWAY_ADMIN_API_KEY", "")
	t.Setenv("ENVIRONMENT", "")
	t.Setenv("ENV", "")

	mw := AdminAPIKeyMiddleware(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/admin/publish", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("dev mode without key: want 200 got %d", rec.Code)
	}
}

// With API key set: missing/wrong key is 401, correct key is 200.
func TestAdminAPIKey_HeaderAuth(t *testing.T) {
	t.Setenv("GATEWAY_ADMIN_API_KEY", "supersecret")
	t.Setenv("ENVIRONMENT", "")
	t.Setenv("ENV", "")
	mw := AdminAPIKeyMiddleware(okHandler())

	cases := []struct {
		name    string
		setKey  string
		setHdr  string
		wantSts int
	}{
		{"no key", "", "", http.StatusUnauthorized},
		{"wrong key", "X-API-Key", "wrong", http.StatusUnauthorized},
		{"correct key", "X-API-Key", "supersecret", http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/admin/publish", nil)
			if tc.setKey != "" {
				req.Header.Set(tc.setKey, tc.setHdr)
			}
			rec := httptest.NewRecorder()
			mw.ServeHTTP(rec, req)
			if rec.Code != tc.wantSts {
				t.Errorf("%s: want %d got %d", tc.name, tc.wantSts, rec.Code)
			}
		})
	}
}

// Query-string auth (?api_key=...) is supported for browser callers that
// can't easily set custom headers. Header takes precedence per the
// middleware's lookup order.
func TestAdminAPIKey_QueryStringAuth(t *testing.T) {
	t.Setenv("GATEWAY_ADMIN_API_KEY", "supersecret")
	t.Setenv("ENVIRONMENT", "")
	t.Setenv("ENV", "")
	mw := AdminAPIKeyMiddleware(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/admin/publish?api_key=supersecret", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("query-string key: want 200 got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/admin/publish?api_key=wrong", nil)
	rec = httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong query-string key: want 401 got %d", rec.Code)
	}
}

// Header beats query string — set both, mismatched: header wins.
func TestAdminAPIKey_HeaderTakesPrecedenceOverQuery(t *testing.T) {
	t.Setenv("GATEWAY_ADMIN_API_KEY", "supersecret")
	t.Setenv("ENVIRONMENT", "")
	t.Setenv("ENV", "")
	mw := AdminAPIKeyMiddleware(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/admin/publish?api_key=wrong", nil)
	req.Header.Set("X-API-Key", "supersecret")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("header-over-query: want 200 got %d", rec.Code)
	}
}
