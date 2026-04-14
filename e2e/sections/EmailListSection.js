'use strict';

const { EmailRowSection }  = require('./EmailRowSection');
const { PaginationSection } = require('./PaginationSection');

class EmailListSection {
  constructor(page) {
    this.page = page;
    this.tbody         = page.locator('[data-testid="email-list-body"]');
    this.deleteAllButton = page.locator('[data-testid="delete-all-button"]');
    this.refreshButton = page.locator('[data-testid="refresh-button"]');
    this.pagination    = new PaginationSection(page);
  }

  // ── Locators (use directly in expect() calls) ────────────────────────────

  /** Locator for all email rows — use in expect() for count / visibility. */
  rows() {
    return this.tbody.locator('[data-testid^="email-row-"]');
  }

  /** Locator for the "No emails found" empty-state message. */
  emptyMessage() {
    return this.tbody.getByText('No emails found');
  }

  // ── Section factories ────────────────────────────────────────────────────

  /** Return the section for the nth email row (0-based). */
  row(index) {
    return new EmailRowSection(this.page, this.rows().nth(index));
  }

  firstRow() {
    return new EmailRowSection(this.page, this.rows().first());
  }

  // ── Actions ──────────────────────────────────────────────────────────────

  async count() {
    return this.rows().count();
  }

  /** Delete all emails; waits for the DELETE response. */
  async deleteAll() {
    const deleteResp = this.page.waitForResponse(
      r => r.url().includes('/api/emails') && r.request().method() === 'DELETE'
    );
    await this.deleteAllButton.click();
    await deleteResp;
  }

  /** Manually refresh the email list. */
  async refresh() {
    const resp = this.page.waitForResponse(r => r.url().includes('/api/emails'));
    await this.refreshButton.click();
    await resp;
  }
}

module.exports = { EmailListSection };
