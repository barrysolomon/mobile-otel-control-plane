# Mobile OTel Control Plane

Management plane for the [OpenTelemetry Android Mobile SDK](https://github.com/barrysolomon/mobile-otel). Provides a visual export policy builder, a Go gateway for device fleet management, and deployment manifests.

## Components

| Component | Path | Description |
|-----------|------|-------------|
| **Control Plane UI** | `control-plane-ui/` | React 18 + TypeScript visual policy builder using React Flow |
| **Gateway** | `gateway/` | Go HTTP server — config versioning, event ingestion, OTLP export |
| **Acceptance Suite** | `acceptance/` | Playwright + simulated SDK. `npm run test:acceptance` boots gateway + UI and validates the 5 user-facing scenarios end-to-end. Spec in [UI_ACCEPTANCE_TESTING_EPIC.md](docs/epics/UI_ACCEPTANCE_TESTING_EPIC.md). |
| **Deployment** | `k8s/` | Kubernetes manifests + Docker Compose for local dev |

## Architecture

```
Android SDK (mobile-otel repo)
    ├─ POST /ingest ──────────► Gateway (Go, :8080)
    ├─ GET  /config  ◄──────── │  ├─ SQLite persistence
    └─ POST /status  ──────────► │  ├─ Config version mgmt
                                 │  └─ OTLP/gRPC export ──► OTEL Collector (:4317)
                                 │
Control Plane UI (React, :3000) ─┘
    ├─ Visual policy builder (React Flow)
    ├─ Device fleet dashboard
    └─ Collector configuration
```

## Quick Start

### Prerequisites

- Go 1.24+
- Node.js 18+
- Docker (for local OTEL Collector)

### Local Development

```bash
# 1. Start the OTEL Collector + Jaeger
docker compose -f k8s/docker-compose.yml up -d

# 2. Start the Gateway
cd gateway
go build -o gateway . && ./gateway
# Listening on :8080

# 3. Start the Control Plane UI
cd control-plane-ui
npm install && npm run dev
# Open http://localhost:3000
```

### Gateway API — v1 surface

The v1 production surface is intentionally small: 3 SDK endpoints, 6 device-fleet endpoints, 5 workflow CRUD endpoints, 3 admin/publish endpoints, plus `/health`. All other handlers are *experimental* and return `503 Service Unavailable` unless the gateway is started with `ENABLE_EXPERIMENTAL=true`.

**SDK contract endpoints** — what every deployed Android, iOS, and React Native SDK polls. Locked-down with contract tests in `gateway/internal/handlers/sdk_contract_test.go`:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/ingest` | POST | Receive event batches from devices |
| `/config` | GET | Return active DSL config (use `?dsl_version=2`) |
| `/status` | POST | Device heartbeat |

**Workflow publish + version management** — the UI's path for shipping policy changes. Admin endpoints require `X-API-Key: <key>` (or `?api_key=`) when `GATEWAY_ADMIN_API_KEY` is set; in dev mode they're open with a warning:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/publish` | POST | Publish a new policy version |
| `/admin/rollback` | POST | Activate a previous version |
| `/admin/versions` | GET | List config version history |
| `/v1/workflows`, `/v1/workflows/detail` | CRUD | Workflow editor's storage layer |

**Device fleet observability**:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/devices`, `/v1/devices/detail`, `/v1/devices/register`, `/v1/devices/group` | various | Device inventory |
| `/v1/device-groups` | GET | List device groups |
| `/v1/heartbeats` | GET | Recent device heartbeats |

**Health**:

| `/health` | GET | Always returns `{"status":"ok"}` |
|---|---|---|

### Experimental endpoints (off by default)

The following endpoint groups are wired but not part of the v1 contract. They exist for ongoing development and return `503` unless `ENABLE_EXPERIMENTAL=true`. Corresponding UI tabs are hidden unless `VITE_ENABLE_EXPERIMENTAL=true` on the UI build.

| Group | Endpoints | Why experimental |
|---|---|---|
| OTEL collector config | `/v1/otel-configs/*` (5) | Per-device-group rollout still has a stubbed status handler |
| Targeting rules | `/v1/targeting-rules` (3) | UI form exists but unwired; semantics undecided |
| Buffer config | `/v1/buffer-configs/*` (3) | UI form exists but unwired |
| Metrics + funnels ingest | `/v1/metrics/ingest`, `/v1/funnels/ingest` | No query/visualization layer yet |
| Fleet intelligence | `/v1/fleet/*` (3) | No tests; no UI |
| Cohorts | `/v1/cohorts/*` (4) | No tests; no UI |
| Cascades | `/v1/cascades`, `/admin/cascade/*` (4) | No tests; no UI |
| Push channel | `/v1/push/status` | No UI consumption |
| Workflow audit | `/v1/workflows/audit` | No UI consumption |
| Journey replay | `/v1/replay/by-trace` | Beta — works, but landed in last 2 weeks |

To enable everything during development: `ENABLE_EXPERIMENTAL=true ./gateway` (server) plus `VITE_ENABLE_EXPERIMENTAL=true npm run dev` (UI).

### Environment Variables (Gateway)

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `DB_PATH` | `./data/gateway.db` | SQLite database path |
| `OTEL_COLLECTOR_ENDPOINT` | (none) | OTLP gRPC target for log forwarding |
| `OTEL_AUTH_TOKEN` | (empty) | Optional bearer token for collector |
| `COLLECTOR_TLS_ENABLED` | `false` | Set `true` for TLS gRPC to collector |
| `GATEWAY_ADMIN_API_KEY` | (none) | When set, `/admin/*` requires `X-API-Key` header; required in production |
| `FLEET_HMAC_SECRET` | (none) | Required when in production mode |
| `ENVIRONMENT` or `ENV` | (empty) | Set either to `production` to enforce strict secrets |
| `ENABLE_EXPERIMENTAL` | `false` | Set `true` to enable the experimental routes listed above |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000,http://localhost:5173` | Comma-separated |

## Relationship to mobile-otel

This repo is the **management plane**. The [mobile-otel](https://github.com/barrysolomon/mobile-otel) repo contains the **data plane** — the Android SDK, instrumentation modules, and collector processor that are submitted to [opentelemetry-android-contrib](https://github.com/open-telemetry/opentelemetry-android-contrib).

The Android SDK works independently without this control plane. Policies can be configured:

1. **Statically** — via `otel-config.json` bundled in the app (see [Configuration Guide](https://github.com/barrysolomon/mobile-otel/blob/main/docs/CONFIGURATION.md))
2. **Dynamically** — via this control plane's gateway (the SDK polls `GET /config`)

## License

Apache 2.0 — see [LICENSE](LICENSE).
