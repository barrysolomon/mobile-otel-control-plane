# UI acceptance testing epic

End-to-end validation that the control-plane UI does what it promises:

1. **Build & publish policies** — a user can draw a workflow in the UI, click publish, and the gateway stores a new version.
2. **Devices receive updated config** — a polling device sees the published policy on its next `/config` call.
3. **Fleet visibility** — devices that report heartbeats show up in the Devices tab with the right config version.
4. **Rollback works** — selecting a previous version flips the active config and the next device poll sees the old policy.
5. **Journey replay** — screenshots and wireframes from device traces render in the Journey Replay tab.
6. **Experimental gate** — when `ENABLE_EXPERIMENTAL=false`, experimental tabs are hidden and the matching server routes return 503.

All six are user-facing guarantees, not just "the code compiles." This epic is the test suite that catches regressions in any of them before a deploy.

## Architecture

Three test surfaces, three runners:

| Surface | Runner | What it asserts |
|---|---|---|
| Component unit / data-path | Vitest + RTL (jsdom) | Parsers, compilers, single-component contracts. Existing 53 tests fall here. |
| User flows (the 5 acceptance scenarios above) | Playwright (Chromium) | Real browser drives the UI; assertions on rendered DOM + screenshots. |
| Device-side gateway contract | Node test harness + Vitest | A simulated SDK polls the gateway and asserts what it received. |

The acceptance tests **always** run against a simulated device by default. The simulated device is a Node module that speaks the same wire protocol as a real SDK (`POST /status` heartbeats every 5s, `GET /config?dsl_version=2` on demand, `POST /ingest` for events). Deterministic, fast, runs in CI.

A `--real-device` flag bridges the same Playwright tests against a booted iOS Simulator or Android Emulator (reusing the UAT matrix harness from the sibling repo). Slow, manual, used for release validation.

## What gets built

### 1. Simulated SDK (`acceptance/simulator/`)

A Node module exposing:

```ts
const device = await createSimulatedDevice({
  gatewayUrl: 'http://localhost:8080',
  deviceId: 'sim-device-1',
  appId: 'demo-app',
})

// Lifecycle
await device.register()                    // POST /v1/devices/register
await device.heartbeat({ bufferUsageMB })  // POST /status
const config = await device.fetchConfig()  // GET /config?dsl_version=2

// Telemetry
await device.sendCrash({ exception: '...', stack: [...] })
await device.sendJourneyEvent({ traceId, screen, screenshot })
```

The simulator's value is precise visibility: a Playwright test can publish a policy and then assert `device.fetchConfig()` returns the new policy ID — without spinning up an emulator.

### 2. Playwright scaffold (`acceptance/playwright/`)

- `playwright.config.ts` — Chromium-only (no need for cross-browser at this stage), `baseURL` from env, 30s timeout per test, screenshot-on-failure.
- `fixtures/` — shared setup: launches gateway, starts UI dev server, mounts simulated device, returns helpers to each test.
- `pages/` — page object models for App, WorkflowBuilder, DeviceFleet, JourneyReplay.
- `tests/` — the 5 acceptance scenarios, one file each.

### 3. Boot orchestration (`acceptance/run.sh` + `package.json` scripts)

`npm run test:acceptance` does:

1. Build gateway, build UI.
2. Start gateway on port 8080 (in-memory SQLite or temp file).
3. Start UI dev server on port 3000 (Vite, proxies `/api` to 8080).
4. Wait for both to be ready.
5. Run Playwright tests, which themselves start the simulated device.
6. Tear down everything.
7. Exit non-zero on any failure.

`npm run test:acceptance:real-device` is the same but skips the simulator and expects an iPhone 17 Simulator or Pixel emulator booted, with the appropriate UAT matrix wiring.

### 4. Vitest acceptance suites (`control-plane-ui/src/__tests__/acceptance/`)

For component-level acceptance that doesn't need a real browser:

- `journeyReplay.acceptance.test.tsx` — full parse → render path for screenshot + wireframe data, using fixture OTLP envelopes from the SDK side.
- `workflowCompiler.acceptance.test.ts` — the v1 + v2 compilers produce DSL the gateway accepts (round-trip via `gateway/internal/config/manager`).

These fill gaps Playwright would be overkill for.

## The 5 acceptance scenarios

### Scenario 1: Build & publish a policy

```
1. UI loads with empty workflow list
2. User clicks "New Workflow"
3. User drags an `event_match` node and a `flush_window` node onto the canvas
4. User wires them together
5. User clicks "Publish"
6. Assert: gateway has a new config version with the compiled DSL
7. Assert: simulated device polls and receives the workflow ID
```

### Scenario 2: Fleet view shows simulated devices

```
1. Three simulated devices send heartbeats
2. User opens the Devices tab
3. Assert: all three appear with their device_id, last_seen, config_version
4. User clicks a device row
5. Assert: device detail shows the right buffer_usage_mb, last_triggers
```

### Scenario 3: Rollback flips the active config

```
1. Publish workflow A (gets version N)
2. Simulated device polls → receives workflow A
3. Publish workflow B (gets version N+1)
4. Simulated device polls → receives workflow B
5. User opens Versions list, clicks "Rollback" on version N
6. Assert: rollback succeeded
7. Simulated device polls → receives workflow A again
```

### Scenario 4: Journey replay renders screenshots and wireframes

```
1. Simulated device emits a journey with one screen view + one screenshot + one wireframe (all attached to the same trace_id)
2. User opens Journey Replay tab, enters the trace_id, clicks "Fetch"
3. Assert: events list shows the screen_view, the screenshot, the wireframe
4. User clicks the screenshot
5. Assert: lightbox opens, shows the data-URL image
6. User clicks the wireframe
7. Assert: wireframe tree inspector renders the UIWindow → children hierarchy
```

### Scenario 5: Experimental gate

```
1. Start gateway with ENABLE_EXPERIMENTAL=false
2. UI loads (built with VITE_ENABLE_EXPERIMENTAL=false)
3. Assert: Configuration tab is NOT visible
4. Assert: Journey Replay tab is NOT visible
5. Direct request to /v1/cohorts returns 503 with feature-disabled body
6. Restart gateway with ENABLE_EXPERIMENTAL=true (skip — would need restart in test)
   — instead: assert the server-side behaviour separately via gateway integration test
```

## Out of scope (defer to follow-up epics)

- **Cross-browser**: Firefox / WebKit. Chromium-only for v1. Easy to add later.
- **Visual regression**: pixel diffs on UI screenshots. Different tooling (Percy/Chromatic), separate epic.
- **Real device matrix**: reuses the existing UAT matrix in mobile-otel; not duplicated here.
- **Load / stress testing**: not acceptance work.
- **Accessibility audits**: separate epic, separate tooling (axe-playwright).

## Done definition

- `npm run test:acceptance` passes locally on `main` with all 5 scenarios green.
- A new acceptance test for any future feature blocks merge until it passes.
- The 53 existing Vitest tests continue to pass.
- README links to the test suite as the "v1 production guarantee" doc.
- Tests run in <3 minutes total (Vitest + Playwright + simulator boot).

## Shipping status — 2026-05-13

- ✅ **6/6 Playwright acceptance tests passing** in 4.6 seconds (chromium-headless, single worker). Suite covers the 5 scenarios plus an experimental-gate ON/OFF split.
- ✅ **56/56 Vitest tests passing** (was 53; +3 new gate visibility tests).
- ✅ **`npm run test:acceptance`** wired in `control-plane-ui/package.json`.
- ✅ **Boot orchestration** (`acceptance/scripts/boot.mjs`) builds gateway, starts both servers, runs the suite, tears down cleanly on signal.

## Real bug caught by the acceptance suite

While Test 2 (fleet view) was iterating, the suite surfaced a genuine gateway bug: `GET /v1/devices` returned 500 with "Failed to list devices" on a fresh SQLite database. Root cause in `gateway/internal/db/db.go:ListDevices`:

```
sql: Scan error on column index 6, name "COALESCE(last_seen,
registered_at)": unsupported Scan, storing driver.Value type string
into type *time.Time
```

The go-sqlite3 driver returns `COALESCE(date_col, date_col)` results as strings rather than parsed `time.Time` — even when both source columns are TIMESTAMP. Scanning the result into `*time.Time` fails.

**Fix:** scan the COALESCE'd columns into intermediary strings and parse in Go via a new `parseSqliteTime` helper. Avoids a connection-string flag that would change semantics for every other query in the package.

This bug was invisible in the existing unit tests because they used fresh DBs with no populated heartbeat rows. The acceptance suite exercises the real round-trip (heartbeats → registration → list) and caught it on first run. This is exactly what the epic was designed to do — and proves the investment immediately.
