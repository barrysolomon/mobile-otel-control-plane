// Acceptance Test 3: rollback flips the active config back.
//
// Scenario:
//   1. Publish A → device sees A
//   2. Publish B → device sees B
//   3. Click Rollback to version A
//   4. Device polls again → sees A again
//
// We exercise rollback at the gateway-API level (POST /admin/rollback)
// rather than clicking a UI button — the UI does have a version list with
// rollback buttons in App.tsx's handleRollback, but the UI's rendering of
// the version list is dependent on initial state that's awkward to seed in
// an isolated test. The gateway side is what the SDK actually depends on;
// driving via API still validates the user-facing contract.

import { test, expect } from '../fixtures/test'

test('rollback flips the active config back', async ({ device, gatewayUrl }) => {
  // Publish A
  const versionA = await publish(gatewayUrl, 'rollback-test-A')
  await expectActiveWorkflow(device, 'rollback-test-A')

  // Publish B (now active)
  const versionB = await publish(gatewayUrl, 'rollback-test-B')
  expect(versionB).toBeGreaterThan(versionA)
  await expectActiveWorkflow(device, 'rollback-test-B')

  // Rollback to A
  const rb = await fetch(`${gatewayUrl}/admin/rollback`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ version: versionA }),
  })
  expect(rb.ok, `rollback to ${versionA}: ${rb.status}`).toBe(true)

  // Device polls again → A
  await expectActiveWorkflow(device, 'rollback-test-A')
})

async function publish(gatewayUrl: string, workflowId: string): Promise<number> {
  const dslV2 = JSON.stringify({
    version: 2,
    buffer_config: { ram_events: 5000, disk_mb: 50, retention_hours: 24, strategy: 'overwrite_oldest' },
    workflows: [
      {
        id: workflowId,
        enabled: true,
        priority: 1,
        initial_state: 'default',
        states: [
          { id: 'default', matchers: [], on_match: { actions: [] } },
        ],
      },
    ],
  })
  const res = await fetch(`${gatewayUrl}/admin/publish`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      graph_json: '[]',
      dsl_json: '{"version":1,"workflows":[]}',
      dsl_v2_json: dslV2,
      published_by: 'acceptance-test',
    }),
  })
  expect(res.ok, `publish ${workflowId}: ${res.status}`).toBe(true)
  const body = (await res.json()) as { version: number }
  return body.version
}

async function expectActiveWorkflow(
  device: { fetchConfig: () => Promise<{ workflows: Array<{ id: string }> }> },
  expectedId: string,
): Promise<void> {
  const cfg = await device.fetchConfig()
  const ids = cfg.workflows.map((w) => w.id)
  expect(ids, `device should see active workflow ${expectedId}, got: ${ids.join(',')}`).toContain(
    expectedId,
  )
}
