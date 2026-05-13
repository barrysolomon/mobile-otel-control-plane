// Shared test fixture. Extends Playwright's base test with:
// - `device`: a fresh simulated SDK device per test (unique deviceId).
// - `gatewayUrl`: pulled from env, helpful for direct gateway assertions.
//
// Why per-test devices instead of one shared one: tests need to assert on
// "this specific device sees the published config" without interference
// from other tests' polls.

import { test as base } from '@playwright/test'
import { createSimulatedDevice, type SimulatedDevice } from '../../simulator/device.ts'

type Fixtures = {
  device: SimulatedDevice
  gatewayUrl: string
}

export const test = base.extend<Fixtures>({
  gatewayUrl: async ({}, use) => {
    const url = process.env.GATEWAY_URL ?? 'http://localhost:8080'
    await use(url)
  },

  device: async ({ gatewayUrl }, use, testInfo) => {
    const deviceId = `acceptance-${testInfo.testId.slice(0, 8)}-${Date.now()}`
    const device = createSimulatedDevice({
      gatewayUrl,
      deviceId,
      appId: 'acceptance-app',
    })
    await use(device)
  },
})

export { expect } from '@playwright/test'
