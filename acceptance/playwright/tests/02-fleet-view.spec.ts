// Acceptance Test 2: Fleet view shows simulated devices.
//
// Scenario:
//   1. Three simulated devices send heartbeats (auto-registers them)
//   2. Gateway directly confirms registration (sanity check)
//   3. User opens the Devices tab
//   4. UI fetch lands; all three deviceIds appear in the fleet table
//
// This validates the device-fleet observability path: heartbeat → DB →
// /v1/devices → UI table. Operators rely on this for "is the fleet
// healthy" answers; broken means broken trust.

import { test, expect } from '../fixtures/test'
import { AppPage } from '../pages/app'
import { createSimulatedDevice } from '../../simulator/device.ts'

test('fleet view shows simulated devices', async ({ page, gatewayUrl }) => {
  // Three devices with deterministic-but-test-unique IDs.
  const ts = Date.now()
  const ids = [`fleet-${ts}-a`, `fleet-${ts}-b`, `fleet-${ts}-c`]
  for (const id of ids) {
    const d = createSimulatedDevice({ gatewayUrl, deviceId: id, appId: 'acceptance-app' })
    await d.heartbeat({ bufferUsageMB: 1.2 })
  }

  // Sanity: gateway directly returns our devices before we touch UI. This
  // isolates "DB doesn't have them" from "UI doesn't show them" if the
  // visual assertion fails later.
  const directRes = await fetch(`${gatewayUrl}/v1/devices?limit=100`)
  const directBody = (await directRes.json()) as { devices: Array<{ device_id: string }> }
  const seenIds = directBody.devices.map((d) => d.device_id)
  for (const id of ids) {
    expect(seenIds, `gateway should have device ${id}; got: ${seenIds.join(',')}`).toContain(id)
  }

  // Navigate to Devices tab. The DeviceFleet useEffect fires fetchDevices()
  // on mount; explicitly wait for the network response to land before
  // asserting the table contents — otherwise the test races React's render
  // cycle.
  const app = new AppPage(page)
  await app.open()

  const devicesResponse = page.waitForResponse(
    (res) => res.url().includes('/v1/devices') && res.request().method() === 'GET',
    { timeout: 10_000 },
  )
  await app.devicesTab.click()
  await devicesResponse

  for (const id of ids) {
    await expect(
      page.getByText(id, { exact: false }),
      `device ${id} should appear in fleet table`,
    ).toBeVisible({ timeout: 5_000 })
  }
})
