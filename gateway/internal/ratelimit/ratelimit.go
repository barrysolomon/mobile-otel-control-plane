// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Limiter tracks request counts per key within a sliding window.
type Limiter struct {
	mu      sync.Mutex
	counts  map[string][]time.Time
	limit   int
	window  time.Duration
	cleanup time.Duration
}

// New creates a rate limiter allowing [limit] requests per [window] per key.
func New(limit int, window time.Duration) *Limiter {
	l := &Limiter{
		counts:  make(map[string][]time.Time),
		limit:   limit,
		window:  window,
		cleanup: window * 2,
	}
	go l.cleanupLoop()
	return l
}

// Allow checks if a request from [key] is within the rate limit.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	// Remove expired entries
	timestamps := l.counts[key]
	start := 0
	for start < len(timestamps) && timestamps[start].Before(cutoff) {
		start++
	}
	timestamps = timestamps[start:]

	if len(timestamps) >= l.limit {
		l.counts[key] = timestamps
		return false
	}

	l.counts[key] = append(timestamps, now)
	return true
}

// cleanupLoop periodically removes stale keys to prevent memory growth.
func (l *Limiter) cleanupLoop() {
	ticker := time.NewTicker(l.cleanup)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		cutoff := time.Now().Add(-l.window)
		for key, timestamps := range l.counts {
			start := 0
			for start < len(timestamps) && timestamps[start].Before(cutoff) {
				start++
			}
			if start >= len(timestamps) {
				delete(l.counts, key)
			} else {
				l.counts[key] = timestamps[start:]
			}
		}
		l.mu.Unlock()
	}
}

// Middleware returns an HTTP middleware that rate-limits by client IP.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientIP(r)
		if !l.Allow(key) {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the client IP from X-Forwarded-For or RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// First IP in the chain is the original client
		if idx := strings.IndexByte(xff, ','); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
