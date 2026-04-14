'use strict';

class BodyVersionsSection {
  constructor(page) {
    this.page = page;
  }

  // ── Locators ─────────────────────────────────────────────────────────────

  /** Locator for all body-version tabs. */
  tabs() {
    return this.page.locator('[data-testid^="email-body-version-tab-"]');
  }

  /** Locator for a specific version tab: 'html', 'plain-text', 'raw', 'watch-html'. */
  tab(version) {
    return this.page.locator(`[data-testid="email-body-version-tab-${version}"]`);
  }

  // ── Actions ──────────────────────────────────────────────────────────────

  /**
   * Find the first non-active tab and click it.
   * The active tab is rendered bold (font-weight 700) with no click handler;
   * clicking it would not trigger a body reload.
   * Returns the tab locator that was clicked, or null if all tabs are active.
   */
  async clickNonActiveTab() {
    const allTabs = this.tabs();
    const count   = await allTabs.count();
    for (let i = 0; i < count; i++) {
      const tab        = allTabs.nth(i);
      const fontWeight = await tab.evaluate(el => window.getComputedStyle(el).fontWeight);
      if (fontWeight !== '700') {
        const resp = this.page.waitForResponse(r => r.url().includes('/body/'));
        await tab.click();
        await resp;
        return tab;
      }
    }
    return null;
  }

  /** Switch to a specific body version by name; waits for the body API call. */
  async switchTo(version) {
    const t    = this.tab(version);
    const resp = this.page.waitForResponse(r => r.url().includes('/body/'));
    await t.click();
    await resp;
    return t;
  }
}

module.exports = { BodyVersionsSection };
