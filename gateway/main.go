// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mobile-observability/gateway/internal/cohort"
	"github.com/mobile-observability/gateway/internal/config"
	"github.com/mobile-observability/gateway/internal/db"
	"github.com/mobile-observability/gateway/internal/fleet"
	"github.com/mobile-observability/gateway/internal/handlers"
	"github.com/mobile-observability/gateway/internal/otel"
	"github.com/mobile-observability/gateway/internal/push"
)

func main() {
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

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Device endpoints
	mux.HandleFunc("POST /ingest", h.HandleIngest)
	mux.HandleFunc("GET /config", h.HandleGetConfig)
	mux.HandleFunc("POST /status", h.HandleStatus)

	// Device management endpoints
	mux.HandleFunc("POST /v1/devices/register", h.HandleRegisterDevice)
	mux.HandleFunc("GET /v1/devices", h.HandleListDevices)
	mux.HandleFunc("GET /v1/devices/detail", h.HandleGetDevice)
	mux.HandleFunc("PATCH /v1/devices/group", h.HandleUpdateDeviceGroup)
	mux.HandleFunc("GET /v1/device-groups", h.HandleListDeviceGroups)
	mux.HandleFunc("GET /v1/heartbeats", h.HandleListHeartbeats)

	// OTEL Configuration management endpoints
	mux.HandleFunc("POST /v1/otel-configs", h.HandleCreateOTELConfig)
	mux.HandleFunc("GET /v1/otel-configs", h.HandleListOTELConfigs)
	mux.HandleFunc("GET /v1/otel-configs/active", h.HandleGetActiveOTELConfig)
	mux.HandleFunc("POST /v1/otel-configs/activate", h.HandleActivateOTELConfig)
	mux.HandleFunc("GET /v1/otel-configs/rollout-status", h.HandleGetConfigRolloutStatus)

	// Admin endpoints
	mux.HandleFunc("POST /admin/publish", h.HandlePublish)
	mux.HandleFunc("POST /admin/rollback", h.HandleRollback)
	mux.HandleFunc("GET /admin/versions", h.HandleVersions)

	// Workflow CRUD endpoints
	mux.HandleFunc("POST /v1/workflows", h.HandleCreateWorkflow)
	mux.HandleFunc("GET /v1/workflows", h.HandleListWorkflows)
	mux.HandleFunc("GET /v1/workflows/detail", h.HandleGetWorkflow)
	mux.HandleFunc("PUT /v1/workflows/detail", h.HandleUpdateWorkflow)
	mux.HandleFunc("DELETE /v1/workflows/detail", h.HandleDeleteWorkflow)

	// Targeting rules endpoints
	mux.HandleFunc("POST /v1/targeting-rules", h.HandleCreateTargetingRule)
	mux.HandleFunc("GET /v1/targeting-rules", h.HandleListTargetingRules)
	mux.HandleFunc("DELETE /v1/targeting-rules", h.HandleDeleteTargetingRule)

	// Buffer config endpoints
	mux.HandleFunc("POST /v1/buffer-configs", h.HandleUpsertBufferConfig)
	mux.HandleFunc("GET /v1/buffer-configs", h.HandleGetBufferConfig)
	mux.HandleFunc("GET /v1/buffer-configs/list", h.HandleListBufferConfigs)

	// Metrics & Funnel ingest endpoints
	mux.HandleFunc("POST /v1/metrics/ingest", h.HandleMetricsIngest)
	mux.HandleFunc("POST /v1/funnels/ingest", h.HandleFunnelsIngest)

	// Fleet Intelligence Routes
	mux.HandleFunc("POST /v1/fleet/events", h.HandleFleetEvents)
	mux.HandleFunc("GET /v1/fleet/rules", h.HandleListFleetRules)
	mux.HandleFunc("GET /v1/fleet/counters", h.HandleGetFleetCounters)

	// Cohort Management
	mux.HandleFunc("GET /v1/cohorts", h.HandleListCohorts)
	mux.HandleFunc("POST /v1/cohorts", h.HandleCreateCohort)
	mux.HandleFunc("DELETE /v1/cohorts", h.HandleDeleteCohort)
	mux.HandleFunc("GET /v1/cohorts/members", h.HandleGetCohortMembers)

	// Cascade Management
	mux.HandleFunc("GET /v1/cascades", h.HandleListCascades)
	mux.HandleFunc("POST /admin/cascade/kill", h.HandleKillSwitch)
	mux.HandleFunc("POST /admin/cascade/resume", h.HandleResumeSwitch)
	mux.HandleFunc("GET /admin/cascade/breaker-state", h.HandleGetBreakerState)

	// Push Channel
	mux.HandleFunc("GET /v1/push/status", h.HandleGetPushStatus)

	// Workflow Audit
	mux.HandleFunc("GET /v1/workflows/audit", h.HandleGetWorkflowAudit)

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Create server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      loggingMiddleware(corsMiddleware(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
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
		log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
		log.Printf("%s %s - %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
