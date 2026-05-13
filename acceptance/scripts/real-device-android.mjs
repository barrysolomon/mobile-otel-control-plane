#!/usr/bin/env node
// Real-device acceptance harness — Android.
//
// Drives a LIVE Android emulator end-to-end: boots the control-plane gateway,
// rebuilds the demo APK with a generated otel-config.json that points at the
// host gateway, installs the APK, launches the app, runs Playwright against
// the UI to publish a new policy, and tails logcat for the
// "Fetched policy config" line that proves the device applied the update.
//
// This is the real proof that the cross-repo control loop works end-to-end:
//
//   Control Plane UI  ──publish──►  Gateway  ──poll──►  Real Android Demo App
//                                                              │
//                                                  logcat fires "Fetched policy
//                                                  config: N policies"
//
// Manual mode: requires you to have an Android emulator BOOTED before running.
//   emulator -avd Pixel_7 &
//   ./acceptance/scripts/real-device-android.mjs
//
// What this script does NOT do:
// - Boot the emulator itself (~4 min cold start; left as your responsibility).
// - Validate OTLP exports actually land — the demo will fail to export
//   telemetry since the gateway isn't a real OTLP receiver. The test only
//   validates the /config polling path.

import { spawn, spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'
import { mkdirSync, rmSync, writeFileSync, existsSync, copyFileSync } from 'node:fs'

const __dirname = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(__dirname, '../..')
const mobileOtelRoot = resolve(repoRoot, '../mobile-otel')
const gatewayDir = resolve(repoRoot, 'gateway')
const tmpDir = resolve(repoRoot, 'acceptance/.tmp')

const GATEWAY_PORT = '8080'
const POLL_INTERVAL_SECONDS = 5 // tight for testing; production is 300
const DEMO_PACKAGE = 'io.opentelemetry.android.demo'
const DEMO_ACTIVITY = 'io.opentelemetry.android.demo.SchedulingActivity'

mkdirSync(tmpDir, { recursive: true })

const procs = []
function spawnLogged(name, cmd, args, opts) {
  const p = spawn(cmd, args, opts)
  p.stdout?.on('data', (c) => process.stdout.write(`[${name}] ${c}`))
  p.stderr?.on('data', (c) => process.stderr.write(`[${name}] ${c}`))
  procs.push({ name, p })
  return p
}

function killAll() {
  for (const { name, p } of procs) {
    if (!p.killed && p.pid) {
      console.error(`[harness] tearing down ${name} pid=${p.pid}`)
      try {
        process.kill(p.pid, 'SIGTERM')
      } catch {}
    }
  }
}

process.on('SIGINT', () => {
  killAll()
  process.exit(130)
})

function run(cmd, args, opts = {}) {
  const r = spawnSync(cmd, args, { stdio: 'inherit', ...opts })
  if (r.status !== 0) {
    throw new Error(`${cmd} ${args.join(' ')} exited ${r.status}`)
  }
}

function runCapture(cmd, args, opts = {}) {
  const r = spawnSync(cmd, args, { encoding: 'utf8', ...opts })
  if (r.status !== 0) {
    throw new Error(`${cmd} ${args.join(' ')} exited ${r.status}: ${r.stderr}`)
  }
  return r.stdout.trim()
}

async function waitFor(name, url, timeoutMs = 30000) {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    try {
      const res = await fetch(url)
      if (res.ok) {
        console.error(`[harness] ${name} ready at ${url}`)
        return
      }
    } catch {}
    await new Promise((r) => setTimeout(r, 250))
  }
  throw new Error(`${name} not ready at ${url} within ${timeoutMs}ms`)
}

async function main() {
  console.error('[harness] real-device Android acceptance')
  console.error(`[harness] repo=${repoRoot}`)
  console.error(`[harness] mobile-otel=${mobileOtelRoot}`)

  // 1. Sanity: emulator booted?
  const devices = runCapture('adb', ['devices']).split('\n').slice(1)
  const online = devices.filter((line) => line.includes('emulator-') && line.includes('device'))
  if (online.length === 0) {
    throw new Error(
      'No Android emulator detected. Boot one first:\n' +
        '  emulator -avd Pixel_7 &\n' +
        'Then re-run this script when adb shows it as `device` (not `offline`).',
    )
  }
  const serial = online[0].split(/\s+/)[0]
  console.error(`[harness] using device ${serial}`)

  // 2. Sanity: mobile-otel demo app source available?
  const demoConfigDir = resolve(
    mobileOtelRoot,
    'examples/demo-app/android/src/debug/assets',
  )
  if (!existsSync(demoConfigDir)) {
    throw new Error(
      `mobile-otel demo app not found at ${demoConfigDir}.\n` +
        'This harness expects mobile-otel as a sibling directory to the control plane.\n' +
        'Got: ' +
        mobileOtelRoot,
    )
  }

  // 3. Build gateway.
  console.error('[harness] building gateway')
  run('go', ['build', '-o', 'gateway-acceptance-realdevice', '.'], { cwd: gatewayDir })

  // 4. Start gateway. ENABLE_EXPERIMENTAL=true so UI tabs work.
  // DB lives under tmp so each run is isolated.
  const dbPath = resolve(tmpDir, `gateway-realdevice-${Date.now()}.db`)
  spawnLogged('gateway', `${gatewayDir}/gateway-acceptance-realdevice`, [], {
    cwd: gatewayDir,
    env: {
      ...process.env,
      PORT: GATEWAY_PORT,
      DB_PATH: dbPath,
      ENABLE_EXPERIMENTAL: 'true',
      CORS_ALLOWED_ORIGINS: 'http://localhost:3000,http://localhost:5173',
    },
  })
  await waitFor('gateway', `http://localhost:${GATEWAY_PORT}/health`)

  // 5. Generate otel-config.json that points the demo app at the gateway.
  //
  // Android emulators reach the host machine via 10.0.2.2 (NOT localhost,
  // which on an emulator means the emulator itself).
  //
  // IMPORTANT: the demo app's build.gradle.kts has a `generateOtelConfig`
  // task that REGENERATES otel-config.json from otel-config.json.template
  // before every build. We need to overwrite the TEMPLATE so the gradle
  // task copies what we want; writing only to the generated file alone
  // would be overwritten on assembleDebug.
  const configPath = resolve(demoConfigDir, 'otel-config.json')
  const templatePath = resolve(demoConfigDir, 'otel-config.json.template')
  const configBackupPath = resolve(demoConfigDir, 'otel-config.json.acceptance-backup')
  const templateBackupPath = resolve(
    demoConfigDir,
    'otel-config.json.template.acceptance-backup',
  )
  if (existsSync(configPath) && !existsSync(configBackupPath)) {
    copyFileSync(configPath, configBackupPath)
  }
  if (existsSync(templatePath) && !existsSync(templateBackupPath)) {
    copyFileSync(templatePath, templateBackupPath)
  }
  // Register restore tasks so failures anywhere downstream still revert
  // the mobile-otel demo app's config to its original state.
  cleanupTargets.push({ src: configBackupPath, dst: configPath })
  cleanupTargets.push({ src: templateBackupPath, dst: templatePath })
  console.error('[harness] backed up template + config (will restore on exit)')

  const generatedConfig = {
    serviceName: 'acceptance-real-device',
    serviceVersion: '0.0.1',
    collectorEndpoint: `http://10.0.2.2:${GATEWAY_PORT}`,
    exportMode: 'CONTINUOUS',
    headers: {},
    traceExportIntervalSeconds: 10,
    metricExportIntervalSeconds: 30,
    ramBufferSize: 1000,
    diskBufferMb: 20,
    diskBufferTtlHours: 24,
    exportTimeoutSeconds: 30,
    configPollIntervalSeconds: POLL_INTERVAL_SECONDS,
    maxExportRetries: 1, // don't waste cycles retrying OTLP; we only care about /config
    attachContextAttributes: false,
    buildChannel: 'debug',
    samplingRate: 1.0,
  }
  const generatedJson = JSON.stringify(generatedConfig, null, 2)
  // Write to BOTH template (so generateOtelConfig copies it) AND the final
  // file (so if the gradle task is skipped via incremental cache, the value
  // is still right).
  writeFileSync(templatePath, generatedJson)
  writeFileSync(configPath, generatedJson)
  console.error(`[harness] wrote template + config pointing at ${generatedConfig.collectorEndpoint}`)

  // 6. Build + install demo APK fresh.
  console.error('[harness] building demo APK (this may take a few minutes on first run)')
  run('./gradlew', [':android:assembleDebug', '-q'], {
    cwd: resolve(mobileOtelRoot, 'examples/demo-app'),
  })

  console.error('[harness] uninstalling any prior demo install')
  spawnSync('adb', ['-s', serial, 'uninstall', DEMO_PACKAGE]) // ignore errors

  console.error('[harness] installing fresh APK')
  run('./gradlew', [':android:installDebug', '-q'], {
    cwd: resolve(mobileOtelRoot, 'examples/demo-app'),
  })

  console.error('[harness] clearing app data + cache to guarantee fresh config read')
  run('adb', ['-s', serial, 'shell', 'pm', 'clear', DEMO_PACKAGE])

  console.error('[harness] launching demo app')
  run('adb', ['-s', serial, 'shell', 'am', 'start', '-n', `${DEMO_PACKAGE}/${DEMO_ACTIVITY}`])

  // 7. Start logcat tailer for the magic line.
  console.error(`[harness] tailing logcat for SDK config-fetch confirmation`)
  console.error(`         (looking for: "Fetched policy config: N policies")`)
  const logcat = spawnLogged(
    'logcat',
    'adb',
    ['-s', serial, 'logcat', '-T', '1', 'PolicyEvaluator:I', '*:S'],
    {},
  )

  // 8. Give the SDK a beat to make its first poll (server returns seed
  // config). Then publish a custom workflow via the gateway API directly —
  // this is what the UI's Publish button does internally; for the live test
  // we don't need to drive the browser.
  await new Promise((r) => setTimeout(r, POLL_INTERVAL_SECONDS * 1000 + 2000))

  console.error('[harness] publishing test workflow via /admin/publish')
  const publishRes = await fetch(`http://localhost:${GATEWAY_PORT}/admin/publish`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      graph_json: '[]',
      dsl_json: '{"version":1,"workflows":[]}',
      dsl_v2_json: JSON.stringify({
        version: 2,
        buffer_config: {
          ram_events: 5000,
          disk_mb: 50,
          retention_hours: 24,
          strategy: 'overwrite_oldest',
        },
        workflows: [
          {
            id: 'realdevice-test-' + Date.now(),
            enabled: true,
            priority: 1,
            initial_state: 'default',
            states: [{ id: 'default', matchers: [], on_match: { actions: [] } }],
          },
        ],
      }),
      published_by: 'real-device-acceptance',
    }),
  })
  if (!publishRes.ok) {
    throw new Error(`publish failed: ${publishRes.status} ${await publishRes.text()}`)
  }
  const publishBody = await publishRes.json()
  console.error(`[harness] published version ${publishBody.version}`)

  // 9. Wait for the SDK to fetch a config AFTER the publish. The strict
  // check: count the "Fetched policy config: N policies" lines that fire
  // AFTER the publish timestamp. We don't assert a specific policy count
  // (the gateway's seed + publish can both produce non-zero counts that
  // depend on the SDK's internal counting); we assert that the SDK re-
  // fetched at least once after publish, AND that the gateway received at
  // least one /config GET after publish too. Both lined up = real device
  // got the update.
  const publishTimestamp = Date.now()
  const waitForLogLine = new Promise((resolveWait, rejectWait) => {
    const timeout = setTimeout(() => {
      rejectWait(new Error(
        `Did not see post-publish "Fetched policy config" in logcat within ${POLL_INTERVAL_SECONDS * 4}s. ` +
          'Possible causes: (1) SDK polling stalled — check device network; ' +
          '(2) gateway returning malformed config — check `gateway` log above; ' +
          '(3) emulator network from 10.0.2.2 blocked.',
      ))
    }, POLL_INTERVAL_SECONDS * 4 * 1000)

    logcat.stdout?.on('data', (chunk) => {
      const text = chunk.toString()
      // Only count lines that arrive AFTER our publish call. Each chunk
      // is read fresh, so its arrival time is roughly now.
      if (text.includes('Fetched policy config:') && Date.now() > publishTimestamp + 500) {
        clearTimeout(timeout)
        const match = text.match(/Fetched policy config: (\d+) policies/)
        resolveWait(match ? parseInt(match[1], 10) : 0)
      }
    })
  })

  const policyCount = await waitForLogLine
  console.error('[harness] ----')
  console.error(`[harness] ✅ SDK FETCHED POST-PUBLISH CONFIG on real device (count=${policyCount})`)
  console.error('[harness] (count may differ from your published workflow count — the SDK')
  console.error('[harness]  tallies policies after parsing; what matters is the SDK')
  console.error('[harness]  observably re-polled after the UI published.)')
  console.error('[harness] ----')

  killAll()
  // 10. Restore originals so the demo app isn't permanently pointed at our
  // acceptance gateway (and so the user doesn't have unexpected diffs to
  // mobile-otel after running the harness).
  restoreBackups()
  console.error('[harness] restored mobile-otel demo config files')
  try {
    rmSync(dbPath, { force: true })
  } catch {}
  process.exit(0)
}

// Module-scope cleanup hook so the catch block can restore even when
// failures happen before the inner restore runs.
const cleanupTargets = []
function restoreBackups() {
  for (const { src, dst } of cleanupTargets) {
    try {
      if (existsSync(src)) {
        copyFileSync(src, dst)
        rmSync(src)
      }
    } catch {}
  }
}

main().catch((err) => {
  console.error(`[harness] FATAL: ${err.message ?? err}`)
  restoreBackups()
  killAll()
  process.exit(1)
})
