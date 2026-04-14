'use strict';

class PaginationSection {
  constructor(page) {
    this.page = page;
    /** Locators exposed for direct assertions in tests */
    this.pageStartEl   = page.locator('#page-start');
    this.pageTotalEl   = page.locator('#page-total');
    this.totalMatchesEl = page.locator('#total-matches');
    this.prevButton    = page.locator('[data-testid="prev-page-button"]');
    this.nextButton    = page.locator('[data-testid="next-page-button"]');
  }

  async currentPage()  { return parseInt(await this.pageStartEl.textContent(), 10); }
  async totalPages()   { return parseInt(await this.pageTotalEl.textContent(), 10); }
  async totalEmails()  { return parseInt(await this.totalMatchesEl.textContent(), 10); }

  async nextPage() {
    const resp = this.page.waitForResponse(r => r.url().includes('/api/emails'));
    await this.nextButton.click();
    await resp;
  }

  async prevPage() {
    const resp = this.page.waitForResponse(r => r.url().includes('/api/emails'));
    await this.prevButton.click();
    await resp;
  }

  /** Navigate forward until the last page is reached. */
  async goToLastPage() {
    const total   = await this.totalPages();
    const current = await this.currentPage();
    for (let i = current; i < total; i++) {
      await this.nextPage();
    }
  }
}

module.exports = { PaginationSection };
