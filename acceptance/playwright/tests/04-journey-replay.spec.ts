// Acceptance Test 4: Journey replay renders screenshots + wireframes.
//
// Scenario:
//   1. Build an OTLP/JSON envelope with one ui.screen_view, one ui.screenshot
//      (with a data-URL image), and one ui.wireframe (with a tree JSON).
//   2. Paste the envelope into the Journey Replay textarea and click Parse.
//   3. Assert the events list shows the three event types.
//   4. Assert the screenshot thumbnail is visible (img with data-URL src).
//   5. Click the screenshot — the lightbox dialog opens.
//
// We exercise the paste-textarea path (not the by-trace_id Dash0 proxy) so
// the test has no external dependencies — Dash0 credentials are not
// required.

import { test, expect } from '../fixtures/test'
import { AppPage } from '../pages/app'
import { JourneyReplayPage } from '../pages/journeyReplay'

// 1×1 transparent PNG, base64-encoded — same fixture the simulator emits.
const TINY_PNG_DATA_URL =
  'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII='

const WIREFRAME_JSON = JSON.stringify({
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

const TRACE_ID = 'a4f225e54cf8a4f6fdf84be3d9dfa1fb'

/**
 * Minimal OTLP/JSON envelope with the three event types Journey Replay
 * renders. Matches the shape `parseEvents` accepts in journeyParser.ts.
 */
function buildOtlpEnvelope(): string {
  const ts = (Date.now() * 1_000_000).toString() // OTLP wants nanoseconds string
  return JSON.stringify({
    resourceLogs: [
      {
        resource: { attributes: [{ key: 'service.name', value: { stringValue: 'acceptance-app' } }] },
        scopeLogs: [
          {
            scope: { name: 'io.dash0.mobile' },
            logRecords: [
              logRecord(ts, 'ui.screen_view', [
                ['event.name', 'ui.screen_view'],
                ['screen.name', 'CheckoutScreen'],
                ['trace_id', TRACE_ID],
              ]),
              logRecord(ts, 'ui.screenshot', [
                ['event.name', 'ui.screenshot'],
                ['screen.name', 'CheckoutScreen'],
                ['mobile.screenshot.data_url', TINY_PNG_DATA_URL],
                ['mobile.screenshot.trigger', 'manual'],
                ['trace_id', TRACE_ID],
              ]),
              logRecord(ts, 'ui.wireframe', [
                ['event.name', 'ui.wireframe'],
                ['screen.name', 'CheckoutScreen'],
                ['mobile.wireframe.data', WIREFRAME_JSON],
                ['mobile.wireframe.trigger', 'screen_view'],
                ['trace_id', TRACE_ID],
              ]),
            ],
          },
        ],
      },
    ],
  })
}

function logRecord(timeUnixNano: string, body: string, attrs: Array<[string, string]>) {
  return {
    timeUnixNano,
    observedTimeUnixNano: timeUnixNano,
    severityNumber: 9,
    severityText: 'INFO',
    body: { stringValue: body },
    attributes: attrs.map(([key, value]) => ({ key, value: { stringValue: value } })),
    traceId: TRACE_ID,
  }
}

test('journey replay renders screenshots and wireframes', async ({ page }) => {
  const app = new AppPage(page)
  await app.open()

  // Journey Replay is an experimental tab; this test relies on the boot
  // script starting the UI with VITE_ENABLE_EXPERIMENTAL=true.
  await expect(app.journeyReplayTab, 'Journey Replay tab must be visible (experimental flag enabled?)').toBeVisible()
  await app.journeyReplayTab.click()

  const replay = new JourneyReplayPage(page)

  // Paste OTLP fixture + parse
  await replay.pasteAndParse(buildOtlpEnvelope())

  // Three event rows render (search for the body strings)
  await expect(page.getByText('ui.screen_view').first()).toBeVisible()
  await expect(page.getByText('ui.screenshot').first()).toBeVisible()
  await expect(page.getByText('ui.wireframe').first()).toBeVisible()

  // Screenshot thumbnail visible
  await expect(replay.screenshotImage).toBeVisible()

  // Click it → lightbox opens
  await replay.screenshotImage.click()
  await expect(replay.screenshotLightbox).toBeVisible()

  // Click the lightbox backdrop to close (clicking it once closes the dialog).
  // Some keyboards close-on-escape; we use the documented click-to-close path.
  await replay.screenshotLightbox.click()
  await expect(replay.screenshotLightbox).not.toBeVisible()
})
