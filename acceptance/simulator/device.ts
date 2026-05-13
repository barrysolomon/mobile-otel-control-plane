// Simulated SDK device. Speaks the gateway's wire protocol exactly as a real
// Android / iOS / RN SDK would — POST /status heartbeats, GET /config polls,
// POST /ingest event uploads. Used by acceptance tests to validate that the
// control-plane UI's changes propagate to a polling device without booting
// a real emulator.
//
// Why a custom simulator instead of running the actual SDK against this
// gateway:
// - Deterministic timing (no Robolectric/Xcode startup variance).
// - Direct test assertions on what the device received without parsing logs.
// - Runs in <100ms per scenario step.
//
// The simulator's contract MUST match the SDK contract tested in
// `gateway/internal/handlers/sdk_contract_test.go`. When that contract
// changes, this file changes in the same commit.

export interface DeviceOptions {
  /** Gateway URL, e.g. "http://localhost:8080" — no trailing slash. */
  gatewayUrl: string
  /** Device ID used in heartbeats + ingest. */
  deviceId: string
  /** App ID used in heartbeats. */
  appId: string
  /** Optional admin API key (only needed for /admin/* calls). */
  adminApiKey?: string
}

export interface MobileEvent {
  event_name: string
  session_id: string
  device_id: string
  trigger_id?: string
  config_version: number
  /** Unix milliseconds. */
  timestamp: number
  attributes: Record<string, unknown>
}

export interface HeartbeatOptions {
  sessionId?: string
  bufferUsageMB?: number
  lastTriggers?: string[]
  configVersion?: number
}

export interface DSLConfigV2 {
  version: number
  buffer_config?: unknown
  workflows: Array<{
    id: string
    enabled: boolean
    priority?: number
    initial_state?: string
    states?: unknown[]
  }>
}

export interface SimulatedDevice {
  /** Device ID for assertions / test isolation. */
  readonly deviceId: string

  /** GET /config?dsl_version=2 — what the SDK calls every 5 min. */
  fetchConfig(): Promise<DSLConfigV2>

  /** POST /status — heartbeat. Auto-registers on first call. */
  heartbeat(opts?: HeartbeatOptions): Promise<void>

  /** POST /ingest — upload event batch. */
  sendEvents(events: MobileEvent[]): Promise<void>

  /**
   * Convenience: send a single crash event. Mirrors what iOS's
   * `emitAnyPendingCrash` and Android's ErrorsInstrumentation emit.
   */
  sendCrash(opts?: { exceptionType?: string; exceptionMessage?: string }): Promise<void>

  /**
   * Convenience: send a journey event with a screenshot + wireframe attached.
   * Used by the journey-replay acceptance test to seed Dash0-like data the UI
   * will fetch back. Returns the trace_id used so the test can query for it.
   */
  sendJourneyWithScreenshotAndWireframe(opts?: {
    screenName?: string
    /** Tiny data URL — keeps test fixtures small. */
    screenshotDataUrl?: string
    wireframeJson?: string
  }): Promise<{ traceId: string }>
}

/**
 * Build a simulated device. The constructor doesn't make any HTTP calls;
 * callers must invoke heartbeat() or fetchConfig() explicitly.
 */
export function createSimulatedDevice(opts: DeviceOptions): SimulatedDevice {
  const base = opts.gatewayUrl.replace(/\/$/, '')
  let sessionId = randomHex(16)

  async function fetchConfig(): Promise<DSLConfigV2> {
    const url = `${base}/config?app_id=${encodeURIComponent(opts.appId)}&device_id=${encodeURIComponent(opts.deviceId)}&dsl_version=2`
    const res = await fetch(url)
    if (!res.ok) {
      throw new Error(`fetchConfig: ${res.status} ${res.statusText}`)
    }
    return (await res.json()) as DSLConfigV2
  }

  async function heartbeat(beat: HeartbeatOptions = {}): Promise<void> {
    const body = {
      device_id: opts.deviceId,
      app_id: opts.appId,
      session_id: beat.sessionId ?? sessionId,
      buffer_usage_mb: beat.bufferUsageMB ?? 0.5,
      last_triggers: beat.lastTriggers ?? [],
      config_version: beat.configVersion ?? 0,
    }
    const res = await fetch(`${base}/status`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(body),
    })
    if (!res.ok) {
      throw new Error(`heartbeat: ${res.status} ${res.statusText}: ${await res.text()}`)
    }
  }

  async function sendEvents(events: MobileEvent[]): Promise<void> {
    const res = await fetch(`${base}/ingest`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ events }),
    })
    if (!res.ok) {
      throw new Error(`sendEvents: ${res.status} ${res.statusText}: ${await res.text()}`)
    }
  }

  async function sendCrash(o: { exceptionType?: string; exceptionMessage?: string } = {}): Promise<void> {
    await sendEvents([
      {
        event_name: 'app.crash',
        session_id: sessionId,
        device_id: opts.deviceId,
        config_version: 0,
        timestamp: Date.now(),
        attributes: {
          'event.name': 'app.crash',
          'exception.type': o.exceptionType ?? 'NullPointerException',
          'exception.message': o.exceptionMessage ?? 'simulated crash from acceptance suite',
        },
      },
    ])
  }

  async function sendJourneyWithScreenshotAndWireframe(o: {
    screenName?: string
    screenshotDataUrl?: string
    wireframeJson?: string
  } = {}): Promise<{ traceId: string }> {
    const traceId = randomHex(16) + randomHex(16) // 32 hex chars = 128 bits
    const now = Date.now()
    const screenName = o.screenName ?? 'CheckoutScreen'
    const screenshot =
      o.screenshotDataUrl ??
      // 1x1 transparent PNG — keeps fixtures tiny but exercises the data-URL path.
      'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII='
    const wireframe =
      o.wireframeJson ??
      JSON.stringify({
        type: 'UIWindow',
        bounds: [0, 0, 400, 800],
        children: [
          {
            type: 'UINavigationController',
            bounds: [0, 0, 400, 800],
            children: [{ type: 'UIViewController', bounds: [0, 64, 400, 736], children: [] }],
          },
        ],
      })

    await sendEvents([
      {
        event_name: 'ui.screen_view',
        session_id: sessionId,
        device_id: opts.deviceId,
        config_version: 0,
        timestamp: now,
        attributes: { 'event.name': 'ui.screen_view', 'screen.name': screenName, trace_id: traceId },
      },
      {
        event_name: 'ui.screenshot',
        session_id: sessionId,
        device_id: opts.deviceId,
        config_version: 0,
        timestamp: now + 1,
        attributes: {
          'event.name': 'ui.screenshot',
          'screen.name': screenName,
          'mobile.screenshot.data_url': screenshot,
          trace_id: traceId,
        },
      },
      {
        event_name: 'ui.wireframe',
        session_id: sessionId,
        device_id: opts.deviceId,
        config_version: 0,
        timestamp: now + 2,
        attributes: {
          'event.name': 'ui.wireframe',
          'screen.name': screenName,
          'mobile.wireframe.data': wireframe,
          trace_id: traceId,
        },
      },
    ])
    return { traceId }
  }

  // Rotate the session for tests that want a fresh session boundary.
  // Not on the public interface yet; expose if/when a test needs it.
  void (function _rotateSession() {
    sessionId = randomHex(16)
  })

  return {
    deviceId: opts.deviceId,
    fetchConfig,
    heartbeat,
    sendEvents,
    sendCrash,
    sendJourneyWithScreenshotAndWireframe,
  }
}

function randomHex(bytes: number): string {
  const arr = new Uint8Array(bytes)
  crypto.getRandomValues(arr)
  return Array.from(arr, (b) => b.toString(16).padStart(2, '0')).join('')
}
