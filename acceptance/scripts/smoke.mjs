#!/usr/bin/env node
// Smoke check: does the gateway respond to the 3 SDK endpoints?
//
// Requires a gateway already running on localhost:8080. Use this when
// something looks off and you want to bisect: "is the gateway broken, or
// is the suite wrong?" Independent of the TypeScript simulator + Playwright
// — pure node + fetch.

const GATEWAY_URL = process.env.GATEWAY_URL ?? 'http://localhost:8080'

async function req(name, url, opts) {
  const res = await fetch(url, opts)
  const body = await res.text()
  if (!res.ok) {
    throw new Error(`${name}: ${res.status} ${res.statusText}: ${body}`)
  }
  return body
}

async function main() {
  console.log(`smoke against ${GATEWAY_URL}`)

  console.log('1. GET /health')
  await req('health', `${GATEWAY_URL}/health`)
  console.log('   ok')

  console.log('2. POST /status (auto-registers smoke device)')
  await req('status', `${GATEWAY_URL}/status`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      device_id: 'smoke-' + Date.now(),
      app_id: 'smoke',
      session_id: 'smoke-s1',
      buffer_usage_mb: 0.5,
      last_triggers: [],
      config_version: 0,
    }),
  })
  console.log('   ok')

  console.log('3. GET /config?dsl_version=2')
  const cfg = await req('config', `${GATEWAY_URL}/config?app_id=smoke&device_id=smoke-1&dsl_version=2`)
  const parsed = JSON.parse(cfg)
  console.log(`   version=${parsed.version} workflows=${parsed.workflows?.length ?? 0}`)

  console.log('4. POST /ingest (1 event)')
  await req('ingest', `${GATEWAY_URL}/ingest`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      events: [
        {
          event_name: 'smoke.ping',
          session_id: 'smoke-s1',
          device_id: 'smoke-1',
          config_version: 0,
          timestamp: Date.now(),
          attributes: {},
        },
      ],
    }),
  })
  console.log('   ok')

  console.log('\nsmoke passed')
}

main().catch((err) => {
  console.error('smoke failed:', err.message ?? err)
  process.exit(1)
})
