// Vitest acceptance: when VITE_ENABLE_EXPERIMENTAL is NOT set to "true",
// the App must NOT render the Configuration or Journey Replay tabs.
//
// This is the OFF-case companion to Playwright test 05 (which validates the
// ON case live in a browser). We test it here in jsdom because:
// 1. Setting Vite env vars per-Playwright-test would need separate UI builds.
// 2. The logic is a plain `import.meta.env` boolean read in App.tsx — no
//    network or DOM rendering nuance.

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'

// Mock the gateway API: App.tsx fires `gatewayAPI.listWorkflows()` on mount.
// We don't want the test to make HTTP requests.
vi.mock('../api/gateway', () => ({
  gatewayAPI: {
    listWorkflows: vi.fn().mockResolvedValue([]),
    createWorkflow: vi.fn(),
    updateWorkflow: vi.fn(),
    deleteWorkflow: vi.fn(),
    publish: vi.fn(),
    rollback: vi.fn(),
    listVersions: vi.fn().mockResolvedValue([]),
  },
}))

describe('App: experimental gate (OFF)', () => {
  beforeEach(() => {
    // Default — no env var set. Vitest's import.meta.env reflects the
    // build-time env at test runtime.
    vi.stubEnv('VITE_ENABLE_EXPERIMENTAL', '')
  })

  it('hides Configuration and Journey Replay tabs when flag is unset', async () => {
    const { App } = await import('../App')
    render(<App />)

    // Core tabs always visible
    expect(screen.getByRole('button', { name: /Workflow Builder/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Devices/i })).toBeInTheDocument()

    // Experimental tabs hidden
    expect(screen.queryByRole('button', { name: /^Configuration/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^Journey Replay/i })).not.toBeInTheDocument()
  })

  it('hides Configuration and Journey Replay tabs when flag is "false"', async () => {
    vi.stubEnv('VITE_ENABLE_EXPERIMENTAL', 'false')
    // Re-import to pick up the new env. Vitest's module cache means the
    // first import already evaluated `import.meta.env` once.
    vi.resetModules()
    const { App } = await import('../App')
    render(<App />)
    expect(screen.queryByRole('button', { name: /^Configuration/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^Journey Replay/i })).not.toBeInTheDocument()
  })
})

describe('App: experimental gate (ON)', () => {
  it('shows Configuration and Journey Replay tabs when flag is "true"', async () => {
    vi.stubEnv('VITE_ENABLE_EXPERIMENTAL', 'true')
    vi.resetModules()
    const { App } = await import('../App')
    render(<App />)

    expect(screen.getByRole('button', { name: /^Configuration/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^Journey Replay/i })).toBeInTheDocument()
  })
})
