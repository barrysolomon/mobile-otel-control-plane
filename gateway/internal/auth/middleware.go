// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"log"
	"net/http"
	"os"
)

// AdminAPIKeyMiddleware returns middleware that requires a valid API key
// for admin endpoints. The key is read from GATEWAY_ADMIN_API_KEY env var.
// Auth is sent via the `X-API-Key: <key>` header (or `?api_key=<key>` query).
//
// Production gating: when GATEWAY_ADMIN_API_KEY is empty, the gateway
// either fails fast (in production) or runs unauthenticated (in dev) with
// a loud warning. "Production" is detected via either ENVIRONMENT=production
// or ENV=production — both are accepted so deployments using either
// convention fail safely. Aligned with main.go's FLEET_HMAC_SECRET gate
// 2026-05-13.
func AdminAPIKeyMiddleware(next http.Handler) http.Handler {
	apiKey := os.Getenv("GATEWAY_ADMIN_API_KEY")
	isProd := os.Getenv("ENVIRONMENT") == "production" || os.Getenv("ENV") == "production"

	if apiKey == "" {
		if isProd {
			log.Fatal("GATEWAY_ADMIN_API_KEY must be set in production")
		}
		log.Println("WARNING: GATEWAY_ADMIN_API_KEY not set — admin endpoints unprotected (dev mode)")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiKey == "" {
			// Dev mode: allow without auth
			next.ServeHTTP(w, r)
			return
		}

		key := r.Header.Get("X-API-Key")
		if key == "" {
			key = r.URL.Query().Get("api_key")
		}

		if key != apiKey {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
