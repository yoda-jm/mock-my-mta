'use strict';

/**
 * The filter syntax help modal, opened via the "?" icon next to the search box.
 * Opening it triggers a GET /api/filters/suggestions call (no term) to populate
 * the table.
 */
class SyntaxHelpModalSection {
  constructor(page) {
    this.page        = page;
    this.locator     = page.locator('#syntaxHelpModal');
    this.openButton  = page.locator('[data-testid="search-syntax-help-button"]');
    this.closeButton = page.locator('[data-testid="syntax-help-modal-close-button"]');
    /** Locator for populated table rows — use in expect() for count assertions. */
    this.tableRows   = page.locator('#syntaxHelpTableBody tr');
  }

  /** Open the modal; waits for the suggestions API response and modal visibility. */
  async open() {
    const resp = this.page.waitForResponse(r => r.url().includes('/api/filters/suggestions'));
    await this.openButton.click();
    await resp;
    await this.locator.waitFor({ state: 'visible' });
  }

  async close() {
    await this.closeButton.click();
  }
}

module.exports = { SyntaxHelpModalSection };
