# Mobile OTel Control Plane

Management plane for the [OpenTelemetry Android Mobile SDK](https://github.com/barrysolomon/mobile-otel). Provides a visual export policy builder, a Go gateway for device fleet management, and deployment manifests.

## Components

| Component | Path | Description |
|-----------|------|-------------|
| **Control Plane UI** | `control-plane-ui/` | React 18 + TypeScript visual policy builder using React Flow |
| **Gateway** | `gateway/` | Go HTTP server — config versioning, event ingestion, OTLP export |
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

### Gateway API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/ingest` | POST | Receive event batches from Android devices |
| `/config` | GET | Return active DSL config for a device |
| `/status` | POST | Device heartbeat |
| `/health` | GET | Health check |
| `/admin/publish` | POST | Publish new policy version |
| `/admin/rollback` | POST | Activate a previous version |
| `/admin/versions` | GET | List config version history |
| `/api/v1/devices/*` | various | Device fleet management |
| `/api/v1/otel-configs/*` | various | OTEL collector configuration |

### Environment Variables (Gateway)

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `DB_PATH` | `./data/gateway.db` | SQLite database path |
| `OTEL_COLLECTOR_ENDPOINT` | `otel-collector...svc.cluster.local:4317` | OTLP gRPC target |
| `OTEL_AUTH_TOKEN` | (empty) | Optional auth token for collector |

## Relationship to mobile-otel

This repo is the **management plane**. The [mobile-otel](https://github.com/barrysolomon/mobile-otel) repo contains the **data plane** — the Android SDK, instrumentation modules, and collector processor that are submitted to [opentelemetry-android-contrib](https://github.com/open-telemetry/opentelemetry-android-contrib).

The Android SDK works independently without this control plane. Policies can be configured:

1. **Statically** — via `otel-config.json` bundled in the app (see [Configuration Guide](https://github.com/barrysolomon/mobile-otel/blob/main/docs/CONFIGURATION.md))
2. **Dynamically** — via this control plane's gateway (the SDK polls `GET /config`)

## License

Apache 2.0 — see [LICENSE](LICENSE).
