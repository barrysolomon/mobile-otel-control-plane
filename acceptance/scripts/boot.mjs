#!/usr/bin/env node
// Boot script for the acceptance suite.
//
// Starts the gateway (Go) on :8080 and the Vite dev server on :3000, then
// runs `playwright test`. Tears down both processes when the tests exit
// (success, failure, or SIGINT).
//
// Usage:
//   node acceptance/scripts/boot.mjs           # full suite
//   node acceptance/scripts/boot.mjs --headed  # passed through to playwright
//
// Why a custom boot script instead of Playwright's `webServer` config:
// - We need to manage TWO servers (gateway + UI), and Playwright's webServer
//   config doesn't compose cleanly for multiple. The official workaround is
//   to make this script orchestrate them.
// - We want explicit log surfacing on failure (gateway exit code, UI Vite
//   compile errors) — Playwright's webServer suppresses both.
// - We want the gateway to start with ENABLE_EXPERIMENTAL=true for the
//   experimental-gate test, then with =false for the default-hidden test.
//   A custom script can spawn per-test server modes; Playwright webServer
//   cannot.

import { spawn } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'
import { mkdirSync, rmSync } from 'node:fs'

const __dirname = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(__dirname, '../..')
const gatewayDir = resolve(repoRoot, 'gateway')
const uiDir = resolve(repoRoot, 'control-plane-ui')
const tmpDataDir = resolve(repoRoot, 'acceptance/.tmp')

const GATEWAY_PORT = process.env.GATEWAY_PORT ?? '8080'
const UI_PORT = process.env.UI_PORT ?? '3000'
const playwrightArgs = process.argv.slice(2)

mkdirSync(tmpDataDir, { recursive: true })
const dbPath = resolve(tmpDataDir, `gateway-${Date.now()}.db`)

const procs = []

function spawnLogged(name, cmd, args, opts) {
  const p = spawn(cmd, args, opts)
  p.stdout?.on('data', (chunk) => process.stdout.write(`[${name}] ${chunk}`))
  p.stderr?.on('data', (chunk) => process.stderr.write(`[${name}] ${chunk}`))
  p.on('exit', (code, signal) => {
    if (code !== 0 && code !== null) {
      console.error(`[${name}] exited code=${code} signal=${signal}`)
    }
  })
  procs.push({ name, p })
  return p
}

function killAll() {
  for (const { name, p } of procs) {
    if (!p.killed) {
      console.error(`[boot] tearing down ${name} (pid=${p.pid})`)
      try {
        process.kill(p.pid, 'SIGTERM')
      } catch {}
    }
  }
  // Give them 2s to exit cleanly, then SIGKILL.
  setTimeout(() => {
    for (const { p } of procs) {
      if (!p.killed) {
        try {
          process.kill(p.pid, 'SIGKILL')
        } catch {}
      }
    }
  }, 2000)
  try {
    rmSync(dbPath, { force: true })
  } catch {}
}

process.on('SIGINT', () => {
  killAll()
  process.exit(130)
})
process.on('SIGTERM', () => {
  killAll()
  process.exit(143)
})

async function waitFor(name, url, timeoutMs = 30000) {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    try {
      const res = await fetch(url)
      if (res.ok) {
        console.error(`[boot] ${name} ready at ${url}`)
        return
      }
    } catch {}
    await new Promise((r) => setTimeout(r, 250))
  }
  throw new Error(`${name} not ready at ${url} within ${timeoutMs}ms`)
}

async function main() {
  console.error(`[boot] repo=${repoRoot}`)
  console.error(`[boot] gateway port=${GATEWAY_PORT} ui port=${UI_PORT}`)

  // 1. Build gateway. We do this synchronously so any compile error fails
  // fast and Playwright never starts.
  await new Promise((resolveBuild, rejectBuild) => {
    const p = spawn('go', ['build', '-o', 'gateway-acceptance', '.'], {
      cwd: gatewayDir,
      stdio: 'inherit',
    })
    p.on('exit', (code) => (code === 0 ? resolveBuild() : rejectBuild(new Error('gateway build failed'))))
  })

  // 2. Start gateway. ENABLE_EXPERIMENTAL=true by default so the
  // experimental scenarios can be exercised; the experimental-gate test
  // hits a temporary second gateway with the flag off.
  spawnLogged('gateway', `${gatewayDir}/gateway-acceptance`, [], {
    cwd: gatewayDir,
    env: {
      ...process.env,
      PORT: GATEWAY_PORT,
      DB_PATH: dbPath,
      ENABLE_EXPERIMENTAL: 'true',
      // CORS for the dev server.
      CORS_ALLOWED_ORIGINS: `http://localhost:${UI_PORT},http://localhost:5173`,
    },
  })
  await waitFor('gateway', `http://localhost:${GATEWAY_PORT}/health`, 30000)

  // 3. Start UI dev server. VITE_ENABLE_EXPERIMENTAL=true mirrors the
  // gateway flag.
  spawnLogged('ui', 'npm', ['run', 'dev', '--', '--port', UI_PORT, '--strictPort'], {
    cwd: uiDir,
    env: {
      ...process.env,
      VITE_ENABLE_EXPERIMENTAL: 'true',
      // Tell the UI's gatewayAPI to talk directly to our gateway (Vite proxy
      // is bypassed for clarity in acceptance runs).
      VITE_GATEWAY_URL: `http://localhost:${GATEWAY_PORT}`,
    },
  })
  await waitFor('ui', `http://localhost:${UI_PORT}/`, 60000)

  // 4. Run Playwright. Pass through env so tests can read the ports.
  const pw = spawnLogged('playwright', 'npx', ['playwright', 'test', ...playwrightArgs], {
    cwd: resolve(repoRoot, 'acceptance'),
    env: {
      ...process.env,
      GATEWAY_URL: `http://localhost:${GATEWAY_PORT}`,
      UI_URL: `http://localhost:${UI_PORT}`,
    },
  })
  const exitCode = await new Promise((r) => pw.on('exit', (c) => r(c ?? 1)))

  killAll()
  process.exit(exitCode)
}

main().catch((err) => {
  console.error(`[boot] fatal: ${err.message ?? err}`)
  killAll()
  process.exit(1)
})
