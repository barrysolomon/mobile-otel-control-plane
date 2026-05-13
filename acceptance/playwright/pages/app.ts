// App-level page object. Encapsulates the top-level tab navigation and
// global UI affordances (publish button, message banner).
//
// Page objects are intentionally THIN: they expose selectors as
// computed-on-demand `Locator`s, not as resolved strings, so timing/staleness
// is handled by Playwright. Methods on the page object are user actions
// expressed in test-vocabulary, not framework internals.

import { type Page, type Locator } from '@playwright/test'

export class AppPage {
  constructor(public readonly page: Page) {}

  // ─── Top-level navigation ─────────────────────────────────────────────

  /** Workflow Builder tab — always visible. */
  get workflowBuilderTab(): Locator {
    return this.page.getByRole('button', { name: 'Workflow Builder', exact: false })
  }

  /** Devices tab — always visible. */
  get devicesTab(): Locator {
    return this.page.getByRole('button', { name: 'Devices', exact: false })
  }

  /** Configuration tab — only visible when VITE_ENABLE_EXPERIMENTAL=true. */
  get configurationTab(): Locator {
    return this.page.getByRole('button', { name: /^Configuration/, exact: false })
  }

  /** Journey Replay tab — only visible when VITE_ENABLE_EXPERIMENTAL=true. */
  get journeyReplayTab(): Locator {
    return this.page.getByRole('button', { name: /^Journey Replay/, exact: false })
  }

  // ─── Global affordances ───────────────────────────────────────────────

  /** Publish button — bottom of WorkflowBuilder. */
  get publishButton(): Locator {
    return this.page.getByRole('button', { name: /publish/i, exact: false })
  }

  /** The header h1 — useful for an initial-load sanity check. */
  get appTitle(): Locator {
    return this.page.getByRole('heading', { name: /Mobile Observability/i })
  }

  /** Success or error toast / banner. */
  get messageBanner(): Locator {
    return this.page.locator('.message')
  }

  // ─── Actions ──────────────────────────────────────────────────────────

  async open(): Promise<void> {
    await this.page.goto('/')
    await this.appTitle.waitFor({ state: 'visible' })
  }
}
