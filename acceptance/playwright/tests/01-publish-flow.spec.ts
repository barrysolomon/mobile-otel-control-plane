// Acceptance Test 1: build & publish a policy.
//
// Scenario:
//   1. UI loads, default workflow ("UI Freeze Handler") is in the editor
//   2. User clicks Publish
//   3. Gateway records a new version
//   4. Simulated device polls /config?dsl_version=2 and receives the workflow
//
// This is the load-bearing v1 control loop: UI → /admin/publish → /config.
// If this breaks, devices stop receiving new policies. The matching
// gateway-side contract test is publish_loop_test.go; this test adds the UI
// half (a real browser actually clicks Publish via the UI's gatewayAPI).

import { test, expect } from '../fixtures/test'
import { AppPage } from '../pages/app'

test('build & publish a policy — simulated device sees the new workflow', async ({
  page,
  device,
  gatewayUrl,
}) => {
  const app = new AppPage(page)
  await app.open()

  // Baseline: what does the device see before publish?
  // (Gateway seeds a default config with one workflow, so this may return
  // workflows.length === 1; we just record it to compare after publish.)
  const before = await device.fetchConfig()
  const versionBefore = await getActiveVersion(gatewayUrl)

  // Click Publish. The UI compiles the default workflow (defaultWorkflow in
  // App.tsx, id="ui-freeze") to DSL v1 + v2 and POSTs to /admin/publish.
  await app.publishButton.click()

  // Success banner — note that the UI message currently includes "Published
  // version N successfully!", but we keep the assertion loose so a copy
  // change doesn't break the test.
  await expect(app.messageBanner).toContainText(/published/i, { timeout: 10_000 })

  // Verify gateway moved the active version forward.
  const versionAfter = await getActiveVersion(gatewayUrl)
  expect(versionAfter).toBeGreaterThan(versionBefore)

  // Verify the simulated device sees the published workflow.
  const after = await device.fetchConfig()
  expect(after.workflows.length).toBeGreaterThan(0)

  const hasUiFreeze = after.workflows.some((w) => w.id === 'ui-freeze')
  expect(hasUiFreeze).toBe(true)

  // Sanity: the published config should NOT equal the seed (workflow ID
  // differs from the seed's auto-generated default).
  const beforeIds = new Set(before.workflows.map((w) => w.id))
  const afterIds = new Set(after.workflows.map((w) => w.id))
  expect(afterIds).not.toEqual(beforeIds)
})

/**
 * Read the active config version directly from the gateway. Used to assert
 * that publish moved the version forward without depending on UI display.
 */
async function getActiveVersion(gatewayUrl: string): Promise<number> {
  const res = await fetch(`${gatewayUrl}/admin/versions?limit=1`)
  if (!res.ok) return 0
  const body = (await res.json()) as { versions?: Array<{ version: number }> }
  return body.versions?.[0]?.version ?? 0
}
