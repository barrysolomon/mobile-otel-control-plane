# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Mobile observability control plane for managing OpenTelemetry configuration on Android devices. The control plane defines what workflows run on devices, configures ring buffers and targeting, receives results (metrics, funnels), and manages fleet-wide configuration. The companion Android SDK project (separate repo, sibling directory) executes the DSL on-device.

Three main components: a React UI for visual workflow editing, a Go gateway API backed by SQLite, and Kubernetes/Docker deployment configs for an OTEL Collector pipeline.

## Build & Run Commands

### Frontend (control-plane-ui/)
```bash
cd control-plane-ui
npm install
npm run dev          # Vite dev server on :3000, proxies /api to :8080
npm run build        # tsc && vite build -> dist/
npm run lint         # ESLint with strict TypeScript rules
```

### Backend (gateway/)
```bash
cd gateway
go build -o gateway .
./gateway            # Runs on :8080
```

Environment variables: `PORT` (default 8080), `DB_PATH` (default ./data/gateway.db), `OTEL_COLLECTOR_ENDPOINT`, `OTEL_AUTH_TOKEN`.

### Infrastructure (k8s/)
```bash
cd k8s
docker-compose up    # Local: OTEL Collector + Jaeger UI
./deploy-native.sh   # Kubernetes deployment
```

## Architecture

```
Android SDK --POST /ingest--> Gateway (Go :8080) --gRPC--> OTEL Collector (:4317)
             <-GET /config--  | SQLite (config versions, devices, workflows, metrics, funnels)
              POST /status--> |
              POST /metrics-> |
              POST /funnels-> |
                              |
Control Plane UI (React :3000) --Vite proxy /api--> Gateway
```

### Key data flow: Workflow Publishing
1. User builds a visual graph in React Flow (WorkflowBuilder component)
2. `graphToDSL.ts` compiles the graph to DSL v1 JSON; `graphToDSLv2.ts` compiles to DSL v2 (state-machine format)
3. Gateway stores graph JSON (for UI reload), DSL v1 JSON, and DSL v2 JSON as versioned entries
4. Android devices poll `GET /config?dsl_version=2` to receive the active DSL version (v1 still supported for backward compatibility)

### DSL Version Negotiation

- `GET /config?dsl_version=1` (or no param) returns the v1 flat trigger/action format
- `GET /config?dsl_version=2` returns the v2 state-machine FSM format
- Both versions are stored on every publish; the gateway serves whichever the device requests

### Frontend (control-plane-ui/src/)

- **App.tsx** -- Tab-based layout: Workflow Builder, Devices, Configuration
- **components/WorkflowBuilder.tsx** -- React Flow canvas with 29 custom node types across 10 palette categories
- **components/nodes/** -- Individual node type components (triggers, actions, states, insights)
- **components/TargetingRules.tsx** -- Per-workflow device targeting rule editor
- **components/BufferConfig.tsx** -- Per-group ring buffer configuration editor
- **types/workflow.ts** -- Core type definitions: `WorkflowGraph`, `GraphNode` (29 variants), `DSLConfig`, `DSLConfigV2`, all matcher/action types
- **api/gateway.ts** -- All gateway HTTP calls (publish, rollback, devices, workflows, OTEL config, targeting, buffer config, metrics ingest, funnel ingest)
- **utils/graphToDSL.ts** -- v1 graph-to-DSL compiler + `validateGraph()` with full validation
- **utils/graphToDSLv2.ts** -- v2 graph-to-DSL compiler (single-state and multi-state FSM support)
- **telemetry.ts** -- Browser-side OpenTelemetry setup exporting to Dash0

### Backend (gateway/)

- **main.go** -- HTTP server setup, route registration, OTEL initialization
- **internal/handlers/handlers.go** -- All HTTP handlers (ingest, config, devices, workflows, admin, targeting, buffer config, metrics, funnels)
- **internal/db/db.go** -- SQLite schema, migrations, and all query methods
- **internal/otel/exporter.go** -- gRPC log exporter to OTEL Collector
- **internal/config/manager.go** -- Config version management with v1/v2 DSL support

### Node Types (29 total, 10 palette categories)

- **Event Triggers**: event_match, log_severity_match, metric_threshold
- **Performance**: ui_freeze, slow_operation, frame_drop
- **Network**: http_error_match, network_loss, slow_request
- **Device Health**: low_memory, battery_drain, thermal_throttling, storage_low
- **Crash/Error**: crash_marker, exception_pattern
- **Predictive**: predictive_risk
- **Logic**: any (OR gate), all (AND gate)
- **States**: state (FSM state container), timeout_matcher (absence-of-event trigger)
- **Actions**: flush_window, set_sampling, annotate_trigger, send_alert, adjust_config
- **Insights**: emit_metric, record_session, create_funnel, create_sankey, take_screenshot

### Gateway API Groups

- `/ingest`, `/config`, `/status` -- Device-facing endpoints
- `/v1/devices/*`, `/v1/device-groups` -- Device fleet management
- `/v1/workflows/*` -- Workflow CRUD
- `/v1/otel-configs/*` -- Collector configuration per device group
- `/v1/targeting-rules` -- Per-workflow device targeting rules
- `/v1/buffer-configs`, `/v1/buffer-configs/list` -- Ring buffer configuration per device group
- `/v1/metrics/ingest` -- Pre-aggregated metric ingestion from devices
- `/v1/funnels/ingest` -- Funnel step event ingestion from devices
- `/admin/publish`, `/admin/rollback`, `/admin/versions` -- Config versioning

### Database
SQLite with tables: `config_versions` (with `dsl_v2_json` column), `devices`, `device_groups`, `device_heartbeats`, `otel_configurations`, `workflows`, `device_metrics`, `funnel_events`, `targeting_rules`, `buffer_configs`. Schema is auto-migrated in `db.go:migrate()`.

## DSL v2 Schema

The v2 DSL is the contract between this control plane and the Android SDK. Each workflow compiles to a finite state machine:

```json
{
  "version": 2,
  "buffer_config": { "ram_events": 5000, "disk_mb": 50, "retention_hours": 24, "strategy": "overwrite_oldest" },
  "workflows": [{
    "id": "crash-handler",
    "name": "Crash Handler",
    "enabled": true,
    "priority": 1,
    "initial_state": "default",
    "states": [{
      "id": "default",
      "matchers": [{ "type": "crash", "config": {} }],
      "on_match": {
        "actions": [
          { "type": "flush_buffer", "config": { "minutes": 5, "scope": "session" } },
          { "type": "record_session", "config": { "max_duration_minutes": 10 } }
        ],
        "transition_to": "recording"
      }
    }]
  }]
}
```

See `docs/DSL_V2_SCHEMA.md` for the complete schema reference with all 21 matcher types and 10 action types.

## Compiler Architecture

The visual graph (React Flow nodes/edges) compiles to DSL through two paths:

1. **v1 compiler** (`graphToDSL.ts`): Flat trigger/action format. Supports original 3 trigger types + 3 action types. Kept for backward compatibility with older SDK versions.

2. **v2 compiler** (`graphToDSLv2.ts`): State-machine format. Supports all 29 node types.
   - **Single-state FSM**: Workflows without StateNodes compile to a single "default" state (backward compatible pattern)
   - **Multi-state FSM**: Workflows with StateNodes compile to proper multi-state machines with transitions, timeouts, and per-state matchers/actions
   - Type narrowing uses `switch` statements on the `GraphNode` discriminated union for type safety

## State Management
The frontend uses Zustand for local state. React Flow manages the visual graph state. No Redux or context-heavy patterns.

## Key Dependencies

- Frontend: React 18, React Flow 11.10, Zustand, Axios, @opentelemetry/sdk-trace-web
- Backend: Go 1.24, mattn/go-sqlite3 (CGo required), go.opentelemetry.io/otel v1.39, google.golang.org/grpc
