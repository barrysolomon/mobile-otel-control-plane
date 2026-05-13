// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mobile-observability/gateway/internal/auth"
	"github.com/mobile-observability/gateway/internal/cohort"
	"github.com/mobile-observability/gateway/internal/config"
	"github.com/mobile-observability/gateway/internal/db"
	"github.com/mobile-observability/gateway/internal/fleet"
	"github.com/mobile-observability/gateway/internal/handlers"
	"github.com/mobile-observability/gateway/internal/otel"
	"github.com/mobile-observability/gateway/internal/push"
	"github.com/mobile-observability/gateway/internal/ratelimit"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "Run migrations and exit without starting the server")
	flag.Parse()

	// Configuration
	port := getEnv("PORT", "8080")
	dbPath := getEnv("DB_PATH", "./data/gateway.db")
	collectorEndpoint := getEnv("OTEL_COLLECTOR_ENDPOINT", "otel-collector.mobile-observability.svc.cluster.local:4317")
	otelAuthToken := getEnv("OTEL_AUTH_TOKEN", "")

	log.Printf("Starting Mobile Observability Gateway")
	log.Printf("Port: %s", port)
	log.Printf("Database: %s", dbPath)
	log.Printf("Collector: %s", collectorEndpoint)
	if otelAuthToken != "" {
		log.Printf("OTEL Auth Token: configured")
	}

	// Ensure database directory exists
	dbDir := "./data"
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	// Initialize database
	database, err := db.NewDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	if *migrateOnly {
		if err := database.MigrateOnly(); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		fmt.Println("Migration completed successfully")
		os.Exit(0)
	}

	// Initialize OTEL exporter
	ctx := context.Background()
	exporter, err := otel.NewLogExporter(ctx, collectorEndpoint, otelAuthToken)
	if err != nil {
		log.Fatalf("Failed to initialize OTEL exporter: %v", err)
	}
	defer exporter.Shutdown(ctx)

	// Initialize config manager
	configMgr := config.NewManager(database)

	// Fleet Intelligence — Split databases
	fleetDB, err := db.NewFleetDB(dbDir + "/fleet_events.db")
	if err != nil {
		log.Fatalf("Failed to open fleet DB: %v", err)
	}
	defer fleetDB.Close()

	auditDB, err := db.NewAuditDB(dbDir + "/cascade_audit.db")
	if err != nil {
		log.Fatalf("Failed to open audit DB: %v", err)
	}
	defer auditDB.Close()

	// Fleet components
	cohortMgr := cohort.NewManager(database.DB())
	ruleEngine := fleet.NewFleetRuleEngine(5 * time.Minute)
	wsRegistry := push.NewConnectionRegistry()
	pushBroker := push.NewBroker(wsRegistry)
	breakerState := fleet.NewBreakerState()
	budgetMgr := fleet.NewBudgetManager(25.0, 10000, 0)
	dedup := fleet.NewEventDeduplicator(6*time.Minute, 100000)

	hmacSecret := []byte(os.Getenv("FLEET_HMAC_SECRET"))
	if len(hmacSecret) == 0 {
		// Production gate accepts both ENVIRONMENT and ENV for the value
		// "production" — keep both so deployments using either convention
		// fail safely. auth/middleware.go uses ENVIRONMENT; previous
		// versions of this file used ENV. Aligned 2026-05-13.
		if os.Getenv("ENVIRONMENT") == "production" || os.Getenv("ENV") == "production" {
			log.Fatalf("FLEET_HMAC_SECRET must be set in production")
		}
		log.Println("WARNING: FLEET_HMAC_SECRET not set, using dev default — do NOT use in production")
		hmacSecret = []byte("dev-secret-change-in-production")
	}

	pipeline := fleet.NewFleetEventPipeline(5000, 4, ruleEngine.OnFleetEvent)
	pipeline.Start()
	defer pipeline.Stop()

	// Initialize handlers
	h := handlers.NewHandlerWithFleet(database, exporter, configMgr, handlers.FleetComponents{
		FleetDB:      fleetDB,
		AuditDB:      auditDB,
		CohortMgr:    cohortMgr,
		RuleEngine:   ruleEngine,
		PushBroker:   pushBroker,
		BreakerState: breakerState,
		BudgetMgr:    budgetMgr,
		Pipeline:     pipeline,
		Dedup:        dedup,
		HmacSecret:   hmacSecret,
	})

	// Setup HTTP routes.
	//
	// v1 MVP scoping (2026-05-13): the gateway is split into a *core* surface
	// (SDK contract + device fleet + workflow publish) and an *experimental*
	// surface (everything else). Experimental routes return 503 Service
	// Unavailable unless ENABLE_EXPERIMENTAL=true. The corresponding UI tabs
	// are hidden behind the same env var in control-plane-ui.
	//
	// Why a server-side gate instead of just leaving them on:
	// - Many experimental handlers are stubs or have no UI consumption today
	//   (rollout-status returns placeholder JSON; fleet/cohorts/cascades have
	//   wired handlers but no tests).
	// - A customer hitting an unfinished endpoint by accident gets a clear
	//   "experimental, not enabled" signal instead of a stub response that
	//   looks real.
	// - Reversible: flip the env var to opt in for development / beta usage.
	mux := http.NewServeMux()
	experimentalEnabled := os.Getenv("ENABLE_EXPERIMENTAL") == "true"
	if experimentalEnabled {
		log.Println("ENABLE_EXPERIMENTAL=true — experimental routes active")
	}

	// experimentalHandlerFunc wraps a handler so it returns 503 when
	// experimental routes are disabled. The route is still registered (so
	// 404 doesn't lie about whether the path exists) but the body explains
	// the gate.
	experimentalHandlerFunc := func(name string, h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !experimentalEnabled {
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
	experimentalHandler := func(name string, h http.Handler) http.Handler {
		return experimentalHandlerFunc(name, h.ServeHTTP)
	}

	// ── CORE — v1 production surface ────────────────────────────────────

	// SDK contract endpoints (Android/iOS/RN poll these)
	mux.HandleFunc("POST /ingest", h.HandleIngest)
	mux.HandleFunc("GET /config", h.HandleGetConfig)
	mux.HandleFunc("POST /status", h.HandleStatus)

	// Device fleet management
	mux.HandleFunc("POST /v1/devices/register", h.HandleRegisterDevice)
	mux.HandleFunc("GET /v1/devices", h.HandleListDevices)
	mux.HandleFunc("GET /v1/devices/detail", h.HandleGetDevice)
	mux.HandleFunc("PATCH /v1/devices/group", h.HandleUpdateDeviceGroup)
	mux.HandleFunc("GET /v1/device-groups", h.HandleListDeviceGroups)
	mux.HandleFunc("GET /v1/heartbeats", h.HandleListHeartbeats)

	// Workflow CRUD (the editor's storage layer)
	mux.HandleFunc("POST /v1/workflows", h.HandleCreateWorkflow)
	mux.HandleFunc("GET /v1/workflows", h.HandleListWorkflows)
	mux.HandleFunc("GET /v1/workflows/detail", h.HandleGetWorkflow)
	mux.HandleFunc("PUT /v1/workflows/detail", h.HandleUpdateWorkflow)
	mux.HandleFunc("DELETE /v1/workflows/detail", h.HandleDeleteWorkflow)

	// Admin/publish (require GATEWAY_ADMIN_API_KEY when set; auth middleware
	// is a no-op in dev mode for `npm run dev` ergonomics)
	adminMW := auth.AdminAPIKeyMiddleware
	mux.Handle("POST /admin/publish", adminMW(http.HandlerFunc(h.HandlePublish)))
	mux.Handle("POST /admin/rollback", adminMW(http.HandlerFunc(h.HandleRollback)))
	mux.Handle("GET /admin/versions", adminMW(http.HandlerFunc(h.HandleVersions)))

	// Health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// ── EXPERIMENTAL — gated behind ENABLE_EXPERIMENTAL=true ────────────

	// OTEL Collector config per device group (rollout-status handler stubbed)
	mux.HandleFunc("POST /v1/otel-configs", experimentalHandlerFunc("otel-configs", h.HandleCreateOTELConfig))
	mux.HandleFunc("GET /v1/otel-configs", experimentalHandlerFunc("otel-configs", h.HandleListOTELConfigs))
	mux.HandleFunc("GET /v1/otel-configs/active", experimentalHandlerFunc("otel-configs", h.HandleGetActiveOTELConfig))
	mux.HandleFunc("POST /v1/otel-configs/activate", experimentalHandlerFunc("otel-configs", h.HandleActivateOTELConfig))
	mux.HandleFunc("GET /v1/otel-configs/rollout-status", experimentalHandlerFunc("otel-configs", h.HandleGetConfigRolloutStatus))

	// Targeting rules (UI form exists but not wired)
	mux.HandleFunc("POST /v1/targeting-rules", experimentalHandlerFunc("targeting-rules", h.HandleCreateTargetingRule))
	mux.HandleFunc("GET /v1/targeting-rules", experimentalHandlerFunc("targeting-rules", h.HandleListTargetingRules))
	mux.HandleFunc("DELETE /v1/targeting-rules", experimentalHandlerFunc("targeting-rules", h.HandleDeleteTargetingRule))

	// Buffer config per device group
	mux.HandleFunc("POST /v1/buffer-configs", experimentalHandlerFunc("buffer-configs", h.HandleUpsertBufferConfig))
	mux.HandleFunc("GET /v1/buffer-configs", experimentalHandlerFunc("buffer-configs", h.HandleGetBufferConfig))
	mux.HandleFunc("GET /v1/buffer-configs/list", experimentalHandlerFunc("buffer-configs", h.HandleListBufferConfigs))

	// Pre-aggregated metrics + funnels (no UI query layer yet)
	mux.HandleFunc("POST /v1/metrics/ingest", experimentalHandlerFunc("metrics-ingest", h.HandleMetricsIngest))
	mux.HandleFunc("POST /v1/funnels/ingest", experimentalHandlerFunc("funnels-ingest", h.HandleFunnelsIngest))

	// Fleet intelligence (no tests; no UI integration)
	mux.HandleFunc("POST /v1/fleet/events", experimentalHandlerFunc("fleet-intelligence", h.HandleFleetEvents))
	mux.HandleFunc("GET /v1/fleet/rules", experimentalHandlerFunc("fleet-intelligence", h.HandleListFleetRules))
	mux.HandleFunc("GET /v1/fleet/counters", experimentalHandlerFunc("fleet-intelligence", h.HandleGetFleetCounters))

	// Cohort management
	mux.HandleFunc("GET /v1/cohorts", experimentalHandlerFunc("cohorts", h.HandleListCohorts))
	mux.HandleFunc("POST /v1/cohorts", experimentalHandlerFunc("cohorts", h.HandleCreateCohort))
	mux.HandleFunc("DELETE /v1/cohorts", experimentalHandlerFunc("cohorts", h.HandleDeleteCohort))
	mux.HandleFunc("GET /v1/cohorts/members", experimentalHandlerFunc("cohorts", h.HandleGetCohortMembers))

	// Cascade management
	mux.HandleFunc("GET /v1/cascades", experimentalHandlerFunc("cascades", h.HandleListCascades))
	mux.Handle("POST /admin/cascade/kill", experimentalHandler("cascades", adminMW(http.HandlerFunc(h.HandleKillSwitch))))
	mux.Handle("POST /admin/cascade/resume", experimentalHandler("cascades", adminMW(http.HandlerFunc(h.HandleResumeSwitch))))
	mux.Handle("GET /admin/cascade/breaker-state", experimentalHandler("cascades", adminMW(http.HandlerFunc(h.HandleGetBreakerState))))

	// Push channel
	mux.HandleFunc("GET /v1/push/status", experimentalHandlerFunc("push-channel", h.HandleGetPushStatus))

	// Workflow audit log (no UI consumption)
	mux.HandleFunc("GET /v1/workflows/audit", experimentalHandlerFunc("workflow-audit", h.HandleGetWorkflowAudit))

	// Journey Replay — proxies trace_id queries to Dash0 so the UI can
	// pull live data without exposing the bearer token to the browser.
	// Beta: works but landed in last 2 weeks; needs real-world soak.
	mux.HandleFunc("GET /v1/replay/by-trace", experimentalHandlerFunc("journey-replay", h.HandleReplayByTrace))

	// Rate limiter: 100 requests per minute per IP
	limiter := ratelimit.New(100, time.Minute)

	// Create server
	server := &http.Server{
		Addr:           ":" + port,
		Handler:        limiter.Middleware(loggingMiddleware(corsMiddleware(mux))),
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB max header size
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	// Start server
	log.Printf("Server listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// Use r.URL.Path only (not RequestURI/String) to omit query params and prevent PII leakage.
		// r.RemoteAddr is redacted to avoid logging user IP addresses.
		logPath := r.URL.Path
		log.Printf("%s %s", r.Method, logPath)
		next.ServeHTTP(w, r)
		log.Printf("%s %s - %v", r.Method, logPath, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000,http://localhost:5173"
	}
	originSet := make(map[string]bool)
	for _, o := range strings.Split(allowedOrigins, ",") {
		originSet[strings.TrimSpace(o)] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if originSet[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
