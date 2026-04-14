'use strict';

class SearchSection {
  constructor(page) {
    this.page = page;
    this.input = page.locator('[data-testid="search-input"]');
    this.submitButton = page.locator('[data-testid="search-submit-button"]');
    this.clearButton = page.locator('[data-testid="search-clear-button"]');
  }

  /** Fill the search box and submit, waiting for the API response. */
  async search(query) {
    const resp = this.page.waitForResponse(r => r.url().includes('/api/emails'));
    await this.input.fill(query);
    await this.submitButton.click();
    await resp;
  }

  /** Clear the search and wait for the full list to reload. */
  async clear() {
    const resp = this.page.waitForResponse(r => r.url().includes('/api/emails'));
    await this.clearButton.click();
    await resp;
  }
}

module.exports = { SearchSection };
