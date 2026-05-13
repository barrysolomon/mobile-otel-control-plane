import { type Page, type Locator } from '@playwright/test'

/**
 * Journey Replay tab. Two input paths:
 *   - by trace_id (gateway proxies to Dash0; needs auth env vars)
 *   - paste raw OTLP/JSON envelope into the textarea (auto-parses on change)
 *
 * Acceptance tests use the paste-textarea path so they don't depend on
 * Dash0 credentials.
 */
export class JourneyReplayPage {
  constructor(public readonly page: Page) {}

  get pasteTextarea(): Locator {
    // The textarea has a multi-line placeholder starting with "Paste OTLP/JSON".
    return this.page.getByPlaceholder(/Paste OTLP/i)
  }

  /** Row representing a parsed event (one per record in the OTLP envelope). */
  eventRowByBody(body: string): Locator {
    return this.page.getByText(body, { exact: false }).first()
  }

  /** Screenshot thumbnail inside the strip (img element with the data URL). */
  get screenshotImage(): Locator {
    return this.page.locator('img[src^="data:image"]').first()
  }

  /** Lightbox dialog opens when a screenshot is clicked. */
  get screenshotLightbox(): Locator {
    return this.page.getByLabel('Full-size screenshot')
  }

  /**
   * Paste OTLP JSON into the textarea. The component auto-parses on change —
   * there is no "Parse" button. After fill(), give the parser one tick.
   */
  async pasteAndParse(otlpJson: string): Promise<void> {
    await this.pasteTextarea.fill(otlpJson)
    await this.page.waitForTimeout(100)
  }
}
